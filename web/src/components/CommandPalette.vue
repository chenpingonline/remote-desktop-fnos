<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowUpRight,
  Command,
  FolderClosed,
  History,
  KeyRound,
  Monitor,
  Moon,
  Plus,
  Search,
  Settings2,
  Sun,
  Zap,
} from 'lucide-vue-next'
import type { Component } from 'vue'
import { state, updateSettings } from '../stores/app'
import { closeCommandPalette, launchRemote, toast, ui } from '../stores/ui'
import { timeAgo } from '../utils/format'
import type { Connection } from '../types'

const router = useRouter()
const inputRef = ref<{ focus: () => void } | null>(null)
const query = ref('')
const selected = ref(0)

interface CommandItem {
  key: string
  label: string
  hint: string
  icon: Component
  path?: string
  run?: () => void
  connection?: Connection
  status?: string
  meta?: string
}

interface CommandGroup {
  label: string
  type: 'nav' | 'connection' | 'action'
  items: CommandItem[]
}

const navItems: CommandItem[] = [
  { key: 'nav-connections', label: '我的连接', hint: '所有远程设备', icon: Monitor, path: '/connections' },
  { key: 'nav-sessions', label: '会话记录', hint: '连接历史与详情', icon: History, path: '/sessions' },
  { key: 'nav-credentials', label: '我的凭据', hint: '密码、密钥与令牌', icon: KeyRound, path: '/credentials' },
  { key: 'nav-wol', label: '网络唤醒', hint: 'WOL 远程开机', icon: Zap, path: '/wol' },
  { key: 'nav-settings', label: '设置', hint: '外观、安全与数据', icon: Settings2, path: '/settings' },
]

const actions = computed<CommandItem[]>(() => [
  {
    key: 'action-new',
    label: '新建连接',
    hint: '添加一台远程设备',
    icon: Plus,
    run: () => router.push({ path: '/connections', query: { new: '1' } }),
  },
  {
    key: 'action-new-group',
    label: '新建分组',
    hint: '在连接页整理设备分组',
    icon: FolderClosed,
    run: () => router.push({ path: '/connections', query: { newGroup: '1' } }),
  },
  {
    key: 'action-new-wol',
    label: '新建唤醒项',
    hint: '通过 MAC 地址远程开机',
    icon: Zap,
    run: () => router.push({ path: '/wol', query: { new: '1' } }),
  },
  {
    key: 'action-theme',
    label: state.settings.theme === 'dark' ? '切换到浅色模式' : '切换到深色模式',
    hint: '立即切换界面主题',
    icon: state.settings.theme === 'dark' ? Sun : Moon,
    run: () => {
      updateSettings({ theme: state.settings.theme === 'dark' ? 'light' : 'dark' })
      toast('已切换界面主题')
    },
  },
])

const filtered = computed<CommandGroup[]>(() => {
  const keyword = query.value.trim().toLowerCase()
  const matches = (...values: Array<string | number | null | undefined>) =>
    !keyword || values.some((value) => String(value || '').toLowerCase().includes(keyword))

  const groups: CommandGroup[] = []
  const navs = navItems.filter((item) => matches(item.label, item.hint))
  if (navs.length) groups.push({ label: '跳转', type: 'nav', items: navs })

  const connections = state.connections
    .filter((item) => matches(item.name, item.host, item.os, item.protocol, ...(item.tags || [])))
    .sort((a, b) => (b.lastConnected ? new Date(b.lastConnected).getTime() : 0) - (a.lastConnected ? new Date(a.lastConnected).getTime() : 0))
    .slice(0, 7)
    .map((item) => ({
      key: `connection-${item.id}`,
      label: item.name,
      hint: `${item.host} · ${item.protocol}`,
      status: item.status,
      icon: Monitor,
      connection: item,
      meta: timeAgo(item.lastConnected),
    }))
  if (connections.length) groups.push({ label: '连接', type: 'connection', items: connections })

  const filteredActions = actions.value.filter((item) => matches(item.label, item.hint))
  if (filteredActions.length) groups.push({ label: '快捷操作', type: 'action', items: filteredActions })
  return groups
})

const flatItems = computed(() => filtered.value.flatMap((group) => group.items))
const activeItem = computed(() => flatItems.value[selected.value])

watch(
  () => ui.commandOpen,
  async (open) => {
    document.body.style.overflow = open ? 'hidden' : ''
    if (open) {
      query.value = ''
      selected.value = 0
      await nextTick()
      inputRef.value?.focus()
    }
  },
)

watch(flatItems, () => {
  if (selected.value >= flatItems.value.length) selected.value = 0
})

function close() {
  closeCommandPalette()
}

function runItem(item: CommandItem | undefined) {
  if (!item) return
  close()
  if (item.path) {
    router.push(item.path)
    return
  }
  if (item.connection) {
    launchRemote(item.connection)
    return
  }
  item.run?.()
}

function onInputKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    if (!flatItems.value.length) return
    selected.value = (selected.value + 1) % flatItems.value.length
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    if (!flatItems.value.length) return
    selected.value = (selected.value - 1 + flatItems.value.length) % flatItems.value.length
  } else if (event.key === 'Enter') {
    event.preventDefault()
    runItem(activeItem.value)
  } else if (event.key === 'Escape') {
    close()
  }
}

function globalKey(event: KeyboardEvent) {
  if (event.key === 'Escape' && ui.commandOpen) close()
}

onMounted(() => window.addEventListener('keydown', globalKey))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', globalKey)
  document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="ui.commandOpen"
        class="fixed inset-0 z-[2100] flex items-start justify-center bg-[#080b09]/55 px-4 pt-[8vh] backdrop-blur-md sm:pt-[10vh]"
        @mousedown.self="close"
        @keydown="onInputKeydown"
      >
        <section class="flex max-h-[min(520px,78vh)] w-full max-w-[590px] flex-col overflow-hidden rounded-[17px] border border-line-strong bg-surface shadow-float" role="dialog" aria-modal="true" aria-label="命令面板">
          <div class="flex min-h-[48px] items-center gap-2 border-b border-line px-3.5">
            <el-input
              ref="inputRef"
              v-model="query"
              size="large"
              placeholder="搜索设备、连接或输入命令…"
              aria-label="搜索设备、连接或命令"
              class="command-search flex-1 [&_.el-input__wrapper]:!min-h-[48px] [&_.el-input__wrapper]:!bg-transparent [&_.el-input__wrapper]:!px-0 [&_.el-input__wrapper]:!shadow-none [&_.el-input__inner]:!text-[15px]"
            >
              <template #prefix><Search :size="18" class="mr-1 text-faint" /></template>
              <template #suffix><kbd class="rounded-[5px] border border-line bg-soft px-1.5 py-1 text-[10px] font-bold text-faint">ESC</kbd></template>
            </el-input>
          </div>

          <div class="flex-1 overflow-y-auto px-2 pb-2 pt-1.5">
            <div v-if="!flatItems.length" class="flex flex-col items-center gap-1.5 px-6 py-9 text-center">
              <span class="mb-1 grid h-11 w-11 place-items-center rounded-xl border border-line bg-soft text-faint"><Search :size="19" /></span>
              <p class="text-[12.5px] font-semibold text-ink">没有找到与「{{ query }}」相关的结果</p>
              <span class="text-[10.5px] text-faint">试试设备名称、主机地址或协议</span>
            </div>

            <div v-for="group in filtered" :key="group.label" class="pt-1.5">
              <p class="px-2 py-1 text-[10px] font-bold uppercase tracking-[0.11em] text-faint">{{ group.label }}</p>
              <button
                v-for="item in group.items"
                :key="item.key"
                type="button"
                class="flex min-h-[40px] w-full items-center gap-2.5 rounded-[9px] px-2 py-1 text-left transition"
                :class="flatItems[selected]?.key === item.key ? 'bg-soft' : ''"
                @mousedown.prevent="runItem(item)"
                @mouseenter="selected = flatItems.findIndex((entry) => entry.key === item.key)"
              >
                <span class="relative grid h-7 w-7 flex-none place-items-center rounded-[8px] border border-line bg-surface text-muted">
                  <component :is="item.icon" :size="16" :stroke-width="1.9" />
                  <span
                    v-if="group.type === 'connection'"
                    class="absolute -bottom-0.5 -right-0.5 h-2 w-2 rounded-full border-[1.5px] border-surface"
                    :class="item.status === 'online' ? 'bg-success' : item.status === 'busy' ? 'bg-warning' : 'bg-faint'"
                  ></span>
                </span>
                <span class="flex min-w-0 flex-1 flex-col gap-px">
                  <span class="truncate text-[12.5px] font-semibold text-ink">{{ item.label }}</span>
                  <span class="truncate text-[10.5px] text-faint">{{ item.hint }}</span>
                </span>
                <span v-if="item.meta" class="whitespace-nowrap text-[10.5px] text-faint">{{ item.meta }}</span>
                <ArrowUpRight :size="14" class="text-transparent" :class="flatItems[selected]?.key === item.key ? '!text-faint' : ''" />
              </button>
            </div>
          </div>

          <footer class="flex min-h-[36px] items-center justify-between border-t border-line bg-soft px-3.5">
            <div class="flex items-center gap-3.5 text-[10.5px] text-faint">
              <span class="flex items-center gap-1"><kbd class="rounded border border-line bg-surface px-1 py-0.5">↑</kbd><kbd class="rounded border border-line bg-surface px-1 py-0.5">↓</kbd> 选择</span>
              <span class="flex items-center gap-1"><kbd class="rounded border border-line bg-surface px-1 py-0.5">↵</kbd> 打开</span>
              <span class="hidden items-center gap-1 sm:flex"><kbd class="flex items-center gap-0.5 rounded border border-line bg-surface px-1 py-0.5"><Command :size="9" />K</kbd> 命令</span>
            </div>
            <span class="text-[11px] font-semibold tracking-[-0.04em] text-faint">relay</span>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>
