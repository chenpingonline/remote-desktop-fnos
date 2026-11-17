const MINUTE = 60 * 1000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

let displayTimeZone: string | undefined

export function setDisplayTimeZone(timeZone: string | undefined): void {
  displayTimeZone = timeZone || undefined
}

export function timeAgo(value: string | null | undefined): string {
  if (!value) return '从未连接'
  const diff = Date.now() - new Date(value).getTime()
  if (diff < MINUTE) return '刚刚'
  if (diff < HOUR) return `${Math.floor(diff / MINUTE)} 分钟前`
  if (diff < DAY) return `${Math.floor(diff / HOUR)} 小时前`
  if (diff < 2 * DAY) return '昨天'
  if (diff < 7 * DAY) return `${Math.floor(diff / DAY)} 天前`
  if (diff < 30 * DAY) return `${Math.floor(diff / (7 * DAY))} 周前`
  return new Intl.DateTimeFormat('zh-CN', { month: 'short', day: 'numeric' }).format(
    new Date(value),
  )
}

export function formatDateTime(value: string | null | undefined, withSeconds = false): string {
  if (!value) return '—'
  const date = new Date(value)
  const parts = new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    ...(withSeconds ? { second: '2-digit' } : {}),
    hourCycle: 'h23',
    ...(displayTimeZone ? { timeZone: displayTimeZone } : {}),
  }).formatToParts(date)
  const read = (type: string) => parts.find((part) => part.type === type)?.value || ''
  return `${read('month')}月${read('day')}日 ${read('hour')}:${read('minute')}${withSeconds ? `:${read('second')}` : ''}`
}

export function formatFullDate(value: string | null | undefined): string {
  if (!value) return '—'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    ...(displayTimeZone ? { timeZone: displayTimeZone } : {}),
  }).format(new Date(value))
}

export function formatDuration(totalSeconds = 0): string {
  const seconds = Math.max(0, Math.round(totalSeconds))
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h} 小时 ${m} 分`
  if (m > 0) return `${m} 分 ${s} 秒`
  return `${s} 秒`
}

export function formatClock(totalSeconds = 0): string {
  const seconds = Math.max(0, Math.round(totalSeconds))
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  return [h, m, s].map((part) => String(part).padStart(2, '0')).join(':')
}

export function formatHours(totalSeconds = 0): string {
  return `${(totalSeconds / 3600).toFixed(1)} 小时`
}

export function greeting(date = new Date()): string {
  const hour = date.getHours()
  if (hour < 6) return '夜深了'
  if (hour < 12) return '上午好'
  if (hour < 14) return '中午好'
  if (hour < 18) return '下午好'
  return '晚上好'
}

export function getInitials(name = ''): string {
  const clean = name.trim()
  if (!clean) return '?'
  if (/^[\u4e00-\u9fa5]/.test(clean)) return clean.slice(0, 2)
  return clean
    .split(/\s+/)
    .map((word) => word[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()
}

export function maskSecret(secret = ''): string {
  if (secret.startsWith('-----BEGIN')) return '••••••••••••••••••••••••'
  return '•'.repeat(Math.min(18, Math.max(8, secret.length)))
}

export interface PasswordStrength {
  score: number
  label: string
}

export function passwordStrength(secret = ''): PasswordStrength {
  let score = 0
  if (secret.length >= 8) score += 1
  if (secret.length >= 14) score += 1
  if (/[a-z]/.test(secret) && /[A-Z]/.test(secret)) score += 1
  if (/[0-9]/.test(secret)) score += 1
  if (/[^A-Za-z0-9]/.test(secret)) score += 1
  if (score <= 2) return { score: 1, label: '较弱' }
  if (score <= 3) return { score: 2, label: '一般' }
  if (score === 4) return { score: 3, label: '良好' }
  return { score: 4, label: '很强' }
}

export function relativeDayLabel(value: string): string {
  const target = new Date(value)
  const today = new Date()
  const start = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  const targetStart = new Date(target.getFullYear(), target.getMonth(), target.getDate())
  const diff = Math.floor((start.getTime() - targetStart.getTime()) / DAY)
  if (diff === 0) return '今天'
  if (diff === 1) return '昨天'
  if (diff < 7) return `${diff} 天前`
  return formatDateTime(value).slice(0, 5)
}

export function uid(prefix = 'id'): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return `${prefix}-${crypto.randomUUID().slice(0, 8)}`
  }
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}
