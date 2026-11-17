<script setup lang="ts">
import { ref, reactive, computed, watch, onBeforeUnmount, onMounted, nextTick } from "vue";
import { X, Plus, Maximize, Minimize, ArrowLeft, Unplug } from "lucide-vue-next";
import Desktop from "../Desktop.vue";
import CertificateDialog from "./CertificateDialog.vue";
import BaseModal from "./BaseModal.vue";
import ConnectionProgress, {
  type ProgressState,
} from "./ConnectionProgress.vue";
import { api, ApiError } from "../api";
import {
  performanceDisplay,
  loadRenderMode,
  loadWallpaper,
  type RenderMode,
  loadDisplayScale,
  loadResolutionMode,
} from "../display";
import { rawConnection, refreshWorkspace, state } from "../stores/app";
import { closeRemote, ui, toast, requestConfirm } from "../stores/ui";
import type { CertificateInspection } from "../certificates";
import "../assets/native-desktop.css";

import { localSessionFor, sessionNeedsReconnect } from "../local-sessions";

type Flow = ProgressState & { inspection?: CertificateInspection };
const sessions = ref<
  {
    id: string;
    name: string;
    connectionId: string;
    protocol: string;
    targetOs: string;
    clipboard: boolean;
    renderMode: RenderMode;
    wallpaper: boolean;
    status: string;
    progress: Flow;
  }[]
>([]);
watch(() => sessions.value.map(({ id, connectionId, status }) => ({ id, connectionId, status })),
  (items) => { ui.remote.sessions = items; }, { immediate: true, flush: "sync" });
const active = ref(""),
  busy = ref(false),
  error = ref(""),
  host = ref<HTMLElement>();
const workspace = ref<HTMLElement>();
const isFullscreen = ref(false);
const ending = ref(false);
function sessionStatus(id: string, status: string) {
  const session = sessions.value.find(s => s.id === id);
  if (session) session.status = status;
}
function syncFullscreen() { isFullscreen.value = document.fullscreenElement === workspace.value; }
async function exitFullscreen() {
  if (document.fullscreenElement === workspace.value) await document.exitFullscreen();
}
async function toggleFullscreen() {
  try {
    if (isFullscreen.value) await exitFullscreen();
    else await workspace.value?.requestFullscreen();
  } catch { toast("浏览器未允许全屏，请在独立标签页重试", { tone: "danger" }); }
}
onMounted(() => {
  document.addEventListener("fullscreenchange", syncFullscreen);
});
const pendingFlow = ref<Flow | null>(null);
const visibleProgress = computed(
  () =>
    pendingFlow.value ||
    sessions.value.find((s) => s.id === active.value)?.progress ||
    null,
);
const certificate = ref<{
  connection: any;
  initial?: CertificateInspection;
  forConnection: boolean;
  flow: Flow;
} | null>(null);
const loginOpen = ref(false),
  pending = ref<{
    connection: any;
    check: string;
    expiresAt: string;
    flow: Flow;
  } | null>(null);
const login = reactive({ username: "", password: "", domain: "", remember: false });
let alive = true,
  generation = 0;
