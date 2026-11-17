<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  Clock3,
  Copy,
  Eye,
  EyeOff,
  KeyRound,
  Link2,
  LockKeyhole,
  MoreHorizontal,
  Pencil,
  ShieldCheck,
  Trash2,
} from 'lucide-vue-next'
import type { Component } from 'vue'
import type { Credential, Group } from '../types'
import { maskSecret, passwordStrength, timeAgo } from '../utils/format'
import { toast } from '../stores/ui'

const props = defineProps<{
  credential: Credential
  group: Group | null
  connectionCount: number
}>()

const emit = defineEmits<{
  edit: [credential: Credential]
  delete: [credential: Credential]
}>()

const revealed = ref(false)

const typeMeta = computed<{ label: string; icon: Component; color: string; base: string }>(() => {
  if (props.credential.type === 'token') return { label: '访问令牌', icon: KeyRound, color: '#359a7d', base: '#66c9ac' }
  return { label: '登录密码', icon: LockKeyhole, color: '#4f8ac9', base: '#7cb3ee' }
})

function handleCommand(command: string) {
  if (command === 'edit') emit('edit', props.credential)
  if (command === 'delete') emit('delete', props.credential)
}
</script>

<template>
  <article
    class="credential-card relative flex min-h-[142px] flex-col rounded-[16px] border border-line bg-surface p-3.5 shadow-card transition duration-300 hover:-translate-y-0.5 hover:border-line-strong hover:shadow-soft"
    :style="{ '--credential-color': typeMeta.color }"
  >
    <div class="flex items-start gap-2.5">
      <span
        class="grid h-[34px] w-[34px] flex-none place-items-center rounded-[10px] border"
        :style="{ color: typeMeta.color, background: `color-mix(in srgb, ${typeMeta.base} 14%, var(--surface-2))`, borderColor: `color-mix(in srgb, ${typeMeta.base} 18%, transparent)` }"
      >
        <component :is="typeMeta.icon" :size="18" :stroke-width="1.9" />
      </span>
      <div class="min-w-0 flex-1 pt-px">
        <h3 class="truncate text-[13px] font-bold tracking-[-0.015em] text-ink">{{ credential.name }}</h3>
        <span class="text-[10px] text-faint">{{ typeMeta.label }}</span>
      </div>
      <el-dropdown trigger="click" placement="bottom-end" @command="handleCommand">
        <button type="button" class="grid h-10 w-10 place-items-center rounded-md text-faint transition hover:bg-soft hover:text-ink sm:h-7 sm:w-7" aria-label="更多操作">
          <MoreHorizontal :size="16" />
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="edit"><span class="flex items-center gap-2"><Pencil :size="13" /> 编辑凭据</span></el-dropdown-item>
            <el-dropdown-item command="delete" class="!text-danger"><span class="flex items-center gap-2"><Trash2 :size="13" /> 删除凭据</span></el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>

    <div class="mt-2.5 flex items-center gap-1.5 text-[10.5px] text-faint">
      <span class="h-1 w-1 rounded-full" :style="{ background: group?.color || 'var(--border-strong)' }"></span>
      {{ group?.name || '未分组' }}
      <span class="h-0.5 w-0.5 rounded-full bg-line-strong"></span>
      <ShieldCheck :size="11" />
      <span class="text-success">NAS 加密保存</span>
    </div>

    <footer class="mt-auto flex items-center justify-between gap-2.5 border-t border-line pt-2.5">
      <span class="flex items-center gap-1 text-[10px] text-faint"><Link2 :size="12" />{{ connectionCount ? `关联 ${connectionCount} 个连接` : '暂未关联连接' }}</span>
      <span class="flex items-center gap-1 text-[10px] text-faint"><Clock3 :size="12" />{{ credential.lastUsed ? timeAgo(credential.lastUsed) : `更新于 ${timeAgo(credential.createdAt)}` }}</span>
    </footer>
  </article>
</template>
