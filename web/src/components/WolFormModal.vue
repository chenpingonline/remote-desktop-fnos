<script setup lang="ts">
import { reactive, watch, ref } from "vue";
import { Zap } from "lucide-vue-next";
import BaseModal from "./BaseModal.vue";
import { saveWol, state } from "../stores/app";
import { toast } from "../stores/ui";
import type { WolEntry } from "../types";

const props = defineProps<{
  open: boolean;
  entry?: WolEntry | null;
}>();

const emit = defineEmits<{
  close: [];
  saved: [entry: WolEntry];
}>();

interface WolFormState {
  id: string | null;
  name: string;
  mac: string;
  broadcast: string;
  port: string;
  connectionId: string;
  notes: string;
}

const form = reactive<WolFormState>({
  id: null,
  name: "",
  mac: "",
  broadcast: "",
  port: "9",
  connectionId: "",
  notes: "",
});
const errors = reactive<Record<string, string>>({});
const busy = ref(false);
const macPattern = /^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$/;

function reset() {
  Object.keys(errors).forEach((key) => delete errors[key]);
  Object.assign(form, {
    id: null,
    name: "",
    mac: "",
    broadcast: "",
    port: "9",
    connectionId: "",
    notes: "",
  });
  if (props.entry) {
    Object.assign(form, {
      id: props.entry.id,
      name: props.entry.name,
      mac: props.entry.mac,
      broadcast: props.entry.broadcast,
      port: String(props.entry.port),
      connectionId: props.entry.connectionId || "",
      notes: props.entry.notes,
    });
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) reset();
  },
  { immediate: true },
);

function onConnectionChange(id: string) {
  if (!id) return;
  const connection = state.connections.find((item) => item.id === id);
  if (connection) form.name = connection.name;
}

async function submit() {
  if (busy.value) return;
  Object.keys(errors).forEach((key) => delete errors[key]);
  if (!form.connectionId) errors.name = "请选择关联连接";
  if (!form.name.trim()) errors.name = "请填写唤醒名称";
  if (!form.mac.trim()) errors.mac = "请填写 MAC 地址";
  else if (!macPattern.test(form.mac.trim()))
    errors.mac = "格式如 AA:BB:CC:DD:EE:FF";
  const port = Number(form.port);
  if (!port || port < 1 || port > 65535)
    errors.port = "端口需在 1 - 65535 之间";
  if (Object.keys(errors).length) return;
  busy.value = true;
  try {
    const saved = await saveWol({
      id: form.id,
      name: form.name.trim(),
      mac: form.mac.trim(),
      broadcast: form.broadcast.trim() || "",
      port,
      connectionId: form.connectionId || null,
      notes: form.notes.trim(),
    });
    toast(props.entry ? "唤醒项已更新" : "唤醒项已创建", {
      description: `${saved.name} · ${saved.mac}`,
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
    :title="entry ? '编辑唤醒项' : '新建唤醒项'"
    description="通过局域网发送 WOL 魔术包，让关机状态的设备自动开机。"
    size="md"
    @close="!busy && emit('close')"
  >
    <el-form
      :model="form"
      label-position="left"
      label-width="96px"
      @submit.prevent="submit"
    >
      <el-form-item label="唤醒名称" :error="errors.name">
        <el-input
          v-model="form.name"
          disabled
          placeholder="例如：设计工作站"
          aria-label="唤醒名称"
          autocomplete="off"
        />
      </el-form-item>
      <el-form-item label="关联连接">
        <el-select
          v-model="form.connectionId"
          :disabled="!!entry"
          class="w-full"
          placeholder="选择要关联的设备（必选）"
          aria-label="关联远程连接"
          @change="onConnectionChange"
        >
          <el-option
            v-for="connection in state.connections"
            :key="connection.id"
            :label="`${connection.name} · ${connection.host}`"
            :value="connection.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="MAC 地址" :error="errors.mac">
        <el-input
          v-model="form.mac"
          class="font-mono"
          placeholder="AA:BB:CC:DD:EE:FF"
          aria-label="MAC 地址"
          autocomplete="off"
          spellcheck="false"
        />
      </el-form-item>
      <el-form-item label="端口" :error="errors.port">
        <el-input
          v-model="form.port"
          disabled
          type="number"
          class="font-mono !w-[140px]"
          aria-label="端口"
        />
      </el-form-item>
      <el-form-item label="广播地址">
        <el-input
          v-model="form.broadcast"
          class="font-mono"
          placeholder="留空自动选择直连子网"
          aria-label="广播地址"
          autocomplete="off"
          spellcheck="false"
        />
      </el-form-item>
      <div
        class="mt-1 flex items-start gap-2 rounded-lg border border-line bg-soft p-2 text-info"
      >
        <Zap :size="13" class="mt-0.5 flex-none" />
        <span class="text-[10.5px] leading-4 text-faint"
          >由 NAS 向直连网段发送 UDP 9
          魔术包，目标设备需启用网络唤醒。发送成功不代表设备已经开机。名称随关联连接保存。</span
        >
      </div>
    </el-form>

    <template #footer>
      <el-button @click="emit('close')">取消</el-button>
      <el-button type="primary" :loading="busy" @click="submit"
        ><Zap :size="14" class="mr-1" />{{
          entry ? "保存修改" : "创建唤醒项"
        }}</el-button
      >
    </template>
  </BaseModal>
</template>
