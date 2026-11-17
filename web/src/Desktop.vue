<script setup lang="ts">
import { displayTransform, performanceDisplay, loadWallpaper, saveWallpaper, loadRenderMode, saveRenderMode, renderModeOptions, type RenderMode, loadDisplayScale, saveDisplayScale, resolutionOptions, loadResolutionMode, saveResolutionMode } from "./display";
import AppSelect from "./components/AppSelect.vue";
import { ref, computed, onMounted, onBeforeUnmount, watch, useId, nextTick } from "vue";
import Guacamole from "./vendor/guacamole.js";
import { sendClipboard, receiveClipboard } from "./clipboard";
import { api, states, reasons } from "./api";
import { Maximize, Unplug, Send, Monitor, SlidersHorizontal, X, Settings, Clipboard } from "lucide-vue-next";
const props = defineProps<{
    sessionId: string;
    name: string;
    protocol: string;
    connectionId?: string;
    targetOs?: string;
    clipboard: boolean;
    toolbarTarget?: string;
    active?: boolean;
    renderMode?: RenderMode;
    wallpaper?: boolean;
  }>(),
  emit = defineEmits<{ "status-change": [status: string]; close: []; reconnect: []; progress: [stage: "connecting" | "waiting" | "ready" | "failed", message?: string] }>();
const initialWallpaper = props.wallpaper ?? loadWallpaper();
const wallpaper = ref(initialWallpaper);
const wallpaperPending = computed(() => props.protocol === "rdp" && wallpaper.value !== initialWallpaper);
const initialRenderMode = props.renderMode || loadRenderMode(props.protocol);
const renderMode = ref<string>(initialRenderMode);
const renderModePending = computed(() => renderMode.value !== initialRenderMode);
function changeRenderMode() { saveRenderMode(props.protocol, renderMode.value); }
function applySettings() {
  // Save all selections together so one reconnect applies the current panel.
  saveRenderMode(props.protocol, renderMode.value);
  if (props.protocol === "rdp") saveDisplayScale(props.protocol, Number(displayScale.value));
  if (props.protocol === "rdp") saveWallpaper(wallpaper.value);
  emit("reconnect");
}
const resolutionMode = ref(loadResolutionMode(props.protocol));
const remoteResolution = ref({ width: 0, height: 0 });
function changeResolutionMode() {
  saveResolutionMode(props.protocol, resolutionMode.value);
  clearTimeout(resizeRetry);
  updateViewport();
}
const initialContentScale = loadDisplayScale(props.protocol);
const displayScale = ref(String(initialContentScale));
const scalePending = computed(() => Number(displayScale.value) !== initialContentScale);
const resizeInfo = ref("");
const reconnectPending = computed(() => wallpaperPending.value || renderModePending.value || (props.protocol === "rdp" && scalePending.value) || resizeInfo.value.includes("未确认"));
const pendingLabels = computed(() => [wallpaperPending.value ? "桌面壁纸" : "", renderModePending.value ? "画质模式" : "", props.protocol === "rdp" && scalePending.value ? "内容大小" : "", resizeInfo.value.includes("未确认") ? "分辨率" : ""].filter(Boolean).join("、"));
function changeDisplayScale() {
  saveDisplayScale(props.protocol, Number(displayScale.value));
}
const localMac = /Mac|iPhone|iPad/.test(navigator.platform);
const pasteStorageKey = `remote-desktop.vnc-paste.${props.connectionId || props.sessionId}`;
function loadPasteModifier() {
  try { const saved=localStorage.getItem(pasteStorageKey);if(saved==='ctrl'||saved==='command')return saved; } catch {}
  // Preserve explicitly configured legacy Mac connections; new auto connections use Ctrl.
  return props.targetOs === 'macos' ? 'command' : 'ctrl';
}
const pasteModifier=ref(loadPasteModifier());
function savePasteModifier(){try{localStorage.setItem(pasteStorageKey,pasteModifier.value)}catch{}}
function remotePasteKey(){return props.protocol==='vnc' && pasteModifier.value==='command' ? 0xffe7 : 0xffe3;}

