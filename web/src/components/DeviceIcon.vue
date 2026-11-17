<script setup lang="ts">
import { computed } from 'vue'
import { Command, Monitor, Terminal } from 'lucide-vue-next'

const props = defineProps<{
  os?: string
  size?: number
}>()

const variant = computed(() => {
  const value = (props.os || '').toLowerCase()
  if (value.includes('mac') || value.includes('ios')) return 'mac'
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
</script>

<template>
  <span class="grid flex-none place-items-center border" :style="style">
    <svg v-if="variant === 'windows'" viewBox="0 0 24 24" fill="currentColor" class="block" :style="{ width: `${(size || 44) * 0.46}px`, height: `${(size || 44) * 0.46}px` }" aria-hidden="true">
      <path d="M3 5.6 10.7 4.5v6.9H3V5.6Zm8.8-1.3L21 3v8.4h-9.2V4.3ZM3 12.6h7.7v6.9L3 18.4v-5.8Zm8.8 0H21V21l-9.2-1.3v-7.1Z" />
    </svg>
    <Command v-else-if="variant === 'mac'" :size="(size || 44) * 0.46" :stroke-width="2.2" />
    <Terminal v-else-if="variant === 'linux'" :size="(size || 44) * 0.45" :stroke-width="2" />
    <Monitor v-else :size="(size || 44) * 0.45" :stroke-width="1.9" />
  </span>
</template>
