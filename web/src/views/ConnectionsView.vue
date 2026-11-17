<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ChevronDown,
  MoreHorizontal,
  BookOpen,
  FolderCog,
  LayoutGrid,
  List,
  Pencil,
  Plus,
  Search,
  SearchX,
  RefreshCw,
  Activity,
  Trash2,
  X,
} from "lucide-vue-next";
import ConnectionPreparation from "../components/ConnectionPreparation.vue";
import ConnectionCard from "../components/ConnectionCard.vue";
import ConnectionFormModal from "../components/ConnectionFormModal.vue";
import DeviceIcon from "../components/DeviceIcon.vue";
import GroupFormModal from "../components/GroupFormModal.vue";
import GroupManageModal from "../components/GroupManageModal.vue";
import PageToolbar from "../components/PageToolbar.vue";
import {
  credentialById,
  groupById,
  groupConnectionCount,
  removeConnection,
  removeGroup,
  state,
  toggleFavorite,
  wolByConnection,
  probeConnection,
  probingConnections,
  connectionStatusLabel,
  latencyLabel,
  probeDescription,
  refreshWorkspace,
} from "../stores/app";
import { launchRemote, remoteActionLabel, requestConfirm, toast } from "../stores/ui";
import { timeAgo } from "../utils/format";
import { compareConnections, connectionSortOptions } from "../connection-sort";
import type { Connection, Group } from "../types";

const route = useRoute();
const router = useRouter();

const search = ref("");
const refreshing = ref(false);
let disposed = false;
onBeforeUnmount(() => { disposed = true; });
onMounted(() => { void refreshConnections(false); });

async function refreshConnections(reload = true) {
  if (refreshing.value) return;
  refreshing.value = true;
  try {
    if (reload) await refreshWorkspace();
    // Snapshot once per visit; polling, filters and view switches must not
    // restart probes. Bound concurrency so large workspaces stay responsive.
    const pending = state.connections.map((connection) => connection.id);
    async function worker() {
      while (!disposed && pending.length) {
        const id = pending.shift()!;
        if (!state.connections.some((connection) => connection.id === id)) continue;
        try {
          await probeConnection(id);
        } catch {
          // Background probes stay silent; a failed target must not interrupt
          // detection of the remaining connections or notify on page entry.
        }
      }
    }
    await Promise.all(Array.from({ length: Math.min(3, pending.length) }, worker));
  } catch (e) {
    if (!disposed) toast((e as Error).message, { tone: "danger" });
  } finally {
    refreshing.value = false;
  }
}
const activeGroup = ref("all");
const statusFilter = ref("all");
const viewMode = ref<"grid" | "list">("grid");
const preparationOpen = ref(false);
const formOpen = ref(false);
const editingConnection = ref<Connection | null>(null);
const presetGroupId = ref("");
const groupFormOpen = ref(false);
const editingGroup = ref<Group | null>(null);
const manageGroupsOpen = ref(false);

const statusOptions = [
  { value: "all", label: "全部状态" },
  { value: "online", label: "端口可达" },
  { value: "busy", label: "会话中" },
  { value: "unknown", label: "未检测" },
  { value: "offline", label: "端口不可达" },
];

const groupTabs = computed(() => {
  const tabs = [
    {
      id: "all",
      name: "全部连接",
      color: "var(--text-3)",
      count: state.connections.length,
    },
    ...state.groups.map((group) => ({
      id: group.id,
      name: group.name,
      color: group.color,
      count: groupConnectionCount(group.id),
    })),
  ];
  const ungrouped = state.connections.filter(
    (connection) => !connection.groupId,
  ).length;
  if (ungrouped)
    tabs.push({
      id: "ungrouped",
      name: "未分组",
      color: "var(--border-strong)",
      count: ungrouped,
    });
  return tabs;
});

