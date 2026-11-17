// Only sessions owned by this page can be returned to. Backend session records
// from another tab/browser cannot safely be reused as live desktop clients.
export interface LocalSession {
  id: string;
  connectionId: string;
  status: string;
}
export function sessionNeedsReconnect(status: string) {
  return ["已断开", "连接失败", "已中断", "正在断开"].includes(status);
}
export function localSessionFor<T extends LocalSession>(sessions: readonly T[], connectionId: string): T | undefined {
  const matching = sessions.filter(s => s.connectionId === connectionId);
  // Prefer a usable session if an older version left duplicate failed tabs.
  return matching.find(s => !sessionNeedsReconnect(s.status)) || matching.at(-1);
}
export function connectionLaunchLabel(sessions: readonly LocalSession[], connectionId: string, status: string) {
  const session = localSessionFor(sessions, connectionId);
  if (session) return sessionNeedsReconnect(session.status) ? "重新连接" : "返回会话";
  return "连接";
}
