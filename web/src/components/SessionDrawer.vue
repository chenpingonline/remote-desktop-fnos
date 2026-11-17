<script setup lang="ts">
import { computed } from 'vue'
import {
  ArrowUpRight,
  CalendarDays,
  Check,
  CircleSlash2,
  Clock3,
  Copy,
  KeyRound,
  LockKeyhole,
  Monitor,
  ShieldCheck,
  Timer,
  Trash2,
  Zap,
} from 'lucide-vue-next'
import DeviceIcon from './DeviceIcon.vue'
import { connectionById } from '../stores/app'
import { sessionQualityDetail, sessionQualityLabel } from '../api'
import { formatDateTime, formatDuration, timeAgo } from '../utils/format'
import { toast } from '../stores/ui'
import type { Session } from '../types'

const props = defineProps<{
  open: boolean
  session: Session | null
}>()

const emit = defineEmits<{
  close: []
  reconnect: [session: Session]
  delete: [session: Session]
}>()

const visible = computed({
  get: () => props.open,
  set: (value: boolean) => {
    if (!value) emit('close')
  },
})

const statusLabel = computed(() => {
  if (!props.session) return ''
  return props.session.status === 'completed' ? '已完成' : props.session.status === 'interrupted' ? '已中断' : '进行中'
})

const deviceOs = computed(() => {
  if (!props.session) return 'Linux'
  return connectionById(props.session.connectionId)?.os || (props.session.protocol === 'RDP' ? 'Windows' : 'Linux')
})

const endedAt = computed(() => {
  if (!props.session) return null
  return new Date(new Date(props.session.startedAt).getTime() + props.session.duration * 1000).toISOString()
})

const statusTone = computed(() => {
  if (!props.session) return 'text-muted'
  return props.session.status === 'completed' ? 'text-success' : props.session.status === 'interrupted' ? 'text-danger' : 'text-info'
})

const statusDot = computed(() => {
  if (!props.session) return 'bg-faint'
  return props.session.status === 'completed' ? 'bg-success' : props.session.status === 'interrupted' ? 'bg-danger' : 'bg-info'
})

function copyId() {
  const sessionId = props.session?.id
  if (!sessionId) return
  navigator.clipboard
    ?.writeText(sessionId)
    .then(() => toast('会话 ID 已复制', { description: sessionId, tone: 'success' }))
    .catch(() => toast('复制失败', { tone: 'danger' }))
}
</script>