// Keep the active group visible while reserving room for the overflow menu.
const visibleGroupTabs = computed(() => {
  const tabs = groupTabs.value.slice(0, 3);
  const selected = groupTabs.value.find((tab) => tab.id === activeGroup.value);
  if (selected && !tabs.some((tab) => tab.id === selected.id)) tabs[2] = selected;
  return tabs;
});
const overflowGroupTabs = computed(() => groupTabs.value.filter(
  (tab) => !visibleGroupTabs.value.some((visible) => visible.id === tab.id),
));
function handleToolbarAction(command: string) {
  if (command === "create-group") openGroupCreate();
  else if (command === "manage-groups") manageGroupsOpen.value = true;
  else if (command === "preparation") preparationOpen.value = true;
}

const filteredConnections = computed(() => {
  const keyword = search.value.trim().toLowerCase();
  return state.connections
    .filter((connection) => {
      if (activeGroup.value === "ungrouped" && connection.groupId) return false;
      if (
        activeGroup.value !== "all" &&
        activeGroup.value !== "ungrouped" &&
        connection.groupId !== activeGroup.value
      )
        return false;
      if (
        statusFilter.value !== "all" &&
        connection.status !== statusFilter.value
      )
        return false;
      if (!keyword) return true;
      return [
        connection.name,
        connection.host,
        connection.os,
        connection.protocol,
        ...(connection.tags || []),
      ]
        .join(" ")
        .toLowerCase()
        .includes(keyword);
    })
    .sort((a, b) => compareConnections(a, b, state.settings.connectionSort));
});

const filtersActive = computed(
  () =>
    Boolean(search.value.trim()) ||
    statusFilter.value !== "all" ||
    activeGroup.value !== "all",
);
const activeGroupName = computed(() => {
  if (activeGroup.value === "all") return null;
  if (activeGroup.value === "ungrouped") return "未分组";
  return groupById(activeGroup.value)?.name || null;
});
const resultLabel = computed(() =>
  filtersActive.value
    ? `找到 ${filteredConnections.value.length} 个连接`
    : `共 ${state.connections.length} 个连接`,
);

function clearQueryKey(key: string) {
  const query = { ...route.query };
  delete query[key];
  router.replace({ query });
}

function setGroup(id: string) {
  activeGroup.value = id;
  if (id === "all") clearQueryKey("group");
  else router.replace({ query: { ...route.query, group: id } });
}

watch(
  () => route.query.group,
  (value) => {
    if (!value) {
      activeGroup.value = "all";
      return;
    }
    if (
      typeof value === "string" &&
      (value === "ungrouped" ||
        state.groups.some((group) => group.id === value))
    )
      activeGroup.value = value;
  },
  { immediate: true },
);

watch(
  () => route.query.new,
  (value) => {
    if (value === "1") {
      openCreate();
      clearQueryKey("new");
    }
  },
  { immediate: true },
);

watch(
  () => route.query.manageGroups,
  (value) => {
    if (value === "1") {
      manageGroupsOpen.value = true;
      clearQueryKey("manageGroups");
    }
  },
  { immediate: true },
);

watch(
  () => route.query.newGroup,
  (value) => {
    if (value === "1") {
      openGroupCreate();
      clearQueryKey("newGroup");
    }
  },
  { immediate: true },
);

function openCreate(groupId = "") {
  editingConnection.value = null;
  presetGroupId.value =
    groupId ||
    (activeGroup.value !== "all" && activeGroup.value !== "ungrouped"
      ? activeGroup.value
      : "");
  formOpen.value = true;
}

function openEdit(connection: Connection) {
  editingConnection.value = connection;
  presetGroupId.value = "";
  formOpen.value = true;
}

function openGroupCreate() {
  editingGroup.value = null;
  groupFormOpen.value = true;
}

function openGroupEdit(group: Group) {
  editingGroup.value = group;
  groupFormOpen.value = true;
}

function handleGroupSaved(group: Group) {
  if (!editingGroup.value) setGroup(group.id);
}