const modifierStorageKey = `remote-desktop.keyboard-mapping.${props.protocol}`;
function savedModifierMode() {
  try {
    const saved = localStorage.getItem(modifierStorageKey);
    if (saved && ["native", "mac", "swap"].includes(saved)) return saved;
  } catch { /* Storage can be disabled in embedded browsers. */ }
  return localMac && props.protocol === "rdp" ? "mac" : "native";
}
const modifierMode = ref(savedModifierMode());
const heldModifiers = new Map<number, number>();
function mappedModifier(key: number) {
  if (modifierMode.value !== "native") {
    if (key === 0xffe7 || key === 0xffeb) return 0xffe3;
    if (key === 0xffe8 || key === 0xffec) return 0xffe4;
    if (modifierMode.value === "swap") {
      if (key === 0xffe3) return props.protocol === "rdp" ? 0xffeb : 0xffe7;
      if (key === 0xffe4) return props.protocol === "rdp" ? 0xffec : 0xffe8;
    }
  }
  return key;
}
function sendKeyboardKey(pressed: boolean, key: number) {
  const modifier = [0xffe3, 0xffe4, 0xffe7, 0xffe8, 0xffeb, 0xffec].includes(key);
  if (!modifier) { client.sendKeyEvent(pressed ? 1 : 0, key); return; }
  if (pressed) {
    if (heldModifiers.has(key)) return;
    const mapped = mappedModifier(key);
    const alreadyHeld = [...heldModifiers.values()].includes(mapped);
    heldModifiers.set(key, mapped);
    if (!alreadyHeld) client.sendKeyEvent(1, mapped);
  } else {
    const mapped = heldModifiers.get(key);
    heldModifiers.delete(key);
    if (mapped !== undefined && ![...heldModifiers.values()].includes(mapped)) client.sendKeyEvent(0, mapped);
  }
}
function changeModifierMode() {
  release();
  try { localStorage.setItem(modifierStorageKey, modifierMode.value); } catch { /* Session choice still works. */ }
}
const toolsOpen = ref(false), tools = ref<HTMLElement>(), settingsOpen = ref(false), settingsButton = ref<HTMLButtonElement>();
const settingsPanel = ref<HTMLElement>();
const settingsMaxHeight = ref<string>();
function updateSettingsBounds() {
  if (!settingsOpen.value || !settingsPanel.value || !panel.value) return;
  const top = settingsPanel.value.getBoundingClientRect().top;
  const bottom = Math.min(window.innerHeight, panel.value.getBoundingClientRect().bottom);
  settingsMaxHeight.value = `${Math.max(0, bottom - top - 8)}px`;
}
watch(settingsOpen, (open) => { if (open) nextTick(updateSettingsBounds); });
watch(() => props.active, (active) => {
  if (active === false) { closeTools(); clipboardOpen.value = false; release(); }
});
const contentLabelId = useId();
const clipboardId = useId();
const clipboardButton = ref<HTMLButtonElement>();
function toggleSettings() { showTools(); settingsOpen.value = !settingsOpen.value; if (settingsOpen.value) clipboardOpen.value = false; }
function toggleClipboard() {
  if (clipboardOpen.value) { closeClipboard(); return; }
  settingsOpen.value = false;
  clipboardOpen.value = true;
  nextTick(() => panel.value?.querySelector<HTMLTextAreaElement>(".clipboard textarea")?.focus());
}
function closeClipboard() {
  clipboardOpen.value = false;
  showTools();
  clipboardButton.value?.focus();
}
function closeSettings() { settingsOpen.value = false; settingsButton.value?.focus(); }
function chooseContentSize(value: number) { displayScale.value = String(value); changeDisplayScale(); }
function contentSizeKey(event: KeyboardEvent) {
  const values = [0.75, 1, 1.5];
  let index = values.indexOf(Number(displayScale.value));
  if (index < 0) index = 1;
  if (["ArrowRight", "ArrowDown"].includes(event.key)) index = (index + 1) % values.length;
  else if (["ArrowLeft", "ArrowUp"].includes(event.key)) index = (index + values.length - 1) % values.length;
  else if (event.key === "Home") index = 0;
  else if (event.key === "End") index = values.length - 1;
  else return;
  event.preventDefault(); event.stopPropagation();
  chooseContentSize(values[index]);
  (event.currentTarget as HTMLElement).querySelectorAll<HTMLButtonElement>("[role=radio]")[index]?.focus();
}
const handleFaded = ref(false), panelSize = ref({ width: 0, height: 0 });
const handlePosition = ref({ x: 1, y: 0.22 });
try {
  const saved = JSON.parse(localStorage.getItem("remote-desktop.tools-position") || "null");
  if (saved && [saved.x, saved.y].every((v: unknown) => typeof v === "number" && v >= 0 && v <= 1)) handlePosition.value = saved;
} catch { /* Position is optional. */ }
const handleLeft = computed(() => 8 + Math.max(0, panelSize.value.width - 60) * handlePosition.value.x);
const handleTop = computed(() => 8 + Math.max(0, panelSize.value.height - 60) * handlePosition.value.y);
let toolsTimer: ReturnType<typeof setTimeout> | undefined,
  fadeTimer: ReturnType<typeof setTimeout> | undefined;