<template>
  <el-drawer v-model="visible" size="min(420px, 100vw)" class="session-drawer max-w-full" append-to-body>
    <template #header>
      <div>
        <p class="mb-0.5 text-[10px] font-bold uppercase tracking-[0.14em] text-faint">Session detail</p>
        <h2 class="text-[17px] font-bold tracking-[-0.03em] text-ink">会话详情</h2>
      </div>
    </template>

    <div v-if="session">
      <div class="rounded-[13px] border border-line bg-soft p-3.5">
        <div class="flex items-start gap-2.5">
          <DeviceIcon :os="deviceOs" :size="40" />
          <div class="min-w-0 flex-1 pt-0.5">
            <h3 class="truncate text-[14px] font-bold text-ink">{{ session.connectionName }}</h3>
            <p class="mt-0.5 truncate font-mono text-[10.5px] text-faint">{{ session.host }}</p>
          </div>
          <span class="inline-flex items-center gap-1.5 rounded-md bg-surface px-2 py-1 text-[10px] font-semibold" :class="statusTone">
            <span class="h-1 w-1 rounded-full" :class="statusDot"></span>{{ statusLabel }}
          </span>
        </div>
        <div class="mt-2.5 flex flex-wrap items-center gap-3 border-t border-line pt-2 text-[10px] text-faint">
          <span class="flex items-center gap-1"><LockKeyhole :size="11" /> NAS 会话记录</span>
          <span class="flex items-center gap-1" :title="sessionQualityDetail(session.quality)"><Monitor :size="11" /> {{ sessionQualityLabel(session.quality) }}</span>
          <span>{{ timeAgo(session.startedAt) }}</span>
        </div>
      </div>

      <div class="mt-3 grid grid-cols-2 gap-px overflow-hidden rounded-[12px] border border-line bg-line">
        <div class="flex flex-col gap-1 bg-surface px-3 py-2.5">
          <span class="flex items-center gap-1.5 text-[10px] text-faint"><CalendarDays :size="12" />开始时间</span>
          <strong class="text-[11px] font-semibold text-ink">{{ formatDateTime(session.startedAt) }}</strong>
        </div>
        <div class="flex flex-col gap-1 bg-surface px-3 py-2.5">
          <span class="flex items-center gap-1.5 text-[10px] text-faint"><Clock3 :size="12" />结束时间</span>
          <strong class="text-[11px] font-semibold text-ink">{{ session.status === 'active' ? '进行中' : formatDateTime(endedAt) }}</strong>
        </div>
        <div class="flex flex-col gap-1 bg-surface px-3 py-2.5">
          <span class="flex items-center gap-1.5 text-[10px] text-faint"><Timer :size="12" />会话时长</span>
          <strong class="text-[11px] font-semibold text-ink">{{ formatDuration(session.duration) }}</strong>
        </div>
        <div class="flex flex-col gap-1 bg-surface px-3 py-2.5">
          <span class="flex items-center gap-1.5 text-[10px] text-faint"><Zap :size="12" />连接协议</span>
          <strong class="text-[11px] font-semibold text-ink">{{ session.protocol }}</strong>
        </div>
      </div>

      <section class="mt-4">
        <div class="mb-2 flex items-baseline justify-between">
          <h4 class="text-[11.5px] font-bold text-ink">会话时间线</h4>
          <span class="text-[9px] font-bold tracking-[0.13em] text-faint">SESSION TIMELINE</span>
        </div>
        <div class="pl-1">
          <div v-for="(step, index) in [
            { title: '发起远程连接', detail: formatDateTime(session.startedAt, true), icon: Check, tone: 'success' },
            { title: session.status === 'active' ? '会话进行中' : session.status === 'interrupted' ? '连接意外中断' : '会话正常结束', detail: session.status === 'active' ? '持续时间实时计算' : formatDateTime(endedAt, true), icon: session.status === 'interrupted' ? CircleSlash2 : session.status === 'active' ? Clock3 : Check, tone: session.status === 'interrupted' ? 'danger' : 'success' },
          ]" :key="index" class="relative flex gap-2.5 pb-3 last:pb-0">
            <span v-if="index < 1" class="absolute bottom-0 left-[8px] top-[18px] w-px bg-line"></span>
            <span
              class="relative z-[1] grid h-[17px] w-[17px] flex-none place-items-center rounded-full border"
              :class="step.tone === 'success' ? 'border-line bg-success-soft text-success' : step.tone === 'danger' ? 'border-line bg-danger-soft text-danger' : 'border-line bg-info-soft text-info'"
            >
              <component :is="step.icon" :size="9" />
            </span>
            <div>
              <p class="text-[10.5px] font-semibold text-muted">{{ step.title }}</p>
              <span class="mt-0.5 block text-[10px] text-faint">{{ step.detail }}</span>
            </div>
          </div>
        </div>
      </section>

      <section class="mt-4">
        <div class="mb-2 flex items-baseline justify-between">
          <h4 class="text-[11.5px] font-bold text-ink">会话信息</h4>
          <span class="text-[9px] font-bold tracking-[0.13em] text-faint">METADATA</span>
        </div>
        <div class="border-t border-line">
          <div class="flex min-h-[32px] items-center gap-2.5 border-b border-line">
            <span class="w-[75px] flex-none text-[10.5px] text-faint">会话 ID</span>
            <code class="min-w-0 flex-1 truncate text-right font-mono text-[10.5px] text-muted">{{ session.id }}</code>
            <button type="button" class="grid h-[22px] w-[22px] place-items-center rounded-[5px] text-faint hover:bg-soft hover:text-ink" aria-label="复制会话 ID" @click="copyId"><Copy :size="12" /></button>
          </div>
          <div class="flex min-h-[32px] items-center gap-2.5 border-b border-line"><span class="w-[75px] flex-none text-[10.5px] text-faint">操作者</span><strong class="flex-1 text-right text-[10.5px] font-medium text-muted">{{ session.user }}</strong></div>
          <div class="flex min-h-[32px] items-center gap-2.5 border-b border-line"><span class="w-[75px] flex-none text-[10.5px] text-faint">远程地址</span><code class="min-w-0 flex-1 truncate text-right font-mono text-[10.5px] text-muted">{{ session.host }}</code></div>
          <div class="flex min-h-[32px] items-center gap-2.5 border-b border-line"><span class="w-[75px] flex-none text-[10.5px] text-faint">状态 / 结束原因</span><strong class="flex-1 truncate text-right text-[10.5px] font-medium text-muted" :title="sessionQualityDetail(session.quality)">{{ sessionQualityLabel(session.quality) }}</strong></div>
        </div>
      </section>

      <div class="mt-4 flex items-start gap-1.5 rounded-lg border border-line bg-soft px-2.5 py-2 text-[10px] leading-5 text-faint">
        <KeyRound :size="13" class="mt-0.5 flex-none" />
        此处仅展示后端记录的创建时间、状态和结束原因；不推断远端认证或加密算法。
      </div>
    </div>

    <template v-if="session" #footer>
      <div class="flex w-full gap-2">
        <el-button disabled title="后端保留记录，暂不支持删除" class="!text-danger" @click="emit('delete', session)"><Trash2 :size="14" class="mr-1.5" />删除记录</el-button>
        <el-button type="primary" class="ml-auto" @click="emit('reconnect', session)"><ArrowUpRight :size="14" class="mr-1.5" />重新连接</el-button>
      </div>
    </template>
  </el-drawer>
</template>
