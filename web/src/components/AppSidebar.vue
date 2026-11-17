<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import {
  History,
  KeyRound,
  Monitor,
  Settings2,
  Zap,
} from 'lucide-vue-next'
import { state, backend } from '../stores/app'
import { toggleSidebar, ui } from '../stores/ui'

const route = useRoute()

const navItems = [
  { label: '我的连接', to: '/connections', name: 'connections', icon: Monitor, count: state.connections.length },
  { label: '会话记录', to: '/sessions', name: 'sessions', icon: History },
  { label: '我的凭据', to: '/credentials', name: 'credentials', icon: KeyRound },
  { label: '网络唤醒', to: '/wol', name: 'wol', icon: Zap },
]

const accountItem = { label: '设置', to: '/settings', name: 'settings', icon: Settings2 }
</script>

<template>
  <aside
    class="sidebar fixed inset-y-0 left-0 z-40 flex h-dvh w-[232px] flex-col overflow-hidden border-r border-sidebar-line bg-sidebar text-sidebar-text shadow-[1px_0_0_rgba(20,32,20,0.02)] transition-transform duration-300 ease-out nav:translate-x-0"
    :class="ui.sidebarOpen ? 'translate-x-0' : '-translate-x-full'"
    aria-label="主导航"
  >
    <nav class="relative flex-1 px-2.5 pb-2.5 pt-3.5 nav:pt-0" aria-label="主导航">
      <RouterLink
        v-for="item in navItems"
        :key="item.name"
        :to="item.to"
        class="group relative mb-1.5 flex h-11 items-center gap-2.5 rounded-[9px] px-2 text-[13px] font-medium tracking-[0.01em] transition nav:h-9"
        :class="route.name === item.name
          ? 'bg-sidebar-active text-sidebar-text'
          : 'text-sidebar-text hover:bg-sidebar-hover'"
        :aria-current="route.name === item.name ? 'page' : undefined"
        @click="toggleSidebar(false)"
      >
        <span
          v-if="route.name === item.name"
          class="absolute -left-[11px] bottom-2 top-2 w-0.5 rounded-sm bg-sidebar-icon"
        ></span>
        <component
          :is="item.icon"
          :size="16"
          :stroke-width="1.8"
          class="flex-none"
          :class="route.name === item.name ? 'text-sidebar-icon' : 'opacity-80'"
        />
        <span class="flex-1">{{ item.label }}</span>
        <span
          v-if="item.count"
          class="grid h-[18px] min-w-[20px] place-items-center rounded-md px-1.5 text-[10px] font-semibold tabular-nums leading-none"
          :class="route.name === item.name ? 'bg-accent text-accent-ink' : 'bg-sidebar-soft text-sidebar-muted'"
        >{{ item.count }}</span>
      </RouterLink>

      <div class="my-3 border-t border-sidebar-line"></div>

      <RouterLink
        :to="accountItem.to"
        class="group relative flex h-11 items-center gap-2.5 rounded-[9px] px-2 text-[13px] font-medium tracking-[0.01em] transition nav:h-9"
        :class="route.name === accountItem.name
          ? 'bg-sidebar-active text-sidebar-text'
          : 'text-sidebar-text hover:bg-sidebar-hover'"
        :aria-current="route.name === accountItem.name ? 'page' : undefined"
        @click="toggleSidebar(false)"
      >
        <span
          v-if="route.name === accountItem.name"
          class="absolute -left-[11px] bottom-2 top-2 w-0.5 rounded-sm bg-sidebar-icon"
        ></span>
        <Settings2
          :size="16"
          :stroke-width="1.8"
          :class="route.name === accountItem.name ? 'text-sidebar-icon' : 'opacity-80'"
        />
        <span>{{ accountItem.label }}</span>
      </RouterLink>
    </nav>

    <div class="flex-none border-t border-sidebar-line px-2.5 py-2">
      <div class="flex items-center gap-2 px-1.5">
        <span class="sidebar-avatar relative grid h-7 w-7 flex-none place-items-center rounded-full text-[11px] font-bold">
          {{ backend.user?.name?.slice(0,1) || 'U' }}
          <span class="absolute -bottom-px -right-px h-2.5 w-2.5 rounded-full border-2 border-sidebar bg-[#70d69a]"></span>
        </span>
        <div class="flex min-w-0 flex-1 flex-col gap-px">
          <span class="text-[12px] font-semibold leading-tight text-sidebar-text">{{ backend.user?.name }}</span>
          <span class="text-[10px] leading-tight tabular-nums text-sidebar-faint">NAS 工作区 · {{ state.connections.length }} 台设备</span>
        </div>
      </div>
    </div>
  </aside>
</template>
