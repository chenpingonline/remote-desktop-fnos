export class ApiError extends Error {
  status: number;
  code: string;
  result: any;
  constructor(message: string, status: number, code: string, result: any) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.result = result;
  }
}
export const base = "/app/remote-desktop/api/v1";
let csrf = "";
let identity = "";
let renewing: Promise<void> | null = null;
export function setCSRF(value: string, uid = "") {
  csrf = value;
  identity = uid;
}
function originHeaders() {
  return {
    "X-Remote-Desktop-Origin":
      typeof window !== "undefined" ? window.location.origin : "",
  };
}
async function renewCSRF(expectedIdentity: string) {
  if (!renewing) {
    renewing = (async () => {
      const response = await fetch(base + "/me", {
        cache: "no-store",
        headers: originHeaders(),
      });
      const me = await response.json();
      if (!response.ok)
        throw new Error(me.error?.message || "登录已失效，请重新打开应用");
      if (
        !expectedIdentity ||
        identity !== expectedIdentity ||
        me.uid !== expectedIdentity
      )
        throw new Error("登录用户已改变，请重新打开应用后保存");
      csrf = me.csrfToken;
    })().finally(() => {
      renewing = null;
    });
  }
  await renewing;
}
export async function api<T = any>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
  const requestIdentity = identity;
  for (let attempt = 0; attempt < 2; attempt++) {
    if (identity !== requestIdentity)
      throw new Error("登录用户已改变，请重新打开应用后保存");
    const requestToken = csrf;
    const response = await fetch(base + path, {
      method,
      headers: {
        ...originHeaders(),
        "Content-Type": "application/json",
        "X-CSRF-Token": requestToken,
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    const result = await response.json();
    // This explicit server error is emitted before mutation. Never replay a
    // network failure, unknown result, origin rejection, or another identity.
    if (
      response.status === 403 &&
      result.error?.code === "CSRF_EXPIRED" &&
      attempt === 0
    ) {
      if (csrf === requestToken) await renewCSRF(requestIdentity);
      continue;
    }
    if (!response.ok) {
      if (response.status === 401 && typeof window !== "undefined")
        window.dispatchEvent(new Event("remote-desktop-auth-expired"));
      throw new ApiError(
        result.error?.message || "请求失败",
        response.status,
        result.error?.code || "",
        result,
      );
    }
    return result;
  }
  throw new Error("安全令牌续取失败，请重新打开应用");
}
export const states: Record<string, string> = {
  CREATED: "等待连接",
  CONNECTING: "连接中",
  ACTIVE: "已连接",
  CLOSING: "正在断开",
  CLOSED: "已断开",
  FAILED: "连接失败",
  INTERRUPTED: "已中断",
};
export const reasons: Record<string, string> = {
  ENGINE_UNAVAILABLE: "桌面引擎不可用，请检查服务状态",
  CONNECT_TIMEOUT: "建立桌面超时，请检查网络、认证和远程服务",
  PROTOCOL_ERROR: "协议协商失败，请检查引擎版本、认证和证书",
  REMOTE_ACCESS_DENIED:
    "Windows 拒绝此账户远程登录，请检查远程桌面权限、账户锁定及登录策略",
  REMOTE_CREDENTIAL_EXPIRED:
    "Windows 账户或密码已过期，需要在 Windows 中处理后再连接",
  REMOTE_SECURITY_FAILED: "RDP 安全协商失败，请检查目标的 NLA 和安全策略",
  REMOTE_AUTH_OR_SECURITY_FAILED:
    "远端认证或安全协商失败，尚不能确定是密码问题",
  REMOTE_TLS_FAILED: "RDP TLS 建连失败，可能是证书不受信任或 TLS 不兼容",
  REMOTE_DNS_FAILED: "NAS 无法解析目标主机名，请检查目标地址或 DNS",
  REMOTE_AUTH_FAILED: "远端认证失败，请检查账号与密码",
  REMOTE_CONNECTION_FAILED: "远端连接失败，请检查认证、证书和目标服务",
  UPSTREAM_DISCONNECTED: "连接已断开",
  TARGET_UNREACHABLE: "目标不可达",
  APP_RESTART: "应用重启，旧会话已中断",
  APP_STOP: "应用已停止",
  TICKET_EXPIRED: "连接等待超时，请重新连接",
  PERMISSION_REVOKED: "连接配置或权限已变更",
  USER_CLOSED: "已主动断开",
  CANCELLED: "已取消连接",
  IO_ERROR: "连接传输失败",
  IDLE_TIMEOUT: "超过 30 分钟无操作，已断开",
  MAX_DURATION: "已达到最长 8 小时会话时长",
};

// 表格/列表里使用的短标签，完整说明仍保留在 reasons 中
export const reasonLabels: Record<string, string> = {
  ENGINE_UNAVAILABLE: "引擎不可用",
  CONNECT_TIMEOUT: "连接超时",
  PROTOCOL_ERROR: "协议错误",
  REMOTE_ACCESS_DENIED: "登录被拒",
  REMOTE_CREDENTIAL_EXPIRED: "凭据过期",
  REMOTE_SECURITY_FAILED: "安全协商失败",
  REMOTE_AUTH_OR_SECURITY_FAILED: "认证/安全失败",
  REMOTE_TLS_FAILED: "TLS 失败",
  REMOTE_DNS_FAILED: "DNS 失败",
  REMOTE_AUTH_FAILED: "认证失败",
  REMOTE_CONNECTION_FAILED: "连接失败",
  UPSTREAM_DISCONNECTED: "上游断开",
  TARGET_UNREACHABLE: "目标不可达",
  APP_RESTART: "应用重启",
  APP_STOP: "应用停止",
  TICKET_EXPIRED: "等待超时",
  PERMISSION_REVOKED: "权限变更",
  USER_CLOSED: "用户断开",
  CANCELLED: "已取消",
  IO_ERROR: "传输失败",
  IDLE_TIMEOUT: "空闲超时",
  MAX_DURATION: "到达时限",
};

export function sessionQualityLabel(quality: string): string {
  return reasonLabels[quality] || states[quality] || quality;
}

export function sessionQualityDetail(quality: string): string {
  return reasons[quality] || states[quality] || quality;
}
