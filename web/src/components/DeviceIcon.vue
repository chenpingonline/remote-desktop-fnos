<script setup lang="ts">
import { computed } from 'vue'
import { Monitor, Laptop, Server, HardDrive, MonitorCog, PanelsTopLeft } from 'lucide-vue-next'

const props = defineProps<{
  icon?: string
  os?: string
  size?: number
}>()

const hardwareIcons = {
  desktop: Monitor,
  laptop: Laptop,
  server: Server,
  nas: HardDrive,
  workstation: MonitorCog,
  virtual: PanelsTopLeft,
}

const variant = computed(() => {
  if (['windows', 'mac', 'linux', 'desktop', 'laptop', 'server', 'nas', 'workstation', 'virtual'].includes(props.icon || '')) return props.icon!
  const value = (props.os || '').toLowerCase()
  if (value.includes('darwin') || value.includes('mac') || value.includes('ios')) return 'mac'
  if (value.includes('win')) return 'windows'
  if (value.includes('linux')) return 'linux'
  return 'unknown'
})

const tint = computed(() => {
  if (variant.value === 'windows') return { color: '#4d8ed4', base: '#4b91df' }
  if (variant.value === 'linux') return { color: '#b47c26', base: '#d6a451' }
  return { color: 'var(--text)', base: '#9aa0a6' }
})

const style = computed(() => {
  const size = props.size || 44
  return {
    width: `${size}px`,
    height: `${size}px`,
    borderRadius: `${Math.round(size * 0.27)}px`,
    color: tint.value.color,
    background: `color-mix(in srgb, ${tint.value.base} 12%, var(--surface-2))`,
    borderColor: `color-mix(in srgb, ${tint.value.base} 16%, transparent)`,
  }
})
const hardwareIcon = computed(() => hardwareIcons[variant.value as keyof typeof hardwareIcons] ?? Monitor)
const glyphSize = computed(() => Math.round((props.size || 44) * 0.62))
</script>

<template>
  <span class="grid flex-none place-items-center border" :style="style" aria-hidden="true">
    <svg v-if="variant === 'windows'" viewBox="0 0 24 24" fill="currentColor" :width="glyphSize" :height="glyphSize">
      <path d="M2 2h9v9H2zM13 2h9v9h-9zM2 13h9v9H2zM13 13h9v9h-9z" />
    </svg>
    <span
      v-else-if="variant === 'mac' || variant === 'linux'"
      class="brand-icon"
      :class="variant"
      :style="{ width: glyphSize + 'px', height: glyphSize + 'px' }"
    />
    <component v-else :is="hardwareIcon" :size="glyphSize" :stroke-width="1.9" />
  </span>
</template>

<style scoped>
/* Brand silhouettes from Simple Icons. */
.brand-icon {
  background: currentColor;
  mask: var(--brand-icon) center / contain no-repeat;
}
.mac { --brand-icon: url('../assets/device-icons/apple.svg'); }
.linux { --brand-icon: url('../assets/device-icons/linux.svg'); }
</style>
