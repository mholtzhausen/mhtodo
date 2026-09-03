<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg, type Status } from '../lib/api'
  import {
    TEMPLATE_FIELDS,
    emptyValues,
    type TemplateFieldKey,
    type TemplateValues
  } from '../lib/templates'
  import StatusPicker from './StatusPicker.svelte'

  // No tri-state control is needed here: the per-field checkbox *is* the null
  // selector (unchecked = the field is not part of the template), so booleans
  // get a plain include checkbox plus a plain value checkbox. TriStateCheck is
  // only used in the settings editor, which has no include column.
  let {
    open,
    /** Values to seed the editable fields from (the form or the task). */
    source,
    /** Fields ticked when the dialog opens. */
    initialSelection = [],
    suggestedName = '',
    onClose,
    onSaved,
    onError
  }: {
    open: boolean
    source: {
      title: string
      description: string
      status: Status
      cwd: string
      slackThread: string
      humanOnly: boolean
      includeInReport: boolean
    }
    initialSelection?: TemplateFieldKey[]
    suggestedName?: string
    onClose: () => void
    onSaved: (name: string) => void
    onError?: (msg: string) => void
  } = $props()

  let name = $state('')
  let selected = $state<Set<TemplateFieldKey>>(new Set())
  let values = $state<TemplateValues>(emptyValues())
  let saving = $state(false)
  let nameError = $state('')
  let nameEl = $state<HTMLInputElement | null>(null)
  let seeded = false

  $effect(() => {
    if (!open) {
      seeded = false
      return
    }
    if (seeded) return
    seeded = true
    name = suggestedName
    nameError = ''
    // Seed every field from the source; the checkbox decides what is kept.
    values = {
      title_prefix: source.title,
      description: source.description,
      status: source.status,
      cwd: source.cwd,
      slack_thread: source.slackThread,
      human_only: source.humanOnly,
      include_in_report: source.includeInReport
    }
    selected = new Set(
      initialSelection.length ? initialSelection : defaultSelection()
    )
  })

  $effect(() => {
    if (open && nameEl) nameEl.focus()
  })

  /** Ticks the fields that carry a non-default value, so the common case is one click. */
  function defaultSelection(): TemplateFieldKey[] {
    const out: TemplateFieldKey[] = []
    if (source.cwd.trim()) out.push('cwd')
    if (source.humanOnly) out.push('human_only')
    if (!source.includeInReport) out.push('include_in_report')
    if (source.status !== 'pending') out.push('status')
    return out
  }

  function toggle(key: TemplateFieldKey) {
    const next = new Set(selected)
    if (next.has(key)) next.delete(key)
    else next.add(key)
    selected = next
  }

  /** Only ticked fields become presets; the rest stay null (unset). */
  function payload(): TemplateValues {
    const out = emptyValues()
    for (const f of TEMPLATE_FIELDS) {
      if (!selected.has(f.key)) continue
      ;(out as Record<string, unknown>)[f.key] = values[f.key]
    }
    return out
  }

  async function submit(e: Event) {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) {
      nameError = 'Name is required'
      nameEl?.focus()
      return
    }
    if (saving) return
    saving = true
    try {
      await api.createTemplate(trimmed, payload())
      onSaved(trimmed)
      onClose()
    } catch (err) {
      const msg = errMsg(err)
      if (msg.toLowerCase().includes('already exists')) nameError = msg
      else onError?.(msg)
    } finally {
      saving = false
    }
  }

  // App.svelte listens for Escape on window to dismiss the detail pane. While
  // this dialog is up it owns Escape, so it is claimed in the capture phase
  // before that listener can close whatever is behind us.
  function onWindowKeydownCapture(e: KeyboardEvent) {
    if (!open || e.key !== 'Escape') return
    e.preventDefault()
    e.stopPropagation()
    onClose()
  }

  const fieldClass =
    'w-full rounded border border-line-soft bg-field px-3 py-1.5 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25'
</script>

<svelte:window onkeydowncapture={onWindowKeydownCapture} />

