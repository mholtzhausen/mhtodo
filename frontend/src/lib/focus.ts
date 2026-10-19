// Focus view pins live in the DB (tasks.focus_rank; CLI `mhtodo focus`).
// Mirrors core.FocusMax — the backend enforces it, this is only for UI hints.
export const FOCUS_MAX = 6

// Pre-DB builds kept pins per-viewer in localStorage under this key.
const LEGACY_KEY = 'mhtodo.focusIds'

/**
 * One-time import of legacy localStorage pins into the DB. Only runs when the
 * DB has no focus yet; the key is removed afterwards either way. Returns true
 * when anything was pinned.
 */
export async function migrateLegacyFocus(
  dbEmpty: boolean,
  focus: (id: string) => Promise<unknown>
): Promise<boolean> {
  let ids: string[] = []
  try {
    const raw = localStorage.getItem(LEGACY_KEY)
    if (!raw) return false
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) ids = parsed.filter((v): v is string => typeof v === 'string' && !!v)
    localStorage.removeItem(LEGACY_KEY)
  } catch {
    return false
  }
  if (!dbEmpty) return false
  let any = false
  for (const id of ids.slice(0, FOCUS_MAX)) {
    try {
      await focus(id)
      any = true
    } catch {
      /* deleted / done since — skip */
    }
  }
  return any
}
