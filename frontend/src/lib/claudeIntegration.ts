import type { GUISettings } from './api'

/** Whether the Claude/Herdr icon should show for a task (not whether open will succeed). */
export function claudeIconVisible(
  task: { cwd?: string; human_only?: boolean; status?: string },
  settings: GUISettings
): boolean {
  if (task.human_only || task.status === 'done') return false
  if (settings.claude.require_cwd && !task.cwd?.trim()) return false
  return true
}
