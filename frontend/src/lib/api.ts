// Thin wrapper over the generated Wails bindings (wailsjs/go/main/App).
// The frontend never touches SQL or business rules — this is the parity
// contract from .agent/plan/02-architecture.md.
import * as App from '../../wailsjs/go/main/App'
import type { core } from '../../wailsjs/go/models'
import { defaultSettings, fromGoSettings, toGoSettings, type GUISettings } from './settings'
import { toGoTemplateInput, type TaskTemplate, type TemplateValues } from './templates'
import { toGoThemeInput, type Theme } from './themes'

export type Status = 'pending' | 'wip' | 'waiting' | 'review' | 'done'

// JSON field names are a stable agent contract (internal/core/task.go).
export interface Task {
  id: string
  title: string
  description: string
  feedback: string
  status: Status
  progress: number
  created_at: string
  updated_at: string
  completed_at: string | null
  archived_at: string | null
  parent_id: string | null
  board_rank: number | null
  cwd: string
  human_only: boolean
  include_in_report: boolean
  slack_thread: string
  todo_session: string
  terminal_pid?: number
}

export interface Activity {
  id: string
  task_id: string
  activity: string
  comment: string
  created_at: string
}

export interface ListFilterInput {
  status?: string
  archived?: boolean
  search?: string
  sort?: 'board' | 'created' | 'updated' | 'status' | 'progress' | 'title'
  ascending?: boolean
  rootsOnly?: boolean
  /** When set, only direct children of this parent (ignores rootsOnly). */
  parentId?: string
  /** When set, overrides the default (include done if no status filter / archived view). */
  includeDone?: boolean
  /** GUI default true; CLI default false — include human-only tasks in results. */
  includeHumanOnly?: boolean
}

export interface ActivityFilterInput {
  taskIds?: string[]
  limit?: number
  includeArchived?: boolean
}

const toGoFilter = (f: ListFilterInput) => ({
  Status: f.status ?? '',
  Search: f.search ?? '',
  Limit: 0,
  Sort: f.sort ?? 'board',
  Ascending: !!f.ascending,
  IncludeDone: f.includeDone ?? (!f.status || !!f.archived),
  Archived: !!f.archived,
  RootsOnly: !!f.rootsOnly,
  ParentID: f.parentId ?? '',
  IncludeHumanOnly: f.includeHumanOnly !== undefined ? f.includeHumanOnly : true
})

const toGoActivityFilter = (f: ActivityFilterInput) => ({
  TaskIDs: f.taskIds ?? [],
  Limit: f.limit ?? 0,
  IncludeArchived: !!f.includeArchived
})

