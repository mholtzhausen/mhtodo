// Thin wrapper over the generated Wails bindings (wailsjs/go/main/App).
// The frontend never touches SQL or business rules — this is the parity
// contract from .agent/plan/02-architecture.md.
import * as App from '../../wailsjs/go/main/App'
import { defaultSettings, fromGoSettings, toGoSettings, type GUISettings } from './settings'

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
  }) {
    return App.CreateTask({
      Title: input.title,
      Description: input.description ?? '',
      Feedback: input.feedback ?? '',
      Status: (input.status ?? 'pending') as unknown as string,
      Progress: input.progress ?? 0,
      ParentID: input.parentId ?? '',
      Cwd: input.cwd ?? '',
      HumanOnly: !!input.humanOnly
    })
  },
  update(id: string, patch: {
    title?: string
    description?: string
    feedback?: string
    progress?: number
    cwd?: string
    humanOnly?: boolean
  }) {
    return App.UpdateTask(id, {
      Title: patch.title ?? null,
      Desc: patch.description ?? null,
      Feedback: patch.feedback ?? null,
      Progress: patch.progress ?? null,
      Cwd: patch.cwd ?? null,
      HumanOnly: patch.humanOnly ?? null
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
  pickDirectory(start = ''): Promise<string> {
    return App.PickDirectory(start)
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
  ensureHerdrWorkspace(taskId: string): Promise<{ ready: boolean; error?: string }> {
    return App.EnsureHerdrWorkspaceForTask(taskId)
  },
  openHerdrTicket(taskId: string): Promise<void> {
    return App.OpenHerdrTicket(taskId)
  }
}

export type { GUISettings } from './settings'

export function errMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  return String(e)
}
