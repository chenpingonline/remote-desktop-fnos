<script setup lang="ts">
import { onMounted, ref } from "vue";
import { Sun, Moon, Laptop, Check, Zap, RefreshCw } from "lucide-vue-next";
import PageToolbar from "../components/PageToolbar.vue";
import CertificateDialog from "../components/CertificateDialog.vue";
import {
  backend,
  isDark,
  state,
  allPages,
  rawConnection,
  refreshWorkspace,
} from "../stores/app";
import { api } from "../api";
import { toast } from "../stores/ui";
import type { AccentName, ThemeMode, Density, PaletteName } from "../types";
import "../assets/native-desktop.css";
const activeTab = ref("general"),
  busy = ref(false),
  trusts = ref<any[]>([]),
  selected = ref<any>(null);
const limits = ref<any>(null);
const tabs = [
  { id: "general", label: "通用" },
  { id: "appearance", label: "外观" },
  { id: "security", label: "安全与证书" },
  { id: "diagnostics", label: "运行与诊断" },
  { id: "about", label: "关于" },
];
const appIcon = import.meta.env.BASE_URL + "apple-touch-icon.png";
const themes = [
  {
    value: "light" as ThemeMode,
    label: "浅色",
    description: "明亮、清晰",
    icon: Sun,
  },
  {
    value: "dark" as ThemeMode,
    label: "深色",
    description: "专注、低光",
    icon: Moon,
  },
  {
    value: "system" as ThemeMode,
    label: "跟随系统",
    description: "自动切换",
    icon: Laptop,
  },
];
const accents = [
  { value: "lime" as AccentName, label: "青柠", color: "#d7fb62" },
  { value: "sky" as AccentName, label: "天蓝", color: "#a5d3ff" },
  { value: "tangerine" as AccentName, label: "橘橙", color: "#ffc38e" },
  { value: "lilac" as AccentName, label: "丁香", color: "#d6c2ff" },
];
const palettes = [
  {
    value: "native" as PaletteName,
    label: "中性白",
    darkLabel: "中性灰",
    description: "贴近白色窗口",
    darkDescription: "中性深灰",
  },
  {
    value: "warm" as PaletteName,
    label: "暖砂",
    darkLabel: "暖砂",
    description: "偏暖米绿",
    darkDescription: "暖调墨绿",
  },
  {
    value: "cool" as PaletteName,
    label: "冷雾",
    darkLabel: "冷雾",
    description: "偏冷雾蓝",
    darkDescription: "冷调雾蓝",
  },
];
const densities = [
  { value: "compact" as Density, label: "紧凑", description: "显示更多内容" },
  {
    value: "comfortable" as Density,
    label: "舒适",
    description: "更宽松的间距",
  },
];
function setTheme(value: ThemeMode) {
  state.settings.theme = value;
}
function setAccent(value: AccentName) {
  state.settings.accent = value;
}
async function reload() {
  trusts.value = await allPages("/certificate-trust");
  if (backend.user?.admin) {
    backend.limits = await api("/settings");
    limits.value = { ...backend.limits };
  }
}
async function saveLimits() {
  if (busy.value) return;
  busy.value = true;
  try {
    backend.limits = await api("/settings", "PATCH", limits.value);
    limits.value = { ...backend.limits };
    toast("运行设置已保存", { tone: "success" });
    await refreshWorkspace();
  } catch (e) {
    toast((e as Error).message, { tone: "danger" });
  } finally {
    busy.value = false;
  }
}
function manage(id: string) {
  selected.value = rawConnection(id);
}
onMounted(() => {
  void reload().catch((e) => toast((e as Error).message, { tone: "danger" }));
});
</script>
<template>
  <div class="page-wrap settings-page">
    <PageToolbar title="设置" />
    <div class="mx-auto w-full max-w-[1000px]">
      <div
        class="mb-3.5 flex gap-1.5 overflow-x-auto rounded-xl border border-line bg-surface p-2 shadow-card"
        role="tablist"
        aria-label="设置分类"
      >
        <button
          v-for="tab in tabs"
          :key="tab.id"
          role="tab"
          :aria-selected="activeTab === tab.id"
          class="flex h-10 flex-1 items-center justify-center whitespace-nowrap rounded-lg px-2 text-xs font-semibold sm:px-3"
          :class="activeTab === tab.id ? 'bg-inset text-ink' : 'text-muted'"
          @click="activeTab = tab.id"
        >
          {{ tab.label }}
        </button>
      </div>
      <section
        class="rounded-2xl border border-line bg-surface p-5 shadow-card"
      >
        <template v-if="activeTab === 'general'">
          <div class="mb-4 flex justify-between text-sm font-bold">
            <h2>基础偏好</h2>
            <span class="text-xs font-normal text-faint">保存在当前浏览器</span>
          </div>
          <el-form
            label-position="left"
            label-width="96px"
            class="grid grid-cols-1 gap-x-8 sm:grid-cols-2"
          >
            <el-form-item label="界面语言">
              <el-input model-value="简体中文" disabled />
            </el-form-item>
            <el-form-item label="显示时区">
              <el-select v-model="state.settings.timezone" aria-label="时区">
                <el-option label="上海" value="Asia/Shanghai" />
                <el-option label="东京" value="Asia/Tokyo" />
                <el-option label="伦敦" value="Europe/London" />
                <el-option label="洛杉矶" value="America/Los_Angeles" />
              </el-select>
            </el-form-item>
            <el-form-item label="启动页面">
              <el-select v-model="state.settings.startPage" aria-label="启动页面">
                <el-option label="我的连接" value="connections" />
                <el-option label="会话记录" value="sessions" />
              </el-select>
            </el-form-item>
            <el-form-item label="断开前确认">
              <el-switch
                v-model="state.settings.confirmDisconnect"
                aria-label="断开前确认"
              />
            </el-form-item>
          </el-form>
        </template>
        <template v-else-if="activeTab === 'appearance'">
          <div>
            <div class="mb-2.5 flex items-baseline justify-between">
              <span class="text-[13px] font-bold text-ink">界面主题</span
              ><small class="text-[9px] font-bold tracking-[0.13em] text-faint"
                >THEME</small
              >
            </div>
            <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
              <button
                v-for="theme in themes"
                :key="theme.value"
                type="button"
                class="rounded-xl border p-1.5 text-left transition"
                :class="
                  state.settings.theme === theme.value
                    ? 'border-ink shadow-[0_0_0_1px_var(--text)]'
                    : 'border-line-strong hover:border-faint'
                "
                @click="setTheme(theme.value)"
              >
                <div class="flex h-[60px] overflow-hidden rounded-md border border-line" aria-hidden="true">
                  <div
                    v-for="mode in theme.value === 'system' ? ['light', 'dark'] : [theme.value]"
                    :key="mode"
                    class="palette-preview flex min-w-0 flex-1 overflow-hidden"
                    :data-theme="mode"
                    :data-palette="state.settings.palette"
                  >
                    <div class="palette-preview-line w-[22%]"></div>
                    <div class="flex flex-1 flex-col gap-1 p-1.5">
                      <span class="palette-preview-line h-1 w-[55%] rounded-sm"></span>
                      <span class="flex gap-1">
                        <i class="palette-preview-surface h-[19px] flex-1 rounded-[3px] border"></i>
                        <i class="palette-preview-surface h-[19px] flex-1 rounded-[3px] border"></i>
                      </span>
                      <span class="palette-preview-line h-[3px] w-[80%] rounded-sm"></span>
                      <span class="palette-preview-line h-[3px] w-[52%] rounded-sm"></span>
                    </div>
                  </div>
                </div>
                <span
                  class="flex items-center justify-between px-1 pb-0.5 pt-2"
                >
                  <span class="flex flex-col gap-px"
                    ><strong class="text-[11.5px] font-semibold text-ink">{{
                      theme.label
                    }}</strong
                    ><small class="text-[10.5px] text-faint">{{
                      theme.description
                    }}</small></span
                  >
                  <span
                    class="grid h-4 w-4 place-items-center rounded-full border"
                    :class="
                      state.settings.theme === theme.value
                        ? 'border-accent-strong bg-accent text-accent-ink'
                        : 'border-line-strong'
                    "
                    ><Check
                      v-if="state.settings.theme === theme.value"
                      :size="11"
                  /></span>
                </span>
              </button>
            </div>
          </div>

          <div class="mt-5 border-t border-line pt-4">
            <div class="mb-2.5 flex items-baseline justify-between">
              <span class="text-[13px] font-bold text-ink">强调色</span
              ><small class="text-[9px] font-bold tracking-[0.13em] text-faint"
                >ACCENT</small
              >
            </div>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="accent in accents"
                :key="accent.value"
                type="button"
                class="flex h-[34px] items-center gap-2 rounded-lg border px-2.5 pr-3 text-[11.5px] font-medium text-muted transition hover:bg-soft"
                :class="
                  state.settings.accent === accent.value
                    ? 'border-faint bg-soft'
                    : 'border-line'
                "
                :aria-pressed="state.settings.accent === accent.value"
                @click="setAccent(accent.value)"
              >
                <span
                  class="grid h-[22px] w-[22px] place-items-center rounded-[7px] text-[#1a220d] shadow-[inset_0_0_0_1px_rgba(0,0,0,0.08)]"
                  :style="{ background: accent.color }"
                  ><Check
                    v-if="state.settings.accent === accent.value"
                    :size="12" /></span
                >{{ accent.label }}
              </button>
            </div>
            <p class="mt-3 flex items-center gap-1.5 text-[11.5px] text-faint">
              <Zap :size="11" />强调色会应用在按钮、图表、品牌标识与高亮状态上。
            </p>
          </div>

          <div class="mt-5 border-t border-line pt-4">
            <div class="mb-2.5 flex items-baseline justify-between">
              <span class="text-[13px] font-bold text-ink">界面配色</span
              ><small class="text-[9px] font-bold tracking-[0.13em] text-faint"
                >PALETTE</small
              >
            </div>
            <div class="flex flex-col gap-2 sm:flex-row">
              <button
                v-for="palette in palettes"
                :key="palette.value"
                type="button"
                class="flex flex-1 items-center gap-2.5 rounded-[10px] border p-2 text-left transition"
                :class="
                  state.settings.palette === palette.value
                    ? 'border-ink bg-soft'
                    : 'border-line-strong hover:bg-soft'
                "
                aria-describedby="palette-hint"
                :aria-pressed="state.settings.palette === palette.value"
                @click="state.settings.palette = palette.value"
              >
                <span
                  class="palette-preview palette-choice-preview flex h-[33px] w-[33px] flex-none items-end gap-1 rounded-md border p-1"
                  :data-theme="isDark ? 'dark' : 'light'"
                  :data-palette="palette.value"
                  aria-hidden="true"
                >
                  <i
                    class="palette-preview-surface h-[17px] w-[12px] rounded-[3px] border"
                  ></i>
                  <i
                    class="palette-preview-line h-[9px] w-[7px] rounded-[2px]"
                  ></i>
                </span>
                <span class="flex flex-col gap-px">
                  <strong class="text-[11.5px] font-semibold text-ink">{{
                    isDark ? palette.darkLabel : palette.label
                  }}</strong>
                  <small class="text-[10.5px] text-faint">{{
                    isDark ? palette.darkDescription : palette.description
                  }}</small>
                </span>
                <span
                  class="ml-auto grid h-4 w-4 place-items-center rounded-full border"
                  :class="
                    state.settings.palette === palette.value
                      ? 'border-accent-strong bg-accent text-accent-ink'
                      : 'border-line-strong'
                  "
                  ><Check
                    v-if="state.settings.palette === palette.value"
                    :size="11"
                /></span>
              </button>
            </div>
            <p id="palette-hint" class="mt-3 flex items-center gap-1.5 text-[11.5px] text-faint">
              <component :is="isDark ? Moon : Sun" :size="11" aria-hidden="true" />
              配色同时适用于浅色与深色模式，预览随当前主题切换。
            </p>
          </div>

          <div class="mt-5 border-t border-line pt-4">
            <div class="mb-2.5 flex items-baseline justify-between">
              <span class="text-[13px] font-bold text-ink">显示密度</span
              ><small class="text-[9px] font-bold tracking-[0.13em] text-faint"
                >DENSITY</small
              >
            </div>
            <div class="flex flex-col gap-2 sm:flex-row">
              <button
                v-for="density in densities"
                :key="density.value"
                type="button"
                class="flex flex-1 items-center gap-2.5 rounded-[10px] border p-2 text-left transition"
                :class="
                  state.settings.density === density.value
                    ? 'border-ink bg-soft'
                    : 'border-line-strong hover:bg-soft'
                "
                @click="state.settings.density = density.value"
              >
                <span
                  class="flex h-[33px] w-[33px] flex-none flex-col justify-center gap-1 rounded-md border border-line bg-surface p-1.5"
                  ><i class="h-[3px] rounded-sm bg-line-strong"></i
                  ><i class="h-[3px] rounded-sm bg-line-strong"></i
                  ><i class="h-[3px] rounded-sm bg-line-strong"></i
                ></span>
                <span class="flex flex-1 flex-col gap-px"
                  ><strong class="text-[12.5px] font-semibold text-ink">{{
                    density.label
                  }}</strong
                  ><small class="text-[10.5px] text-faint">{{
                    density.description
                  }}</small></span
                >
                <span
                  class="h-3 w-3 flex-none rounded-full border"
                  :class="
                    state.settings.density === density.value
                      ? 'border-[3px] border-ink bg-accent'
                      : 'border-line-strong'
                  "
                ></span>
              </button>
            </div>
          </div>

          <div class="mt-5 border-t border-line pt-4">
            <div class="mb-2.5 flex items-baseline justify-between">
              <span class="text-[13px] font-bold text-ink">动效</span
              ><small class="text-[9px] font-bold tracking-[0.13em] text-faint"
                >MOTION</small
              >
            </div>
            <div class="flex items-center justify-between gap-4 sm:gap-6">
              <div>
                <p class="text-[13px] font-semibold text-ink">减少动态效果</p>
                <span
                  class="mt-1 block max-w-[470px] text-[11.5px] leading-5 text-faint"
                  >关闭页面过渡、图表生长和悬停位移等动画。</span
                >
              </div>
              <el-switch
                v-model="state.settings.reduceMotion"
                aria-label="减少动态效果"
              />
            </div>
          </div>
        </template>

        <template v-else-if="activeTab === 'security'">
          <h2 class="mb-3 text-sm font-bold">凭据与认证</h2>
          <p class="mb-5 text-xs leading-6 text-muted">
            使用 NAS
            登录身份隔离每位用户的连接与凭据。账号、域和密码由后端加密保存。用户名和域可查看及修改，密码不回显。RDP
            使用 NLA 与证书校验。
          </p>
          <div class="mb-3 flex items-center justify-between">
            <h2 class="text-sm font-bold">证书信任</h2>
            <el-button size="small" @click="reload"
              ><RefreshCw :size="13" class="mr-1" />刷新</el-button
            >
          </div>
          <p v-if="!trusts.length" class="text-xs text-faint">
            暂无已保存的证书信任。首次连接时会检查证书。
          </p>
          <div
            v-for="t in trusts"
            :key="t.connectionId"
            class="flex items-center justify-between gap-3 border-b border-line py-3 text-xs"
          >
            <div class="min-w-0">
              <p class="font-semibold">
                {{ t.name || t.connectionName || t.host }}
              </p>
              <p class="mt-1 break-all text-faint">{{ t.host }}:{{ t.port }}</p>
            </div>
            <el-button size="small" @click="manage(t.connectionId)"
              >查看与撤销</el-button
            >
          </div>
        </template>
        <template v-else-if="activeTab === 'diagnostics'">
          <h2 class="mb-3 text-sm font-bold">
            运行状态 · {{ backend.capabilities.version }}
          </h2>
          <p class="mb-5 text-xs text-muted">
            桌面引擎：{{
              backend.capabilities.engineReachable ? "端口可达" : "不可达"
            }}。端口可达不代表远端桌面已验证。
          </p>
          <el-form
            v-if="backend.user?.admin && limits"
            label-position="left"
            label-width="160px"
          >
            <el-form-item label="每用户会话数"
              ><el-input-number
                v-model="limits.perUserLimit"
                :min="1"
                :max="8"
                aria-label="每用户会话数"
            /></el-form-item>
            <el-form-item label="全局会话数"
              ><el-input-number
                v-model="limits.totalLimit"
                :min="limits.perUserLimit"
                :max="16"
                aria-label="全局会话数"
            /></el-form-item>
            <el-form-item label="空闲超时（分钟）"
              ><el-input-number
                v-model="limits.idleTimeoutMinutes"
                :min="1"
                :max="120"
                aria-label="空闲超时"
            /></el-form-item>
            <el-form-item label="最长时长（小时）"
              ><el-input-number
                v-model="limits.maxDurationHours"
                :min="1"
                :max="24"
                aria-label="最长会话时长"
            /></el-form-item>
            <el-button type="primary" :loading="busy" @click="saveLimits"
              >保存运行设置</el-button
            >
          </el-form>
          <p v-else class="text-xs text-faint">
            仅 NAS 管理员可调整会话配额与超时。
          </p>
          <h2 class="mb-2 mt-6 text-sm font-bold">最近操作</h2>
          <div
            v-for="e in backend.audit"
            :key="e.id"
            class="flex justify-between gap-3 border-b border-line py-2 text-xs text-muted"
          >
            <span>{{ e.action }}</span
            ><time>{{ new Date(e.createdAt).toLocaleString() }}</time>
          </div>
        </template>
        <div v-else-if="activeTab === 'about'" class="flex items-center gap-3">
          <img :src="appIcon" alt="" width="48" height="48" class="h-12 w-12 rounded-xl" />
          <div class="flex-1">
            <h2 class="text-sm font-bold text-ink">远程桌面</h2>
            <p class="mt-1 text-xs tabular-nums text-muted">当前版本 {{ backend.capabilities.version || '—' }}</p>
          </div>
          <a
            href="https://github.com/chenpingonline/remote-desktop-fnos"
            target="_blank"
            rel="noopener noreferrer"
            aria-label="打开远程桌面 GitHub 仓库"
            title="GitHub 仓库"
            class="grid h-7 w-7 flex-none place-items-center rounded-[7px] border border-transparent text-ink hover:border-line-strong hover:bg-soft focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2"
          >
            <!-- Primer Octicons mark-github-16, licensed in public/licenses/primer-octicons.txt. -->
            <svg width="20" height="20" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true" focusable="false">
              <path d="M6.766 11.328c-2.063-.25-3.516-1.734-3.516-3.656 0-.781.281-1.625.75-2.188-.203-.515-.172-1.609.063-2.062.625-.078 1.468.25 1.968.703.594-.187 1.219-.281 1.985-.281.765 0 1.39.094 1.953.265.484-.437 1.344-.765 1.969-.687.218.422.25 1.515.046 2.047.5.593.766 1.39.766 2.203 0 1.922-1.453 3.375-3.547 3.64.531.344.89 1.094.89 1.954v1.625c0 .468.391.734.86.547C13.781 14.359 16 11.53 16 8.03 16 3.61 12.406 0 7.984 0 3.563 0 0 3.61 0 8.031a7.88 7.88 0 0 0 5.172 7.422c.422.156.828-.125.828-.547v-1.25c-.219.094-.5.156-.75.156-1.031 0-1.64-.562-2.078-1.609-.172-.422-.36-.672-.719-.719-.187-.015-.25-.093-.25-.187 0-.188.313-.328.625-.328.453 0 .844.281 1.25.86.313.452.64.655 1.031.655s.641-.14 1-.5c.266-.265.47-.5.657-.656" />
            </svg>
          </a>
        </div>
      </section>
    </div>
    <div class="native-desktop">
      <CertificateDialog
        v-if="selected"
        :connection="selected"
        :for-connection="false"
        @close="selected = null"
        @changed="reload"
      />
    </div>
  </div>
</template>
