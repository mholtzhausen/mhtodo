import type { Status } from './api'

const BOARD_COLUMNS: Status[] = ['pending', 'wip', 'waiting', 'review', 'done']

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
