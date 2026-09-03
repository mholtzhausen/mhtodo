// Task templates (v0.5): named sets of presets applied when creating a task.
//
// Every preset is nullable. `null` means "not part of this template", so task
// creation falls back to the normal default (the GUI settings defaults, or
// core's own). A present-but-empty string or an explicit `false` is a real
// value that overrides that default. Keeping that distinction is why the
// pointer/null shape is carried all the way from SQLite to this module rather
// than being flattened to '' / false anywhere in between.
import type { Status } from './api'

/** Mirrors core.Template (internal/core/template.go) — JSON names are a contract. */
export interface TaskTemplate {
  id: string
  name: string
  title_prefix: string | null
  description: string | null
  status: Status | null
  cwd: string | null
  slack_thread: string | null
  human_only: boolean | null
  include_in_report: boolean | null
  created_at: string
  updated_at: string
}

/** The subset of TaskTemplate that is an actual task preset. */
export type TemplateFieldKey =
  | 'title_prefix'
  | 'description'
  | 'status'
  | 'cwd'
  | 'slack_thread'
  | 'human_only'
  | 'include_in_report'

export type TemplateFieldKind = 'text' | 'textarea' | 'path' | 'url' | 'bool' | 'status'

export interface TemplateFieldMeta {
  key: TemplateFieldKey
  /** Field label in the settings editor and the save-as-template modal. */
  label: string
  /** Short label used for the picker chips, where space is tight. */
  chip: string
  kind: TemplateFieldKind
  hint?: string
  placeholder?: string
}

/**
 * Single source of truth for which fields a template carries, in the order the
 * new-task dialog shows them. Drives the settings editor, the save-as-template
 * checkboxes and the picker chips, so adding a field is a one-line change here
 * plus the Go struct.
 */
export const TEMPLATE_FIELDS: readonly TemplateFieldMeta[] = [
  {
    key: 'title_prefix',
    label: 'Title prefix',
    chip: 'prefix',
    kind: 'text',
    hint: 'Prepended to whatever you type as the title. Trailing spaces are kept.',
    placeholder: 'BUG: '
  },
  {
    key: 'description',
    label: 'Description',
    chip: 'description',
    kind: 'textarea',
    placeholder: 'Boilerplate notes for tasks from this template…'
  },
  { key: 'status', label: 'Status', chip: 'status', kind: 'status' },
  {
    key: 'cwd',
    label: 'Working directory',
    chip: 'cwd',
    kind: 'path',
    placeholder: '/home/you/project'
  },
  {
    key: 'slack_thread',
    label: 'Slack thread',
    chip: 'slack',
    kind: 'url',
    placeholder: 'https://…'
  },
  {
    key: 'human_only',
    label: 'Human only',
    chip: 'human-only',
    kind: 'bool',
    hint: 'Agents skip tasks marked human-only.'
  },
  {
    key: 'include_in_report',
    label: 'Include in Slack report',
    chip: 'in-report',
    kind: 'bool',
    hint: 'Whether the task shows up in the board summary copy.'
  }
] as const

/** Preset values keyed by field, with null meaning "not part of this template". */
export type TemplateValues = {
  title_prefix: string | null
  description: string | null
  status: Status | null
  cwd: string | null
  slack_thread: string | null
  human_only: boolean | null
  include_in_report: boolean | null
}

export function emptyValues(): TemplateValues {
  return {
    title_prefix: null,
    description: null,
    status: null,
    cwd: null,
    slack_thread: null,
    human_only: null,
    include_in_report: null
  }
}

export function valuesOf(t: TaskTemplate): TemplateValues {
  return {
    title_prefix: t.title_prefix,
    description: t.description,
    status: t.status,
    cwd: t.cwd,
    slack_thread: t.slack_thread,
    human_only: t.human_only,
    include_in_report: t.include_in_report
  }
}

/** True when the template defines this field at all. */
export function isSet(v: TemplateValues, key: TemplateFieldKey): boolean {
  return v[key] !== null && v[key] !== undefined
}

/** The fields a template presets, in TEMPLATE_FIELDS order. */
export function setFields(t: TaskTemplate): TemplateFieldMeta[] {
  const v = valuesOf(t)
  return TEMPLATE_FIELDS.filter((f) => isSet(v, f.key))
}

/**
 * Chip text for one preset field. Booleans must carry their value: a template
 * forcing human-only off would otherwise be indistinguishable from one forcing
 * it on.
 */
export function chipLabel(t: TaskTemplate, f: TemplateFieldMeta): string {
  const value = valuesOf(t)[f.key]
  if (f.kind === 'bool') return `${f.chip}: ${value ? 'on' : 'off'}`
  if (f.kind === 'status') return `${f.chip}: ${value}`
  if (typeof value === 'string' && value === '') return `${f.chip}: none`
  return f.chip
}

/** The shape the new-task dialog holds while the user edits it. */
export interface NewTaskFormValues {
  title: string
  description: string
  status: Status
  cwd: string
  slackThread: string
  humanOnly: boolean
  includeInReport: boolean
}

/**
 * Overlays a template's set fields onto the current form values, leaving
 * fields the template does not define untouched so the GUI-settings defaults
 * already seeded into the form survive. Mirrors core.Template.Apply.
 *
 * appliedPrefix is the prefix a previously picked template contributed; it is
 * stripped first so switching templates swaps the prefix instead of stacking
 * two of them.
 *
 * Returns the patched values plus the new prefix length, so the caller can put
 * the caret after the prefix when it focuses the title input.
 */
export function applyTemplate(
  t: TaskTemplate,
  current: NewTaskFormValues,
  appliedPrefix = ''
): { values: NewTaskFormValues; titlePrefix: string } {
  const next: NewTaskFormValues = { ...current }
  let titlePrefix = ''

  if (t.title_prefix !== null) {
    const bare =
      appliedPrefix && current.title.startsWith(appliedPrefix)
        ? current.title.slice(appliedPrefix.length)
        : current.title
    titlePrefix = t.title_prefix
    next.title = titlePrefix + bare
  } else if (appliedPrefix && current.title.startsWith(appliedPrefix)) {
    // The new template defines no prefix, so drop the old one.
    next.title = current.title.slice(appliedPrefix.length)
  }

  if (t.description !== null) next.description = t.description
  if (t.status !== null) next.status = t.status
  if (t.cwd !== null) next.cwd = t.cwd
  if (t.slack_thread !== null) next.slackThread = t.slack_thread
  if (t.human_only !== null) next.humanOnly = t.human_only
  if (t.include_in_report !== null) next.includeInReport = t.include_in_report

  return { values: next, titlePrefix }
}

/** Builds the Wails core.TemplateInput payload from a name plus preset values. */
export function toGoTemplateInput(name: string, v: TemplateValues) {
  return {
    Name: name,
    TitlePrefix: v.title_prefix,
    Description: v.description,
    Status: v.status as unknown as string | null,
    Cwd: v.cwd,
    SlackThread: v.slack_thread,
    HumanOnly: v.human_only,
    IncludeInReport: v.include_in_report
  }
}
