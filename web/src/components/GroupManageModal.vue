<script setup lang="ts">
import { ArrowUpRight, FolderClosed, Pencil, Plus, Trash2 } from 'lucide-vue-next'
import BaseModal from './BaseModal.vue'
import GroupIcon from './GroupIcon.vue'
import { groupConnectionCount, state } from '../stores/app'
import type { Group } from '../types'

defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  close: []
  create: []
  edit: [group: Group]
  delete: [group: Group]
  view: [group: Group]
}>()
</script>

<template>
  <BaseModal
    :open="open"
    title="管理分组"
    description="新建、编辑或删除分组；删除分组不会移除其中的设备。"
    size="lg"
    @close="emit('close')"
  >
    <div v-if="state.groups.length" class="flex flex-col gap-1.5">
      <div
        v-for="group in state.groups"
        :key="group.id"
        class="flex items-center gap-2.5 rounded-lg border border-line bg-surface px-2.5 py-2 transition hover:border-line-strong"
      >
        <GroupIcon :icon="group.icon" :color="group.color" :size="32" />
        <div class="flex min-w-0 flex-1 flex-col gap-px">
          <p class="truncate text-[12.5px] font-semibold text-ink">{{ group.name }}</p>
          <p class="truncate text-[10.5px] text-faint">{{ group.description || '暂无描述' }} · {{ groupConnectionCount(group.id) }} 台设备</p>
        </div>
        <span class="hidden h-1 w-[54px] flex-none overflow-hidden rounded-sm bg-inset sm:block">
          <span class="block h-full rounded-sm opacity-75" :style="{ width: `${Math.max(10, Math.min(100, groupConnectionCount(group.id) * 18))}%`, background: group.color }"></span>
        </span>
        <div class="flex flex-none items-center gap-0.5">
          <el-tooltip :show-arrow="false" content="查看该分组的连接" placement="top" :show-after="300">
            <button type="button" class="grid h-10 w-10 place-items-center rounded-md text-faint transition hover:bg-soft hover:text-ink sm:h-8 sm:w-8" :aria-label="`查看 ${group.name} 的连接`" @click="emit('view', group)"><ArrowUpRight :size="14" /></button>
          </el-tooltip>
          <el-tooltip :show-arrow="false" content="编辑分组" placement="top" :show-after="300">
            <button type="button" class="grid h-10 w-10 place-items-center rounded-md text-faint transition hover:bg-soft hover:text-ink sm:h-8 sm:w-8" :aria-label="`编辑分组 ${group.name}`" @click="emit('edit', group)"><Pencil :size="14" /></button>
          </el-tooltip>
          <el-tooltip :show-arrow="false" content="删除分组" placement="top" :show-after="300">
            <button type="button" class="grid h-10 w-10 place-items-center rounded-md text-faint transition hover:bg-danger-soft hover:text-danger sm:h-8 sm:w-8" :aria-label="`删除分组 ${group.name}`" @click="emit('delete', group)"><Trash2 :size="14" /></button>
          </el-tooltip>
        </div>
      </div>
    </div>
    <div v-else class="flex flex-col items-center gap-2 px-4 py-10 text-center">
      <span class="grid h-11 w-11 place-items-center rounded-xl border border-line bg-soft text-faint"><FolderClosed :size="22" /></span>
      <p class="text-[12.5px] font-semibold text-ink">还没有分组</p>
      <span class="text-[10.5px] text-faint">创建第一个分组，把设备按团队或环境整理起来。</span>
    </div>

    <template #footer>
      <el-button @click="emit('close')">关闭</el-button>
      <el-button type="primary" @click="emit('create')"><Plus :size="14" class="mr-1" />新建分组</el-button>
    </template>
  </BaseModal>
</template>
