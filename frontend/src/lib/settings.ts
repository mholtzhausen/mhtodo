// GUI settings types — mirror internal/settings/settings.go (Wails bindings).
import { settings as goSettings } from '../../wailsjs/go/models'

export type TaskStatusId = 'pending' | 'wip' | 'waiting' | 'review' | 'pr' | 'done'

export interface IntegrationConfig {
  enabled: boolean
  binary: string
  env_start: string
}

export interface NotificationsConfig {
  tray_label_statuses: TaskStatusId[]
  tray_menu_statuses: TaskStatusId[]
  max_items_per_status: number
  notify_send_wip: boolean
  notify_send_waiting: boolean
  notify_send_review: boolean
  notify_send_done: boolean
}

export interface GUISettings {
  default_cwd: string
  default_human_only: boolean
  default_include_in_report: boolean
  archive_done_subtasks: boolean
  start_hidden: boolean
  notifications: NotificationsConfig
  zed: IntegrationConfig
}

export const STATUS_OPTIONS: { id: TaskStatusId; label: string }[] = [
  { id: 'pending', label: 'Pending' },
  { id: 'wip', label: 'WIP' },
  { id: 'waiting', label: 'Waiting' },
  { id: 'review', label: 'Review' },
  { id: 'pr', label: 'Pull Request' },
  { id: 'done', label: 'Done' }
]

const statusSet = new Set(STATUS_OPTIONS.map((s) => s.id))

export function normalizeStatusList(list: unknown, fallback: TaskStatusId[]): TaskStatusId[] {
  const raw = Array.isArray(list) ? list : []
  const seen = new Set<string>()
  const out: TaskStatusId[] = []
  for (const item of raw) {
    const id = String(item ?? '')
      .trim()
      .toLowerCase() as TaskStatusId
    if (!statusSet.has(id) || seen.has(id)) continue
    seen.add(id)
    out.push(id)
  }
  if (out.length === 0) return [...fallback]
  return STATUS_OPTIONS.map((s) => s.id).filter((id) => out.includes(id))
}

export function toggleStatusInOrder(
  list: TaskStatusId[],
  id: TaskStatusId,
  on: boolean
): TaskStatusId[] {
  if (on) {
    if (list.includes(id)) return list
    return STATUS_OPTIONS.map((s) => s.id).filter((s) => s === id || list.includes(s))
  }
  return list.filter((s) => s !== id)
}

export const defaultNotifications = (): NotificationsConfig => ({
  tray_label_statuses: ['waiting', 'review'],
  tray_menu_statuses: ['waiting', 'review'],
  max_items_per_status: 10,
  notify_send_wip: false,
  notify_send_waiting: false,
  notify_send_review: true,
  notify_send_done: false
})

export const defaultSettings = (): GUISettings => ({
  default_cwd: '',
  default_human_only: false,
  default_include_in_report: true,
  archive_done_subtasks: false,
  start_hidden: false,
  notifications: defaultNotifications(),
  zed: { enabled: false, binary: 'zed', env_start: '' }
})

function fromGoNotifications(n: goSettings.NotificationsConfig | undefined): NotificationsConfig {
  const d = defaultNotifications()
  if (!n) return d
  const max = Number(n.max_items_per_status)
  return {
    tray_label_statuses: normalizeStatusList(n.tray_label_statuses, d.tray_label_statuses),
    tray_menu_statuses: normalizeStatusList(n.tray_menu_statuses, d.tray_menu_statuses),
    max_items_per_status: Number.isFinite(max) && max > 0 ? Math.min(20, Math.floor(max)) : 10,
    notify_send_wip: !!n.notify_send_wip,
    notify_send_waiting: !!n.notify_send_waiting,
    notify_send_review: n.notify_send_review !== false,
    notify_send_done: !!n.notify_send_done
  }
}

// Wails codegen uses json struct tags → snake_case field names on the wire.
export function fromGoSettings(s: goSettings.GUISettings): GUISettings {
  return {
    default_cwd: s.default_cwd ?? '',
    default_human_only: !!s.default_human_only,
    default_include_in_report: s.default_include_in_report !== false,
    archive_done_subtasks: !!s.archive_done_subtasks,
    start_hidden: !!s.start_hidden,
    notifications: fromGoNotifications(s.notifications),
    zed: {
      enabled: !!s.zed?.enabled,
      binary: s.zed?.binary ?? 'zed',
      env_start: s.zed?.env_start ?? ''
    }
  }
}

export function toGoSettings(s: GUISettings): goSettings.GUISettings {
  const n = s.notifications ?? defaultNotifications()
  return new goSettings.GUISettings({
    default_cwd: s.default_cwd,
    default_human_only: s.default_human_only,
    default_include_in_report: s.default_include_in_report,
    archive_done_subtasks: s.archive_done_subtasks,
    start_hidden: s.start_hidden,
    notifications: {
      tray_label_statuses: n.tray_label_statuses,
      tray_menu_statuses: n.tray_menu_statuses,
      max_items_per_status: n.max_items_per_status,
      notify_send_wip: n.notify_send_wip,
      notify_send_waiting: n.notify_send_waiting,
      notify_send_review: n.notify_send_review,
      notify_send_done: n.notify_send_done
    },
    zed: {
      enabled: s.zed.enabled,
      binary: s.zed.binary,
      env_start: s.zed.env_start
    }
  })
}
