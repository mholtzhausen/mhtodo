// Ticket priority (v20). Visual + filter only: never reorders the board or
// Focus lanes. Levels mirror internal/core/priority.go (ascending).
export type Priority = 'none' | 'low' | 'medium' | 'high' | 'urgent'

export const PRIORITIES: readonly Priority[] = ['none', 'low', 'medium', 'high', 'urgent']

export const PRIORITY_LABELS: Record<Priority, string> = {
  none: 'No priority',
  low: 'Low',
  medium: 'Medium',
  high: 'High',
  urgent: 'Urgent'
}

/** Theme-token tint per level (reuses existing tokens so every theme covers it). */
export const PRIORITY_TEXT: Record<Priority, string> = {
  none: 'text-ink-3',
  low: 'text-ink-2',
  medium: 'text-st-pr',
  high: 'text-st-waiting',
  urgent: 'text-danger'
}

export function normPriority(p: unknown): Priority {
  return typeof p === 'string' && (PRIORITIES as readonly string[]).includes(p)
    ? (p as Priority)
    : 'none'
}

const STORAGE_KEY = 'mhtodo.priorityFilter'

/** Empty array = no filter. */
export function loadPriorityFilter(): Priority[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return PRIORITIES.filter((p) => parsed.includes(p))
  } catch {
    return []
  }
}

export function savePriorityFilter(filter: Priority[]) {
  try {
    if (filter.length) localStorage.setItem(STORAGE_KEY, JSON.stringify(filter))
    else localStorage.removeItem(STORAGE_KEY)
  } catch {
    /* ignore */
  }
}

/**
 * Keeps tasks whose priority is in `filter` (empty = everything). A root
 * stays visible when any of its sub-tasks match, so the match has context.
 * Input order is preserved — filtering never reorders.
 */
export function applyPriorityFilter<
  T extends { id: string; parent_id?: string | null; priority?: string }
>(tasks: T[], filter: readonly Priority[]): T[] {
  if (!filter.length) return tasks
  const want = new Set<string>(filter)
  const matches = (t: T) => want.has(normPriority(t.priority))
  const parentsWithMatch = new Set<string>()
  for (const t of tasks) if (t.parent_id && matches(t)) parentsWithMatch.add(t.parent_id)
  return tasks.filter((t) => matches(t) || (!t.parent_id && parentsWithMatch.has(t.id)))
}
