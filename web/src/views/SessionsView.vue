<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  Activity,
  ArrowUpRight,
  Clock3,
  Download,
  History,
  Search,
  ShieldCheck,
  Timer,
  Trash2,
} from 'lucide-vue-next'
import DeviceIcon from '../components/DeviceIcon.vue'
import PageToolbar from '../components/PageToolbar.vue'
import SessionDrawer from '../components/SessionDrawer.vue'
import { clearSessions, connectionById, removeSession, state, stats } from '../stores/app'
import { sessionQualityDetail, sessionQualityLabel } from '../api'
import { launchRemote, requestConfirm, toast } from '../stores/ui'
import { formatDateTime, formatDuration, timeAgo } from '../utils/format'
import type { Session } from '../types'

const search = ref('')
const statusFilter = ref('all')
const rangeFilter = ref('all')
const drawerOpen = ref(false)
const selectedId = ref<string | null>(null)

const tableHost = ref<HTMLElement | null>(null)
// 列的最小内容宽度；表格铺满时把多余宽度平均分给所有列，避免单列被拉宽
const COLUMN_BASE = [180, 68, 140, 100, 80, 100, 48]
const columnWidths = ref<number[]>([...COLUMN_BASE])
let tableObserver: ResizeObserver | null = null

function distributeColumns(total: number) {
  if (!total) return
  const base = COLUMN_BASE
  const minimum = base.reduce((sum, width) => sum + width, 0)
  const extra = Math.max(0, total - minimum)
  const share = Math.floor(extra / base.length)
  let remainder = extra - share * base.length
  columnWidths.value = base.map((width) => {
    const bonus = remainder > 0 ? 1 : 0
    remainder -= bonus
    return width + share + bonus
  })
}

onMounted(() => {
  const host = tableHost.value
  if (!host || typeof ResizeObserver === 'undefined') return
  tableObserver = new ResizeObserver((entries) => {
    distributeColumns(Math.floor(entries[0].contentRect.width))
  })
  tableObserver.observe(host)
  distributeColumns(host.clientWidth)
})

onBeforeUnmount(() => tableObserver?.disconnect())

const statusOptions = [
  { value: 'all', label: '全部状态' },
  { value: 'completed', label: '已完成' },
  { value: 'interrupted', label: '已中断' },
  { value: 'active', label: '进行中' },
]

const rangeOptions = [
  { value: 'all', label: '全部时间' },
  { value: '7', label: '近 7 天' },
  { value: '30', label: '近 30 天' },
]

const filteredSessions = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  const now = Date.now()
  const days = rangeFilter.value === 'all' ? null : Number(rangeFilter.value)
  return [...state.sessions]
    .filter((session) => {
      if (statusFilter.value !== 'all' && session.status !== statusFilter.value) return false
      if (days && now - new Date(session.startedAt).getTime() > days * 24 * 60 * 60 * 1000) return false
      if (!keyword) return true
      return [session.connectionName, session.host, session.protocol, session.user, session.id].join(' ').toLowerCase().includes(keyword)
    })
    .sort((a, b) => new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime())
})

const selectedSession = computed(() => state.sessions.find((session) => session.id === selectedId.value) || null)
const averageMinutes = computed(() => {
  const completed = state.sessions.filter((session) => session.status !== 'active')
  if (!completed.length) return 0
  return Math.round(completed.reduce((sum, item) => sum + item.duration, 0) / completed.length / 60)
})
const thisWeek = computed(() => {
  const since = Date.now() - 7 * 24 * 60 * 60 * 1000
  return state.sessions.filter((session) => new Date(session.startedAt).getTime() >= since).length
})
const activeCount = computed(() => state.sessions.filter((session) => session.status === 'active').length)

function openDetail(session: Session) {
  selectedId.value = session.id
  drawerOpen.value = true
}

function statusLabel(status: string) {
  if (status === 'completed') return '已完成'
  if (status === 'interrupted') return '已中断'
  return '进行中'
}

