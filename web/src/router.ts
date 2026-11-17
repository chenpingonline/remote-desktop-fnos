import { createRouter, createWebHashHistory } from "vue-router";
import type { RouteRecordRaw } from "vue-router";

const routes: RouteRecordRaw[] = [
  { path: "/", redirect: "/connections" },
  { path: "/overview", redirect: "/connections" },
  {
    path: "/connections",
    name: "connections",
    component: () => import("./views/ConnectionsView.vue"),
    meta: { title: "我的连接" },
  },
  {
    path: "/sessions",
    name: "sessions",
    component: () => import("./views/SessionsView.vue"),
    meta: { title: "会话记录" },
  },
  {
    path: "/credentials",
    name: "credentials",
    component: () => import("./views/CredentialsView.vue"),
    meta: { title: "我的凭据" },
  },
  { path: "/groups", redirect: "/connections" },
  {
    path: "/wol",
    name: "wol",
    component: () => import("./views/WolView.vue"),
    meta: { title: "网络唤醒" },
  },
  {
    path: "/settings",
    name: "settings",
    component: () => import("./views/SettingsView.vue"),
    meta: { title: "设置" },
  },
  { path: "/:pathMatch(.*)*", redirect: "/connections" },
];

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior() {
    return { top: 0 };
  },
});

router.afterEach((to) => {
  document.title = `${to.meta.title || "Relay"} · Relay 远程桌面`;
});

export default router;
