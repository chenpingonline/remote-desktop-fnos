<script setup lang="ts">
import { computed } from 'vue'
import { Briefcase, Building2, FlaskConical, Folder, House, Layers, Palette, Server } from 'lucide-vue-next'
import type { Component } from 'vue'

const props = defineProps<{
  icon?: string
  color?: string
  size?: number
}>()

const icons: Record<string, Component> = {
  palette: Palette,
  server: Server,
  flask: FlaskConical,
  building: Building2,
  home: House,
  briefcase: Briefcase,
  layers: Layers,
  folder: Folder,
}

const iconComponent = computed(() => icons[props.icon || 'folder'] || icons.folder)
const iconStyle = computed(() => {
  const size = props.size || 42
  const color = props.color || '#c9ec58'
  return {
    width: `${size}px`,
    height: `${size}px`,
    borderRadius: `${Math.round(size * 0.28)}px`,
    background: `color-mix(in srgb, ${color} 35%, var(--surface))`,
    color: `color-mix(in srgb, var(--text) 84%, ${color})`,
    borderColor: `color-mix(in srgb, ${color} 70%, var(--text-2))`,
  }
})
</script>

<template>
  <span class="grid flex-none place-items-center border" :style="iconStyle">
    <component :is="iconComponent" :size="(size || 42) * 0.44" :stroke-width="1.9" />
  </span>
</template>