let handleDrag: { id: number; x: number; y: number; left: number; top: number } | undefined;
let draggedHandle = false;
function fadeHandleLater() {
  clearTimeout(fadeTimer);
  fadeTimer = setTimeout(() => { if (!toolsOpen.value && !handleDrag) handleFaded.value = true; }, 1800);
}
function wakeHandle() { clearTimeout(fadeTimer); handleFaded.value = false; }
function showTools() { clearTimeout(toolsTimer); wakeHandle(); toolsOpen.value = true; }
function closeTools() { toolsOpen.value = false; settingsOpen.value = false; fadeHandleLater(); }
function operateDesktop() {
  closeTools();
  // A click may return to an already-focused canvas (no focus event).
  syncLocalClipboard();
}
function closeToolsAndFocus() {
  closeTools();
  if (props.toolbarTarget) { settingsButton.value?.focus(); return; }
  panel.value?.querySelector<HTMLButtonElement>(".desktop-tools-handle")?.focus();
}
function toggleTools(event: MouseEvent) {
  if (draggedHandle && event.detail !== 0) { draggedHandle = false; return; }
  toolsOpen.value ? closeTools() : showTools();
}
function startHandleDrag(event: PointerEvent) {
  if (event.button !== 0) return;
  event.stopPropagation();
  wakeHandle();
  draggedHandle = false;
  handleDrag = { id: event.pointerId, x: event.clientX, y: event.clientY, left: handleLeft.value, top: handleTop.value };
  (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
}
function moveHandle(event: PointerEvent) {
  if (!handleDrag || handleDrag.id !== event.pointerId) return;
  event.stopPropagation();
  const dx = event.clientX - handleDrag.x, dy = event.clientY - handleDrag.y;
  if (!draggedHandle && Math.hypot(dx, dy) < 5) return;
  draggedHandle = true;
  handlePosition.value = {
    x: Math.max(0, Math.min(1, (handleDrag.left + dx - 8) / Math.max(panelSize.value.width - 60, 1))),
    y: Math.max(0, Math.min(1, (handleDrag.top + dy - 8) / Math.max(panelSize.value.height - 60, 1))),
  };
}
function endHandleDrag(event: PointerEvent) {
  if (handleDrag?.id !== event.pointerId) return;
  event.stopPropagation();
  handleDrag = undefined;
  if ((event.currentTarget as HTMLElement).hasPointerCapture(event.pointerId)) (event.currentTarget as HTMLElement).releasePointerCapture(event.pointerId);
  if (draggedHandle) {
    try { localStorage.setItem("remote-desktop.tools-position", JSON.stringify(handlePosition.value)); } catch { /* Session position still works. */ }
  }
  fadeHandleLater();
}
function hideToolsLater() {
  if (props.toolbarTarget) return;
  clearTimeout(toolsTimer);
  toolsTimer = setTimeout(() => {
    if (status.value === "已连接" && !error.value && !tools.value?.contains(document.activeElement) && !tools.value?.matches(":hover")) closeTools();
  }, 1800);
}
const surface = ref<HTMLElement>();
let renderedScale = { x: 1, y: 1 };
const canvas = ref<HTMLElement>(),
  panel = ref<HTMLElement>(),
  status = ref("连接中"),
  error = ref(""),
  text = ref(""),
  zoom = ref("fit"),
  clipboardOpen = ref(false),
  clipboardInfo = ref("已启用文字双向同步");
watch(status, (value) => emit("status-change", value), { immediate: true });
let client: any,
  keyboard: any,
  timer: ReturnType<typeof setInterval>,
  observer: ResizeObserver,
  closed = false,
  connected = false,
  firstFrame = false,
  resizeTimer: ReturnType<typeof setTimeout> | undefined,
  resizeRetry: ReturnType<typeof setTimeout> | undefined,
  requestedSize: { width: number; height: number } | undefined,
  resizeSentAt = 0,
  queuedSize: { width: number; height: number } | undefined;
watch([status, error], () => {
  if (error.value || status.value !== "已连接") showTools();
  else hideToolsLater();
});
function release() {
  keyboard?.reset();
  for (const mapped of new Set(heldModifiers.values())) client?.sendKeyEvent(0, mapped);
  heldModifiers.clear();
}
function fit() {
  if (!client || !canvas.value) return;
  const d = client.getDisplay();
  if (!d.getWidth() || !d.getHeight()) return;
  const width = canvas.value.clientWidth, height = canvas.value.clientHeight;
  if (!width || !height) return; // Inactive tabs have no usable viewport.
  if (!surface.value) return;
  renderedScale = displayTransform(width, height, d.getWidth(), d.getHeight(), zoom.value);
  // Own the scaled layout bounds. A CSS transform alone leaves the unscaled
  // Guacamole descendants contributing to scrollable overflow.
  d.scale(1);
  const el = d.getElement();
  el.style.transformOrigin = "0 0";
  el.style.transform = `scale(${renderedScale.x}, ${renderedScale.y})`;
  surface.value.style.width = `${Math.min(width, d.getWidth() * renderedScale.x)}px`;
  if (!["fit", "fill"].includes(zoom.value)) surface.value.style.width = `${d.getWidth() * renderedScale.x}px`;
  surface.value.style.height = `${d.getHeight() * renderedScale.y}px`;
}
function remoteSizeMatches() {
  if (!requestedSize || !client) return false;
  const display = client.getDisplay();
  // RDP rounds width to a multiple of 2 or 4 pixels.
  return Math.abs(display.getWidth() - requestedSize.width) <= 4 &&
    Math.abs(display.getHeight() - requestedSize.height) <= 4;
}
function requestRemoteSize() {
  clearTimeout(resizeRetry);
  if (!connected || closed || !canvas.value?.clientWidth || !canvas.value.clientHeight) return;
  if (requestedSize) {
    if (remoteSizeMatches()) { requestedSize = undefined; resizeInfo.value = ""; }
    else if (Date.now() - resizeSentAt < 8000) {
      resizeRetry = setTimeout(requestRemoteSize, 500);
      return;
    } else {
      // An unconfirmed update must not be retried indefinitely.
      resizeInfo.value = "远端未确认分辨率，请重连应用";
      return;
    }
  }
  if (!queuedSize) return;
  const size = queuedSize;
  const display = client.getDisplay();
  if (Math.abs(display.getWidth() - size.width) <= 4 && Math.abs(display.getHeight() - size.height) <= 4) {
    queuedSize = undefined;
    resizeInfo.value = "";
    return;
  }
  // Coalesce rapid selections and wait between acknowledged display updates.
  if (Date.now() - resizeSentAt < 1000) {
    resizeRetry = setTimeout(requestRemoteSize, 1000 - (Date.now() - resizeSentAt));
    return;
  }
  queuedSize = undefined;
  requestedSize = size;
  resizeSentAt = Date.now();
  resizeInfo.value = "正在调整分辨率";
  client.sendSize(size.width, size.height);
  resizeRetry = setTimeout(requestRemoteSize, 500);
}
function updateViewport() {
  nextTick(updateSettingsBounds);
  if (panel.value?.clientWidth && panel.value.clientHeight) panelSize.value = { width: panel.value.clientWidth, height: panel.value.clientHeight };
  fit();
  clearTimeout(resizeTimer);
  if (closed || !connected || props.protocol !== "rdp") {
    clearTimeout(resizeRetry);
    return;
  }
  resizeTimer = setTimeout(() => {
    if (closed || !connected || !canvas.value) return;
    const { clientWidth: width, clientHeight: height } = canvas.value;
    if (!width || !height) return;
    queuedSize = performanceDisplay(width, height, initialContentScale, resolutionMode.value, initialRenderMode);
    requestRemoteSize();
  }, 600);
}
async function disconnect() {
  closed = true;
  connected = false;
  stopClipboard();
  clearTimeout(resizeTimer);
  clearTimeout(resizeRetry);
  release();
  client?.disconnect();
  try {
    await api("/sessions/" + props.sessionId, "DELETE");
  } catch (e) {
    error.value = String(e);
  }
  status.value = "已断开";
}
function special(keys: number[]) {
  if (status.value !== "已连接") return;
  release();
  keys.forEach((k) => client.sendKeyEvent(1, k));
  [...keys].reverse().forEach((k) => client.sendKeyEvent(0, k));
}
async function fullscreen() {
  try {
    await panel.value?.requestFullscreen();
  } catch {
    error.value = "浏览器未允许全屏，请在独立标签页重试";
  }
}
let clipboardQueue = Promise.resolve(), clipboardRevision = 0;
const clipboardTransfers = new Set<() => void>();
function stopClipboard() {
  ++clipboardRevision;
  for (const cancel of clipboardTransfers) cancel();
  clipboardTransfers.clear();
}
let lastRemoteText: string | undefined;
function clipboardActive() {
  return props.clipboard && connected && !closed && !document.hidden &&
    document.hasFocus() && !!canvas.value?.getClientRects().length &&
    document.activeElement === canvas.value;
}
function transferText(value: string, paste = false) {
  const operation = clipboardQueue.catch(() => {}).then(async () => {
    if (!clipboardActive()) return;
    const bytes = new TextEncoder().encode(value).length;
    // The gateway limits total base64 payload, including padding per 4096-byte chunk.
    const encodedBytes = Math.floor(bytes / 4096) * 5464 + Math.ceil((bytes % 4096) / 3) * 4;
    if (encodedBytes > 1 << 20) {
      clipboardInfo.value = "文字过长，未同步；请分段复制";
      return;
    }
    if (value !== lastRemoteText) {
      await new Promise<void>((resolve, reject) => {
        const stream = client.createClipboardStream("text/plain");
        const finish = (message?: string) => {
          clearTimeout(timeout);
          clipboardTransfers.delete(cancel);
          stream.onack = () => {};
          if (message) { stream.sendEnd(); reject(new Error(message)); }
          else resolve();
        };
        const cancel = () => finish("剪贴板发送已取消");
        const timeout = setTimeout(() => finish("剪贴板发送超时，请重试"), 10000);
        clipboardTransfers.add(cancel);
        sendClipboard(stream, value, finish, () => finish());
      });
      if (!clipboardActive()) return;
      lastRemoteText = value;
    }
    if (paste) special([remotePasteKey(), 0x76]);
    clipboardInfo.value = "文字已同步";
  });
  clipboardQueue = operation.catch((e) => { if (!closed) clipboardInfo.value = String(e.message || e); });
}
async function syncLocalClipboard() {
  if (!clipboardActive()) return;
  const revision = ++clipboardRevision;
  try {
    const value = await navigator.clipboard.readText();
    if (revision !== clipboardRevision || !clipboardActive()) return;
    text.value = value;
    transferText(value);
  } catch {
    clipboardInfo.value = "自动读取受限；可直接在桌面粘贴，或使用面板发送";
  }
}
function pasteText(event: ClipboardEvent) {
  if (!clipboardActive() || !event.clipboardData) return;
  event.preventDefault();
  event.stopImmediatePropagation();
  release();
  ++clipboardRevision; // Discard any older read initiated by focus.
  if (!event.clipboardData.types.includes("text/plain")) {
    clipboardInfo.value = "仅支持文字剪贴板";
    return;
  }
  text.value = event.clipboardData.getData("text/plain");
  transferText(text.value, true);
}
async function pasteWithControl() {
  const revision = ++clipboardRevision;
  try {
    // Read within the key gesture: Control+V does not generate native paste on Mac.
    const value = await navigator.clipboard.readText();
    if (revision !== clipboardRevision || !clipboardActive()) return;
    text.value = value;
    transferText(value, true);
  } catch {
    if (revision !== clipboardRevision || !clipboardActive()) return;
    // The remote clipboard may already contain text sent from the manual panel.
    special([remotePasteKey(), 0x76]);
    clipboardInfo.value = "无法读取本机文字；已执行远端粘贴，新文字请用 Command+V 或面板发送";
  }
}
function clipboardShortcut(event: KeyboardEvent) {
  if (!clipboardActive()) return;
  const key = event.key.toLowerCase();
  if (((event.ctrlKey || event.metaKey) && key === "v") || (event.shiftKey && key === "insert")) {
    if (localMac && event.ctrlKey && !event.metaKey && key === "v" && !event.altKey) {
      event.preventDefault();
      event.stopImmediatePropagation();
      release();
      if (!event.repeat) void pasteWithControl();
      return;
    }
    // Allow the browser's native paste event, but keep Guacamole from sending V
    // before the clipboard stream has finished.
    event.stopImmediatePropagation();
  } else if (event.metaKey && ["c", "x"].includes(key) &&
      (props.protocol !== "rdp" || modifierMode.value !== "native")) {
    event.preventDefault();
    event.stopImmediatePropagation();
    release();
    special([mappedModifier(0xffe7), key.charCodeAt(0)]);
  }
}
async function receiveRemoteText(value: string) {
  if (closed || !connected) return;
  ++clipboardRevision;
  text.value = value;
  lastRemoteText = value;
  if (!clipboardActive()) return;
  try {
    await navigator.clipboard.writeText(value);
    clipboardInfo.value = "远端文字已复制到本机";
  } catch {
    clipboardInfo.value = "自动写入受限；请点击“复制到本机”";
  }
}
function sendText() {
  canvas.value?.focus({ preventScroll: true });
  ++clipboardRevision;
  transferText(text.value);
}
async function copyText() {
  try {
    await navigator.clipboard.writeText(text.value);
    clipboardInfo.value = "已复制到本机";
  } catch {
    // User-triggered fallback for HTTP and browsers without the async API.
    const field = panel.value?.querySelector<HTMLTextAreaElement>(".clipboard textarea");
    field?.focus();
    field?.select();
    clipboardInfo.value = field && document.execCommand("copy")
      ? "已复制到本机" : "请选中文字后复制";
  }
}
onMounted(async () => {
  fadeHandleLater();
  updateViewport();
  window.addEventListener("blur", release);
  window.addEventListener("focus", syncLocalClipboard);
  window.addEventListener("resize", updateViewport);
  document.addEventListener("fullscreenchange", updateViewport);
  try {
    const t = await api("/sessions/" + props.sessionId + "/ticket", "POST", {});
    if (closed) return;
    const tunnel = new Guacamole.WebSocketTunnel(t.webSocketPath);
    client = new Guacamole.Client(tunnel);
    tunnel.onstatechange = (state: number) => {
      if (closed) return;
      if (state === 0) emit('progress', 'connecting');
      if (state === 1) emit('progress', 'waiting');
      if (state === 2) status.value = firstFrame ? '已中断' : '连接失败';
      if (state === 3) status.value = '连接不稳定';
    };
    const el = client.getDisplay().getElement();
    surface.value!.appendChild(el);
    client.onstatechange = (state: number) => {
      connected = state === 3; // Guacamole.Client.State.CONNECTED
      if (state === 1) emit('progress', 'connecting');

      if (!closed && state >= 4) status.value = firstFrame ? '已中断' : '连接失败';
      if (connected) updateViewport();
      else { clearTimeout(resizeTimer); clearTimeout(resizeRetry); stopClipboard(); }
    };
    client.onsync = () => {
      if (firstFrame || closed) return;
      // A sync only schedules rendering. Wait for the display queue to flush.
      client.getDisplay().flush(() => {
        if (!closed && !firstFrame && connected && client.getDisplay().getWidth() > 0 && client.getDisplay().getHeight() > 0) {
          firstFrame = true;
          status.value = '已连接';
          emit('progress', 'ready');
        }
      });
    };
    client.onerror = () => {
      status.value = "连接失败";
      error.value = "桌面连接失败，正在读取详细状态";
      emit("progress", "failed", error.value);
      release();
    };
    client.getDisplay().onresize = () => {
      remoteResolution.value = { width: client.getDisplay().getWidth(), height: client.getDisplay().getHeight() };
      fit();
      if (connected && props.protocol === "rdp") requestRemoteSize();
    };
    client.onclipboard = (stream: any, mime: string) => {
      if (!props.clipboard || mime !== "text/plain") {
        client.sendAck(stream.index, "不支持该类型", 256);
        return;
      }
      receiveClipboard(stream, receiveRemoteText);
    };
    keyboard = new Guacamole.Keyboard(canvas.value);
    keyboard.onkeydown = (k: number) => {
      if (status.value === "已连接") sendKeyboardKey(true, k);
      return false;
    };
    keyboard.onkeyup = (k: number) => sendKeyboardKey(false, k);
    const mouse = new Guacamole.Mouse(el);
    mouse.onEach(["mousedown", "mouseup", "mousemove"], (e: any) => {
      if (status.value === "已连接") {
        canvas.value?.focus();
        const remote = new Guacamole.Mouse.State(e.state);
        remote.x /= renderedScale.x;
        remote.y /= renderedScale.y;
        client.sendMouseState(remote);
      }
    });
    observer = new ResizeObserver(updateViewport);
    observer.observe(canvas.value!);
    client.connect(new URLSearchParams({ ticket: t.ticket }).toString());
    timer = setInterval(async () => {
      try {
        const s = await api("/sessions/" + props.sessionId);
        status.value = s.state === "ACTIVE" && !connected ? "已中断" : states[s.state] || s.state;
        if (["FAILED", "CLOSED", "INTERRUPTED"].includes(s.state)) {
          error.value = reasons[s.endReason] || s.endReason || "连接已结束";
          emit("progress", "failed", error.value);
          release();
          client?.disconnect();
          clearInterval(timer);
        }
      } catch (e) {
        error.value = String(e);
      }
    }, 1000);
  } catch (e) {
    error.value = String(e);
    status.value = "连接失败";
    emit("progress", "failed", (e as Error).message || String(e));
  }
});
onBeforeUnmount(() => {
  closed = true;
  clearTimeout(toolsTimer);
  clearTimeout(fadeTimer);
  handleDrag = undefined;
  stopClipboard();
  clearInterval(timer);
  clearTimeout(resizeTimer);
  clearTimeout(resizeRetry);
  observer?.disconnect();
  window.removeEventListener("resize", updateViewport);
  document.removeEventListener("fullscreenchange", updateViewport);
  window.removeEventListener("blur", release);
  window.removeEventListener("focus", syncLocalClipboard);
  release();
  client?.disconnect();
  api("/sessions/" + props.sessionId, "DELETE").catch(() => {});
});
</script>
<template>
  <section ref="panel" class="desktop-panel">
    <button v-if="!toolbarTarget" class="desktop-tools-handle" :class="{ 'handle-faded': handleFaded && !toolsOpen, 'handle-behind-settings': settingsOpen }"
      :style="{ left: handleLeft + 'px', top: handleTop + 'px' }"
      :aria-label="toolsOpen ? '收起桌面工具栏' : '显示桌面工具栏'" :aria-expanded="toolsOpen"
      title="桌面工具 · 拖动可移动位置" @click="toggleTools"
      @pointerdown="startHandleDrag" @pointermove="moveHandle" @pointerup="endHandleDrag" @pointercancel="endHandleDrag"
      @pointerenter="wakeHandle" @pointerleave="fadeHandleLater" @focus="wakeHandle" @blur="fadeHandleLater">
      <X v-if="toolsOpen" :size="18"/><SlidersHorizontal v-else :size="18"/>
    </button>
    <div ref="tools" role="region" aria-label="桌面工具" :aria-hidden="!toolbarTarget && !toolsOpen" @keydown.esc.stop.prevent="closeToolsAndFocus" class="desktop-overlay" :class="{ 'tools-open': toolsOpen || toolbarTarget, 'integrated-overlay': toolbarTarget }" @mouseenter="showTools" @mouseleave="hideToolsLater" @focusin="showTools" @focusout="hideToolsLater">
      <Teleport :to="toolbarTarget || 'body'" :disabled="!toolbarTarget">
      <div v-show="!toolbarTarget || active" class="desktop-tools" :class="{ 'integrated-tools': toolbarTarget }">
        <div v-if="!toolbarTarget" class="desktop-identity"><Monitor v-if="!toolbarTarget" :size="18"/><strong v-if="!toolbarTarget">{{ name }}</strong><span class="status" :class="{ good: status === '已连接' }">{{ status }}</span></div>
        <div class="desktop-actions">
          <button ref="settingsButton" aria-label="会话设置" title="会话设置" :class="{ 'session-settings-active': settingsOpen }" :aria-expanded="settingsOpen" @click="toggleSettings"><Settings :size="16" aria-hidden="true"/><span v-if="!toolbarTarget">会话设置</span></button>
          <button v-if="clipboard" ref="clipboardButton" aria-label="剪贴板" :class="{ 'session-settings-active': clipboardOpen }" :title="'剪贴板 · ' + clipboardInfo" :aria-description="clipboardInfo" :aria-expanded="clipboardOpen" :aria-controls="clipboardId + '-panel'" @click="toggleClipboard"><Clipboard :size="16"/><span v-if="!toolbarTarget" class="secondary-label">剪贴板</span><span v-if="clipboardInfo.includes('受限') || clipboardInfo.includes('过长')" class="clipboard-attention" aria-hidden="true"/></button>
          <button v-if="!toolbarTarget" aria-label="全屏" @click="fullscreen"><Maximize :size="16"/></button>
          <button v-if="!toolbarTarget" aria-label="断开" @click="disconnect"><Unplug :size="16"/><span class="secondary-label">断开</span></button>
          <button v-if="['已断开', '连接失败', '已中断'].includes(status)" @click="emit('reconnect')">重新连接</button>
        </div>
      </div>
      </Teleport>
      <div v-if="error" class="desktop-message error" role="alert">{{ error }}</div>
      <section ref="settingsPanel" v-show="settingsOpen" class="session-settings" :style="{ maxHeight: settingsMaxHeight }" role="dialog" aria-label="会话设置" aria-modal="false" @keydown.esc.stop.prevent="closeSettings">
        <div class="session-settings-heading"><h2>会话设置</h2><button aria-label="关闭会话设置" @click="closeSettings"><X :size="16"/></button></div>
        <div class="session-settings-body">
          <section class="session-settings-group"><h3>画质与性能</h3>
            <label class="session-row">画质模式<AppSelect v-model="renderMode" @change="changeRenderMode" aria-label="画质模式" :options="renderModeOptions"/></label>
            <p class="session-performance-hint">{{ renderMode === 'clear' ? '无损画面，适合文字和代码；视频可能较慢。' : renderMode === 'smooth' ? (protocol === 'rdp' ? '优先动态画面；自动分辨率最高 1920 × 1080，固定分辨率保持所选尺寸。' : '降低图像质量与压缩负担，优先动态画面。') : '自动平衡画面清晰度和传输速度，适合日常操作。' }}</p>
          </section>
          <section class="session-settings-group"><h3>显示</h3>
            <label v-if="protocol === 'rdp'" class="session-row">桌面壁纸<input class="session-wallpaper-switch" type="checkbox" role="switch" aria-label="桌面壁纸" v-model="wallpaper" @change="saveWallpaper(wallpaper)" /></label>
            <label class="session-row">画面适配<AppSelect v-model="zoom" @change="updateViewport" aria-label="画面缩放"
              :options="[
                { value: 'fit', label: '适应窗口（保持比例）' }, { value: 'fill', label: '铺满窗口（拉伸）' }, { value: '1', label: '像素原始尺寸' },
                { value: '1.25', label: '放大 125%' }, { value: '1.5', label: '放大 150%' },
                { value: '2', label: '放大 200%' }, { value: '0.75', label: '缩小 75%' }, { value: '0.5', label: '缩小 50%' },
              ]"/></label>
            <label v-if="protocol === 'rdp'" class="session-row">分辨率<AppSelect v-model="resolutionMode" @change="changeResolutionMode" aria-label="分辨率" :options="resolutionOptions"/></label>
            <div class="session-row session-resolution-readout"><span>实际分辨率</span><output aria-label="实际分辨率">{{ remoteResolution.width && remoteResolution.height ? `${remoteResolution.width} × ${remoteResolution.height}` : '等待远端返回' }}{{ resizeInfo ? ` · ${resizeInfo}` : '' }}</output></div>
            <div v-if="protocol === 'rdp'" class="session-row session-content-size"><span :id="contentLabelId">内容大小</span><fieldset title="内容大小保存后，通过重连应用；Windows 将按匹配的显示比例绘制">
              <div class="content-size-options" role="radiogroup" :aria-labelledby="contentLabelId" @keydown="contentSizeKey">
                <button v-for="preset in [{ value: 0.75, label: '小' }, { value: 1, label: '标准' }, { value: 1.5, label: '大' }]" :key="preset.value" role="radio" :aria-checked="Number(displayScale) === preset.value" @click="chooseContentSize(preset.value)">{{ preset.label }}</button>
              </div>
              <details class="session-scale-details"><summary>更多比例<span>{{ Math.round(Number(displayScale) * 100) }}%</span></summary>
                <AppSelect v-model="displayScale" @change="changeDisplayScale" aria-label="显示缩放" :options="[
                  { value: '0.5', label: '50% · 更多内容' }, { value: '0.75', label: '75%' },
                  { value: '1', label: '100% · 标准' }, { value: '1.25', label: '125%' },
                  { value: '1.5', label: '150% · 大字' }, { value: '1.75', label: '175%' }, { value: '2', label: '200%' },
                ]"/>
              </details>
            </fieldset></div>
          </section>
          <section class="session-settings-group"><h3>键盘</h3>
            <label class="session-row">按键映射<AppSelect v-model="modifierMode" @change="changeModifierMode" aria-label="键盘映射" :options="[
              { value: 'native', label: '沿用本机设置' }, { value: 'mac', label: 'Command / Win 作为 Ctrl' },
              { value: 'swap', label: '交换 Ctrl 与 Command / Win' },
            ]"/></label>
            <label v-if="protocol==='vnc'" class="session-row">远端粘贴快捷键<AppSelect v-model="pasteModifier" @change="savePasteModifier" aria-label="远端粘贴快捷键" :options="[{value:'ctrl',label:'Ctrl+V · Windows / Linux'},{value:'command',label:'Command+V · macOS'}]"/></label>
            <div class="session-row"><span>特殊按键</span><div class="session-key-actions">
              <button :disabled="status !== '已连接'" @click="special([0xffe3, 0xffe9, 0xffff])">Ctrl+Alt+Del</button>
              <button :disabled="status !== '已连接'" @click="special([protocol === 'rdp' ? 0xffeb : 0xffe7])">{{ protocol === 'rdp' ? 'Win' : 'Win / Command' }}</button>
              <button :disabled="status !== '已连接'" @click="special([0xffe9])">{{ protocol === 'rdp' ? 'Alt' : 'Option' }}</button>
            </div></div>
          </section>
        </div>
        <footer v-if="reconnectPending" class="session-settings-footer">
          <p>{{ pendingLabels }}待应用</p>
          <button @click="applySettings">重新连接并应用</button>
        </footer>
      </section>
    </div>
    <div
      ref="canvas"
      tabindex="0"
      class="remote-canvas"
      :class="{ 'remote-canvas-fit': zoom === 'fit' || zoom === 'fill' }"
      aria-label="远程桌面画布"
      @pointerdown="operateDesktop"
      @blur="release"
      @focus="syncLocalClipboard"
      @keydown.capture="clipboardShortcut"
      @paste.capture="pasteText"
    ><div ref="surface" class="remote-surface" /></div>
    <section v-if="clipboardOpen" :id="clipboardId + '-panel'" class="clipboard" role="dialog" :aria-labelledby="clipboardId + '-title'" aria-modal="false" @keydown.esc.stop.prevent="closeClipboard">
      <div class="clipboard-heading">
        <h2 :id="clipboardId + '-title'">文本剪贴板</h2>
        <button type="button" aria-label="关闭剪贴板" @click="closeClipboard"><X :size="16" /></button>
      </div>
      <div class="clipboard-body">
        <textarea
          :id="clipboardId"
          aria-label="剪贴板文本"
          v-model="text"
          maxlength="262144"
          placeholder="粘贴文本后发送到远端"
          @focus="release"
        />
        <div class="clipboard-actions">
          <button class="primary" :disabled="status !== '已连接'" @click="sendText">
            <Send :size="16" />发送到远端
          </button>
          <button @click="copyText">复制到本机</button>
        </div>
        <small role="status">{{ clipboardInfo }}</small>
      </div>
    </section>
  </section>
</template>
