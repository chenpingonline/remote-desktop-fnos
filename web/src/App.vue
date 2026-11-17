<script setup lang="ts">
import { onBeforeUnmount, onMounted, watch, ref } from "vue";
import { RouterView, useRoute, useRouter } from "vue-router";
import AppSidebar from "./components/AppSidebar.vue";
import CommandPalette from "./components/CommandPalette.vue";
import RemoteSessionOverlay from "./components/RemoteSessionOverlay.vue";
import { openCommandPalette, toggleSidebar, ui } from "./stores/ui";

import {
  backend,
  initialize,
  login,
  refreshWorkspace,
  state,
} from "./stores/app";
import { toast } from "./stores/ui";
const route = useRoute();
const router = useRouter();
const token = ref("");
const workspaceMain = ref<HTMLElement | null>(null);
const homeEntry =
  !window.location.hash ||
  /^#\/?(?:\?.*)?$/.test(window.location.hash) ||
  window.location.hash === "#/overview";
let startupApplied = false;
watch(
  () => backend.ready,
  (ready) => {
    if (!ready || startupApplied) return;
    startupApplied = true;
    if (homeEntry) router.replace({ name: state.settings.startPage });
  },
);
let poll: ReturnType<typeof setInterval> | undefined;
async function signIn() {
  try {
    const value = token.value;
    token.value = "";
    await login(value);
  } catch (e) {
    backend.error = (e as Error).message;
  }
}
onMounted(async () => {
  await initialize();
  poll = setInterval(() => {
    if (backend.ready && !document.hidden)
      refreshWorkspace().catch((e) => {
        backend.error = e.message;
      });
  }, 10000);
});
onBeforeUnmount(() => clearInterval(poll));

watch(
  () => ui.sidebarOpen,
  (open) => {
    document.body.classList.toggle("nav-open", open);
  },
);

onBeforeUnmount(() => document.body.classList.remove("nav-open"));

watch(
  () => route.fullPath,
  () => {
    toggleSidebar(false);
    workspaceMain.value?.scrollTo({ top: 0 });
  },
);

function handleGlobalKey(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
    event.preventDefault();
    openCommandPalette();
  }
  if (event.key === "Escape" && ui.sidebarOpen) toggleSidebar(false);
}

onMounted(() => window.addEventListener("keydown", handleGlobalKey));
onBeforeUnmount(() => window.removeEventListener("keydown", handleGlobalKey));
</script>

<template>
  <div
    v-if="!backend.ready"
    class="grid min-h-screen place-items-center bg-canvas p-6 text-ink"
  >
    <section class="panel w-full max-w-sm space-y-4 p-6">
      <h1 class="text-lg font-bold">远程桌面</h1>
      <p class="text-sm text-muted">
        {{
          backend.busy
            ? "正在读取 NAS 工作区…"
            : backend.error || "请从飞牛应用入口打开。"
        }}
      </p>
      <el-form v-if="!backend.busy && backend.devLoginAllowed" @submit.prevent="signIn">
        <el-input
          v-model="token"
          type="password"
          aria-label="开发访问令牌"
          placeholder="本地开发访问令牌"
          autocomplete="off"
        />
        <div class="mt-4 flex gap-2">
          <el-button type="primary" :disabled="!token" @click="signIn"
            >登录</el-button
          ><el-button @click="initialize">重试</el-button>
        </div>
      </el-form>
      <el-button v-else-if="!backend.busy" @click="initialize">重试</el-button>
    </section>
  </div>
  <div v-else class="app-shell min-h-screen text-ink">
    <AppSidebar />

    <button
      v-if="ui.sidebarOpen"
      type="button"
      aria-label="关闭导航"
      class="fixed inset-0 z-[35] bg-black/50 backdrop-blur-sm nav:hidden"
      @click="toggleSidebar(false)"
    ></button>

    <div class="app-body flex min-h-screen flex-col nav:pl-[232px]">
      <main ref="workspaceMain" class="flex-1">
        <RouterView v-slot="{ Component }">
          <Transition
            mode="out-in"
            enter-active-class="transition duration-200 ease-out"
            enter-from-class="translate-y-2 opacity-0"
            enter-to-class="translate-y-0 opacity-100"
            leave-active-class="transition duration-150 ease-in"
            leave-from-class="opacity-100"
            leave-to-class="opacity-0"
          >
            <component :is="Component" />
          </Transition>
        </RouterView>
      </main>
    </div>

    <CommandPalette />
    <RemoteSessionOverlay />
  </div>
</template>
