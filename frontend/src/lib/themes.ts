// GUI themes (v0.6): categorized design tokens applied as CSS custom properties.
// Mirrors internal/core/theme_tokens.go — keep keys and categories in sync.

/** Mirrors core.Theme — JSON names are a contract. */
export interface Theme {
  id: string
  name: string
  builtin_key: string | null
  tokens: Record<string, string>
  created_at: string
  updated_at: string
  active: boolean
}

export type ThemeTokenKind = 'color' | 'length'

export interface ThemeFieldMeta {
  key: string
  category: string
  label: string
  kind: ThemeTokenKind
  allowRgba?: boolean
  minPx?: number
  maxPx?: number
}

/** Single FE source of truth for Settings → Themes field order/pickers. */
export const THEME_FIELDS: readonly ThemeFieldMeta[] = [
  { key: 'color.canvas', category: 'Surfaces', label: 'Canvas', kind: 'color' },
  { key: 'color.chrome', category: 'Surfaces', label: 'Chrome', kind: 'color' },
  { key: 'color.col', category: 'Surfaces', label: 'Column', kind: 'color' },
  { key: 'color.card', category: 'Surfaces', label: 'Card', kind: 'color' },
  { key: 'color.cardHi', category: 'Surfaces', label: 'Card elevated', kind: 'color' },
  { key: 'color.field', category: 'Surfaces', label: 'Field', kind: 'color' },
  { key: 'color.line', category: 'Borders', label: 'Line', kind: 'color' },
  { key: 'color.lineSoft', category: 'Borders', label: 'Line soft', kind: 'color' },
  { key: 'color.ink', category: 'Text', label: 'Ink', kind: 'color' },
  { key: 'color.ink2', category: 'Text', label: 'Ink secondary', kind: 'color' },
  { key: 'color.ink3', category: 'Text', label: 'Ink muted', kind: 'color' },
  { key: 'color.accent', category: 'Accent', label: 'Accent', kind: 'color' },
  { key: 'color.accentHi', category: 'Accent', label: 'Accent hover', kind: 'color' },
  { key: 'color.accentInk', category: 'Accent', label: 'Accent ink', kind: 'color' },
  { key: 'color.stPending', category: 'Status', label: 'Pending', kind: 'color' },
  { key: 'color.stWip', category: 'Status', label: 'WIP', kind: 'color' },
  { key: 'color.stWaiting', category: 'Status', label: 'Waiting', kind: 'color' },
  { key: 'color.stReview', category: 'Status', label: 'Review', kind: 'color' },
  { key: 'color.stDone', category: 'Status', label: 'Done', kind: 'color' },
  { key: 'color.danger', category: 'Feedback', label: 'Danger', kind: 'color' },
  {
    key: 'color.track',
    category: 'Feedback',
    label: 'Progress track',
    kind: 'color',
    allowRgba: true
  },
  { key: 'radius.control', category: 'Radius', label: 'Control', kind: 'length', minPx: 0, maxPx: 24 },
  { key: 'radius.card', category: 'Radius', label: 'Card', kind: 'length', minPx: 0, maxPx: 32 },
  { key: 'radius.panel', category: 'Radius', label: 'Panel', kind: 'length', minPx: 0, maxPx: 40 },
  { key: 'radius.chip', category: 'Radius', label: 'Chip', kind: 'length', minPx: 0, maxPx: 16 },
  { key: 'space.gapSm', category: 'Space', label: 'Gap small', kind: 'length', minPx: 0, maxPx: 48 },
  { key: 'space.gapMd', category: 'Space', label: 'Gap medium', kind: 'length', minPx: 0, maxPx: 64 },
  { key: 'space.gapLg', category: 'Space', label: 'Gap large', kind: 'length', minPx: 0, maxPx: 96 },
  { key: 'space.padSm', category: 'Space', label: 'Pad small', kind: 'length', minPx: 0, maxPx: 48 },
  { key: 'space.padMd', category: 'Space', label: 'Pad medium', kind: 'length', minPx: 0, maxPx: 64 },
  { key: 'space.padLg', category: 'Space', label: 'Pad large', kind: 'length', minPx: 0, maxPx: 96 }
]

