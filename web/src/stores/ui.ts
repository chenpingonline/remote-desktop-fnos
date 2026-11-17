import { h, reactive } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import type { RemoteConnection, WolEntry } from "../types";
import { connectionById, wolById } from "./app";
import { api } from "../api";
import { connectionLaunchLabel, type LocalSession } from "../local-sessions";
import { uid } from "../utils/format";

export type ToastTone = "default" | "success" | "danger" | "info";

export interface ToastOptions {
  description?: string;
  tone?: ToastTone;
  duration?: number;
}

export interface ConfirmOptions {
  appendTo?: HTMLElement;
  title?: string;
  description?: string;
  confirmText?: string;
  cancelText?: string;
  tone?: "danger" | "default";
}

interface RemoteState {
  sessions: LocalSession[];
  open: boolean;
  connection: RemoteConnection | null;
  nonce: number;
}

interface UIState {
  sidebarOpen: boolean;
  commandOpen: boolean;
  remote: RemoteState;
}

export const ui = reactive<UIState>({
  sidebarOpen: false,
  commandOpen: false,
  remote: {
    sessions: [],
    open: false,
    connection: null,
    nonce: 0,
  },
});

export function toast(message: string, options: ToastOptions = {}): string {
  const id = uid("toast");
  const type =
    options.tone === "danger"
      ? "error"
      : options.tone === "success"
        ? "success"
        : options.tone === "info"
          ? "info"
          : "info";
  ElMessage({
    type,
    duration: options.duration ?? 3200,
    showClose: true,
    grouping: true,
    customClass: "relay-message",
    message: h("div", { class: "relay-message-content" }, [
      h("p", message),
      options.description ? h("p", options.description) : null,
    ]),
  });
  return id;
}

export function requestConfirm(options: ConfirmOptions): Promise<boolean> {
  const tone = options.tone || "danger";
  return ElMessageBox.confirm(
    options.description || "",
    options.title || "确认操作",
    {
      appendTo: options.appendTo || document.body,
      confirmButtonText: options.confirmText || "确认",
      cancelButtonText: options.cancelText || "取消",
      type: tone === "danger" ? "warning" : "info",
      customClass: "relay-messagebox",
      confirmButtonClass: tone === "danger" ? "relay-messagebox__danger" : "",
      distinguishCancelAndClose: true,
      autofocus: false,
      closeOnClickModal: false,
    },
  )
    .then(() => true)
    .catch(() => false);
}

export function openCommandPalette(): void {
  ui.commandOpen = true;
}

export function closeCommandPalette(): void {
  ui.commandOpen = false;
}

export function toggleSidebar(value?: boolean): void {
  ui.sidebarOpen = typeof value === "boolean" ? value : !ui.sidebarOpen;
}

export function remoteActionLabel(connection: { id: string; status: string }) {
  return connectionLaunchLabel(ui.remote.sessions, connection.id, connection.status);
}

export function launchRemote(
  connection: RemoteConnection | null | undefined,
): void {
  if (!connection) return;
  if (!connection.id) {
    toast("请先在我的连接中保存设备，再发起连接", { tone: "info" });
    return;
  }
  ui.remote.connection = { ...connection };
  ui.remote.open = true;
  ui.remote.nonce += 1;
  ui.sidebarOpen = false;
}

export function closeRemote(): void {
  ui.remote.open = false;
  ui.remote.connection = null;
}

export async function wakeWolEntry(entry: WolEntry): Promise<void> {
  try {
    if (!entry.connectionId) throw new Error("请先关联远程连接");
    const result = await api(
      "/connections/" + entry.connectionId + "/wake",
      "POST",
      {},
    );
    const current = wolById(entry.id);
    if (current) current.lastWokenAt = result.sentAt || new Date().toISOString();
    toast(result.message || "魔术包已发送", {
      description: "已广播唤醒请求，设备是否开机需另行确认",
      tone: "success",
    });
  } catch (e) {
    toast((e as Error).message, { tone: "danger" });
  }
}
