// GUI settings types — mirror internal/settings/settings.go (Wails bindings).
import { settings as goSettings } from '../../wailsjs/go/models'

export interface IntegrationConfig {
  enabled: boolean
  binary: string
  env_start: string
}

export interface ClaudeConfig extends IntegrationConfig {
  ticket_prompt: string
  close_tab_on_done: boolean
  require_cwd: boolean
}

export interface HerdrConfig extends IntegrationConfig {
  space_name: string
}

export interface GUISettings {
  default_cwd: string
  default_human_only: boolean
  archive_done_subtasks: boolean
  claude: ClaudeConfig
  herdr: HerdrConfig
}

export const DEFAULT_CLAUDE_TICKET_PROMPT =
  'read todo {{todo-hash}} and start on the ticket. if there is not enough information to start working, gather as much information about the issue on your own (read-only) and ask your human for input. When starting the task, remember to create subtasks and notify about activities on the task.'

export const DEFAULT_HERDR_SPACE_NAME = 'mhtodo'

export function effectiveTicketPrompt(s: GUISettings): string {
  const p = s.claude.ticket_prompt.trim()
  return p || DEFAULT_CLAUDE_TICKET_PROMPT
}

export function effectiveHerdrSpaceName(s: GUISettings): string {
  const n = s.herdr.space_name.trim()
  return n || DEFAULT_HERDR_SPACE_NAME
}

export const defaultSettings = (): GUISettings => ({
  default_cwd: '',
  default_human_only: false,
  archive_done_subtasks: false,
  claude: {
    enabled: false,
    binary: 'claude',
    env_start: '',
    ticket_prompt: '',
    close_tab_on_done: false,
    require_cwd: true
  },
  herdr: { enabled: false, binary: 'herdr', env_start: '', space_name: '' }
})

// Wails codegen uses json struct tags → snake_case field names on the wire.
export function fromGoSettings(s: goSettings.GUISettings): GUISettings {
  return {
    default_cwd: s.default_cwd ?? '',
    default_human_only: !!s.default_human_only,
    archive_done_subtasks: !!s.archive_done_subtasks,
    claude: {
      enabled: !!s.claude?.enabled,
      binary: s.claude?.binary ?? 'claude',
      env_start: s.claude?.env_start ?? '',
      ticket_prompt: s.claude?.ticket_prompt ?? '',
      close_tab_on_done: !!s.claude?.close_tab_on_done,
      require_cwd: s.claude?.require_cwd !== false
    },
    herdr: {
      enabled: !!s.herdr?.enabled,
      binary: s.herdr?.binary ?? 'herdr',
      env_start: s.herdr?.env_start ?? '',
      space_name: s.herdr?.space_name ?? ''
    }
  }
}

export function toGoSettings(s: GUISettings): goSettings.GUISettings {
  return new goSettings.GUISettings({
    default_cwd: s.default_cwd,
    default_human_only: s.default_human_only,
    archive_done_subtasks: s.archive_done_subtasks,
    claude: {
      enabled: s.claude.enabled,
      binary: s.claude.binary,
      env_start: s.claude.env_start,
      ticket_prompt: s.claude.ticket_prompt,
      close_tab_on_done: s.claude.close_tab_on_done,
      require_cwd: s.claude.require_cwd
    },
    herdr: {
      enabled: s.herdr.enabled,
      binary: s.herdr.binary,
      env_start: s.herdr.env_start,
      space_name: s.herdr.space_name
    }
  })
}
