<script setup lang="ts">
import { reactive, watch, ref } from "vue";
import {
  Briefcase,
  Building2,
  FlaskConical,
  Folder,
  House,
  Layers,
  Palette,
  Server,
} from "lucide-vue-next";
import type { Component } from "vue";
import BaseModal from "./BaseModal.vue";
import GroupIcon from "./GroupIcon.vue";
import { saveGroup } from "../stores/app";
import { toast } from "../stores/ui";
import type { Group } from "../types";

const props = defineProps<{
  open: boolean;
  group: Group | null;
  presetColor?: string;
}>();

const emit = defineEmits<{
  close: [];
  saved: [group: Group];
}>();

interface GroupFormState {
  version: number;
  id: string | null;
  name: string;
  description: string;
  color: string;
  icon: string;
}

const icons: { value: string; label: string; icon: Component }[] = [
  { value: "folder", label: "文件夹", icon: Folder },
  { value: "palette", label: "设计", icon: Palette },
  { value: "server", label: "服务", icon: Server },
  { value: "flask", label: "测试", icon: FlaskConical },
  { value: "building", label: "办公", icon: Building2 },
  { value: "home", label: "家庭", icon: House },
  { value: "briefcase", label: "财务", icon: Briefcase },
  { value: "layers", label: "分组", icon: Layers },
];

const colors = [
  { value: "#c9ec58", label: "青柠绿" },
  { value: "#9bc8ff", label: "天蓝" },
  { value: "#ffc48e", label: "杏橙" },
  { value: "#d3b9ff", label: "淡紫" },
  { value: "#8fe0c9", label: "薄荷绿" },
  { value: "#f3a9bc", label: "玫瑰粉" },
  { value: "#f2d46b", label: "金黄" },
  { value: "#8ec6c5", label: "青瓷绿" },
];
const form = reactive<GroupFormState>({
  version: 0,
  id: null,
  name: "",
  description: "",
  color: colors[0].value,
  icon: "folder",
});
const errors = reactive<Record<string, string>>({});
const busy = ref(false);
function reset() {
  Object.keys(errors).forEach((key) => delete errors[key]);
  Object.assign(form, {
    version: 0,
    id: null,
    name: "",
    description: "",
    color: props.presetColor || colors[0].value,
    icon: "folder",
  });
  if (props.group) Object.assign(form, props.group);
}

watch(
  () => props.open,
  (open) => {
    if (open) reset();
  },
  { immediate: true },
);

async function submit() {
  if (busy.value) return;
  Object.keys(errors).forEach((key) => delete errors[key]);
  if (!form.name.trim()) {
    errors.name = "请填写分组名称";
    return;
  }
  if (form.name.trim().length > 20) {
    errors.name = "分组名称不能超过 20 个字符";
    return;
  }
  busy.value = true;
  try {
    const saved = await saveGroup({ ...form, name: form.name.trim() });
    toast(props.group ? "分组已更新" : "新分组已创建", {
      description: saved.name,
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
    :title="group ? '编辑分组' : '新建分组'"
    :description="
      group
        ? '名称保存到 NAS；图标、颜色和描述保存在当前浏览器。'
        : '用分组整理设备，让连接查找更快、更清晰。'
    "
    size="md"
    @close="!busy && emit('close')"
  >
    <div
      class="mb-3 flex items-center gap-3 rounded-lg border border-line bg-soft p-3"
    >
      <GroupIcon :icon="form.icon" :color="form.color" :size="42" />
      <div class="flex min-w-0 flex-1 flex-col gap-px">
        <span
          class="text-xs font-semibold text-muted"
          >分组预览</span
        >
        <strong class="truncate text-base font-bold text-ink">{{
          form.name || "未命名分组"
        }}</strong>
        <p v-if="form.description" class="break-words text-xs text-muted">
          {{ form.description }}
        </p>
      </div>
    </div>

    <el-form
      :model="form"
      label-position="left"
      label-width="96px"
      @submit.prevent="submit"
    >
      <el-form-item label="分组名称" :error="errors.name">
        <el-input
          v-model="form.name"
          type="text"
          maxlength="20"
          placeholder="例如：设计团队"
          autocomplete="off"
          aria-label="分组名称"
        />
      </el-form-item>
      <el-form-item label="描述（可选）">
        <el-input
          v-model="form.description"
          placeholder="这个分组用来存放哪些设备？"
          aria-label="分组描述"
        />
      </el-form-item>

      <el-form-item label="图标">
        <div class="grid w-full grid-cols-[repeat(auto-fit,minmax(42px,1fr))] gap-1.5">
          <button
            v-for="item in icons"
            :key="item.value"
            type="button"
            class="flex flex-col items-center justify-center gap-1 rounded-lg border py-2 text-xs leading-tight"
            :class="form.icon === item.value
              ? 'border-ink bg-ink text-surface'
              : 'border-line bg-surface text-muted hover:border-line-strong hover:text-ink'"
            :aria-pressed="form.icon === item.value"
            @click="form.icon = item.value"
          >
            <component :is="item.icon" :size="18" aria-hidden="true" />
            {{ item.label }}
          </button>
        </div>
      </el-form-item>

      <el-form-item label="标识颜色">
        <div class="flex w-full flex-wrap items-center gap-2">
          <button
            v-for="color in colors"
            :key="color.value"
            type="button"
            class="grid h-6 w-6 place-items-center rounded-full text-[11px] font-extrabold text-[#243012] shadow-[inset_0_0_0_1px_rgba(0,0,0,0.08)] transition hover:scale-105"
            :class="
              form.color === color.value
                ? 'ring-2 ring-ink ring-offset-2 ring-offset-surface'
                : ''
            "
            :style="{ background: color.value }"
            :aria-label="`选择颜色：${color.label}`"
            :aria-pressed="form.color === color.value"
            :title="color.label"
            @click="form.color = color.value"
          >
            <span v-if="form.color === color.value">✓</span>
          </button>
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="emit('close')">取消</el-button>
      <el-button type="primary" :loading="busy" @click="submit">{{
        group ? "保存修改" : "创建分组"
      }}</el-button>
    </template>
  </BaseModal>
</template>
