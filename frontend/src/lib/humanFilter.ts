export type HumanFilter = 'all' | 'exclude' | 'only'

export function loadHumanFilter(): HumanFilter {
  try {
    const stored = localStorage.getItem('mhtodo.humanFilter')
    if (stored === 'all' || stored === 'exclude' || stored === 'only') return stored
  } catch {
    /* ignore */
  }
  return 'all'
}

export function applyHumanFilter<T extends { human_only?: boolean }>(
  tasks: T[],
  filter: HumanFilter
): T[] {
  if (filter === 'all') return tasks
  if (filter === 'only') return tasks.filter((t) => !!t.human_only)
  return tasks.filter((t) => !t.human_only)
}
