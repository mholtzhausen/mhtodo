import type { GUISettings } from './settings'
import { claudeSpawnEnabled, normalizeSpawn } from './settings'

/** Whether the Claude icon should show for a task (not whether open will succeed). */
export function claudeIconVisible(
  task: { cwd?: string; human_only?: boolean; status?: string },
  settings: GUISettings
): boolean {
  if (!claudeSpawnEnabled(settings)) return false
  if (task.human_only || task.status === 'done') return false
  if (settings.claude.require_cwd && !task.cwd?.trim()) return false
  return true
}

/** Backend binary readiness for the current spawn mode. */
export async function claudeBackendReady(
  settings: GUISettings,
  checkBinary: (path: string) => Promise<boolean>
): Promise<{ claude: boolean; backend: boolean }> {
  const spawn = normalizeSpawn(settings.claude.spawn)
  if (spawn === 'disabled') {
    return { claude: false, backend: false }
  }
  const claude = await checkBinary(settings.claude.binary)
  if (!claude) {
    return { claude: false, backend: false }
  }
  if (spawn === 'herdr') {
    const backend = await checkBinary(settings.herdr.binary)
    return { claude, backend }
  }
  // terminal: optional preferred emulator; empty binary means auto-pick at launch
  if (settings.terminal.binary.trim()) {
    const backend = await checkBinary(settings.terminal.binary)
    return { claude, backend }
  }
  return { claude, backend: true }
}
