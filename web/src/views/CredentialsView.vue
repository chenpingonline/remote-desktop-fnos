<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  Eye,
  Fingerprint,
  KeyRound,
  LockKeyhole,
  Plus,
  Search,
  ShieldAlert,
  ShieldCheck,
  Unlock,
} from 'lucide-vue-next'
import CredentialCard from '../components/CredentialCard.vue'
import CredentialFormModal from '../components/CredentialFormModal.vue'
import PageToolbar from '../components/PageToolbar.vue'
import { credentialUsage, groupById, removeCredential, state } from '../stores/app'
import { requestConfirm, toast } from '../stores/ui'
import { passwordStrength } from '../utils/format'
import type { Credential } from '../types'

const search = ref('')
const typeFilter = ref('all')
const locked = ref(false)
const formOpen = ref(false)
const editingCredential = ref<Credential | null>(null)

const typeTabs = computed(() => [{id:'all',label:'全部凭据',count:state.credentials.length,icon:KeyRound}])

const filteredCredentials = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  return state.credentials.filter((credential) => {
    if (typeFilter.value !== 'all' && credential.type !== typeFilter.value) return false
    if (!keyword) return true
    const groupName = groupById(credential.groupId)?.name || ''
    return [credential.name, credential.username, groupName, credential.notes].join(' ').toLowerCase().includes(keyword)
  })
})

const weakCount = computed(() => 0)

function openCreate() {
  if (locked.value) {
    toast('保险库已锁定，请先解锁', { tone: 'danger' })
    return
  }
  editingCredential.value = null
  formOpen.value = true
}

function openEdit(credential: Credential) {
  if (locked.value) {
    toast('保险库已锁定，请先解锁', { tone: 'danger' })
    return
  }
  editingCredential.value = credential
  formOpen.value = true
}

async function remove(credential: Credential) {
  if (locked.value) {
    toast('保险库已锁定，请先解锁', { tone: 'danger' })
    return
  }
  const usage = credentialUsage(credential.id)
  const ok = await requestConfirm({
    title: '删除这项凭据？',
    description: usage
      ? `「${credential.name}」被 ${usage} 个连接使用。请先编辑连接，解除凭据引用后再删除。`
      : `「${credential.name}」将被永久删除，此操作无法撤销。`,
    confirmText: '删除凭据',
  })
  if (!ok) return
  await removeCredential(credential.id)
  toast('凭据已删除', { description: credential.name })
}

function toggleLock() {
  locked.value = !locked.value
  toast(locked.value ? '保险库已锁定' : '保险库已解锁', {
    description: locked.value ? '凭据内容已隐藏，展开后可管理。' : '可管理凭据元数据。',
    tone: locked.value ? 'info' : 'success',
  })
}

function runSecurityCheck() { toast('密码由后端加密保存，不提供明文读取或密码强度扫描', {tone:'info'}) }

</script>

<template>
  <div class="page-wrap">
    <PageToolbar title="我的凭据" />

    <h2 class="sr-only">凭据列表</h2>

    <div class="mb-2.5 flex flex-col gap-2.5 rounded-[15px] border border-line bg-surface p-2 shadow-card xl:flex-row xl:items-center xl:justify-between">
      <div class="flex flex-wrap items-center gap-0.5 xl:flex-nowrap xl:overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        <button
          v-for="tab in typeTabs"
          :key="tab.id"
          type="button"
          class="flex h-10 flex-none items-center gap-1.5 rounded-lg px-2.5 text-[11px] font-semibold transition sm:h-[30px]"
          :class="typeFilter === tab.id ? 'bg-inset text-ink' : 'text-faint hover:bg-soft hover:text-muted'"
          @click="typeFilter = tab.id"
        >
          <component :is="tab.icon" :size="13" />{{ tab.label }}<span class="count-chip">{{ tab.count }}</span>
        </button>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <el-input v-model="search" clearable placeholder="搜索名称或用户名" class="min-w-[140px] flex-1">
          <template #prefix><Search :size="14" class="text-faint" /></template>
        </el-input>
        <span class="hidden items-center gap-1 whitespace-nowrap text-[10px] font-medium sm:flex" :class="weakCount ? 'text-warning' : 'text-success'">
          <ShieldAlert v-if="weakCount" :size="12" />
          <ShieldCheck v-else :size="12" />
          {{ weakCount ? `${weakCount} 项强度较弱` : '后端加密保存' }}
        </span>
        <el-tooltip :show-arrow="false" content="安全检查" placement="top" :show-after="300">
          <button type="button" class="relay-icon-btn flex-none disabled:cursor-not-allowed disabled:opacity-40" :disabled="locked" aria-label="安全检查" @click="runSecurityCheck"><ShieldCheck :size="15" /></button>
        </el-tooltip>
        <el-tooltip :show-arrow="false" :content="locked ? '解锁保险库' : '锁定保险库'" placement="top" :show-after="300">
          <button type="button" class="relay-icon-btn flex-none" :aria-label="locked ? '解锁保险库' : '锁定保险库'" @click="toggleLock">
            <Unlock v-if="locked" :size="15" />
            <LockKeyhole v-else :size="15" />
          </button>
        </el-tooltip>
        <el-button type="primary" class="!h-10 w-full flex-none sm:!h-[30px] sm:w-auto" @click="openCreate"><Plus :size="14" class="mr-1" />新建凭据</el-button>
      </div>
    </div>

    <template v-if="!locked">
      <div v-if="filteredCredentials.length" class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
        <CredentialCard
          v-for="(credential, index) in filteredCredentials"
          :key="credential.id"
          :credential="credential"
          :group="groupById(credential.groupId)"
          :connection-count="credentialUsage(credential.id)"
          class="animate-[fade-up_0.4s_cubic-bezier(0.22,1,0.36,1)_both]"
          :style="{ animationDelay: `${Math.min(index * 35, 210)}ms` }"
          @edit="openEdit"
          @delete="remove"
        />
      </div>
      <div v-else class="panel flex flex-col items-center gap-2 px-6 py-10 text-center">
        <span class="mb-1 grid h-12 w-12 place-items-center rounded-xl border border-line bg-soft text-faint"><Eye :size="22" /></span>
        <h3 class="text-[14px] font-bold text-ink">没有找到相关凭据</h3>
        <p class="text-[10.5px] text-faint">尝试更换搜索关键词或凭据类型，也可以新建一项凭据。</p>
        <el-button size="small" @click="search = ''; typeFilter = 'all'">重置筛选</el-button>
      </div>
    </template>

    <div v-else class="panel flex flex-col items-center gap-2 px-6 py-10 text-center">
      <span class="mb-2 grid h-12 w-12 place-items-center rounded-[15px] border border-line bg-soft text-muted"><LockKeyhole :size="26" /></span>
      <p class="text-[11px] font-bold uppercase tracking-[0.16em] text-faint">Vault locked</p>
      <h3 class="text-[18px] font-bold text-ink">保险库已锁定</h3>
      <p class="mb-2 max-w-[360px] text-[11px] leading-6 text-faint">为保护你的凭据，所有内容已隐藏。展开后可编辑或删除凭据。</p>
      <el-button type="primary" @click="toggleLock"><Unlock :size="15" class="mr-1.5" />解锁保险库</el-button>
      <span class="mt-1 text-[10px] text-faint">这里只折叠列表；访问权限由 NAS 登录身份控制。</span>
    </div>

    <CredentialFormModal :open="formOpen" :credential="editingCredential" @close="formOpen = false" @saved="formOpen = false" />
  </div>
</template>
