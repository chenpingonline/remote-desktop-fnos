<script setup lang="ts">
import { computed, reactive, watch, ref } from "vue";
import { KeyRound, LockKeyhole } from "lucide-vue-next";
import BaseModal from "./BaseModal.vue";
import { saveCredential, state } from "../stores/app";
import { passwordStrength } from "../utils/format";
import { toast } from "../stores/ui";
import type { Credential } from "../types";

const props = defineProps<{
  open: boolean;
  credential?: Credential | null;
  presetGroupId?: string;
}>();

const emit = defineEmits<{
  close: [];
  saved: [credential: Credential];
}>();

const types = [
  {
    value: "password",
    label: "登录密码",
    hint: "用户名 + 密码",
    icon: LockKeyhole,
  },
];

interface CredentialFormState {
  version: number;
  id: string | null;
  name: string;
  protocol: string;
  domain: string;
  type: string;
  username: string;
  secret: string;
  groupId: string;
  notes: string;
}

const form = reactive<CredentialFormState>({} as CredentialFormState);
const errors = reactive<Record<string, string>>({});
const busy = ref(false);
const strength = computed(() => passwordStrength(form.secret || ""));

function blankForm(): CredentialFormState {
  return {
    version: 0,
    id: null,
    name: "",
    type: "password",
    protocol: "rdp",
    domain: "",
    username: "",
    secret: "",
    groupId: props.presetGroupId || "",
    notes: "",
  };
}

function reset() {
  Object.keys(errors).forEach((key) => delete errors[key]);
  Object.assign(form, blankForm());
  if (props.credential) Object.assign(form, props.credential);
}

watch(
  () => props.open,
  (open) => {
    if (open) reset();
    else form.secret = "";
  },
  { immediate: true },
);

function validate() {
  Object.keys(errors).forEach((key) => delete errors[key]);
  if (!form.name.trim()) errors.name = "请填写凭据名称";
  if (form.protocol === "rdp" && !form.username.trim())
    errors.username = "请填写用户名";
  if (!props.credential && !form.secret) errors.secret = "请填写登录密码";
  return !Object.keys(errors).length;
}

async function submit() {
  if (busy.value) return;
  if (!validate()) return;
  busy.value = true;
  try {
    const saved = await saveCredential({
      ...form,
      username: form.protocol === "rdp" ? form.username : "",
      domain: form.protocol === "rdp" ? form.domain : "",
    });
    form.secret = "";
    toast(props.credential ? "凭据已更新" : "凭据已安全保存", {
      description: `${saved.name} · ${types.find((type) => type.value === saved.type)?.label}`,
      tone: "success",
    });
    emit("saved", saved);
    emit("close");
  } catch (e) {
    toast((e as Error).message, { tone: "danger" });
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <BaseModal
    :open="open"
    :title="credential ? '编辑凭据' : '新建凭据'"
    :description="
      credential
        ? '更新凭据内容后，关联的连接会立即使用新凭据。'
        : '凭据加密保存在 NAS，密码不会回显。'
    "
    size="lg"
    @close="!busy && emit('close')"
  >
    <el-form
      :model="form"
      label-position="left"
      label-width="96px"
      @submit.prevent="submit"
    >
      <el-form-item label="凭据名称" :error="errors.name">
        <el-input
          v-model="form.name"
          placeholder="例如：生产数据库密钥"
          aria-label="凭据名称"
          autocomplete="off"
        />
      </el-form-item>
      <el-form-item label="凭据协议">
        <el-select
          v-model="form.protocol"
          class="w-full"
          :disabled="!!credential"
          aria-label="凭据协议"
        >
          <el-option label="RDP · Windows" value="rdp" />
          <el-option label="VNC · macOS" value="vnc" />
        </el-select>
      </el-form-item>
      <el-form-item v-if="form.protocol === 'rdp'" label="域（可选）">
        <el-input v-model="form.domain" aria-label="凭据域" />
      </el-form-item>
      <el-form-item
        v-if="form.protocol === 'rdp'"
        label="用户名"
        :error="errors.username"
      >
        <el-input
          v-model="form.username"
          placeholder="例如：deploy 或 CORP\chenping"
          aria-label="用户名"
          class="font-mono"
          autocomplete="off"
          spellcheck="false"
        />
      </el-form-item>

      <el-form-item label="密码" :error="errors.secret">
        <el-input
          v-model="form.secret"
          type="password"
          show-password
          class="font-mono"
          :placeholder="credential ? '已保存密码，留空保持不变' : '输入登录密码'"
          autocomplete="new-password"
        />
        <div
          v-if="form.type === 'password' && form.secret"
          class="mt-1.5 flex items-center gap-2 text-[10px] text-faint"
        >
          <div class="flex w-[95px] gap-[3px]">
            <span
              v-for="index in 4"
              :key="index"
              class="h-[3px] flex-1 rounded-sm"
              :class="
                strength.score >= index
                  ? strength.score <= 1
                    ? 'bg-danger'
                    : strength.score === 2
                      ? 'bg-warning'
                      : 'bg-success'
                  : 'bg-inset'
              "
            ></span>
          </div>
          <span>强度：{{ strength.label }}</span>
        </div>
      </el-form-item>

      <div
        class="mt-0.5 flex items-start gap-2 rounded-lg border border-line bg-soft p-2 text-faint"
      >
        <LockKeyhole :size="14" class="mt-0.5 flex-none text-success" />
        <div>
          <p class="text-[10.5px] font-semibold text-muted">加密保存到 NAS</p>
          <span class="mt-0.5 block text-[10.5px] leading-5"
            >用户名和域可查看及修改。密码加密保存在 NAS，不回显；编辑时留空保留原密码，填写新密码则替换。</span
          >
        </div>
      </div>
    </el-form>

    <template #footer>
      <el-button @click="emit('close')">取消</el-button>
      <el-button type="primary" :loading="busy" @click="submit">
        <LockKeyhole :size="15" class="mr-1.5" />
        {{ credential ? "保存修改" : "保存凭据" }}
      </el-button>
    </template>
  </BaseModal>
</template>