/** Slate factory defaults (must match core.FactoryTokens("slate")). */
export const SLATE_FACTORY_TOKENS: Record<string, string> = {
  'color.canvas': '#252b37',
  'color.chrome': '#1f242f',
  'color.col': '#2c3340',
  'color.card': '#343c4d',
  'color.cardHi': '#3d465a',
  'color.field': '#28303d',
  'color.line': '#4a556b',
  'color.lineSoft': '#3a4356',
  'color.ink': '#eef1f7',
  'color.ink2': '#b6bed0',
  'color.ink3': '#8791a5',
  'color.accent': '#7b8cff',
  'color.accentHi': '#93a0ff',
  'color.accentInk': '#0e1230',
  'color.stPending': '#a3adbf',
  'color.stWip': '#7b8cff',
  'color.stWaiting': '#e8ab4a',
  'color.stReview': '#c084fc',
  'color.stDone': '#4cc48e',
  'color.danger': '#ff8492',
  'color.track': 'rgba(255, 255, 255, 0.13)',
  'radius.control': '4px',
  'radius.card': '6px',
  'radius.panel': '8px',
  'radius.chip': '3px',
  'space.gapSm': '8px',
  'space.gapMd': '10px',
  'space.gapLg': '14px',
  'space.padSm': '8px',
  'space.padMd': '12px',
  'space.padLg': '20px'
}

/** Ordered unique category names from THEME_FIELDS. */
export const THEME_CATEGORIES: readonly string[] = [
  ...new Set(THEME_FIELDS.map((f) => f.category))
]

/** Map dotted token key → CSS custom property (mirrors core.TokenToCSSVar). */
export function tokenToCSSVar(key: string): string {
  const i = key.indexOf('.')
  if (i < 0) return '--' + kebabToken(key)
  const prefix = key.slice(0, i)
  const rest = kebabToken(key.slice(i + 1))
  switch (prefix) {
    case 'color':
      return '--color-' + rest
    case 'radius':
      return '--radius-' + rest
    case 'space':
      return '--spacing-' + rest
    default:
      return '--' + prefix + '-' + rest
  }
}

function kebabToken(s: string): string {
  let out = ''
  for (let i = 0; i < s.length; i++) {
    const ch = s[i]
    if (ch >= 'A' && ch <= 'Z') {
      if (i > 0) out += '-'
      out += ch.toLowerCase()
      continue
    }
    if (ch >= '0' && ch <= '9' && i > 0) {
      const prev = s[i - 1]
      if (prev >= 'a' && prev <= 'z') out += '-'
    }
    out += ch
  }
  return out
}

/** Apply theme tokens onto documentElement for Tailwind @theme utilities. */
export function applyThemeTokens(tokens: Record<string, string> | null | undefined): void {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  const merged = { ...SLATE_FACTORY_TOKENS, ...(tokens ?? {}) }
  for (const field of THEME_FIELDS) {
    const val = merged[field.key]
    if (val) root.style.setProperty(tokenToCSSVar(field.key), val)
  }
}

export function toGoThemeInput(name: string, tokens: Record<string, string>) {
  return { Name: name, Tokens: tokens }
}

export function parsePx(value: string): number {
  const m = String(value).trim().match(/^(\d+(?:\.\d+)?)\s*px$/i)
  if (m) return Math.round(parseFloat(m[1]))
  const n = parseFloat(value)
  return Number.isFinite(n) ? Math.round(n) : 0
}

export function formatPx(n: number): string {
  return `${Math.max(0, Math.round(n))}px`
}

/** Hex for <input type="color">; falls back to #000000 for rgba. */
export function colorInputValue(value: string): string {
  const v = value.trim()
  if (/^#[0-9a-fA-F]{6}$/.test(v)) return v.toLowerCase()
  if (/^#[0-9a-fA-F]{3}$/.test(v)) {
    const r = v[1],
      g = v[2],
      b = v[3]
    return `#${r}${r}${g}${g}${b}${b}`.toLowerCase()
  }
  return '#000000'
}
