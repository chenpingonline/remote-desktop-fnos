<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  open: boolean
  title?: string
  description?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
  closeOnBackdrop?: boolean
}>()

const emit = defineEmits<{ close: [] }>()

const visible = computed({
  get: () => props.open,
  set: (value: boolean) => {
    if (!value) emit('close')
  },
})

const width = computed(() => {
  const widths = { sm: 400, md: 540, lg: 650, xl: 820 }
  return `min(${widths[props.size || 'md']}px, calc(100vw - 32px))`
})
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="title"
    :aria-label="title"
    append-to-body
    align-center
    destroy-on-close
    :width="width"
    :close-on-click-modal="closeOnBackdrop !== false"
    class="relay-dialog"
  >
    <template #header>
      <div class="min-w-0">
        <h2 class="text-[17px] font-bold leading-tight tracking-[-0.03em] text-ink">{{ title }}</h2>
        <p v-if="description" class="mt-0.5 max-w-[540px] text-[12px] leading-[1.55] text-muted">{{ description }}</p>
      </div>
    </template>

    <slot />

    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </el-dialog>
</template>
