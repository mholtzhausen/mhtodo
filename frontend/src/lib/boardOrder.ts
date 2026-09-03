import type { Status } from './api'

const BOARD_COLUMNS: Status[] = ['pending', 'wip', 'waiting', 'review', 'done']

function rootRank(t: any): number {
  return t.board_rank ?? Number.POSITIVE_INFINITY
}

function compareRoots(a: any, b: any): number {
  const ra = rootRank(a)
  const rb = rootRank(b)
  if (ra !== rb) return ra - rb
  return String(b.updated_at).localeCompare(String(a.updated_at))
}

/** Root tasks and nested sub-tasks in board visual order (columns L→R, cards T→B). */
export function boardTaskOrder(tasks: any[], showSubtasks: boolean): string[] {
  const byStatus: Record<string, any[]> = {}
  for (const s of BOARD_COLUMNS) byStatus[s] = []

  const childrenOf: Record<string, any[]> = {}
  for (const t of tasks) {
    if (t.parent_id) {
      if (showSubtasks) (childrenOf[t.parent_id] ??= []).push(t)
      continue
    }
    ;(byStatus[t.status] ??= []).push(t)
  }

  for (const s of BOARD_COLUMNS) {
    byStatus[s].sort(compareRoots)
  }

  const ids: string[] = []
  for (const s of BOARD_COLUMNS) {
    for (const t of byStatus[s]) {
      ids.push(t.id)
      if (showSubtasks) {
        for (const c of childrenOf[t.id] ?? []) ids.push(c.id)
      }
    }
  }
  return ids
}

export function boardAdjacentTaskId(
  tasks: any[],
  showSubtasks: boolean,
  currentId: string,
  dir: -1 | 1
): string | null {
  const order = boardTaskOrder(tasks, showSubtasks)
  const idx = order.indexOf(currentId)
  if (idx < 0) return null
  const next = idx + dir
  if (next < 0 || next >= order.length) return null
  return order[next]
}

/** List-view visual order: roots then nested children when showSubtasks. */
export function listTaskOrder(tasks: any[], showSubtasks: boolean): string[] {
  const roots = tasks.filter((t) => !t.parent_id)
  const childrenOf = (pid: string) => tasks.filter((t) => t.parent_id === pid)
  const out: string[] = []
  if (!showSubtasks) {
    for (const t of roots) out.push(t.id)
    return out
  }
  const rootIds = new Set(roots.map((t) => t.id))
  for (const t of roots) {
    out.push(t.id)
    for (const c of childrenOf(t.id)) out.push(c.id)
  }
  for (const t of tasks) {
    if (t.parent_id && !rootIds.has(t.parent_id) && !roots.some((r) => r.id === t.id)) {
      if (!out.includes(t.id)) out.push(t.id)
    }
  }
  return out
}

export function listAdjacentTaskId(
  tasks: any[],
  showSubtasks: boolean,
  currentId: string,
  dir: -1 | 1
): string | null {
  const order = listTaskOrder(tasks, showSubtasks)
  const idx = order.indexOf(currentId)
  if (idx < 0) return null
  const next = idx + dir
  if (next < 0 || next >= order.length) return null
  return order[next]
}

/** Unique task ids in activity-list order (first appearance wins). */
export function activityAdjacentTaskId(
  activities: { task_id: string }[],
  currentId: string,
  dir: -1 | 1
): string | null {
  const order: string[] = []
  const seen = new Set<string>()
  for (const a of activities) {
    if (seen.has(a.task_id)) continue
    seen.add(a.task_id)
    order.push(a.task_id)
  }
  const idx = order.indexOf(currentId)
  if (idx < 0) return null
  const next = idx + dir
  if (next < 0 || next >= order.length) return null
  return order[next]
}