function valid(flow: Flow) {
  return alive && flow.token === generation && flow.mode !== "cancelled";
}
function fail(e: unknown, flow?: Flow) {
  const message = (e as Error).message || String(e);
  if (flow && !valid(flow)) return;
  error.value = message;
  if (flow) {
    flow.mode = "failed";
    flow.error = message;
  }
  toast(message, { tone: "danger" });
}
function clearPending() {
  generation++;
  pendingFlow.value = null;
  certificate.value = null;
  pending.value = null;
  loginOpen.value = false;
  login.password = "";
  busy.value = false;
}
async function start(
  c: any,
  check: string,
  flow: Flow,
  ephemeralCredentialRef = "",
) {
  if (!valid(flow)) return;
  flow.step = 2;
  flow.mode = "running";
  flow.error = "";
  const renderMode = loadRenderMode(c.protocol);
  const wallpaper = loadWallpaper();
  const s = await api("/sessions", "POST", {
    renderMode,
    ...(c.protocol === "rdp" ? { wallpaper } : {}),
    connectionId: c.id,
    certificateCheck: check,
    ephemeralCredentialRef,
    display:
      performanceDisplay(
        host.value?.clientWidth || innerWidth,
        host.value?.clientHeight || innerHeight - 44,
        loadDisplayScale(c.protocol),
        loadResolutionMode(c.protocol),
        renderMode,
      ),
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  });
  if (!valid(flow)) {
    await api("/sessions/" + s.id, "DELETE").catch(() => {});
    return;
  }
  sessions.value.push({
    id: s.id,
    renderMode,
    wallpaper,
    status: "连接中",
    name: c.name,
    connectionId: c.id,
    protocol: c.protocol,
    targetOs: c.targetOs,
    clipboard: c.settings.clipboard === "text",
    progress: flow,
  });
  active.value = s.id;
  pendingFlow.value = null;
  loginOpen.value = false;
  login.password = "";
  pending.value = null;
  (document.activeElement as HTMLElement)?.blur();
  refreshWorkspace().catch((e) =>
    toast((e as Error).message, { tone: "danger" }),
  );
}
async function authorized(
  c: any,
  check: string,
  expiresAt: string,
  flow: Flow,
) {
  if (!valid(flow)) return;
  flow.step = 1;
  flow.mode = "running";
  pending.value = { connection: c, check, expiresAt, flow };
  if (!c.credentialId) {
    Object.assign(login, { username: "", password: "", domain: "", remember: false });
    flow.mode = "login";
    loginOpen.value = true;
    return;
  }
  await start(c, check, flow);
}
async function prepareConnection(c: any, flow: Flow) {
  if (c.protocol === "rdp") {
    flow.step = 0;
    const inspect = () => api<CertificateInspection>(
      "/connections/" + c.id + "/certificate", "POST", {},
    );
    let result: CertificateInspection;
    try {
      result = await inspect();
    } catch (e) {
      // Rapid connections (including a fresh check after saving credentials)
      // can hit the one-second probe limit. Retry this read-only check once.
      if (!(e instanceof ApiError) || e.status !== 429 || e.code !== "RATE_LIMIT") throw e;
      await new Promise((resolve) => setTimeout(resolve, 1000));
      if (!valid(flow)) return;
      result = await inspect();
    }
    if (!valid(flow)) return;
    flow.inspection = result;
    if (!["valid", "trusted"].includes(result.status)) {
      flow.mode = result.status === "invalid" ? "failed" : "certificate";
      if (result.status === "invalid")
        flow.error = "远程证书已过期或尚未生效，请先在远端更新证书";
      certificate.value = {
        connection: c,
        initial: result,
        forConnection: true,
        flow,
      };
      return;
    }
    await authorized(c, result.check, result.expiresAt, flow);
  } else await authorized(c, "", "", flow);
}
async function connect() {
  if (busy.value || !ui.remote.connection?.id) return;
  const existing = localSessionFor(sessions.value, ui.remote.connection.id);
  if (existing && !sessionNeedsReconnect(existing.status)) {
    active.value = existing.id;
    return;
  }
  const c = rawConnection(ui.remote.connection.id);
  const flow: Flow = reactive({
    token: ++generation,
    connection: c,
    step: c.protocol === "rdp" ? 0 : 1,
    mode: "running",
    error: "",
  });
  pendingFlow.value = flow;
  certificate.value = null;
  pending.value = null;
  loginOpen.value = false;
  login.password = "";
  busy.value = true;
  error.value = "";
  try {
    await prepareConnection(c, flow);
  } catch (e) {
    fail(e, flow);
  } finally {
    if (valid(flow)) busy.value = false;
  }
}
async function certificateReady(check: string) {
  const context = certificate.value;
  if (!context || !valid(context.flow)) return;
  certificate.value = null;
  busy.value = true;
  try {
    await nextTick();
    await authorized(
      context.connection,
      check,
      context.initial?.expiresAt || new Date(Date.now() + 60000).toISOString(),
      context.flow,
    );
  } catch (e) {
    fail(e, context.flow);
  } finally {
    if (valid(context.flow)) busy.value = false;
  }
}
async function temporaryConnect() {
  if (busy.value || !pending.value) return;
  const context = pending.value;
  if (!valid(context.flow)) return;
  if (
    context.connection.protocol === "rdp" &&
    Date.now() >= Date.parse(context.expiresAt)
  ) {
    login.password = "";
    loginOpen.value = false;
    await connect();
    return;
  }
  busy.value = true;
  error.value = "";
  context.flow.mode = "running";
  context.flow.error = "";
  try {
    const c = context.connection;
    const credential = {
      username: c.protocol === "rdp" ? login.username : "",
      password: login.password,
      domain: c.protocol === "rdp" ? login.domain : "",
    };
    if (login.remember) {
      // The backend commits encrypted credentials and the connection together.
      // Use this flow's version so a concurrent edit cannot be overwritten.
      const saved = await api("/connections/" + c.id, "PATCH", {
        ...c,
        credentialId: "",
        credential,
      });
      login.password = "";
      if (!valid(context.flow)) return;
      context.connection = saved;
      context.flow.connection = saved;
      login.remember = false;
      toast("凭据已加密保存，下次连接将自动使用", { tone: "success" });
      await refreshWorkspace();
      if (!valid(context.flow)) return;
      loginOpen.value = false;
      // Saving changes the connection version and invalidates the old RDP
      // certificate check. Run the normal check again before creating a session.
      await prepareConnection(saved, context.flow);
      return;
    }
    const r = await api("/ephemeral-credentials", "POST", {
      connectionId: c.id,
      ...credential,
    });
    if (!valid(context.flow)) return;
    login.password = "";
    await start(c, context.check, context.flow, r.ref);
  } catch (e) {
    fail(e, context.flow);
  } finally {
    if (valid(context.flow)) busy.value = false;
  }
}
function desktopProgress(
  id: string,
  stage: "connecting" | "waiting" | "ready" | "failed",
  message = "",
) {
  const flow = sessions.value.find((s) => s.id === id)?.progress;
  if (!flow || flow.mode === "cancelled") return;
  if (stage === "failed") {
    if (flow.mode === "ready") return;
    flow.mode = "failed";
    flow.error = message || "远程连接失败";
    return;
  }
  if (flow.mode === "failed" || flow.mode === "ready") return;
  flow.step = stage === "connecting" ? 2 : 3;
  flow.mode = stage === "ready" ? "ready" : "running";
}
function resume() {
  const flow = visibleProgress.value;
  if (!flow || !valid(flow)) return;
  if (flow.mode === "certificate")
    certificate.value = {
      connection: flow.connection,
      initial: flow.inspection,
      forConnection: true,
      flow,
    };
  else if (flow.mode === "login") loginOpen.value = true;
}
async function end(id: string) {
  try {
    await api("/sessions/" + id, "DELETE");
    const index = sessions.value.findIndex((s) => s.id === id);
    sessions.value = sessions.value.filter((s) => s.id !== id);
    if (active.value === id) active.value = sessions.value[Math.min(index, sessions.value.length - 1)]?.id || "";
    await refreshWorkspace();
    if (!sessions.value.length && !pendingFlow.value) { await exitFullscreen(); closeRemote(); }
    return true;
  } catch (e) {
    fail(e);
    return false;
  }
}
async function cancelProgress() {
  const flow = visibleProgress.value;
  const session = sessions.value.find((s) => s.progress.token === flow?.token);
  if (flow) flow.mode = "cancelled";
  clearPending();
  if (session) await end(session.id);
  ui.remote.open = false;
}
async function retry() {
  const flow = visibleProgress.value;
  if (!flow) return;
  const c = rawConnection(flow.connection.id);
  const session = sessions.value.find((s) => s.progress.token === flow.token);
  if (session) {
    flow.mode = "cancelled";
    await end(session.id);
  }
  clearPending();
  ui.remote.connection = { ...c };
  ui.remote.open = true;
  await nextTick();
  await connect();
}
async function returnManage() {
  await exitFullscreen();
  if (pendingFlow.value) await cancelProgress();
  else ui.remote.open = false;
}
async function closeSession(id: string) {
  if (ending.value || pendingFlow.value) return;
  const session = sessions.value.find((s) => s.id === id);
  if (!session) return;
  ending.value = true;
  try {
    if (state.settings.confirmDisconnect && !(await requestConfirm({
      appendTo: workspace.value,
      title: `断开“${session.name}”？`,
      description: "将结束此远程连接并关闭标签，其他会话继续保持连接。",
      confirmText: "断开连接",
    }))) return;
    await end(id);
  } finally { ending.value = false; }
}
function tabKey(event: KeyboardEvent, index: number) {
  let next = index;
  if (event.key === "ArrowRight") next = (index + 1) % sessions.value.length;
  else if (event.key === "ArrowLeft") next = (index - 1 + sessions.value.length) % sessions.value.length;
  else if (event.key === "Home") next = 0;
  else if (event.key === "End") next = sessions.value.length - 1;
  else return;
  event.preventDefault();
  active.value = sessions.value[next]!.id;
  nextTick(() => document.getElementById(`remote-tab-${active.value}`)?.focus());
}
const reconnecting = new Set<string>();
async function reconnect(connectionId: string, id: string) {
  if (reconnecting.has(connectionId)) return;
  reconnecting.add(connectionId);
  try {
    const c = rawConnection(connectionId);
    if (!(await end(id))) return;
    clearPending();
    ui.remote.connection = { ...c };
    ui.remote.open = true;
    await nextTick();
    await connect();
  } finally { reconnecting.delete(connectionId); }
}
async function openRequestedConnection() {
  const connectionId = ui.remote.connection?.id;
  if (!connectionId) return;
  const existing = localSessionFor(sessions.value, connectionId);
  if (existing) {
    if (pendingFlow.value) clearPending();
    active.value = existing.id;
    if (sessionNeedsReconnect(existing.status)) await reconnect(connectionId, existing.id);
    return;
  }
  // Repeated clicks while checking certificates or waiting for credentials
  // return to that flow instead of cancelling it and creating another one.
  if (pendingFlow.value?.connection.id === connectionId && !["cancelled", "failed"].includes(pendingFlow.value.mode)) {
    resume();
    return;
  }
  if (pendingFlow.value || busy.value) clearPending();
  await connect();
}
let handlingLaunch = false;
let requestedConnection = ui.remote.connection;
let handledNonce = ui.remote.nonce;
async function handleLaunch() {
  if (handlingLaunch) return;
  handlingLaunch = true;
  try {
    // Serialize launch requests so rapid clicks cannot race through session
    // creation/reconnection. Coalesce repeated requests to the latest target.
    while (alive && handledNonce !== ui.remote.nonce) {
      await nextTick();
      handledNonce = ui.remote.nonce;
      // A reconnect may temporarily close/restore ui.remote.connection. Retain
      // the actual latest launch target independently of those UI mutations.
      if (!requestedConnection) continue;
      ui.remote.connection = { ...requestedConnection };
      ui.remote.open = true;
      await nextTick();
      await openRequestedConnection();
    }
  } catch (e) { fail(e); }
  finally { handlingLaunch = false; }
}
watch(() => ui.remote.nonce, () => {
  requestedConnection = ui.remote.connection ? { ...ui.remote.connection } : null;
  void handleLaunch();
});
onBeforeUnmount(() => {
  document.removeEventListener("fullscreenchange", syncFullscreen);
  alive = false;
  ui.remote.sessions = [];
  clearPending();
  for (const s of sessions.value)
    api("/sessions/" + s.id, "DELETE").catch(() => {});
});
</script>
<template>
  <Teleport to="body">
    <button
      v-if="sessions.length && !ui.remote.open"
      class="fixed bottom-4 right-4 z-[100] rounded-xl border border-line bg-surface px-4 py-3 text-sm text-ink shadow-float"
      @click="ui.remote.open = true"
    >
      返回远程会话 · {{ sessions.length }}
    </button>
    <section
      v-show="ui.remote.open"
      ref="workspace"
      class="relay-remote fixed inset-0 z-[100] flex flex-col bg-canvas"
      aria-label="远程会话工作区"
    >
      <header class="remote-session-header remote-compact native-desktop">
        <div class="remote-tab-group">
          <nav class="remote-session-tabs" role="tablist" aria-label="桌面会话">
            <div v-for="(s, index) in sessions" :key="s.id" class="remote-session-tab" :class="{ 'is-active': active === s.id }">
              <button :id="`remote-tab-${s.id}`" role="tab" :aria-selected="active === s.id" :aria-controls="`remote-panel-${s.id}`"
                :tabindex="active === s.id ? 0 : -1" :disabled="!!pendingFlow" :title="s.name + ' · ' + s.status" :aria-label="s.name + '，' + s.status"
                @click="active = s.id" @keydown="tabKey($event, index)">
                <span class="remote-session-status" :data-state="s.status === '已连接' ? 'connected' : ['连接失败', '已中断', '已断开'].includes(s.status) ? 'disconnected' : 'connecting'" aria-hidden="true"/><span>{{ s.name }}</span>
              </button>
              <button class="remote-tab-close" :aria-label="'断开并关闭会话 ' + s.name" :title="'断开并关闭 ' + s.name"
                :disabled="ending || !!pendingFlow" @click="closeSession(s.id)"><X :size="14" aria-hidden="true"/></button>
            </div>
          </nav>
          <button class="remote-header-button remote-new-session" aria-label="新建连接" title="返回管理选择其他电脑，现有会话保持连接" @click="returnManage"><Plus :size="18" aria-hidden="true"/></button>
          <span v-if="busy" class="remote-connecting" role="status">正在连接…</span>
          <span v-else-if="!sessions.length" class="remote-connecting">{{ error || '等待连接' }}</span>
        </div>
        <div class="remote-header-actions">
          <div id="remote-session-toolbar" />
          <button class="remote-header-button" :aria-label="isFullscreen ? '退出全屏' : '全屏'" :title="isFullscreen ? '退出全屏（Esc）' : '全屏'" :aria-pressed="isFullscreen" @click="toggleFullscreen">
            <Minimize v-if="isFullscreen" :size="16" aria-hidden="true"/><Maximize v-else :size="16" aria-hidden="true"/>
          </button>
          <span class="remote-header-divider" aria-hidden="true"/>
          <button class="remote-header-button" aria-label="返回管理" title="返回管理，保留所有已连接会话" @click="returnManage"><ArrowLeft :size="16" aria-hidden="true"/></button>
          <button class="remote-header-button remote-disconnect" aria-label="断开当前会话" title="断开当前会话"
            :disabled="!active || ending || !!pendingFlow" @click="closeSession(active)"><Unplug :size="16" aria-hidden="true"/></button>
        </div>
      </header>
      <div ref="host" class="native-desktop relative min-h-0 min-w-0 flex-1 overflow-hidden">
        <Desktop
          v-for="s in sessions"
          v-show="active === s.id"
          :key="s.id"
          :id="`remote-panel-${s.id}`"
          role="tabpanel"
          :aria-labelledby="`remote-tab-${s.id}`"
          toolbar-target="#remote-session-toolbar"
          :active="ui.remote.open && active === s.id"
          :session-id="s.id"
          :name="s.name"
          :protocol="s.protocol"
          :connection-id="s.connectionId"
          :target-os="s.targetOs"
          :clipboard="s.clipboard"
          :render-mode="s.renderMode"
          :wallpaper="s.wallpaper"
          @status-change="sessionStatus(s.id, $event)"
          @close="closeSession(s.id)"
          @reconnect="reconnect(s.connectionId, s.id)"
          @progress="(stage, message) => desktopProgress(s.id, stage, message)"
        />
        <ConnectionProgress
          v-if="
            visibleProgress &&
            !['ready', 'cancelled'].includes(visibleProgress.mode)
          "
          :progress="visibleProgress"
          @retry="retry"
          @cancel="cancelProgress"
          @resume="resume"
        />
      </div>
    </section>
    <div class="native-desktop">
      <CertificateDialog
        v-if="certificate"
        :connection="certificate.connection"
        :initial="certificate.initial"
        :for-connection="certificate.forConnection"
        @close="certificate = null"
        @ready="certificateReady"
      />
    </div>
    <BaseModal
      :open="loginOpen"
      title="远程登录"
      :description="login.remember
        ? '凭据将加密保存在 NAS，并关联当前连接。'
        : '本次凭据仅用于当前会话；勾选记住后，下次无需再次输入。'"
      @close="
        loginOpen = false;
        login.password = '';
      "
    >
      <el-form
        label-position="left"
        label-width="96px"
        :disabled="busy"
        @submit.prevent="temporaryConnect"
      >
        <el-form-item
          v-if="pending?.connection.protocol === 'rdp'"
          label="用户名"
          ><el-input
            v-model="login.username"
            autocomplete="off"
            aria-label="远程用户名"
        /></el-form-item>
        <el-form-item label="密码"
          ><el-input
            v-model="login.password"
            type="password"
            show-password
            autocomplete="new-password"
            aria-label="远程密码"
        /></el-form-item>
        <el-form-item
          v-if="pending?.connection.protocol === 'rdp'"
          label="域（可选）"
          ><el-input v-model="login.domain" aria-label="远程域"
        /></el-form-item>
        <el-form-item>
          <el-checkbox v-model="login.remember" :disabled="busy">记住凭据，下次自动使用</el-checkbox>
        </el-form-item>
        <p v-if="error" role="alert" class="text-xs text-danger">{{ error }}</p>
      </el-form>
      <template #footer
        ><el-button
          :disabled="busy"
          @click="
            loginOpen = false;
            login.password = '';
          "
          >取消</el-button
        ><el-button
          type="primary"
          :loading="busy"
          :disabled="!login.password"
          @click="temporaryConnect"
          >连接</el-button
        ></template
      >
    </BaseModal>
  </Teleport>
</template>