async function removeGroupAndReset(group: Group) {
  const used = groupConnectionCount(group.id);
  const ok = await requestConfirm({
    title: "删除这个分组？",
    description: used
      ? `「${group.name}」中有 ${used} 台设备。删除分组后，这些设备会移动到“未分组”，不会被删除。`
      : `「${group.name}」是一个空分组，删除后无法恢复。`,
    confirmText: "删除分组",
  });
  if (!ok) return;
  await removeGroup(group.id);
  if (activeGroup.value === group.id) setGroup("all");
  toast("分组已删除", {
    description: used ? `${used} 台设备已移到未分组` : group.name,
  });
}

function viewGroupConnections(group: Group) {
  manageGroupsOpen.value = false;
  setGroup(group.id);
}

function createGroupFromManage() {
  manageGroupsOpen.value = false;
  openGroupCreate();
}

function editGroupFromManage(group: Group) {
  manageGroupsOpen.value = false;
  openGroupEdit(group);
}

async function remove(connection: Connection) {
  const ok = await requestConfirm({
    title: "删除这个连接？",
    description: `「${connection.name}」将从工作区移除，已有会话记录会保留。此操作无法撤销。`,
    confirmText: "删除连接",
  });
  if (!ok) return;
  await removeConnection(connection.id);
  toast("连接已删除", { description: connection.name });
}

function favorite(connection: Connection) {
  toggleFavorite(connection.id);
  toast(connection.favorite ? "已添加到收藏" : "已取消收藏", {
    description: connection.name,
    tone: "success",
  });
}

function clearFilters() {
  search.value = "";
  statusFilter.value = "all";
  setGroup("all");
}

async function probe(row: Connection) {
  try {
    const result = await probeConnection(row.id);
    if (result)
      toast(result.message, { tone: result.reachable ? "success" : "danger" });
  } catch (e) {
    toast((e as Error).message, { tone: "danger" });
  }
}
</script>

