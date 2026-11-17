<script setup lang="ts">
import { Menu } from 'lucide-vue-next'
import { toggleSidebar } from '../stores/ui'

defineProps<{
  title: string
}>()
</script>

<template>
  <h1 class="sr-only">{{ title }}</h1>

  <div class="page-toolbar mb-2.5 flex items-center justify-between gap-2 nav:hidden">
    <div class="flex min-w-0 items-center gap-1.5">
      <button class="relay-icon-btn" type="button" aria-label="打开导航" @click="toggleSidebar()">
        <Menu :size="18" />
      </button>
      <span class="truncate text-[13px] font-semibold text-ink">{{ title }}</span>
    </div>
  </div>
</template>
