<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Check,
  Clock3,
  Copy,
  MoreHorizontal,
  Pencil,
  Plus,
  Power,
  Search,
  Trash2,
  Zap,
} from 'lucide-vue-next'
import PageToolbar from '../components/PageToolbar.vue'
import WolFormModal from '../components/WolFormModal.vue'
import { connectionById, removeWol, state } from '../stores/app'
import { requestConfirm, toast, wakeWolEntry } from '../stores/ui'
import { timeAgo } from '../utils/format'
import type { Connection, WolEntry } from '../types'

const route = useRoute()
const router = useRouter()

const search = ref('')
const formOpen = ref(false)
const editingEntry = ref<WolEntry | null>(null)
const copiedId = ref<string | null>(null)

const filteredWol = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return state.wol
  return state.wol.filter((entry) => {
    const connection = connectionById(entry.connectionId)
    return [entry.name, entry.mac, entry.broadcast, connection?.name || ''].join(' ').toLowerCase().includes(keyword)
  })
})

function connectionFor(entry: WolEntry): Connection | null {
  return connectionById(entry.connectionId)
}

function statusLabel(status: string): string {
  if (status === 'online') return '端口可达'
  if (status === 'busy') return '会话中'
  if (status === 'offline') return '端口不可达'
  return '未检测'
}

function statusDot(status: string): string {
  if (status === 'online') return 'bg-success'
  if (status === 'busy') return 'bg-warning'
  if (status === 'offline') return 'bg-danger'
  return 'bg-faint'
}

function clearQueryKey(key: string) {
  const query = { ...route.query }
  delete query[key]
  router.replace({ query })
}

function openCreate() {
  editingEntry.value = null
  formOpen.value = true
}

function openEdit(entry: WolEntry) {
  editingEntry.value = entry
  formOpen.value = true
}

async function remove(entry: WolEntry) {
  const ok = await requestConfirm({
    title: '删除这个唤醒项？',
    description: `「${entry.name}」的 WOL 配置将被删除，关联的远程连接不会被删除。`,
    confirmText: '删除唤醒项',
  })
  if (!ok) return
  await removeWol(entry.id)
  toast('唤醒项已删除', { description: entry.name })
}

function copyMac(entry: WolEntry) {
  navigator.clipboard
    ?.writeText(entry.mac)
    .then(() => {
      copiedId.value = entry.id
      toast('MAC 地址已复制', { description: entry.mac, tone: 'success' })
      window.setTimeout(() => {
        if (copiedId.value === entry.id) copiedId.value = null
      }, 1600)
    })
    .catch(() => toast('复制失败，请手动选择地址', { tone: 'danger' }))
}

function handleCommand(command: string, entry: WolEntry) {
  if (command === 'edit') openEdit(entry)
  if (command === 'delete') remove(entry)
}

