<script setup lang="ts">
import { computed, reactive, watch, ref } from "vue";
import { Check } from "lucide-vue-next";
import { loadWallpaper } from "../display";
import DeviceIcon from "./DeviceIcon.vue";
import { deviceIcons } from "../device-icons";
import BaseModal from "./BaseModal.vue";
import {
  pendingLogin,
  saveConnection,
  state,
  wolByConnection,
} from "../stores/app";
import { toast } from "../stores/ui";
import type { Connection } from "../types";

const props = defineProps<{
  open: boolean;
  connection?: Connection | null;
  presetGroupId?: string;
}>();

const emit = defineEmits<{
  close: [];
  saved: [connection: Connection];
}>();

const protocols = [
  { value: "RDP", label: "RDP", hint: "Windows 桌面", port: 3389 },
  { value: "VNC", label: "VNC", hint: "跨平台桌面", port: 5900 },
];

interface ConnectionFormState {
  icon: string;
  version: number;
  id: string | null;
  name: string;
  host: string;
  port: number | string;
  protocol: string;
  groupId: string;
  credentialId: string;
  status: string;
  tags: string;
  notes: string;
  favorite: boolean;
  wallpaper: boolean;
  wolEnabled: boolean;
  wolMac: string;
  wolBroadcast: string;
}

const form = reactive<ConnectionFormState>({} as ConnectionFormState);
const errors = reactive<Record<string, string>>({});
const busy = ref(false);
const authModes = [
  { value: "prompt", label: "连接时输入" },
  { value: "existing", label: "使用已有" },
  { value: "new", label: "新建" },
] as const;
const authMode = ref<"prompt" | "existing" | "new">("prompt");
const pickerOpen = ref(false);
const query = ref("");
const login = reactive({ username: "", domain: "", password: "" });
const remember = ref(false);

function clearLogin() {
  Object.assign(login, { username: "", domain: "", password: "" });
  remember.value = false;
}

function chooseMode(mode: typeof authMode.value) {
  if (mode === "existing") {
    query.value = "";
    pickerOpen.value = true;
    return;
  }
  if (mode !== authMode.value) clearLogin();
  form.credentialId = "";
  authMode.value = mode;
}

function selectCredential(id: string) {
  clearLogin();
  form.credentialId = id;
  authMode.value = "existing";
  pickerOpen.value = false;
}

const availableCredentials = computed(() =>
  state.credentials.filter((credential) => credential.protocol === form.protocol?.toLowerCase()),
);

const matches = computed(() => {
  const term = query.value.trim().toLowerCase();
  return availableCredentials.value.filter(c =>
    [c.name, c.username, c.domain].some(value => value?.toLowerCase().includes(term)),
  );
});
const selected = computed(() => availableCredentials.value.find(c => c.id === form.credentialId));

function blankForm(): ConnectionFormState {
  return {
    icon: "auto",
    version: 0,
    id: null,
    name: "",
    host: "",
    port: 3389,
    protocol: "RDP",
    groupId: props.presetGroupId || "",
    credentialId: "",
    status: "unknown",
    tags: "",
    notes: "",
    favorite: false,
    wallpaper: loadWallpaper(),
    wolEnabled: false,
    wolMac: "",
    wolBroadcast: "",
  };
}

function resetForm() {
  Object.keys(errors).forEach((key) => delete errors[key]);
  clearLogin();
  authMode.value = "prompt";
  Object.assign(form, blankForm());
  if (props.connection) {
    Object.assign(form, {
      ...props.connection,
      tags: Array.isArray(props.connection.tags)
        ? props.connection.tags.join(", ")
        : "",
      icon: props.connection.settings?.icon || "auto",
      favorite: Boolean(props.connection.favorite),
      wallpaper: props.connection.settings?.wallpaper ?? loadWallpaper(),
    });
    const once = pendingLogin(props.connection.id, props.connection.version);
    if (once) Object.assign(login, once);
    authMode.value = form.credentialId ? "existing" : once ? "new" : "prompt";
    const linked = wolByConnection(props.connection.id);
    form.wolEnabled = Boolean(linked);
    form.wolMac = linked?.mac || "";
    form.wolBroadcast = linked?.broadcast || "";
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) resetForm();
    else { pickerOpen.value = false; clearLogin(); }
  },
  { immediate: true },
);

function setProtocol(protocol: string) {
  if (protocol === form.protocol) return;
  const previous = protocols.find((item) => item.value === form.protocol);
  form.protocol = protocol;
  authMode.value = "prompt";
  clearLogin();
  form.credentialId = "";
  const next = protocols.find((item) => item.value === protocol);
  if (!Number(form.port) || Number(form.port) === previous?.port) {
    form.port = next?.port || form.port;
  }
}

