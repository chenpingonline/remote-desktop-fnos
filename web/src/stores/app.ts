import { computed, reactive, ref, watch } from "vue";
import { api, setCSRF, ApiError } from "../api";
import { defaultSettings } from "../data/seed";
import type {
  AccentName,
  Connection,
  ConnectionInput,
  ConnectionProbe,
  Credential,
  CredentialInput,
  Group,
  GroupInput,
  Settings,
  Session,
  SessionStats,
  WolEntry,
  WolInput,
  WorkspaceState,
} from "../types";
import { setDisplayTimeZone } from "../utils/format";

export const backend = reactive({
  ready: false,
  busy: false,
  devLoginAllowed: false,
  error: "",
  user: null as null | { uid: string; name: string; admin: boolean },
  capabilities: {} as Record<string, any>,
  audit: [] as any[],
  limits: null as any,
});
export const state = reactive<WorkspaceState>({
  connections: [],
  groups: [],
  credentials: [],
  sessions: [],
  wol: [],
  activity: Array(7).fill(0),
  settings: {
    ...defaultSettings,
    notifications: { ...defaultSettings.notifications },
  },
});
const rawConnections = new Map<string, any>();
const metadata = reactive<{
  connections: Record<string, any>;
  groups: Record<string, any>;
}>({ connections: {}, groups: {} });
function preferenceKey() {
  return "remote-desktop.relay.preferences." + backend.user?.uid;
}
function loadPreferences() {
  let saved: any = {};
  try {
    saved = JSON.parse(localStorage.getItem(preferenceKey()) || "{}");
  } catch {}
  Object.assign(state.settings, defaultSettings, saved.settings || {});
  if (!["connections", "sessions"].includes(state.settings.startPage))
    state.settings.startPage = "connections";
  state.settings.notifications = {
    ...defaultSettings.notifications,
    ...(saved.settings?.notifications || {}),
  };
  metadata.connections = saved.connections || {};
  metadata.groups = saved.groups || {};
}
function persistPreferences() {
  if (!backend.user) return;
  try {
    localStorage.setItem(
      preferenceKey(),
      JSON.stringify({
        settings: state.settings,
        connections: metadata.connections,
        groups: metadata.groups,
      }),
    );
  } catch {}
}
watch(() => [state.settings, metadata], persistPreferences, { deep: true });
const accentMap: Record<AccentName, string[]> = {
  lime: ["#d7fb62", "#c2eb43", "#182006", "#4c7b35"],
  sky: ["#a5d3ff", "#7bbcff", "#0a263e", "#2f6fa8"],
  tangerine: ["#ffc38e", "#ffa75f", "#2f1705", "#b26a2a"],
  lilac: ["#d6c2ff", "#bfa1ff", "#201136", "#7a5cc0"],
};
const systemTheme = window.matchMedia("(prefers-color-scheme: dark)");
const systemDark = ref(systemTheme.matches);
export const isDark = computed(
  () => state.settings.theme === "dark" ||
    (state.settings.theme === "system" && systemDark.value),
);
export function applyPreferences() {
  const root = document.documentElement;
  const dark = isDark.value;
  root.dataset.theme = dark ? "dark" : "light";
  root.classList.toggle("dark", dark);
  root.dataset.palette = state.settings.palette;
  root.dataset.density = state.settings.density;
  root.dataset.reduceMotion = String(state.settings.reduceMotion);
  const colors = accentMap[state.settings.accent] || accentMap[defaultSettings.accent];
  ["--accent", "--accent-strong", "--accent-ink", "--accent-deep"].forEach(
    (key, i) => root.style.setProperty(key, colors[i]),
  );
  setDisplayTimeZone(state.settings.timezone);
}
watch(() => state.settings, applyPreferences, { deep: true });
applyPreferences();
systemTheme.addEventListener("change", (event) => {
  systemDark.value = event.matches;
  applyPreferences();
});
export async function allPages(path: string): Promise<any[]> {
  const items: any[] = [];
  for (let offset = 0; ; offset += 100) {
    const page = await api(path + "?limit=100&offset=" + offset);
    items.push(...page.items);
    if (page.items.length < 100) return items;
  }
}
export function rawConnection(id: string) {
  const c = rawConnections.get(id);
  if (!c) throw new Error("连接不存在，请刷新后重试");
  return c;
}
const probeResults = new Map<
  string,
  { target: string; result: ConnectionProbe }