{#if open}
  <!-- This dialog is mounted inside TaskDetail, which in modal mode sits in a
       backdrop whose click handler clears the selected task. Swallowing clicks
       here keeps interacting with the form from closing whatever is behind it. -->
  <!-- Purely a click sink, not a control: Escape is handled at window capture. -->
  <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
  <div
    onclick={(e) => e.stopPropagation()}
    class="fixed inset-0 z-[60] flex items-center justify-center bg-black/55 p-4 backdrop-blur-[2px]"
  >
    <form
      in:fly={{ y: 12, duration: 150 }}
      onsubmit={submit}
      class="flex max-h-[92vh] w-full max-w-md flex-col rounded-lg border border-line bg-col shadow-2xl"
    >
      <div class="flex flex-none items-center gap-2.5 border-b border-line-soft px-5 py-3.5">
        <h2 class="flex-1 text-base font-semibold text-ink">Save as template</h2>
        <button
          type="button"
          onclick={onClose}
          title="Close (esc)"
          class="rounded p-1.5 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
        >
          ✕
        </button>
      </div>

      <div class="flex min-h-0 flex-col gap-3.5 overflow-y-auto p-5">
        <label class="block">
          <span class="micro mb-1.5">Template name <em class="not-italic text-danger">*</em></span>
          <input
            bind:this={nameEl}
            bind:value={name}
            maxlength="80"
            placeholder="Event Logger"
            class="{fieldClass} {nameError
              ? 'border-danger focus:border-danger focus:ring-danger/25'
              : ''}"
          />
          {#if nameError}
            <p class="mt-1.5 text-xs text-danger">{nameError}</p>
          {/if}
        </label>

        <p class="text-xs leading-relaxed text-ink-3">
          Tick the fields to include. Unticked fields are left out, so tasks made from this
          template keep their normal defaults for them.
        </p>

        <div class="flex flex-col gap-2.5">
          {#each TEMPLATE_FIELDS as field (field.key)}
            {@const on = selected.has(field.key)}
            <div class="rounded border border-line-soft bg-field/30 p-2.5">
              <label class="flex cursor-pointer items-center gap-2.5">
                <input
                  type="checkbox"
                  checked={on}
                  onchange={() => toggle(field.key)}
                  class="h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
                />
                <span class="text-sm {on ? 'text-ink' : 'text-ink-3'}">{field.label}</span>
              </label>

              {#if on}
                <div class="mt-2 pl-[26px]">
                  {#if field.kind === 'bool'}
                    <label class="flex cursor-pointer items-center gap-2.5">
                      <input
                        type="checkbox"
                        checked={values[field.key] as boolean}
                        onchange={(e) =>
                          (values = {
                            ...values,
                            [field.key]: (e.currentTarget as HTMLInputElement).checked
                          })}
                        class="h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
                      />
                      <span class="text-sm text-ink-2"
                        >{values[field.key] ? 'On' : 'Off'} for tasks from this template</span
                      >
                    </label>
                  {:else if field.kind === 'status'}
                    <StatusPicker
                      value={values.status as Status}
                      onPick={(s) => (values = { ...values, status: s })}
                    />
                  {:else if field.kind === 'textarea'}
                    <textarea
                      rows="3"
                      value={values[field.key] as string}
                      placeholder={field.placeholder ?? ''}
                      oninput={(e) =>
                        (values = {
                          ...values,
                          [field.key]: (e.currentTarget as HTMLTextAreaElement).value
                        })}
                      class="{fieldClass} resize-y leading-relaxed"
                    ></textarea>
                  {:else}
                    <input
                      value={values[field.key] as string}
                      placeholder={field.placeholder ?? ''}
                      oninput={(e) =>
                        (values = {
                          ...values,
                          [field.key]: (e.currentTarget as HTMLInputElement).value
                        })}
                      class="{fieldClass} {field.kind === 'url' ? 'font-mono text-xs' : ''}"
                    />
                  {/if}
                  {#if field.hint}
                    <p class="mt-1.5 text-xs leading-relaxed text-ink-3">{field.hint}</p>
                  {/if}
                </div>
              {/if}
            </div>
          {/each}
        </div>
      </div>

      <div class="flex flex-none items-center justify-end gap-2 border-t border-line-soft px-5 py-3">
        <button
          type="button"
          onclick={onClose}
          class="rounded px-3 py-1.5 text-sm text-ink-2 transition-colors hover:bg-white/5 hover:text-ink"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={!name.trim() || saving}
          class="btn-primary rounded bg-accent px-4 py-1.5 text-sm font-medium text-accent-ink shadow-sm transition-colors hover:bg-accent-hi disabled:cursor-not-allowed disabled:opacity-40"
        >
          Save template
        </button>
      </div>
    </form>
  </div>
{/if}