watch(
  () => route.query.new,
  (value) => {
    if (value === '1') {
      openCreate()
      clearQueryKey('new')
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="page-wrap">
    <PageToolbar title="网络唤醒" />

    <div class="mb-2.5 flex items-center gap-2 rounded-[15px] border border-line bg-surface p-2 shadow-card">
      <el-input v-model="search" clearable placeholder="搜索名称、MAC 或关联设备" aria-label="搜索唤醒项" class="min-w-[140px] flex-1">
        <template #prefix><Search :size="14" class="text-faint" /></template>
      </el-input>
      <span class="hidden items-center gap-1 whitespace-nowrap text-[10px] text-faint sm:flex"><Zap :size="12" />魔术包 · UDP 9</span>
      <el-button type="primary" class="!h-10 flex-none sm:!h-[30px]" @click="openCreate"><Plus :size="14" class="mr-1" />新建唤醒</el-button>
    </div>

    <div v-if="filteredWol.length" class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="entry in filteredWol"
        :key="entry.id"
        class="group relative flex min-h-[168px] flex-col rounded-2xl border border-line bg-surface p-3.5 device-card-motion shadow-card"
      >
        <div class="flex items-start gap-2.5">
          <span class="grid h-[38px] w-[38px] flex-none place-items-center rounded-xl border border-line bg-soft text-ink"><Power :size="18" /></span>
          <div class="flex min-w-0 flex-1 flex-col gap-0.5 pt-0.5">
            <h3 class="truncate text-[13px] font-bold tracking-[-0.015em] text-ink">{{ entry.name }}</h3>
            <button type="button" class="flex max-w-full items-center gap-1 text-left font-mono text-[10.5px] text-faint hover:text-muted" title="复制 MAC 地址" @click="copyMac(entry)">
              <span class="truncate">{{ entry.mac }}</span>
              <Check v-if="copiedId === entry.id" :size="11" class="flex-none" />
              <Copy v-else :size="11" class="flex-none opacity-0 transition group-hover:opacity-100" />
            </button>
          </div>
          <el-dropdown trigger="click" placement="bottom-end" @command="(command: string) => handleCommand(command, entry)">
            <button type="button" class="grid h-10 w-10 place-items-center rounded-md text-faint transition hover:bg-soft hover:text-ink sm:h-7 sm:w-7" aria-label="更多操作">
              <MoreHorizontal :size="16" />
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="edit"><span class="flex items-center gap-2"><Pencil :size="13" /> 编辑唤醒项</span></el-dropdown-item>
                <el-dropdown-item command="delete" class="!text-danger"><span class="flex items-center gap-2"><Trash2 :size="13" /> 删除唤醒项</span></el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <div class="mt-2.5 flex flex-wrap items-center gap-1.5 text-[10.5px] text-faint">
          <span class="meta-pill">{{ entry.broadcast }}:{{ entry.port }}</span>
          <span v-if="connectionFor(entry)" class="inline-flex min-w-0 items-center gap-1.5">
            <span class="h-[6px] w-[6px] flex-none rounded-full" :class="statusDot(connectionFor(entry)!.status)"></span>
            <span class="truncate">{{ connectionFor(entry)!.name }} · {{ statusLabel(connectionFor(entry)!.status) }}</span>
          </span>
          <span v-else>未关联远程连接</span>
        </div>

        <div class="mt-1.5 flex items-center gap-1 text-[10px] text-faint">
          <Clock3 :size="11" />
          {{ entry.lastWokenAt ? `上次发送唤醒包 ${timeAgo(entry.lastWokenAt)}` : '暂无唤醒包发送记录' }}
        </div>

        <footer class="mt-auto flex items-center justify-between gap-2 border-t border-line pt-2.5">
          <span class="inline-flex h-[22px] items-center rounded-[5px] border border-line px-1.5 font-mono text-[9.5px] font-bold tracking-[0.04em] text-faint">WOL</span>
          <div class="flex items-center gap-1.5">
            <button
              type="button"
              class="inline-flex h-10 items-center gap-1 rounded-lg bg-soft px-2.5 text-[11px] font-semibold text-muted transition hover:bg-ink hover:text-canvas disabled:cursor-not-allowed disabled:opacity-40 sm:h-[27px]"
              :disabled="connectionFor(entry)?.status === 'online'"
              :aria-label="`通过网络唤醒 ${entry.name}`"
              :title="connectionFor(entry)?.status === 'online' ? '设备端口可达，无需再次唤醒' : '发送唤醒包，开机结果需另行确认'"
              @click="wakeWolEntry(entry)"
            ><Power :size="13" />开机</button>

          </div>
        </footer>
      </article>
    </div>

    <div v-else class="panel flex flex-col items-center gap-2 px-6 py-10 text-center">
      <span class="mb-1 grid h-12 w-12 place-items-center rounded-xl border border-line bg-soft text-faint"><Zap :size="22" /></span>
      <h3 class="text-[14px] font-bold text-ink">{{ state.wol.length ? '没有匹配的唤醒项' : '还没有网络唤醒项' }}</h3>
      <p class="mb-2 max-w-[360px] text-[11px] leading-5 text-faint">
        {{ state.wol.length ? '试试其他关键词，或清除搜索后重试。' : '填写设备的 MAC 地址，就能在关机状态下通过魔术包远程开机；也可以在新建远程连接时勾选 WOL。' }}
      </p>
      <el-button v-if="state.wol.length" size="small" @click="search = ''">清除搜索</el-button>
      <el-button v-else type="primary" size="small" @click="openCreate"><Plus :size="13" class="mr-1" />新建唤醒项</el-button>
    </div>

    <div class="mt-2.5 flex items-start gap-1.5 px-0.5 text-[10px] text-faint">
      <Zap :size="11" class="mt-0.5 flex-none" />
      魔术包需要目标设备在 BIOS / 网卡中启用 Wake-on-LAN，并通过同一局域网广播；由 NAS 发送唤醒请求，设备开机状态需另行确认。
    </div>

    <WolFormModal :open="formOpen" :entry="editingEntry" @close="formOpen = false" @saved="formOpen = false" />
  </div>
</template>
