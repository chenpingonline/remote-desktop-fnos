<script setup lang="ts">
import { reactive, watch, ref } from "vue";
import { Check, Star, Zap } from "lucide-vue-next";
import BaseModal from "./BaseModal.vue";
import {
  removeWol,
  saveConnection,
  saveWol,
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
  clipboard: boolean;
  wolEnabled: boolean;
  wolMac: string;
  wolBroadcast: string;
}

const form = reactive<ConnectionFormState>({} as ConnectionFormState);
const errors = reactive<Record<string, string>>({});
const busy = ref(false);

function blankForm(): ConnectionFormState {
  return {
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
    clipboard: false,
    wolEnabled: false,
    wolMac: "",
    wolBroadcast: "",
  };
}

function resetForm() {
  Object.keys(errors).forEach((key) => delete errors[key]);
  Object.assign(form, blankForm());
  if (props.connection) {
    Object.assign(form, {
      ...props.connection,
      tags: Array.isArray(props.connection.tags)
        ? props.connection.tags.join(", ")
        : "",
      favorite: Boolean(props.connection.favorite),
      clipboard: props.connection.settings?.clipboard === "text",
    });
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
  },
  { immediate: true },
);

function setProtocol(protocol: string) {
  const previous = protocols.find((item) => item.value === form.protocol);
  form.protocol = protocol;
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
    });
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
    :description="
      connection
        ? '更新设备信息与连接参数。'
        : '保存一台远程设备，之后可以一键发起安全会话。'
    "
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
      <el-form-item label="连接名称" :error="errors.name">
        <el-input
          v-model="form.name"
          placeholder="例如：设计工作站"
          aria-label="连接名称"
          autocomplete="off"
        />
      </el-form-item>
      <el-form-item label="所属分组">
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
      <el-form-item label="端口" :error="errors.port">
        <el-input
          v-model="form.port"
          type="number"
          class="font-mono !w-[140px]"
          aria-label="端口"
        />
      </el-form-item>

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

      <el-form-item label="文本剪贴板"
        ><el-switch v-model="form.clipboard" aria-label="启用文本剪贴板"
      /></el-form-item>
      <el-form-item label="默认凭据">
        <el-select
          v-model="form.credentialId"
          class="w-full"
          placeholder="不保存凭据"
          aria-label="默认凭据"
        >
          <el-option label="不保存凭据" value="" />
          <el-option
            v-for="credential in state.credentials.filter(
              (c) => c.protocol === form.protocol.toLowerCase(),
            )"
            :key="credential.id"
            :label="`${credential.name} · ${credential.protocol?.toUpperCase()}`"
            :value="credential.id"
          />
        </el-select>
      </el-form-item>

      <div class="rounded-xl border border-line bg-soft px-3 py-2.5">
        <div class="flex items-center justify-between gap-3">
          <div class="flex min-w-0 flex-1 items-center gap-2 text-faint">
            <Zap :size="15" class="flex-none" />
            <p class="text-[12px] font-semibold leading-5 text-ink">
              网络唤醒 (WOL)<span
                class="ml-1.5 text-[11px] font-normal text-faint max-[479px]:ml-0 max-[479px]:block"
                >关机后可用魔术包远程开机</span
              >
            </p>
          </div>
          <el-switch
            v-model="form.wolEnabled"
            class="flex-none"
            aria-label="启用网络唤醒"
          />
        </div>
        <div
          v-if="form.wolEnabled"
          class="mt-2.5 grid grid-cols-1 gap-x-3 gap-y-2 border-t border-line pt-2.5 sm:grid-cols-2"
        >
          <el-form-item
            label="MAC 地址"
            label-width="68px"
            :error="errors.wolMac"
            class="!mb-0"
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
          <el-form-item label="广播地址" label-width="68px" class="!mb-0">
            <el-input
              v-model="form.wolBroadcast"
              class="font-mono"
              placeholder="留空自动选择直连子网"
              aria-label="WOL 广播地址"
              autocomplete="off"
              spellcheck="false"
            />
          </el-form-item>
        </div>
      </div>

      <div
        class="mt-2 flex items-center justify-between gap-3 rounded-xl border border-line bg-soft px-3 py-2.5"
      >
        <div class="flex min-w-0 flex-1 items-center gap-2 text-faint">
          <Star :size="15" class="flex-none" />
          <p class="text-[12px] font-semibold leading-5 text-ink">
            添加到收藏<span
              class="ml-1.5 text-[11px] font-normal text-faint max-[479px]:ml-0 max-[479px]:block"
              >收藏的连接会固定显示在侧边栏快速访问中。</span
            >
          </p>
        </div>
        <el-switch
          v-model="form.favorite"
          class="flex-none"
          aria-label="添加到收藏"
        />
      </div>

      <div
        class="mb-2.5 mt-4 flex flex-wrap items-baseline gap-x-2 gap-y-0.5 border-t border-dashed border-line pt-3"
      >
        <span class="text-[11.5px] font-bold text-muted">本地信息</span>
        <span class="text-[11px] leading-4 text-faint"
          >标签与备注仅保存在此浏览器，不会写入 NAS。</span
        >
      </div>
      <el-form-item label="标签">
        <el-input
          v-model="form.tags"
          placeholder="逗号分隔，例如：设计, 主力设备"
          aria-label="标签"
        />
      </el-form-item>
      <el-form-item label="备注" class="!mb-0">
        <el-input
          v-model="form.notes"
          type="textarea"
          :rows="2"
          placeholder="补充说明、设备用途或其他信息…"
          aria-label="备注"
        />
      </el-form-item>
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
</template>