<template>
  <div class="page-wrap">
    <PageToolbar title="我的连接" />

    <div class="connections-toolbar rounded-[15px] border border-line bg-surface p-2 shadow-card">
      <div class="connection-group-row">
        <div class="connection-group-tabs" role="group" aria-label="连接分组">
          <button
            v-for="tab in visibleGroupTabs" :key="tab.id" type="button"
            class="connection-group-tab"
            :class="activeGroup === tab.id ? 'bg-inset text-ink' : 'text-faint hover:bg-soft hover:text-muted'"
            :aria-pressed="activeGroup === tab.id" :title="tab.name"
            @click="setGroup(tab.id)"
          >
            <span class="h-1 w-1 flex-none rounded-full" :style="{ background: tab.color }"></span>
            <span class="min-w-0 truncate">{{ tab.name }}</span>
            <span class="count-chip flex-none">{{ tab.count }}</span>
          </button>
          <el-dropdown v-if="overflowGroupTabs.length" trigger="click" @command="setGroup" popper-class="connection-group-menu">
            <button type="button" class="connection-group-tab text-muted" aria-label="更多分组">
              更多分组 <ChevronDown :size="14" />
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-for="tab in overflowGroupTabs" :key="tab.id" :command="tab.id">
                  <span class="connection-group-option"><span>{{ tab.name }}</span><span class="count-chip">{{ tab.count }}</span></span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
        <div class="connection-group-picker">
          <el-select :model-value="activeGroup" @change="setGroup" filterable aria-label="选择连接分组" placeholder="选择分组" popper-class="connection-group-menu" :teleported="true">
            <el-option v-for="tab in groupTabs" :key="tab.id" :value="tab.id" :label="tab.name">
              <span class="connection-group-option"><span>{{ tab.name }}</span><span class="count-chip">{{ tab.count }}</span></span>
            </el-option>
          </el-select>
        </div>
        <el-button class="connection-group-manage" @click="manageGroupsOpen = true"><FolderCog :size="14" class="mr-1.5" />管理分组</el-button>
        <el-dropdown trigger="click" @command="handleToolbarAction">
          <button type="button" class="relay-icon-btn" aria-label="更多工具栏操作"><MoreHorizontal :size="16" /></button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="create-group"><Plus :size="14" class="mr-2" />新建分组</el-dropdown-item>
              <el-dropdown-item command="manage-groups"><FolderCog :size="14" class="mr-2" />管理分组</el-dropdown-item>
              <el-dropdown-item command="preparation" divided><BookOpen :size="14" class="mr-2" />连接前准备</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
      <div class="connection-action-row">
          <el-input v-model="search" clearable placeholder="搜索名称、地址或标签" aria-label="搜索连接" class="connection-search">
            <template #prefix><Search :size="14" class="text-faint" /></template>
          </el-input>
          <el-select v-model="statusFilter" class="connection-status" aria-label="按状态筛选">
            <el-option v-for="option in statusOptions" :key="option.value" :label="option.label" :value="option.value" />
          </el-select>
          <div
            class="flex h-10 flex-none items-center gap-0.5 rounded-lg border border-line bg-soft p-0.5 sm:h-8"
          >
            <el-tooltip :show-arrow="false" content="网格视图" placement="top" :show-after="300">
              <button
                type="button"
                class="grid h-10 w-10 place-items-center rounded-md transition sm:h-6 sm:w-6"
                :class="
                  viewMode === 'grid'
                    ? 'bg-surface text-ink shadow-sm'
                    : 'text-faint hover:text-muted'
                "
                aria-label="网格视图"
                :aria-pressed="viewMode === 'grid'"
                @click="viewMode = 'grid'"
              >
                <LayoutGrid :size="14" />
              </button>
            </el-tooltip>
            <el-tooltip :show-arrow="false" content="列表视图" placement="top" :show-after="300">
              <button
                type="button"
                class="grid h-10 w-10 place-items-center rounded-md transition sm:h-6 sm:w-6"
                :class="
                  viewMode === 'list'
                    ? 'bg-surface text-ink shadow-sm'
                    : 'text-faint hover:text-muted'
                "
                aria-label="列表视图"
                :aria-pressed="viewMode === 'list'"
                @click="viewMode = 'list'"
              >
                <List :size="14" />
              </button>
            </el-tooltip>
          </div>
          <el-tooltip :show-arrow="false" content="刷新连接列表" placement="top" :show-after="300">
            <button type="button" class="relay-icon-btn flex-none disabled:opacity-40" aria-label="刷新连接列表" :title="refreshing ? '正在检测连通性和延迟…' : '刷新列表并检测连通性和延迟'" :disabled="refreshing" :aria-busy="refreshing" @click="refreshConnections()">
              <RefreshCw :size="15" :class="{ 'animate-spin': refreshing }" />
            </button>
          </el-tooltip>
          <el-button
            type="primary"
            class="connection-create !h-10 flex-none sm:!h-[30px]"
            @click="openCreate()"
            ><Plus :size="14" class="mr-1" />新建连接</el-button
          >
      </div>
    </div>

    <div class="flex min-h-[34px] flex-wrap items-center justify-between gap-2 px-0.5">
      <h2 class="text-[11px] font-medium text-muted">
        {{ resultLabel
        }}<span v-if="activeGroupName" class="text-faint">
          · {{ activeGroupName }}</span
        >
      </h2>
      <div class="ml-auto flex items-center gap-3">
      <button
        v-if="filtersActive"
        type="button"
        class="flex items-center gap-1 text-[10.5px] font-semibold text-faint hover:text-ink"
        @click="clearFilters"
      >
        <X :size="11" />清除筛选
      </button>
      <el-select
        v-model="state.settings.connectionSort"
        class="connection-sort"
        aria-label="连接排序方式"
        size="small"
      >
        <el-option v-for="option in connectionSortOptions" :key="option.value" :label="option.label" :value="option.value" />
      </el-select>
      </div>
    </div>

    <template v-if="filteredConnections.length">
      <div
        v-if="viewMode === 'grid'"
        class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3"
      >
        <ConnectionCard
          v-for="connection in filteredConnections"
          :key="connection.id"
          :connection="connection"
          :group="groupById(connection.groupId)"
          :credential="credentialById(connection.credentialId)"
          :wol="wolByConnection(connection.id)"
          @connect="launchRemote"
          @edit="openEdit"
          @delete="remove"
          @favorite="favorite"
        />
      </div>

      <div v-else class="panel overflow-hidden">
        <el-table :data="filteredConnections">
          <el-table-column label="设备" min-width="170">
            <template #default="{ row }: { row: Connection }">
              <div class="flex items-center gap-2.5">
                <DeviceIcon :os="row.os" :size="30" />
                <div class="flex min-w-0 flex-col gap-px">
                  <span class="truncate text-[12px] font-semibold text-ink">{{
                    row.name
                  }}</span>
                  <span class="truncate font-mono text-[10.5px] text-faint">{{
                    row.host
                  }}</span>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="78">
            <template #default="{ row }: { row: Connection }">
              <span
                class="flex items-center gap-1.5 text-[11px] font-semibold text-muted"
                ><span
                  class="h-[7px] w-[7px] rounded-full"
                  :class="
                    row.status === 'online'
                      ? 'bg-success'
                      : row.status === 'busy'
                        ? 'bg-warning'
                        : row.status === 'offline'
                          ? 'bg-danger'
                          : 'bg-faint'
                  "
                ></span
                >{{
                  probingConnections.has(row.id)
                    ? "检测中…"
                    : connectionStatusLabel(row.status)
                }}</span
              >
            </template>
          </el-table-column>
          <el-table-column label="分组" min-width="95">
            <template #default="{ row }: { row: Connection }">
              <span class="flex items-center gap-1.5 whitespace-nowrap"
                ><span
                  class="h-1 w-1 rounded-full"
                  :style="{
                    background:
                      groupById(row.groupId)?.color || 'var(--border-strong)',
                  }"
                ></span
                >{{ groupById(row.groupId)?.name || "未分组" }}</span
              >
            </template>
          </el-table-column>
          <el-table-column label="协议" width="66">
            <template #default="{ row }: { row: Connection }">
              <span
                class="inline-flex h-[21px] items-center rounded-[5px] border border-line px-1.5 font-mono text-[10px] font-bold text-faint"
                >{{ row.protocol }}</span
              >
            </template>
          </el-table-column>
          <el-table-column label="TCP 延迟" width="104">
            <template #default="{ row }: { row: Connection }">
              <span class="tabular-nums" :title="probeDescription(row)">{{
                probingConnections.has(row.id) ? "检测中…" : latencyLabel(row)
              }}</span>
            </template>
          </el-table-column>
          <el-table-column label="最近连接" min-width="92">
            <template #default="{ row }: { row: Connection }">{{
              timeAgo(row.lastConnected)
            }}</template>
          </el-table-column>
          <el-table-column width="176" align="right">
            <template #default="{ row }: { row: Connection }">
              <div class="flex items-center justify-end gap-0.5">
                <el-tooltip :show-arrow="false" content="检查 TCP 连通性" placement="top"
                  ><button
                    type="button"
                    class="grid h-7 w-7 place-items-center rounded-md text-faint hover:bg-soft disabled:opacity-40"
                    :disabled="probingConnections.has(row.id)"
                    :aria-label="'检测连接 ' + row.name"
                    @click="probe(row)"
                  >
                    <Activity :size="14" /></button
                ></el-tooltip>
                <el-tooltip :show-arrow="false" content="编辑" placement="top" :show-after="300"
                  ><button
                    type="button"
                    class="grid h-7 w-7 place-items-center rounded-md text-faint hover:bg-soft hover:text-ink"
                    @click="openEdit(row)"
                  >
                    <Pencil :size="14" /></button
                ></el-tooltip>
                <el-tooltip :show-arrow="false" content="删除" placement="top" :show-after="300"
                  ><button
                    type="button"
                    class="grid h-7 w-7 place-items-center rounded-md text-faint hover:bg-danger-soft hover:text-danger"
                    @click="remove(row)"
                  >
                    <Trash2 :size="14" /></button
                ></el-tooltip>
                <button
                  type="button"
                  class="connection-action ml-1 inline-flex h-[29px] items-center whitespace-nowrap rounded-[7px] px-2 text-[10.5px] font-semibold sm:h-[26px]"
                  :data-status="row.status"
                  @click="launchRemote(row)"
                >
                  {{ remoteActionLabel(row) }}
                </button>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </template>

    <div
      v-else
      class="panel flex flex-col items-center gap-2 px-6 py-10 text-center"
    >
      <span
        class="mb-1 grid h-12 w-12 place-items-center rounded-xl border border-line bg-soft text-faint"
        ><SearchX :size="24"
      /></span>
      <h3 class="text-[15px] font-bold text-ink">没有匹配的连接</h3>
      <p class="mb-2 max-w-[350px] text-[11.5px] leading-6 text-faint">
        没有找到符合当前搜索和筛选条件的设备。试试调整关键词或清除筛选。
      </p>
      <div class="flex gap-2">
        <el-button size="small" @click="clearFilters"
          ><X :size="13" class="mr-1" />清除筛选</el-button
        >
        <el-button type="primary" size="small" @click="openCreate()"
          ><Plus :size="13" class="mr-1" />新建连接</el-button
        >
      </div>
    </div>

    <ConnectionPreparation :open="preparationOpen" @close="preparationOpen = false" />
    <ConnectionFormModal
      :open="formOpen"
      :connection="editingConnection"
      :preset-group-id="presetGroupId"
      @close="formOpen = false"
      @saved="formOpen = false"
    />
    <GroupFormModal
      :open="groupFormOpen"
      :group="editingGroup"
      @close="groupFormOpen = false"
      @saved="handleGroupSaved"
    />
    <GroupManageModal
      :open="manageGroupsOpen"
      @close="manageGroupsOpen = false"
      @create="createGroupFromManage"
      @edit="editGroupFromManage"
      @delete="removeGroupAndReset"
      @view="viewGroupConnections"
    />
  </div>
