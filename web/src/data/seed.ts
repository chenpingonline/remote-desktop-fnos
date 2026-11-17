import type { Settings } from '../types'

export const defaultSettings: Settings = {
  connectionSort: 'recent',
  language: 'zh-CN',
  timezone: 'Asia/Shanghai',
  startPage: 'connections',
  autoReconnect: true,
  rememberWindow: true,
  confirmDisconnect: true,
  autoLock: '15',
  clipboardClear: true,
  auditEvents: true,
  theme: 'light',
  accent: 'sky',
  palette: 'native',
  density: 'compact',
  reduceMotion: false,
  notifications: {
    connect: true,
    failure: true,
    weekly: false,
    desktop: false,
  },
  quietHours: '22:00 - 08:00',
}
