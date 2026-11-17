export type Protocol = "RDP" | "VNC";
export type ConnectionStatus = "online" | "busy" | "offline" | "unknown";
export type ConnectionOS = "macOS" | "Windows" | "Linux";
export type CredentialType = "password" | "token";
export type SessionStatus = "completed" | "interrupted" | "active";
export type ThemeMode = "light" | "dark" | "system";
export type AccentName = "lime" | "sky" | "tangerine" | "lilac";
export type PaletteName = "native" | "warm" | "cool";
export type Density = "comfortable" | "compact";

export interface Group {
  version?: number;
  id: string;
  name: string;
  description: string;
  color: string;
  icon: string;
}

export interface ConnectionProbe {
  reachable: boolean;
  latencyMs: number | null;
  checkedAt: string;
  message: string;
}

export interface Connection {
  probe?: ConnectionProbe;
  version?: number;
  settings?: {
    icon?: string;
    security: string;
    clipboard: string;
    certificatePolicy: string;
    authMode: string;
  };
  wol?: { enabled: boolean; mac: string; broadcast: string };
  id: string;
  name: string;
  host: string;
  port: number;
  protocol: Protocol;
  groupId: string | null;
  credentialId: string | null;
  os: ConnectionOS | string;
  osVersion: string;
  status: ConnectionStatus;
  latency: number | null;
  lastConnected: string | null;
  favorite: boolean;
  tags: string[];
  notes: string;
  createdAt: string;
}

export interface RemoteConnection {
  id?: string | null;
  name: string;
  host: string;
  protocol: Protocol | string;
  os?: ConnectionOS | string;
  osVersion?: string;
  latency?: number | null;
  status?: ConnectionStatus | string;
  favorite?: boolean;
  tags?: string[];
  groupId?: string | null;
  credentialId?: string | null;
  port?: number;
  notes?: string;
  createdAt?: string;
}

export interface WolEntry {
  id: string;
  name: string;
  mac: string;
  broadcast: string;
  port: number;
  connectionId: string | null;
  notes: string;
  createdAt: string;
  lastWokenAt: string | null;
}

export interface WolInput {
  id?: string | null;
  name: string;
  mac: string;
  broadcast?: string;
  port?: number | string;
  connectionId?: string | null;
  notes?: string;
}

export interface Credential {
  domain?: string;
  version?: number;
  protocol?: string;
  id: string;
  name: string;
  type: CredentialType;
  username: string;
  secret: string;
  groupId: string | null;
  lastUsed: string | null;
  notes: string;
  createdAt: string;
}

export interface Session {
  id: string;
  connectionId: string | null;
  connectionName: string;
  host: string;
  protocol: Protocol | string;
  startedAt: string;
  duration: number;
  status: SessionStatus;
  quality: string;
  user: string;
}

export interface NotificationSettings {
  connect: boolean;
  failure: boolean;
  weekly: boolean;
  desktop: boolean;
}

export interface Settings {
  connectionSort: import("./connection-sort").ConnectionSort;
  language: string;
  timezone: string;
  startPage: string;
  autoReconnect: boolean;
  rememberWindow: boolean;
  confirmDisconnect: boolean;
  autoLock: string;
  clipboardClear: boolean;
  auditEvents: boolean;
  theme: ThemeMode;
  accent: AccentName;
  palette: PaletteName;
  density: Density;
  reduceMotion: boolean;
  notifications: NotificationSettings;
  quietHours: string;
}

export interface WorkspaceState {
  connections: Connection[];
  sessions: Session[];
  credentials: Credential[];
  groups: Group[];
  wol: WolEntry[];
  activity: number[];
  settings: Settings;
}

export interface ConnectionInput {
  icon?: string;
  version?: number;
  clipboard?: boolean;
  wolEnabled?: boolean;
  wolMac?: string;
  wolBroadcast?: string;
  id?: string | null;
  name: string;
  host: string;
  port: number | string;
  protocol?: string;
  groupId?: string | null;
  credentialId?: string | null;
  os?: string;
  osVersion?: string;
  status?: ConnectionStatus | string;
  latency?: number | null;
  lastConnected?: string | null;
  favorite?: boolean;
  tags?: string[] | string;
  notes?: string;
  createdAt?: string;
}

export interface CredentialInput {
  version?: number;
  protocol?: string;
  domain?: string;
  id?: string | null;
  name: string;
  type?: CredentialType | string;
  username?: string;
  secret?: string;
  groupId?: string | null;
  lastUsed?: string | null;
  notes?: string;
  createdAt?: string;
}

export interface GroupInput {
  version?: number;
  id?: string | null;
  name: string;
  description?: string;
  color?: string;
  icon?: string;
}

export interface SessionStats {
  online: number;
  busy: number;
  offline: number;
  available: number;
  total: number;
  totalSeconds: number;
  successRate: number;
  avgSeconds: number;
}

export interface ExportSnapshot {
  app: string;
  version: number;
  exportedAt: string;
  data: WorkspaceState;
}
