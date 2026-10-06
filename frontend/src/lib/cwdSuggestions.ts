// Warm cwd typeahead candidates: template-backed paths + previously used ticket
// cwds. Built at App startup and refreshed on tasks:changed / templates:changed
// so focus only filters/ranks in memory (no fetch lag).
import { api } from './api'
import { fuzzyScore } from './fuzzy'
import type { TaskTemplate } from './templates'

export type CwdCandidateKind = 'template' | 'history'

export interface CwdCandidate {
  kind: CwdCandidateKind
  /** Template id, or the cwd string for history rows. */
  id: string
  /** Template name, or '' for history. */
  name: string
  cwd: string
}

const NAME_MATCH_BOOST = 10_000
const DEFAULT_CAP = 40

let candidates: CwdCandidate[] = []
let ready = false
let loading = false
let refreshSeq = 0

type Listener = () => void
const listeners = new Set<Listener>()

function notify() {
  for (const l of listeners) l()
}

/** Subscribe to candidate-list updates. Returns an unsubscribe function. */
export function subscribeCwdSuggestions(listener: Listener): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function getCwdCandidates(): readonly CwdCandidate[] {
  return candidates
}

export function cwdSuggestionsReady(): boolean {
  return ready
}

export function cwdSuggestionsLoading(): boolean {
  return loading
}

export function mergeCwdCandidates(
  templates: TaskTemplate[],
  taskCwds: string[]
): CwdCandidate[] {
  const templateRows: CwdCandidate[] = []
  const covered = new Set<string>()

  for (const t of templates) {
    const cwd = (t.cwd ?? '').trim()
    if (!cwd) continue
    templateRows.push({ kind: 'template', id: t.id, name: t.name, cwd })
    covered.add(cwd)
  }

  const historyRows: CwdCandidate[] = []
  const seenHistory = new Set<string>()
  for (const raw of taskCwds) {
    const cwd = raw.trim()
    if (!cwd || covered.has(cwd) || seenHistory.has(cwd)) continue
    seenHistory.add(cwd)
    historyRows.push({ kind: 'history', id: cwd, name: '', cwd })
  }

  // Stable empty-query order: all templates, then history (alpha within each).
  templateRows.sort((a, b) => a.name.localeCompare(b.name) || a.cwd.localeCompare(b.cwd))
  historyRows.sort((a, b) => a.cwd.localeCompare(b.cwd))
  return [...templateRows, ...historyRows]
}

function scoreCandidate(query: string, row: CwdCandidate): number {
  if (!query) {
    // Empty query: templates before history; preserve merge order via 0.
    return row.kind === 'template' ? 1 : 0
  }
  const nameScore = row.kind === 'template' ? fuzzyScore(query, row.name) : -1
  const cwdScore = fuzzyScore(query, row.cwd)
  if (nameScore < 0 && cwdScore < 0) return -1
  const base = Math.max(nameScore, cwdScore)
  return nameScore >= 0 ? base + NAME_MATCH_BOOST : base
}

/** Filter and rank warm candidates for the typeahead dropdown. */
export function rankCwdCandidates(
  query: string,
  source: readonly CwdCandidate[] = candidates,
  cap = DEFAULT_CAP
): CwdCandidate[] {
  const q = query.trim()
  if (!q) {
    return source.slice(0, cap)
  }
  const scored: { row: CwdCandidate; score: number }[] = []
  for (const row of source) {
    const score = scoreCandidate(q, row)
    if (score < 0) continue
    scored.push({ row, score })
  }
  scored.sort((a, b) => {
    if (b.score !== a.score) return b.score - a.score
    if (a.row.name !== b.row.name) return a.row.name.localeCompare(b.row.name)
    return a.row.cwd.localeCompare(b.row.cwd)
  })
  return scored.slice(0, cap).map((s) => s.row)
}

/** Reload templates + wide task list and replace the warm candidate store. */
export async function refreshCwdSuggestions(): Promise<void> {
  const seq = ++refreshSeq
  loading = true
  notify()
  try {
    const [templates, tasks] = await Promise.all([
      api.listTemplates(),
      api.list({ includeDone: true, includeHumanOnly: true })
    ])
    if (seq !== refreshSeq) return
    candidates = mergeCwdCandidates(
      templates,
      tasks.map((t) => t.cwd)
    )
    ready = true
  } finally {
    if (seq === refreshSeq) {
      loading = false
      notify()
    }
  }
}