</template>

<style>
.connections-toolbar { container-type: inline-size; container-name: connection-tools; }
.connection-group-row { display: flex; align-items: center; gap: 8px; padding-bottom: 8px; border-bottom: 1px solid var(--border); }
.connection-group-tabs { display: flex; align-items: center; gap: 4px; min-width: 0; flex: 1; }
.connection-group-tab { display: inline-flex; align-items: center; gap: 6px; height: 34px; max-width: 160px; padding: 0 10px; border-radius: 8px; font-size: 13px; font-weight: 600; white-space: nowrap; }
.connection-group-picker { display: none; min-width: 0; flex: 1; }
.connection-group-picker .el-select { width: 100%; }
.connection-group-picker .el-select__input:focus-visible { outline: none; }
.connection-group-picker .el-select__wrapper.is-focused {
  box-shadow: 0 0 0 1px var(--border-strong) inset !important;
}
.connection-action-row { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; padding-top: 8px; }
.connection-search { flex: 1; min-width: 160px; }
.connection-sort { width: 148px; flex: none; }
.connection-status { width: 116px; flex: none; }
.connection-group-menu { max-width: calc(100vw - 32px); }
.connection-group-menu .el-select-dropdown__item,
.connection-group-menu .el-dropdown-menu__item { height: auto; min-height: 36px; white-space: normal; }
.connection-group-menu .el-dropdown-menu { max-height: min(320px, 40dvh); overflow-y: auto; }
.connection-group-menu .el-select-dropdown__wrap { max-height: min(274px, 40dvh); }
.connection-group-option { display: flex; align-items: center; justify-content: space-between; gap: 12px; width: 100%; padding: 6px 0; line-height: 1.5; }
.connection-group-option > span:first-child { min-width: 0; overflow-wrap: anywhere; white-space: normal; }
.connection-group-option .count-chip { flex: none; }
@container connection-tools (max-width: 760px) {
  .connection-group-tabs, .connection-group-manage { display: none; }
  .connection-group-picker { display: block; }
}
@container connection-tools (max-width: 460px) {
  .connection-search { flex-basis: 100%; }
  .connection-create { margin-left: auto; }
}
@container connection-tools (max-width: 340px) {
  .connection-create { width: 100%; margin-left: 0; }
}
</style>