>();
export const probingConnections = reactive(new Set<string>());
function probeTarget(c: any) {
  return JSON.stringify([c.host, c.port, c.protocol.toLowerCase()]);
}
function latestProbe(c: any) {
  const saved = probeResults.get(c.id);
  if (saved && saved.target !== probeTarget(c)) {
    probeResults.delete(c.id);
    return undefined;
  }
  return saved?.result;
}
export function connectionStatusLabel(status: string) {
  return status === "online"
    ? "端口可达"
    : status === "offline"
      ? "端口不可达"
      : status === "busy"
        ? "会话中"
        : "未检测";
}
export function latencyLabel(c: Connection) {
  if (c.latency !== null)
    return `${c.latency < 1 ? c.latency.toFixed(2) : c.latency.toFixed(1)} ms`;
  return c.probe?.reachable === false ? "检测失败" : "未测延迟";
}
export function probeDescription(c: Connection) {
  return c.probe
    ? `${new Date(c.probe.checkedAt).toLocaleString()} · ${c.probe.message}。延迟为 NAS 到目标端口的 TCP 建连耗时（含 DNS），不代表桌面操作延迟。`
    : "尚未检查 NAS 到目标端口的 TCP 连通性";
}
export async function probeConnection(
  id: string,
): Promise<ConnectionProbe | undefined> {
  if (probingConnections.has(id)) return;
  const connection = rawConnection(id);
  const target = probeTarget(connection),
    uid = backend.user?.uid;
  probingConnections.add(id);
  try {
    let result: ConnectionProbe;
    try {
      result = await api<ConnectionProbe>(
        "/connections/" + id + "/probe",
        "POST",
        {},
      );
    } catch (e) {
      if (
        !(e instanceof ApiError) ||
        e.status !== 422 ||
        e.code !== "TARGET_UNREACHABLE" ||
        !e.result.probe
      )
        throw e;
      result = e.result.probe;
    }
    const current = rawConnections.get(id);
    if (
      backend.user?.uid !== uid ||
      !current ||
      probeTarget(current) !== target
    )
      return;
    probeResults.set(id, { target, result });
    const card = connectionById(id);
    if (card) {
      card.probe = result;
      card.latency = result.latencyMs;
      if (
        !state.sessions.some(
          (s) => s.connectionId === id && s.status === "active",
        )
      )
        card.status = result.reachable ? "online" : "offline";
    }
    return result;
  } finally {
    probingConnections.delete(id);
  }
}
function mapConnection(c: any): Connection {
  const probe = latestProbe(c);
  return {
    ...c,
    ...(metadata.connections[c.id] || {}),
    id: c.id,
    version: c.version,
    name: c.name,
    host: c.host,
    port: c.port,
    protocol: c.protocol.toUpperCase(),
    groupId: c.groupId || null,
    credentialId: c.credentialId || null,
    os: c.protocol === "rdp" ? "Windows" : "系统未识别",
    osVersion: "",
    status: probe ? (probe.reachable ? "online" : "offline") : "unknown",
    latency: probe?.latencyMs ?? null,
    probe,
    lastConnected: null,
    favorite: !!metadata.connections[c.id]?.favorite,
    tags: metadata.connections[c.id]?.tags || [],
    notes: metadata.connections[c.id]?.notes || "",
    createdAt: metadata.connections[c.id]?.createdAt || "",
    settings: c.settings,
    wol: c.wol,
  };
}
function mapSession(s: any): Session {
  const c = state.connections.find((c) => c.id === s.connectionId);
  const active = ["CREATED", "CONNECTING", "ACTIVE", "CLOSING"].includes(
    s.state,
  );
  return {
    id: s.id,
    connectionId: s.connectionId,
    connectionName: c?.name || "已删除的连接",
    host: c?.host || "",
    protocol: c?.protocol || "",
    startedAt: s.createdAt,
    duration: Math.max(
      0,
      Math.round(
        ((s.endedAt ? Date.parse(s.endedAt) : Date.now()) -
          Date.parse(s.createdAt)) /
          1000,
      ),
    ),
    status: active
      ? "active"
      : s.state === "CLOSED" && s.endReason === "USER_CLOSED"
        ? "completed"
        : "interrupted",
    quality: s.endReason || s.state,
    user: backend.user?.name || "",
  };
}
let refreshGeneration = 0;
export function invalidateIdentity() {
  refreshGeneration++;
  backend.ready = false;
  backend.user = null;
  backend.limits = null;
  backend.capabilities = {};
  backend.audit = [];
  setCSRF("", "");
  state.connections = [];
  state.groups = [];
  state.credentials = [];
  state.sessions = [];
  state.wol = [];
  state.activity = Array(7).fill(0);
  rawConnections.clear();
  probeResults.clear();
  metadata.connections = {};
  metadata.groups = {};
}
window.addEventListener("remote-desktop-auth-expired", invalidateIdentity);
let refreshQueue: Promise<void> = Promise.resolve();
export function refreshWorkspace(): Promise<void> {
  const next = refreshQueue.catch(() => {}).then(loadWorkspace);
  refreshQueue = next;
  return next;
}
async function loadWorkspace() {
  const generation = ++refreshGeneration;
  const uid = backend.user?.uid;
  const me = await api("/me");
  if (me.uid !== uid) {
    invalidateIdentity();
    throw new Error("登录用户已改变，请重新打开应用");
  }
  const [connections, groups, credentials, sessions, capabilities, audit, wakeHistory] =
    await Promise.all([
      allPages("/connections"),
      allPages("/groups"),
      allPages("/credentials"),
      allPages("/sessions"),
      api("/capabilities"),
      api("/audit-events?limit=50"),
      api<{ items: Record<string, string> }>("/wake-history"),
    ]);
  if (generation !== refreshGeneration || backend.user?.uid !== uid) return;
  rawConnections.clear();
  connections.forEach((c) => rawConnections.set(c.id, c));
  for (const id of probeResults.keys())
    if (!rawConnections.has(id)) probeResults.delete(id);
  state.connections = connections.map(mapConnection);
  state.groups = groups.map((g) => ({
    ...(metadata.groups[g.id] || {}),
    ...g,
    description: metadata.groups[g.id]?.description || "",
    color: metadata.groups[g.id]?.color || "#c9ec58",
    icon: metadata.groups[g.id]?.icon || "folder",
  }));
  state.credentials = credentials.map((c) => ({
    ...c,
    protocol: c.type,
    type: "password",
    username: c.username || "",
    domain: c.domain || "",
    secret: "",
    groupId: null,
    lastUsed: null,
    notes: "",
    createdAt: c.updatedAt,
  }));
  state.sessions = sessions.map(mapSession);
  state.connections.forEach((c) => {
    const records = state.sessions
      .filter((s) => s.connectionId === c.id)
      .sort((a, b) => Date.parse(b.startedAt) - Date.parse(a.startedAt));
    c.lastConnected = records[0]?.startedAt || null;
    if (records.some((s) => s.status === "active")) c.status = "busy";
  });
  // Preserve a send completed while this refresh's history request was in flight.
  const lastWakeTimes = new Map(state.wol.map(entry => [entry.connectionId, entry.lastWokenAt]));
  for (const [id, sentAt] of Object.entries(wakeHistory.items)) {
    const previous = lastWakeTimes.get(id);
    if (!previous || Date.parse(sentAt) > Date.parse(previous)) lastWakeTimes.set(id, sentAt);
  }
  state.wol = state.connections
    .filter((c) => c.wol?.enabled)
    .map((c) => ({
      id: c.id,
      connectionId: c.id,
      name: c.name,
      mac: c.wol!.mac,
      broadcast: c.wol!.broadcast,
      port: 9,
      notes: "",
      createdAt: "",
      lastWokenAt: lastWakeTimes.get(c.id) || null,
    }));
  state.activity = Array.from({ length: 7 }, (_, i) => {
    const day = new Date();
    day.setDate(day.getDate() - 6 + i);
    return state.sessions.filter(
      (s) => new Date(s.startedAt).toDateString() === day.toDateString(),
    ).length;
  });
  backend.capabilities = capabilities;
  backend.audit = audit.items;
}
export async function initialize() {
  if (backend.busy) return;
  backend.busy = true;
  backend.devLoginAllowed = false;
  backend.error = "";
  try {
    const me = await api("/me");
    setCSRF(me.csrfToken, me.uid);
    backend.user = me;
    loadPreferences();
    await refreshWorkspace();
    if (me.admin) backend.limits = await api("/settings");
    backend.ready = true;
  } catch (e) {
    backend.devLoginAllowed = e instanceof ApiError && e.status === 401 &&
      e.result?.error?.devLoginAllowed === true;
    backend.error = (e as Error).message;
    backend.ready = false;
  } finally {
    backend.busy = false;
  }
}
export async function login(token: string) {
  await api("/dev-login", "POST", { token });
  await initialize();
}
export const groupById = (id?: string | null) =>
  state.groups.find((g) => g.id === id) || null;
