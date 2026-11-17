<script setup lang="ts">
import { computed, ref } from "vue";
import {
  Check,
  Clock3,
  Copy,
  KeyRound,
  MoreHorizontal,
  Pencil,
  Star,
  Trash2,
  Wifi,
  Power,
} from "lucide-vue-next";
import DeviceIcon from "./DeviceIcon.vue";
import type { Connection, Credential, Group, WolEntry } from "../types";
import { timeAgo } from "../utils/format";
import { toast, wakeWolEntry, remoteActionLabel } from "../stores/ui";

const props = defineProps<{
  connection: Connection;
  group?: Group | null;
  credential?: Credential | null;
  wol?: WolEntry | null;
}>();

const emit = defineEmits<{
  connect: [connection: Connection];
  edit: [connection: Connection];
  delete: [connection: Connection];
  favorite: [connection: Connection];
}>();

import CertificateDialog from "./CertificateDialog.vue";
import {
  rawConnection,
  probeConnection,
  probingConnections,
  connectionStatusLabel,
  latencyLabel,
  probeDescription,
} from "../stores/app";
const certificateOpen = ref(false);
const probing = computed(() => probingConnections.has(props.connection.id));
async function probe() {
  try {
    const r = await probeConnection(props.connection.id);
    if (r) toast(r.message, { tone: r.reachable ? "success" : "danger" });
  } catch (e) {
    toast((e as Error).message, { tone: "danger" });
  }
}
const copied = ref(false);

const statusLabel = computed(() =>
  probing.value ? "检测中…" : connectionStatusLabel(props.connection.status),
);
const statusClass = computed(() =>
  props.connection.status === "online"
    ? "bg-success"
    : props.connection.status === "busy"
      ? "bg-warning"
      : props.connection.status === "offline"
        ? "bg-danger"
        : "bg-faint",
);

function copyHost() {
  navigator.clipboard
    ?.writeText(props.connection.host)
    .then(() => {
      copied.value = true;
      toast("主机地址已复制", {
        description: props.connection.host,
        tone: "success",
      });
      setTimeout(() => (copied.value = false), 1600);
    })
    .catch(() => toast("复制失败，请手动选择地址", { tone: "danger" }));
}

function handleCommand(command: string) {
  if (command === "certificate") certificateOpen.value = true;
  if (command === "probe") probe();
  if (command === "edit") emit("edit", props.connection);
  if (command === "delete") emit("delete", props.connection);
}
</script>