export const api = {
  list(f: ListFilterInput): Promise<Task[]> {
    return App.ListTasks(toGoFilter(f)).then((t) => t ?? []) as Promise<Task[]>
  },
  get(ref: string) {
    return App.GetTask(ref)
  },
  create(input: {
    title: string
    description?: string
    feedback?: string
    status?: Status
    progress?: number
    parentId?: string
    cwd?: string
    humanOnly?: boolean
    includeInReport?: boolean
    slackThread?: string
    todoSession?: string
  }): Promise<Task> {
    return App.CreateTask({
      Title: input.title,
      Description: input.description ?? '',
      Feedback: input.feedback ?? '',
      Status: (input.status ?? 'pending') as unknown as string,
      Progress: input.progress ?? 0,
      ParentID: input.parentId ?? '',
      Cwd: input.cwd ?? '',
      HumanOnly: !!input.humanOnly,
      IncludeInReport: input.includeInReport,
      SlackThread: input.slackThread ?? '',
      TodoSession: input.todoSession ?? ''
    }) as Promise<Task>
  },
  update(id: string, patch: {
    title?: string
    description?: string
    feedback?: string
    progress?: number
    cwd?: string
    humanOnly?: boolean
    includeInReport?: boolean
    slackThread?: string
    todoSession?: string
  }) {
    return App.UpdateTask(id, {
      Title: patch.title ?? null,
      Desc: patch.description ?? null,
      Feedback: patch.feedback ?? null,
      Progress: patch.progress ?? null,
      Cwd: patch.cwd ?? null,
      HumanOnly: patch.humanOnly ?? null,
      IncludeInReport: patch.includeInReport ?? null,
      SlackThread: patch.slackThread ?? null,
      TodoSession: patch.todoSession ?? null
    })
  },
  setStatus(id: string, status: Status) {
    return App.SetStatus(id, status as unknown as string)
  },
  reorderTask(id: string, beforeId?: string | null) {
    return App.ReorderBoardTask(id, beforeId ?? '') as Promise<Task>
  },
  archiveDone(): Promise<Task[]> {
    return App.ArchiveDone().then((t) => t ?? []) as Promise<Task[]>
  },
  archive(id: string) {
    return App.Archive(id) as Promise<Task>
  },
  unarchive(id: string) {
    return App.Unarchive(id)
  },
  remove(id: string) {
    return App.DeleteTask(id)
  },
  countChildren(id: string): Promise<number> {
    return App.CountChildren(id)
  },
  addActivity(taskId: string, input: { activity?: string; comment?: string }): Promise<Activity> {
    return App.AddActivity(taskId, {
      Activity: input.activity ?? '',
      Comment: input.comment ?? ''
    }) as Promise<Activity>
  },
  listActivity(f: ActivityFilterInput = {}): Promise<Activity[]> {
    return App.ListActivity(toGoActivityFilter(f)).then((a) => a ?? []) as Promise<Activity[]>
  },
  deleteActivity(id: string): Promise<Activity> {
    return App.DeleteActivity(id) as Promise<Activity>
  },
  dbPath(): Promise<string> {
    return App.DBPath()
  },
  quit(): void {
    App.Quit()
  },
  hideWindow(): Promise<void> {
    return App.HideWindow()
  },
  getAlwaysOnTop(): Promise<boolean> {
    return App.GetAlwaysOnTop()
  },
  setAlwaysOnTop(on: boolean): Promise<void> {
    return App.SetAlwaysOnTop(on)
  },
  async pickDirectory(start = ''): Promise<string> {
    const trimmed = start.trim()
    try {
      return await App.PickDirectory(trimmed)
    } catch (e) {
      if (trimmed) return App.PickDirectory('')
      throw e
    }
  },
  getSettings(): Promise<GUISettings> {
    return App.GetGUISettings().then((s) => fromGoSettings(s))
  },
  setSettings(s: GUISettings): Promise<void> {
    return App.SetGUISettings(toGoSettings(s))
  },
  checkBinary(path: string): Promise<boolean> {
    return App.CheckBinary(path)
  },
  ensureHerdrReady(): Promise<{ ready: boolean; error?: string }> {
    return App.EnsureHerdrReady()
  },
  ensureHerdrWorkspace(taskId: string): Promise<{ ready: boolean; error?: string }> {
    return App.EnsureHerdrWorkspaceForTask(taskId)
  },
  openHerdrTicket(taskId: string): Promise<void> {
    return App.OpenHerdrTicket(taskId)
  },
  openZedTicket(taskId: string): Promise<void> {
    return App.OpenZedTicket(taskId)
  },
  zedTicketCommand(taskId: string): Promise<string> {
    return App.ZedTicketCommand(taskId)
  },
  slackReport(): Promise<string> {
    return App.SlackReport()
  },
  taskMarkdownReport(id: string): Promise<string> {
    return App.TaskMarkdownReport(id)
  },

  // --- task templates (v0.5) ---
  listTemplates(): Promise<TaskTemplate[]> {
    return App.ListTemplates().then((t) => t ?? []) as Promise<TaskTemplate[]>
  },
  getTemplate(ref: string): Promise<TaskTemplate> {
    return App.GetTemplate(ref) as Promise<TaskTemplate>
  },
  // Wails types Go pointer fields as `?: T` (optional/undefined), but null is
  // what actually marshals back to a nil pointer, i.e. "field not set". The
  // cast reconciles the generated shape with the value the backend needs.
  createTemplate(name: string, values: TemplateValues): Promise<TaskTemplate> {
    const input = toGoTemplateInput(name, values) as unknown as core.TemplateInput
    return App.CreateTemplate(input) as Promise<TaskTemplate>
  },
  /** Full replace: values left null are cleared on the stored template. */
  updateTemplate(id: string, name: string, values: TemplateValues): Promise<TaskTemplate> {
    const input = toGoTemplateInput(name, values) as unknown as core.TemplateInput
    return App.UpdateTemplate(id, input) as Promise<TaskTemplate>
  },
  deleteTemplate(id: string): Promise<TaskTemplate> {
    return App.DeleteTemplate(id) as Promise<TaskTemplate>
  },

  // --- themes (v0.6) ---
  listThemes(): Promise<Theme[]> {
    return App.ListThemes().then((t) => t ?? []) as Promise<Theme[]>
  },
  getTheme(ref: string): Promise<Theme> {
    return App.GetTheme(ref) as Promise<Theme>
  },
  getActiveTheme(): Promise<Theme> {
    return App.GetActiveTheme() as Promise<Theme>
  },
  createTheme(name: string, tokens: Record<string, string>): Promise<Theme> {
    const input = toGoThemeInput(name, tokens) as unknown as core.ThemeInput
    return App.CreateTheme(input) as Promise<Theme>
  },
  updateTheme(id: string, name: string, tokens: Record<string, string>): Promise<Theme> {
    const input = toGoThemeInput(name, tokens) as unknown as core.ThemeInput
    return App.UpdateTheme(id, input) as Promise<Theme>
  },
  deleteTheme(id: string): Promise<Theme> {
    return App.DeleteTheme(id) as Promise<Theme>
  },
  activateTheme(ref: string): Promise<Theme> {
    return App.ActivateTheme(ref) as Promise<Theme>
  },
  duplicateTheme(ref: string, name = ''): Promise<Theme> {
    return App.DuplicateTheme(ref, name) as Promise<Theme>
  },
  resetTheme(ref: string): Promise<Theme> {
    return App.ResetTheme(ref) as Promise<Theme>
  },

  // --- install / update (header Install control) ---
  getInstallStatus(force = false): Promise<InstallStatus> {
    return App.GetInstallStatus(force).then((raw: any) => ({
      show: !!(raw?.show ?? raw?.Show),
      current_version: String(raw?.current_version ?? raw?.CurrentVersion ?? ''),
      latest_version: String(raw?.latest_version ?? raw?.LatestVersion ?? ''),
      up_to_date: !!(raw?.up_to_date ?? raw?.UpToDate),
      has_service: !!(raw?.has_service ?? raw?.HasService),
      ephemeral: !!(raw?.ephemeral ?? raw?.Ephemeral),
      install_path: String(raw?.install_path ?? raw?.InstallPath ?? ''),
      prefix: String(raw?.prefix ?? raw?.Prefix ?? ''),
      message: String(raw?.message ?? raw?.Message ?? ''),
      cached_at: String(raw?.cached_at ?? raw?.CachedAt ?? ''),
      fresh: !!(raw?.fresh ?? raw?.Fresh)
    }))
  },
  runInstallActions(opts: {
    updateApp: boolean
    installService: boolean
    integrationZsh: boolean
    integrationBash: boolean
  }): Promise<InstallActionsResult> {
    return App.RunInstallActions({
      UpdateApp: opts.updateApp,
      InstallService: opts.installService,
      IntegrationZsh: opts.integrationZsh,
      IntegrationBash: opts.integrationBash
    }).then((raw: any) => ({
      message: String(raw?.message ?? raw?.Message ?? ''),
      updated: !!(raw?.updated ?? raw?.Updated),
      service: !!(raw?.service ?? raw?.Service),
      integration: !!(raw?.integration ?? raw?.Integration)
    }))
  }
}

export interface InstallStatus {
  show: boolean
  current_version: string
  latest_version: string
  up_to_date: boolean
  has_service: boolean
  ephemeral: boolean
  install_path: string
  prefix: string
  message: string
  cached_at: string
  fresh: boolean
}

export interface InstallActionsResult {
  message: string
  updated: boolean
  service: boolean
  integration: boolean
}

export type { GUISettings } from './settings'
export type { TaskTemplate, TemplateValues } from './templates'
export type { Theme } from './themes'

export function errMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  return String(e)
}
