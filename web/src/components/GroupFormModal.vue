<script setup lang="ts">
import { computed, reactive, watch, ref } from "vue";
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
  "#c9ec58",
  "#9bc8ff",
  "#ffc48e",
  "#d3b9ff",
  "#8fe0c9",
  "#f3a9bc",
  "#f2d46b",
  "#8ec6c5",
];
const form = reactive<GroupFormState>({
  version: 0,
  id: null,
  name: "",
  description: "",
  color: colors[0],
  icon: "folder",
});
const errors = reactive<Record<string, string>>({});
const busy = ref(false);
const previewIcon = computed(
  () => icons.find((item) => item.value === form.icon) || icons[0],
);

function reset() {
  Object.keys(errors).forEach((key) => delete errors[key]);
  Object.assign(form, {
    version: 0,
    id: null,
    name: "",
    description: "",
    color: props.presetColor || colors[0],
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
          class="text-[9.5px] font-bold uppercase tracking-[0.1em] text-faint"
          >分组预览</span
        >
        <strong class="truncate text-[13px] font-bold text-ink">{{
          form.name || "未命名分组"
        }}</strong>
        <p class="truncate text-[10.5px] text-faint">
          {{ form.description || "为你的远程设备创建一个清晰的空间" }}
        </p>
      </div>
      <span
        class="self-start text-[8px] font-bold tracking-[0.11em]"
        :style="{ color: form.color }"
        >PREVIEW</span
      >
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
        <div class="flex w-full flex-wrap gap-1.5">
          <el-tooltip :show-arrow="false"
            v-for="item in icons"
            :key="item.value"
            :content="item.label"
            placement="top"
            :show-after="350"
          >
            <button
              type="button"
              class="grid h-8 w-8 place-items-center rounded-lg border bg-surface transition"
              :class="
                form.icon === item.value
                  ? 'border-ink bg-ink text-canvas'
                  : 'border-line text-faint hover:border-line-strong hover:text-ink'
              "
              :aria-label="item.label"
              @click="form.icon = item.value"
            >
              <component :is="item.icon" :size="15" :stroke-width="1.9" />
            </button>
          </el-tooltip>
        </div>
      </el-form-item>

      <el-form-item label="标识颜色">
        <div class="flex w-full flex-wrap items-center gap-2">
          <button
            v-for="color in colors"
            :key="color"
            type="button"
            class="grid h-6 w-6 place-items-center rounded-full text-[11px] font-extrabold text-[#243012] shadow-[inset_0_0_0_1px_rgba(0,0,0,0.08)] transition hover:scale-105"
            :class="
              form.color === color
                ? 'ring-2 ring-ink ring-offset-2 ring-offset-surface'
                : ''
            "
            :style="{ background: color }"
            :aria-label="`选择颜色 ${color}`"
            @click="form.color = color"
          >
            <span v-if="form.color === color">✓</span>
          </button>
          <span
            class="ml-auto flex items-center gap-2 text-[10.5px] text-faint"
          >
            <GroupIcon
              :icon="previewIcon.value"
              :color="form.color"
              :size="28"
            />
            {{ form.name || "分组" }}
          </span>
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