<template>
  <article
    class="connection-card group relative flex min-h-[188px] flex-col rounded-2xl border border-line bg-surface p-3.5 device-card-motion shadow-card"
  >
    <div class="flex items-start gap-2.5">
      <DeviceIcon :icon="connection.settings?.icon" :os="connection.os" :size="38" />
      <div class="flex min-w-0 flex-1 flex-col gap-0.5 pt-0.5">
        <h3
          class="truncate text-[13.5px] font-bold tracking-[-0.015em] text-ink"
        >
          {{ connection.name }}
        </h3>
        <div class="flex min-w-0 items-center gap-1 font-mono text-[10.5px] text-faint">
          <span class="select-text truncate">{{ connection.host }}</span>
          <button
            type="button"
            class="grid h-6 w-6 flex-none place-items-center rounded transition hover:bg-soft hover:text-muted"
            title="复制主机地址"
            aria-label="复制主机地址"
            @click.stop="copyHost"
          >
            <Check v-if="copied" :size="11" />
            <Copy v-else :size="11" />
          </button>
        </div>
      </div>
      <div class="-mr-1 -mt-0.5 flex items-center gap-0.5">
        <el-tooltip :show-arrow="false"
          :content="connection.favorite ? '取消收藏' : '收藏连接'"
          placement="top"
          :show-after="350"
        >
          <button
            type="button"
            class="grid h-10 w-10 place-items-center rounded-md transition hover:bg-soft sm:h-7 sm:w-7"
            :class="
              connection.favorite
                ? 'text-[#c69a2e]'
                : 'text-faint opacity-60 group-hover:opacity-100'
            "
            :aria-label="connection.favorite ? '取消收藏' : '收藏连接'"
            @click="emit('favorite', connection)"
          >
            <Star
              :size="14"
              :fill="connection.favorite ? 'currentColor' : 'none'"
            />
          </button>
        </el-tooltip>
        <el-dropdown
          trigger="click"
          placement="bottom-end"
          @command="handleCommand"
        >
          <button
            type="button"
            class="grid h-10 w-10 place-items-center rounded-md text-faint transition hover:bg-soft hover:text-ink sm:h-7 sm:w-7"
            aria-label="更多操作"
          >
            <MoreHorizontal :size="16" />
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="edit">
                <span class="flex items-center gap-2"
                  ><Pencil :size="13" /> 编辑连接</span
                >
              </el-dropdown-item>
              <el-dropdown-item command="delete" class="!text-danger">
                <span class="flex items-center gap-2"
                  ><Trash2 :size="13" /> 删除连接</span
                >
              </el-dropdown-item>
              <el-dropdown-item command="probe" divided :disabled="probing"
                ><span>{{
                  probing ? "检测中…" : "检查 TCP 连通性"
                }}</span></el-dropdown-item
              >
              <el-dropdown-item
                v-if="connection.protocol === 'RDP'"
                command="certificate"
                ><span>证书与信任</span></el-dropdown-item
              >
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <div class="mt-2.5 flex items-center gap-2 text-[10.5px] text-faint">
      <span
        class="inline-flex items-center gap-1.5 font-semibold text-muted"
        role="status"
        aria-live="polite"
        :aria-busy="probing"
        :title="probeDescription(connection)"
      >
        <span class="h-[7px] w-[7px] rounded-full" :class="statusClass"></span
        >{{ statusLabel }}
      </span>
      <span class="h-0.5 w-0.5 rounded-full bg-line-strong"></span>
      <span>{{ connection.osVersion || connection.os }}</span>
    </div>

    <div class="mt-2 flex min-h-[21px] flex-wrap items-center gap-1">
      <span
        v-if="group"
        class="inline-flex h-[21px] items-center gap-1.5 rounded-md bg-soft px-1.5 text-[10.5px] font-semibold text-muted"
      >
        <span
          class="h-1 w-1 rounded-full"
          :style="{ background: group.color }"
        ></span
        >{{ group.name }}
      </span>
      <span
        v-for="tag in (connection.tags || []).slice(0, 2)"
        :key="tag"
        class="inline-flex h-[21px] items-center rounded-md bg-soft px-1.5 text-[11px] font-semibold text-faint"
        >{{ tag }}</span
      >
      <span
        v-if="(connection.tags || []).length > 2"
        class="inline-flex h-[21px] items-center rounded-md bg-soft px-1.5 text-[11px] font-semibold text-faint"
        >+{{ connection.tags.length - 2 }}</span
      >
    </div>

    <div class="mt-2 flex min-w-0 items-center gap-3 text-[10.5px] text-faint">
      <span
        class="flex items-center gap-1 whitespace-nowrap"
        :title="probeDescription(connection)"
        ><Wifi :size="12" class="opacity-70" />{{
          probing ? "检测中…" : latencyLabel(connection)
        }}</span
      >
      <span class="flex items-center gap-1 whitespace-nowrap"
        ><Clock3 :size="12" class="opacity-70" />{{
          timeAgo(connection.lastConnected)
        }}</span
      >
      <span
        v-if="credential"
        class="hidden min-w-0 items-center gap-1 sm:flex"
        :title="credential.name"
      >
        <KeyRound :size="12" class="flex-none opacity-70" /><span
          class="truncate"
          >{{ credential.name }}</span
        >
      </span>
    </div>

    <footer
      class="mt-auto flex items-center justify-between gap-2 border-t border-line pt-2.5"
    >
      <div class="flex min-w-0 items-center gap-1.5">
        <span
          class="inline-flex h-[22px] items-center rounded-[5px] border border-line px-1.5 font-mono text-[9.5px] font-bold tracking-[0.04em] text-faint"
          >{{ connection.protocol }}</span
        >
        <template v-if="wol">
          <el-tooltip :show-arrow="false"
            content="通过网络唤醒开机"
            placement="top"
            :show-after="300"
          >
            <button
              type="button"
              class="grid h-10 w-10 place-items-center rounded-md border border-line bg-surface text-faint transition hover:border-line-strong hover:bg-soft hover:text-ink disabled:cursor-not-allowed disabled:opacity-35 sm:h-[27px] sm:w-[27px]"
              :disabled="connection.status === 'online'"
              aria-label="通过网络唤醒开机"
              @click.stop="wakeWolEntry(wol)"
            >
              <Power :size="13" />
            </button>
          </el-tooltip>

        </template>
      </div>
      <button
        type="button"
        class="connection-action inline-flex h-10 items-center rounded-lg px-2.5 text-[11px] font-semibold sm:h-[27px]"
        :data-status="connection.status"
        @click="emit('connect', connection)"
      >
        {{ remoteActionLabel(connection) }}
      </button>
    </footer>
    <div class="native-desktop">
      <CertificateDialog
        v-if="certificateOpen"
        :connection="rawConnection(connection.id)"
        :for-connection="false"
        @close="certificateOpen = false"
      />
    </div>
  </article>
</template>
