// GUI settings types — mirror internal/settings/settings.go (Wails bindings).
import { settings as goSettings } from '../../wailsjs/go/models'

export type ClaudeSpawn = 'herdr' | 'terminal' | 'disabled'

export interface IntegrationConfig {
  enabled: boolean
  binary: string
  env_start: string
}

export interface ClaudeConfig extends IntegrationConfig {
  spawn: ClaudeSpawn
  ticket_prompt: string
  close_tab_on_done: boolean
  require_cwd: boolean
}

export interface HerdrConfig extends IntegrationConfig {
  space_name: string
}

export interface TerminalConfig {
  binary: string
  env_start: string
}

export interface GUISettings {
  default_cwd: string
  default_human_only: boolean
  default_include_in_report: boolean
  archive_done_subtasks: boolean
  start_hidden: boolean
  claude: ClaudeConfig
  herdr: HerdrConfig
  terminal: TerminalConfig
  zed: IntegrationConfig
}

export const DEFAULT_CLAUDE_TICKET_PROMPT =
  'read todo {{todo-hash}} and start on the ticket. if there is not enough information to start working, gather as much information about the issue on your own (read-only) and ask your human for input. When starting the task, remember to create subtasks and notify about activities on the task.'

export const DEFAULT_HERDR_SPACE_NAME = 'mhtodo'

export function normalizeSpawn(s: string | undefined | null): ClaudeSpawn {
  switch ((s ?? '').trim().toLowerCase()) {
    case 'herdr':
      return 'herdr'
    case 'terminal':
      return 'terminal'
    default:
      return 'disabled'
  }
}

/** Claude actions are available when spawn is herdr or terminal. */
export function claudeSpawnEnabled(s: GUISettings): boolean {
  return normalizeSpawn(s.claude.spawn) !== 'disabled'
}

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
  default_include_in_report: true,
  archive_done_subtasks: false,
  start_hidden: false,
  claude: {
    enabled: false,
    spawn: 'disabled',
    binary: 'claude',
    env_start: '',
    ticket_prompt: '',
    close_tab_on_done: false,
    require_cwd: true
  },
  herdr: { enabled: false, binary: 'herdr', env_start: '', space_name: '' },
  terminal: { binary: '', env_start: '' },
  zed: { enabled: false, binary: 'zed', env_start: '' }
})

// Wails codegen uses json struct tags → snake_case field names on the wire.
export function fromGoSettings(s: goSettings.GUISettings): GUISettings {
  const spawn = normalizeSpawn(s.claude?.spawn)
  return {
    default_cwd: s.default_cwd ?? '',
    default_human_only: !!s.default_human_only,
    default_include_in_report: s.default_include_in_report !== false,
    archive_done_subtasks: !!s.archive_done_subtasks,
    start_hidden: !!s.start_hidden,
    claude: {
      enabled: spawn !== 'disabled',
      spawn,
      binary: s.claude?.binary ?? 'claude',
      env_start: s.claude?.env_start ?? '',
      ticket_prompt: s.claude?.ticket_prompt ?? '',
      close_tab_on_done: !!s.claude?.close_tab_on_done,
      require_cwd: s.claude?.require_cwd !== false
    },
    herdr: {
      enabled: spawn === 'herdr',
      binary: s.herdr?.binary ?? 'herdr',
      env_start: s.herdr?.env_start ?? '',
      space_name: s.herdr?.space_name ?? ''
    },
    terminal: {
      binary: s.terminal?.binary ?? '',
      env_start: s.terminal?.env_start ?? ''
    },
    zed: {
      enabled: !!s.zed?.enabled,
      binary: s.zed?.binary ?? 'zed',
      env_start: s.zed?.env_start ?? ''
    }
  }
}

export function toGoSettings(s: GUISettings): goSettings.GUISettings {
  const spawn = normalizeSpawn(s.claude.spawn)
  return new goSettings.GUISettings({
    default_cwd: s.default_cwd,
    default_human_only: s.default_human_only,
    default_include_in_report: s.default_include_in_report,
    archive_done_subtasks: s.archive_done_subtasks,
    start_hidden: s.start_hidden,
    claude: {
      enabled: spawn !== 'disabled',
      spawn,
      binary: s.claude.binary,
      env_start: s.claude.env_start,
      ticket_prompt: s.claude.ticket_prompt,
      close_tab_on_done: s.claude.close_tab_on_done,
      require_cwd: s.claude.require_cwd
    },
    herdr: {
      enabled: spawn === 'herdr',
      binary: s.herdr.binary,
      env_start: s.herdr.env_start,
      space_name: s.herdr.space_name
    },
    terminal: {
      binary: s.terminal.binary,
      env_start: s.terminal.env_start
    },
    zed: {
      enabled: s.zed.enabled,
      binary: s.zed.binary,
      env_start: s.zed.env_start
    }
  })
}
