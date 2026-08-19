// Thin wrapper over the generated Wails bindings (wailsjs/go/main/App).
// The frontend never touches SQL or business rules — this is the parity
// contract from .agent/plan/02-architecture.md.
import * as App from '../../wailsjs/go/main/App'

export type Status = 'pending' | 'wip' | 'done' | 'waiting'

// JSON field names are a stable agent contract (internal/core/task.go).
export interface Task {
  id: string
  title: string
  description: string
  status: Status
  progress: number
  created_at: string
  updated_at: string
  completed_at: string | null
}

export interface ListFilterInput {
  status?: string // '' = all (shows done too)
  search?: string
  sort?: 'created' | 'updated' | 'status' | 'progress' | 'title'
  ascending?: boolean
}

// core.ListFilter has no json tags → Go field names verbatim. Keep in sync
// with internal/core/task.go.
const toGoFilter = (f: ListFilterInput) => ({
  Status: f.status ?? '',
  Search: f.search ?? '',
  Limit: 0,
  Sort: f.sort ?? 'updated',
  Ascending: !!f.ascending,
  IncludeDone: !f.status // "all" view includes done; explicit status chips don't need it
})

export const api = {
  list(f: ListFilterInput): Promise<Task[]> {
    // Defensive: a nil Go slice arrives as null over the Wails bridge; the
    // app normalizes it, but never trust an external boundary.
    return App.ListTasks(toGoFilter(f)).then((t) => t ?? []) as Promise<Task[]>
  },
  get(ref: string) {
    return App.GetTask(ref)
  },
  create(input: { title: string; description?: string; status?: Status; progress?: number }) {
    return App.CreateTask({
      Title: input.title,
      Description: input.description ?? '',
      Status: (input.status ?? 'pending') as unknown as string,
      Progress: input.progress ?? 0
    })
  },
  // Nil fields are left unchanged on the Go side (*string/*int).
  update(id: string, patch: { title?: string; description?: string; progress?: number }) {
    return App.UpdateTask(id, {
      Title: patch.title ?? null,
      Desc: patch.description ?? null,
      Progress: patch.progress ?? null
    })
  },
  setStatus(id: string, status: Status) {
    return App.SetStatus(id, status as unknown as string)
  },
  remove(id: string) {
    return App.DeleteTask(id)
  },
  dbPath(): Promise<string> {
    return App.DBPath()
  },
  // Real exit (Ctrl+Q). Window close hides to tray instead.
  quit(): void {
    App.Quit()
  }
}

export function errMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  return String(e)
}
