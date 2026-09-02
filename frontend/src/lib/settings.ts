// GUI settings types — mirror internal/settings/settings.go (Wails bindings).
import { settings as goSettings } from '../../wailsjs/go/models'

export interface IntegrationConfig {
  enabled: boolean
  binary: string
  env_start: string
}

export interface HerdrConfig extends IntegrationConfig {
  space_name: string
}

export interface GUISettings {
  default_cwd: string
  default_human_only: boolean
  claude: IntegrationConfig
  herdr: HerdrConfig
}

export const defaultSettings = (): GUISettings => ({
  default_cwd: '',
  default_human_only: false,
  claude: { enabled: false, binary: 'claude', env_start: '' },
  herdr: { enabled: false, binary: 'herdr', env_start: '', space_name: 'mhtodo' }
})

// Wails codegen uses json struct tags → snake_case field names on the wire.
export function fromGoSettings(s: goSettings.GUISettings): GUISettings {
  return {
    default_cwd: s.default_cwd ?? '',
    default_human_only: !!s.default_human_only,
    claude: {
      enabled: !!s.claude?.enabled,
      binary: s.claude?.binary ?? 'claude',
      env_start: s.claude?.env_start ?? ''
    },
    herdr: {
      enabled: !!s.herdr?.enabled,
      binary: s.herdr?.binary ?? 'herdr',
      env_start: s.herdr?.env_start ?? '',
      space_name: s.herdr?.space_name ?? 'mhtodo'
    }
  }
}

export function toGoSettings(s: GUISettings): goSettings.GUISettings {
  return new goSettings.GUISettings({
    default_cwd: s.default_cwd,
    default_human_only: s.default_human_only,
    claude: {
      enabled: s.claude.enabled,
      binary: s.claude.binary,
      env_start: s.claude.env_start
    },
    herdr: {
      enabled: s.herdr.enabled,
      binary: s.herdr.binary,
      env_start: s.herdr.env_start,
      space_name: s.herdr.space_name
    }
  })
}