function validate() {
  Object.keys(errors).forEach((key) => delete errors[key]);
  if (!form.name.trim()) errors.name = "请填写连接名称";
  if (!form.host.trim()) errors.host = "请填写主机地址或 IP";
  const port = Number(form.port);
  if (!port || port < 1 || port > 65535)
    errors.port = "端口需在 1 - 65535 之间";
  if (form.host.includes(" ")) errors.host = "主机地址不能包含空格";
  if (authMode.value === "new") {
    if (form.protocol === "RDP" && !login.username.trim()) errors.username = "请填写用户名";
    if (!login.password) errors.password = "请填写密码";
  }
  if (form.wolEnabled) {
    const mac = form.wolMac.trim();
    if (!mac) errors.wolMac = "请填写 MAC 地址";
    else if (!/^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$/.test(mac))
      errors.wolMac = "格式如 AA:BB:CC:DD:EE:FF";
  }
  return !Object.keys(errors).length;
}

async function submit() {
  if (busy.value) return;
  if (!validate()) return;
  busy.value = true;
  try {
    const saved = await saveConnection({
      ...form,
      port: Number(form.port),
      tags: form.tags
        .split(/[,，]/)
        .map((tag) => tag.trim())
        .filter(Boolean),
    }, authMode.value === "new" ? { login: { ...login, username: login.username.trim(), domain: login.domain.trim() }, remember: remember.value } : undefined);
    toast(props.connection ? "连接已更新" : "新连接已创建", {
      description: `${saved.name} · ${saved.host}`,
      tone: "success",
    });
    emit("saved", saved);
    emit("close");
  } catch (e) {
    toast((e as Error).message, { tone: "danger" });
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <BaseModal
    :open="open"
    :title="connection ? '编辑连接' : '新建连接'"
    size="lg"
    @close="!busy && emit('close')"
  >
    <el-form
      :model="form"
      label-position="left"
      label-width="96px"
      class="connection-form"
      @submit.prevent="submit"
    >
      <div class="connection-fields">
        <el-form-item class="connection-name" label="连接名称" :error="errors.name">
          <el-input
            v-model="form.name"
            placeholder="例如：设计工作站"
            aria-label="连接名称"
            autocomplete="off"
          />
        </el-form-item>
        <el-form-item label="设备图标">
          <el-select v-model="form.icon" class="w-full" aria-label="设备图标">
            <template #prefix>
              <DeviceIcon :icon="form.icon" :os="form.protocol === 'RDP' ? 'Windows' : ''" :size="22" />
            </template>
            <el-option
              v-for="option in deviceIcons"
              :key="option.value"
              :value="option.value"
              :label="option.label"
            >
              <span class="flex h-full items-center gap-2">
                <DeviceIcon :icon="option.value" :os="form.protocol === 'RDP' ? 'Windows' : ''" :size="22" />
                {{ option.label }}
              </span>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item class="secondary-field" label="所属分组" label-width="64px">
          <el-select
            v-model="form.groupId"
            class="w-full"
            placeholder="未分组"
            aria-label="所属分组"
          >
            <el-option label="未分组" value="" />
            <el-option
              v-for="group in state.groups"
              :key="group.id"
              :label="group.name"
              :value="group.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="主机地址 / IP" :error="errors.host">
          <el-input
            v-model="form.host"
            placeholder="192.168.1.10 或 device.local"
            aria-label="主机地址"
            class="font-mono"
            autocomplete="off"
            spellcheck="false"
          />
        </el-form-item>
        <el-form-item class="secondary-field" label="端口" label-width="64px" :error="errors.port">
          <el-input
            v-model="form.port"
            type="number"
            class="font-mono"
            aria-label="端口"
          />
        </el-form-item>
      </div>

      <el-form-item label="连接协议">
        <div class="grid w-full grid-cols-1 gap-2 sm:grid-cols-2">
          <button
            v-for="item in protocols"
            :key="item.value"
            type="button"
            class="relative flex items-center gap-2.5 rounded-lg border p-2 text-left transition"
            :class="
              form.protocol === item.value
                ? 'border-ink bg-soft shadow-[inset_0_0_0_1px_var(--text)]'
                : 'border-line-strong bg-surface hover:border-faint'
            "
            @click="setProtocol(item.value)"
          >
            <span class="flex min-w-0 flex-col gap-0.5 leading-tight">
              <span class="font-mono text-[12px] font-bold text-ink">{{
                item.label
              }}</span>
              <span class="text-[10.5px] text-faint">{{ item.hint }}</span>
            </span>
            <span
              class="ml-auto h-3 w-3 rounded-full border"
              :class="
                form.protocol === item.value
                  ? 'border-[3px] border-ink bg-accent'
                  : 'border-line-strong bg-surface'
              "
            ></span>
          </button>
        </div>
      </el-form-item>

      <el-form-item label="认证方式">
        <div class="flex flex-wrap items-center gap-2">
          <div role="group" aria-label="认证方式" class="inline-flex rounded-lg border border-line bg-soft p-0.5">
            <button v-for="mode in authModes" :key="mode.value" type="button"
              class="rounded-md px-2.5 py-1 text-xs font-semibold"
              :class="authMode === mode.value ? 'bg-ink text-surface' : 'text-muted hover:text-ink'"
              :aria-pressed="authMode === mode.value" @click="chooseMode(mode.value)">
              {{ mode.label }}
            </button>
          </div>
          <button v-if="authMode === 'existing'" type="button" class="max-w-full truncate text-xs text-muted underline"
            @click="chooseMode('existing')">{{ selected?.name || '选择凭据' }}</button>
        </div>
      </el-form-item>
      <template v-if="authMode === 'new'">
        <template v-if="form.protocol === 'RDP'">
          <el-form-item label="用户名" :error="errors.username">
            <el-input v-model="login.username" aria-label="用户名" autocomplete="off" />
          </el-form-item>
          <el-form-item label="域（可选）">
            <el-input v-model="login.domain" aria-label="域（可选）" autocomplete="off" />
          </el-form-item>
        </template>
        <el-form-item label="密码" :error="errors.password">
          <el-input v-model="login.password" type="password" show-password aria-label="密码" autocomplete="new-password" />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="remember">记住凭据</el-checkbox>
        </el-form-item>
      </template>

      <div class="connection-fields">
        <el-form-item label="网络唤醒">
          <el-switch v-model="form.wolEnabled" aria-label="启用网络唤醒" />
        </el-form-item>
        <el-form-item v-if="form.protocol === 'RDP'" class="secondary-field" label="桌面壁纸" label-width="64px">
          <el-switch v-model="form.wallpaper" aria-label="显示桌面壁纸" />
        </el-form-item>
      </div>
      <div v-if="form.wolEnabled" class="connection-fields">
        <el-form-item
          label="MAC 地址"
          :error="errors.wolMac"
        >
          <el-input
            v-model="form.wolMac"
            class="font-mono"
            placeholder="AA:BB:CC:DD:EE:FF"
            aria-label="WOL MAC 地址"
            autocomplete="off"
            spellcheck="false"
          />
        </el-form-item>
        <el-form-item class="secondary-field" label="广播地址" label-width="64px">
          <el-input
            v-model="form.wolBroadcast"
            class="font-mono"
            placeholder="自动选择"
            aria-label="WOL 广播地址"
            autocomplete="off"
            spellcheck="false"
          />
        </el-form-item>
      </div>

    </el-form>

    <template #footer>
      <el-button @click="emit('close')">取消</el-button>
      <el-button
        type="primary"
        class="min-w-[104px]"
        :loading="busy"
        @click="submit"
      >
        <Check :size="15" class="mr-1.5" />
        {{ connection ? "保存修改" : "创建连接" }}
      </el-button>
    </template>
  </BaseModal>
  <BaseModal :open="pickerOpen" title="选择已有凭据" size="sm" @close="pickerOpen = false">
    <el-input v-model="query" clearable placeholder="搜索名称、用户名或域" aria-label="搜索凭据" />
    <div class="mt-2 max-h-64 overflow-y-auto">
      <button v-for="item in matches" :key="item.id" type="button"
        class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left hover:bg-soft"
        @click="selectCredential(item.id)">
        <span class="min-w-0 flex-1">
          <strong class="block truncate text-sm text-ink">{{ item.name }}</strong>
          <span v-if="item.username" class="block truncate text-xs text-muted">{{ [item.domain, item.username].filter(Boolean).join('\\') }}</span>
        </span>
        <Check v-if="item.id === form.credentialId" :size="16" />
      </button>
      <p v-if="!matches.length" class="p-3 text-sm text-muted">
        {{ availableCredentials.length ? '没有匹配的凭据' : '暂无可用于 ' + form.protocol + ' 的凭据' }}
      </p>
    </div>
  </BaseModal>
</template>

<style scoped>
.connection-fields {
  display: grid;
  grid-template-columns: minmax(0, 1.3fr) minmax(0, 1fr);
  gap: 8px 40px;
  margin-bottom: 12px;
}

.connection-name {
  grid-column: 1 / -1;
}

.connection-fields :deep(.el-form-item) {
  min-width: 0;
  margin: 0;
}

.connection-fields :deep(.el-input:has(input[type="number"])) {
  max-width: 100px;
}

@media (min-width: 640px) {
  .secondary-field :deep(.el-form-item__label) {
    justify-content: flex-end;
    padding-right: 8px;
  }
}

@media (max-width: 639px) {
  .connection-fields {
    grid-template-columns: minmax(0, 1fr);
  }

  .relay-dialog .connection-form.el-form--label-left :deep(.el-form-item) {
    display: flex;
  }

  .relay-dialog .connection-form.el-form--label-left :deep(.el-form-item__label) {
    width: 84px !important;
    height: 32px;
    align-items: center;
    line-height: 1.3;
    padding: 0 8px 0 0;
  }
}
</style>
