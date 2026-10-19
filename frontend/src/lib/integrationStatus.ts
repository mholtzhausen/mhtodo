// Shared Zed readiness for board cards + detail. Avoids per-card
// getSettings/checkBinary storms when many TaskActivityActions mount.
import { api, type GUISettings } from './api'
import { defaultSettings } from './settings'

let settings: GUISettings = defaultSettings()
const binaryOk = new Map<string, boolean>()
const listeners = new Set<() => void>()

function notify() {
  for (const fn of listeners) fn()
}

export function subscribeIntegrationStatus(fn: () => void): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

export function getIntegrationSettings(): GUISettings {
  return settings
}

export function setIntegrationSettings(s: GUISettings) {
  settings = s
  notify()
}

export async function loadIntegrationSettings(): Promise<GUISettings> {
  settings = await api.getSettings()
  notify()
  return settings
}

export function invalidateBinaryCache() {
  binaryOk.clear()
  notify()
}

export async function checkBinaryCached(path: string): Promise<boolean> {
  const key = path.trim()
  if (!key) return false
  if (binaryOk.has(key)) return binaryOk.get(key)!
  try {
    const ok = await api.checkBinary(key)
    binaryOk.set(key, ok)
    return ok
  } catch {
    binaryOk.set(key, false)
    return false
  }
}

export type BinaryReadiness = { zed: boolean }

export async function refreshBinaryReadiness(s: GUISettings = settings): Promise<BinaryReadiness> {
  const out: BinaryReadiness = { zed: false }
  if (s.zed.enabled) {
    out.zed = await checkBinaryCached(s.zed.binary)
  }
  notify()
  return out
}

// Claude launcher kind ("herdr", a terminal name, or "" when unavailable).
// Detected once per GUI session; shared by every TaskActivityActions.
let claudeLauncher = ''
let claudeLauncherLoad: Promise<string> | null = null

export function getClaudeLauncher(): string {
  return claudeLauncher
}

export function loadClaudeLauncher(): Promise<string> {
  if (!claudeLauncherLoad) {
    claudeLauncherLoad = api
      .claudeLauncher()
      .catch(() => '')
      .then((kind) => {
        claudeLauncher = kind
        notify()
        return kind
      })
  }
  return claudeLauncherLoad
}