function statusTagType(status: string) {
  if (status === 'completed') return 'success'
  if (status === 'interrupted') return 'danger'
  return 'info'
}

function reconnect(session: Session) {
  const existing = connectionById(session.connectionId)
  drawerOpen.value = false
  launchRemote(existing || {
    id: null,
    name: session.connectionName,
    host: session.host,
    protocol: session.protocol,
    os: 'Linux',
    osVersion: '',
    latency: null,
    status: 'online',
  })
}

function remove(_: Session) { toast('后端保留会话记录 90 天，暂不支持手动删除', {tone:'info'}) }
function clearAll() { toast('后端保留会话记录 90 天，暂不支持手动清空', {tone:'info'}) }

function exportCsv() {
  if (!filteredSessions.value.length) {
    toast('没有可导出的会话记录', { tone: 'danger' })
    return
  }
  const header = ['会话 ID', '设备', '主机地址', '协议', '开始时间', '时长（秒）', '状态', '会话状态']
  const rows = filteredSessions.value.map((session) => [
    session.id,
    session.connectionName,
    session.host,
    session.protocol,
    formatDateTime(session.startedAt, true),
    session.duration,
    statusLabel(session.status),
    session.quality,
  ])
  const escape = (value: unknown) => `"${String(value).replace(/"/g, '""')}"`
  const csv = `\ufeff${[header, ...rows].map((row) => row.map(escape).join(',')).join('\n')}`
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `relay-sessions-${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
  toast('会话记录已导出', { description: `${filteredSessions.value.length} 条记录 · CSV`, tone: 'success' })
}

function resetFilters() {
  search.value = ''
  statusFilter.value = 'all'
  rangeFilter.value = 'all'
}
</script>

<template>
  <div class="page-wrap">
    <PageToolbar title="会话记录" />

    <section class="mb-3 grid grid-cols-2 gap-2 xl:grid-cols-4">
      <article class="flex min-h-[60px] items-center gap-2.5 rounded-[13px] border border-line bg-surface px-3 py-2.5 shadow-card">
        <span class="grid h-[30px] w-[30px] flex-none place-items-center rounded-[8px] border border-line bg-soft text-faint"><History :size="16" /></span>
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <span class="text-[10.5px] text-faint">全部会话</span>
          <strong class="text-[18px] font-semibold leading-none tracking-[-0.04em] text-ink">{{ String(state.sessions.length).padStart(2, '0') }}<small class="ml-1 text-[10px] font-normal text-faint">次</small></strong>
        </div>
        <span class="hidden self-start text-[10px] text-faint xl:block">本周 +{{ thisWeek }}</span>
      </article>
      <article class="flex min-h-[60px] items-center gap-2.5 rounded-[13px] border border-line bg-surface px-3 py-2.5 shadow-card">
        <span class="grid h-[30px] w-[30px] flex-none place-items-center rounded-[8px] border border-line bg-soft text-faint"><Timer :size="16" /></span>
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <span class="text-[10.5px] text-faint">累计连接时长</span>
          <strong class="text-[18px] font-semibold leading-none tracking-[-0.04em] text-ink">{{ (stats.totalSeconds / 3600).toFixed(1) }}<small class="ml-1 text-[10px] font-normal text-faint">小时</small></strong>
        </div>
        <span class="hidden self-start text-[10px] text-faint xl:block">{{ state.sessions.length }} 次会话</span>
      </article>
      <article class="flex min-h-[60px] items-center gap-2.5 rounded-[13px] border border-line bg-surface px-3 py-2.5 shadow-card">
        <span class="grid h-[30px] w-[30px] flex-none place-items-center rounded-[8px] border border-line bg-soft text-faint"><ShieldCheck :size="16" /></span>
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <span class="text-[10.5px] text-faint">正常结束比例</span>
          <strong class="text-[18px] font-semibold leading-none tracking-[-0.04em] text-ink">{{ stats.successRate }}<small class="ml-1 text-[10px] font-normal text-faint">%</small></strong>
        </div>
        <span class="hidden self-start rounded bg-success-soft px-1.5 py-0.5 text-[10px] text-success xl:block">后端记录</span>
      </article>
      <article class="flex min-h-[60px] items-center gap-2.5 rounded-[13px] border border-line bg-surface px-3 py-2.5 shadow-card">
        <span class="grid h-[30px] w-[30px] flex-none place-items-center rounded-[8px] border border-line bg-soft text-faint"><Activity :size="16" /></span>
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <span class="text-[10.5px] text-faint">平均会话时长</span>
          <strong class="text-[18px] font-semibold leading-none tracking-[-0.04em] text-ink">{{ averageMinutes }}<small class="ml-1 text-[10px] font-normal text-faint">分钟</small></strong>
        </div>
        <span class="hidden self-start text-[10px] text-faint xl:block">{{ activeCount }} 个进行中</span>
      </article>
    </section>

    <section class="panel overflow-hidden">
      <div class="flex flex-col gap-2 border-b border-line p-3 xl:flex-row xl:items-center xl:justify-between">
        <div class="flex items-center gap-2">
          <History :size="16" class="text-faint" />
          <h2 class="text-[13.5px] font-bold text-ink">全部会话</h2>
          <span class="count-chip">{{ filteredSessions.length }}</span>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <el-button class="!h-10 flex-none sm:!h-[34px]" disabled title="后端保留记录 90 天，暂不支持清空" @click="clearAll"><Trash2 :size="14" class="mr-1" />清除记录</el-button>
          <el-button type="primary" class="!h-10 flex-none sm:!h-[34px]" @click="exportCsv"><Download :size="14" class="mr-1" />导出 CSV</el-button>
          <el-input v-model="search" clearable placeholder="搜索设备、地址或 ID" aria-label="搜索会话" class="min-w-[160px] flex-1 sm:!w-[200px] sm:flex-none">
            <template #prefix><Search :size="14" class="text-faint" /></template>
          </el-input>
          <el-select v-model="statusFilter" class="flex-none sm:!w-[100px]" aria-label="按状态筛选">
            <el-option v-for="option in statusOptions" :key="option.value" :label="option.label" :value="option.value" />
          </el-select>
          <el-select v-model="rangeFilter" class="flex-none sm:!w-[100px]" aria-label="按时间筛选">
            <el-option v-for="option in rangeOptions" :key="option.value" :label="option.label" :value="option.value" />
          </el-select>
        </div>
      </div>

      <div ref="tableHost" class="hidden md:block">
      <el-table :data="filteredSessions" class="session-table" @row-click="openDetail">
        <el-table-column label="设备" :width="columnWidths[0]">
          <template #default="{ row }: { row: Session }">
            <div class="flex items-center gap-2.5">
              <DeviceIcon :os="connectionById(row.connectionId)?.os || (row.protocol === 'RDP' ? 'Windows' : 'Linux')" :size="30" />
              <div class="flex min-w-0 flex-col gap-px">
                <span class="truncate text-[12px] font-semibold text-ink">{{ row.connectionName }}</span>
                <span class="truncate font-mono text-[10.5px] text-faint">{{ row.host }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="协议" :width="columnWidths[1]">
          <template #default="{ row }: { row: Session }">
            <span class="inline-flex h-[21px] items-center rounded-[5px] border border-line px-1.5 font-mono text-[10px] font-bold text-faint">{{ row.protocol }}</span>
          </template>
        </el-table-column>
        <el-table-column label="开始时间" :width="columnWidths[2]">
          <template #default="{ row }: { row: Session }">
            <div class="flex flex-col gap-px">
              <span class="whitespace-nowrap text-[11px] text-muted">{{ formatDateTime(row.startedAt) }}</span>
              <small class="text-[10.5px] text-faint">{{ timeAgo(row.startedAt) }}</small>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="会话时长" :width="columnWidths[3]">
          <template #default="{ row }: { row: Session }">
            <span class="whitespace-nowrap text-[11px] text-muted">{{ row.status === 'active' ? '进行中' : formatDuration(row.duration) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" :width="columnWidths[4]">
          <template #default="{ row }: { row: Session }">
            <el-tag :type="statusTagType(row.status)" size="small" effect="light" round>{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="会话状态" :width="columnWidths[5]">
          <template #default="{ row }: { row: Session }">
            <span
              class="block truncate text-[11px] text-faint"
              :title="sessionQualityDetail(row.quality)"
              >{{ sessionQualityLabel(row.quality) }}</span
            >
          </template>
        </el-table-column>
        <el-table-column :width="columnWidths[6]" align="right">
          <template #header><span class="sr-only">操作</span></template>
          <template #default="{ row }: { row: Session }">
            <el-tooltip :show-arrow="false" content="查看会话详情" placement="top" :show-after="350">
              <button type="button" class="grid h-7 w-7 place-items-center rounded-md border border-transparent text-faint transition hover:border-line hover:bg-surface hover:text-ink" aria-label="查看会话详情" @click.stop="openDetail(row)">
                <ArrowUpRight :size="15" />
              </button>
            </el-tooltip>
          </template>
        </el-table-column>
        <template #empty>
          <el-empty description="没有匹配的会话">
            <el-button size="small" @click="resetFilters">重置筛选</el-button>
          </el-empty>
        </template>
      </el-table>
      </div>

      <div class="divide-y divide-line md:hidden">
        <button
          v-for="session in filteredSessions"
          :key="session.id"
          type="button"
          class="flex w-full items-start gap-3 px-3 py-3 text-left transition hover:bg-soft"
          @click="openDetail(session)"
        >
          <DeviceIcon :os="connectionById(session.connectionId)?.os || (session.protocol === 'RDP' ? 'Windows' : 'Linux')" :size="38" />
          <span class="flex min-w-0 flex-1 flex-col gap-1.5">
            <span class="flex items-center justify-between gap-2">
              <span class="truncate text-[12px] font-semibold text-ink">{{ session.connectionName }}</span>
              <el-tag :type="statusTagType(session.status)" size="small" effect="light" round>{{ statusLabel(session.status) }}</el-tag>
            </span>
            <span class="truncate font-mono text-[10px] text-faint">{{ session.host }}</span>
            <span class="flex flex-wrap items-center gap-x-3 gap-y-1 text-[10px] text-muted">
              <span class="flex items-center gap-1"><Clock3 :size="11" />{{ formatDateTime(session.startedAt) }}</span>
              <span class="flex items-center gap-1"><Timer :size="11" />{{ session.status === 'active' ? '进行中' : formatDuration(session.duration) }}</span>
              <span class="rounded border border-line px-1.5 font-mono text-[9px] font-bold text-faint">{{ session.protocol }}</span>
            </span>
          </span>
          <ArrowUpRight :size="16" class="mt-1 flex-none text-faint" />
        </button>
        <el-empty v-if="!filteredSessions.length" description="没有匹配的会话">
          <el-button size="small" @click="resetFilters">重置筛选</el-button>
        </el-empty>
      </div>

      <footer class="flex min-h-[34px] flex-wrap items-center gap-2 border-t border-line bg-soft px-3 text-[10.5px] text-faint">
        <span>显示 {{ filteredSessions.length }} 条记录</span>
        <span class="h-0.5 w-0.5 rounded-full bg-line-strong"></span>
        <span>会话记录默认保留 90 天</span>
        <span class="ml-auto flex items-center gap-1"><Clock3 :size="11" /> 时间以本地时区显示</span>
      </footer>
    </section>

    <SessionDrawer :open="drawerOpen" :session="selectedSession" @close="drawerOpen = false" @reconnect="reconnect" @delete="remove" />
  </div>
</template>