export const credentialById = (id?: string | null) =>
  state.credentials.find((c) => c.id === id) || null;
export const connectionById = (id?: string | null) =>
  state.connections.find((c) => c.id === id) || null;
export const groupConnectionCount = (id: string) =>
  state.connections.filter((c) => c.groupId === id).length;
export const groupOnlineCount = (id: string) =>
  state.connections.filter((c) => c.groupId === id && c.status === "busy")
    .length;
export const credentialUsage = (id: string) =>
  state.connections.filter((c) => c.credentialId === id).length;
export const wolById = (id?: string | null) =>
  state.wol.find((w) => w.id === id) || null;
export const wolByConnection = (id?: string | null) =>
  state.wol.find((w) => w.connectionId === id) || null;
export const stats = computed<SessionStats>(() => {
  const active = state.connections.filter((c) => c.status === "busy").length;
  const totalSeconds = state.sessions.reduce((n, s) => n + s.duration, 0);
  return {
    online: active,
    busy: active,
    offline: state.connections.length - active,
    available: active,
    total: state.connections.length,
    totalSeconds,
    successRate: state.sessions.length
      ? Math.round(
          (state.sessions.filter((s) => s.status === "completed").length /
            state.sessions.length) *
            100,
        )
      : 0,
    avgSeconds: state.sessions.length
      ? Math.round(totalSeconds / state.sessions.length)
      : 0,
  };
});
export async function saveConnection(p: ConnectionInput): Promise<Connection> {
  if (!["RDP", "VNC"].includes(p.protocol || "RDP"))
    throw new Error("后端仅支持 RDP 和 VNC");
  const old = p.id ? rawConnection(p.id) : null;
  const body = {
    name: p.name.trim(),
    host: p.host.trim(),
    port: Number(p.port),
    protocol: (p.protocol || "RDP").toLowerCase(),
    targetOs: p.protocol === "VNC" ? "auto" : "windows",
    groupId: p.groupId || "",
    credentialId: p.credentialId || "",
    version: p.version ?? old?.version ?? 0,
    settings: {
      ...(old?.settings || {
        security: "nla",
        certificatePolicy: "verify",
        authMode: "password",
      }),
      clipboard:
        p.clipboard === undefined
          ? old?.settings?.clipboard || "disabled"
          : p.clipboard
            ? "text"
            : "disabled",
    },
    wol:
      p.wolEnabled === undefined
        ? old?.wol || { enabled: false, mac: "", broadcast: "" }
        : {
            enabled: p.wolEnabled,
            mac: p.wolMac || "",
            broadcast: p.wolBroadcast || "",
          },
  };
  const saved = await api(
    "/connections" + (p.id ? "/" + p.id : ""),
    p.id ? "PATCH" : "POST",
    body,
  );
  metadata.connections[saved.id] = {
    ...metadata.connections[saved.id],
    favorite: !!p.favorite,
    tags: Array.isArray(p.tags) ? p.tags : [],
    notes: p.notes || "",
    createdAt: p.createdAt || new Date().toISOString(),
  };
  await refreshWorkspace();
  return mapConnection(saved);
}
export async function removeConnection(id: string) {
  await api("/connections/" + id, "DELETE");
  delete metadata.connections[id];
  await refreshWorkspace();
}
export function toggleFavorite(id: string) {
  const c = connectionById(id);
  if (c) {
    c.favorite = !c.favorite;
    metadata.connections[id] = {
      ...metadata.connections[id],
      favorite: c.favorite,
    };
  }
}
export async function saveGroup(p: GroupInput): Promise<Group> {
  const old = groupById(p.id);
  const saved = await api(
    "/groups" + (p.id ? "/" + p.id : ""),
    p.id ? "PATCH" : "POST",
    { name: p.name, version: p.version ?? old?.version ?? 0 },
  );
  metadata.groups[saved.id] = {
    color: p.color,
    icon: p.icon,
    description: p.description,
  };
  await refreshWorkspace();
  return groupById(saved.id)!;
}
export async function removeGroup(id: string) {
  await api("/groups/" + id, "DELETE");
  delete metadata.groups[id];
  await refreshWorkspace();
}
export async function saveCredential(p: CredentialInput): Promise<Credential> {
  const old = credentialById(p.id);
  const saved = await api(
    "/credentials" + (p.id ? "/" + p.id : ""),
    p.id ? "PATCH" : "POST",
    {
      name: p.name,
      type: old?.protocol || p.protocol || "rdp",
      username: p.username || "",
      password: p.secret || "",
      domain: p.domain || "",
      version: p.version ?? old?.version ?? 0,
    },
  );
  await refreshWorkspace();
  return credentialById(saved.id)!;
}
export async function removeCredential(id: string) {
  await api("/credentials/" + id, "DELETE");
  await refreshWorkspace();
}
export async function saveWol(p: WolInput): Promise<WolEntry> {
  if (!p.connectionId)
    throw new Error("请选择关联连接；后端按连接保存网络唤醒配置");
  if (Number(p.port || 9) !== 9) throw new Error("后端使用 UDP 9 端口");
  const c = rawConnection(p.connectionId);
  await api("/connections/" + c.id, "PATCH", {
    ...c,
    wol: { enabled: true, mac: p.mac, broadcast: p.broadcast || "" },
  });
  await refreshWorkspace();
  return wolByConnection(c.id)!;
}
export async function removeWol(id: string) {
  const c = rawConnection(id);
  await api("/connections/" + id, "PATCH", {
    ...c,
    wol: { ...c.wol, enabled: false },
  });
  await refreshWorkspace();
}
export async function removeSession(id: string) {
  const s = state.sessions.find((s) => s.id === id);
  if (s?.status !== "active")
    throw new Error("后端保留已结束的会话记录，暂不支持删除");
  await api("/sessions/" + id, "DELETE");
  await refreshWorkspace();
}
export function clearSessions() {
  throw new Error("后端保留会话记录 90 天，暂不支持手动清空");
}
export function updateSettings(p: Partial<Settings>) {
  Object.assign(state.settings, p);
}
export function exportSnapshot() {
  return {
    app: "remote-desktop",
    version: 1,
    exportedAt: new Date().toISOString(),
    data: {
      ...state,
      credentials: state.credentials.map(({ secret, username, ...c }) => c),
    },
    note: "仅供查看的配置与记录，不包含凭据密文或主密钥，不能用于完整恢复",
  };
}
export function importSnapshot(_: unknown) {
  throw new Error(
    "暂不支持导入浏览器工作区，请使用 NAS 数据与密钥配对备份恢复",
  );
}
export function resetWorkspace() {
  Object.assign(state.settings, defaultSettings);
  applyPreferences();
}
export function storageUsage() {
  try {
    return new Blob([localStorage.getItem(preferenceKey()) || ""]).size;
  } catch {
    return 0;
  }
}
