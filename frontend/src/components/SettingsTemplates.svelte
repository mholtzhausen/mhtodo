<script lang="ts">
  import { onDestroy } from 'svelte'
  import { api, errMsg, type Status } from '../lib/api'
  import {
    TEMPLATE_FIELDS,
    valuesOf,
    type TaskTemplate,
    type TemplateValues
  } from '../lib/templates'
  import StatusPicker from './StatusPicker.svelte'
  import TriStateCheck from './TriStateCheck.svelte'

  let {
    template,
    defaultHumanOnly,
    defaultIncludeInReport,
    onSaved,
    onDeleted,
    onStatus,
    onError
  }: {
    template: TaskTemplate
    /** Resolved values shown as the "default (…)" hint on unset booleans. */
    defaultHumanOnly: boolean
    defaultIncludeInReport: boolean
    onSaved: (t: TaskTemplate) => void
    onDeleted: (id: string) => void
    onStatus: (s: 'idle' | 'saving' | 'dirty') => void
    onError: (msg: string) => void
  } = $props()

  // The parent remounts this component per template ({#key}), so each instance
  // owns exactly one template for its lifetime. Capturing the initial prop
  // value is therefore intended, not a missed derived.
  // svelte-ignore state_referenced_locally
  let name = $state(template.name)
  // svelte-ignore state_referenced_locally
  let values = $state<TemplateValues>(valuesOf(template))
  let confirmDelete = $state(false)
  let nameError = $state('')
  let deleted = false

  let persistTimer: ReturnType<typeof setTimeout> | undefined
  /** Serializes writes so a debounced save and a flush cannot interleave. */
  let inFlight: Promise<void> = Promise.resolve()
  /** Snapshot of what the backend last confirmed, to skip no-op saves. */
  // svelte-ignore state_referenced_locally
  let lastSaved = $state(snapshot(template.name, valuesOf(template)))

  function snapshot(n: string, v: TemplateValues) {
    return JSON.stringify({ n, v })
  }

  // Debounced autosave, matching the rest of the settings dialog.
  $effect(() => {
    const snap = snapshot(name, values)
    if (snap === lastSaved) {
      onStatus('idle')
      return
    }
    onStatus('dirty')
    clearTimeout(persistTimer)
    persistTimer = setTimeout(() => void queueSave(), 400)
    return () => clearTimeout(persistTimer)
  })

  // Losing the component (dialog closed, another template selected) must not
  // drop an edit that is still sitting in the debounce window.
  onDestroy(() => {
    clearTimeout(persistTimer)
    if (!deleted && snapshot(name, values) !== lastSaved) void queueSave()
  })

  function queueSave(): Promise<void> {
    const n = name
    const v = $state.snapshot(values) as TemplateValues
    inFlight = inFlight.then(() => save(n, v))
    return inFlight
  }

  async function save(n: string, v: TemplateValues) {
    if (deleted) return
    const trimmed = n.trim()
    if (!trimmed) {
      // An empty name cannot be persisted; surface it rather than silently
      // dropping the edit, and leave the field marked dirty.
      nameError = 'Name is required'
      onStatus('dirty')
      return
    }
    onStatus('saving')
    try {
      const saved = await api.updateTemplate(template.id, trimmed, v)
      nameError = ''
      // Only settle if nothing changed while the write was in flight.
      if (snapshot(name, values) === snapshot(n, v)) {
        lastSaved = snapshot(n, v)
        onStatus('idle')
      } else {
        onStatus('dirty')
      }
      onSaved(saved)
    } catch (err) {
      const msg = errMsg(err)
      if (msg.toLowerCase().includes('already exists')) nameError = msg
      else onError(msg)
      onStatus('dirty')
    }
  }

  export async function flush() {
    clearTimeout(persistTimer)
    if (snapshot(name, values) !== lastSaved) await queueSave()
    else await inFlight
  }

  async function pickCwd() {
    try {
      const path = await api.pickDirectory((values.cwd ?? '').trim())
      if (path) values = { ...values, cwd: path }
    } catch (err) {
      onError(errMsg(err))
    }
  }

  async function remove() {
    clearTimeout(persistTimer)
    deleted = true // stops the destroy-time flush from recreating pending edits
    try {
      await api.deleteTemplate(template.id)
      onDeleted(template.id)
    } catch (err) {
      deleted = false
      onError(errMsg(err))
    }
  }

  function clearField(key: keyof TemplateValues) {
    values = { ...values, [key]: null }
  }

  /** Enabling a text field starts it as an empty string, which is "set to empty". */
  function enableField(key: keyof TemplateValues, initial: string | Status | boolean) {
    values = { ...values, [key]: initial }
  }

  const fieldClass =
    'w-full rounded border border-line-soft bg-field px-3 py-2 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25'
</script>

<section>
  <div class="mb-4 flex items-start gap-3">
    <div class="min-w-0 flex-1">
      <h3 class="text-sm font-semibold text-ink">{template.name || 'Untitled template'}</h3>
      <p class="mt-1 text-xs leading-relaxed text-ink-3">
        Only the fields you set below are applied when you create a task from this template.
        Everything else keeps its normal default.
      </p>
    </div>
  </div>

  <div class="flex flex-col gap-4">
    <label class="block">
      <span class="micro mb-1.5">Template name <em class="not-italic text-danger">*</em></span>
      <input
        bind:value={name}
        placeholder="Event Logger"
        maxlength="80"
        class="{fieldClass} {nameError ? 'border-danger focus:border-danger focus:ring-danger/25' : ''}"
      />
      {#if nameError}
        <p class="mt-1.5 text-xs text-danger">{nameError}</p>
      {/if}
    </label>

    <div class="h-px bg-line-soft"></div>

    {#each TEMPLATE_FIELDS as field (field.key)}
      {@const set = values[field.key] !== null}
      <div class="block">
        {#if field.kind === 'bool'}
          <TriStateCheck
            value={values[field.key] as boolean | null}
            label={field.label}
            fallback={field.key === 'human_only' ? defaultHumanOnly : defaultIncludeInReport}
            hint={field.hint ?? ''}
            onChange={(v) => (values = { ...values, [field.key]: v })}
          />
        {:else}
          <div class="mb-1.5 flex items-center gap-2">
            <span class="micro">{field.label}</span>
            {#if set}
              <button
                type="button"
                onclick={() => clearField(field.key)}
                title="Remove this field from the template"
                aria-label="Remove {field.label} from the template"
                class="rounded p-0.5 text-ink-3 transition-colors hover:bg-white/10 hover:text-danger"
              >
                <svg
                  class="h-3.5 w-3.5"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <path d="M3 6h18" />
                  <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" />
                  <path d="M10 11v6" />
                  <path d="M14 11v6" />
                </svg>
              </button>
            {:else}
              <span class="text-[11px] text-ink-3">not set</span>
            {/if}
          </div>

          {#if !set}
            <button
              type="button"
              onclick={() =>
                enableField(field.key, field.kind === 'status' ? ('pending' as Status) : '')}
              class="w-full rounded border border-dashed border-line-soft bg-field/30 px-3 py-2 text-left text-sm text-ink-3 transition-colors hover:border-accent/50 hover:text-ink-2"
            >
              + Add {field.label.toLowerCase()} to this template
            </button>
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
                (values = { ...values, [field.key]: (e.currentTarget as HTMLTextAreaElement).value })}
              class="{fieldClass} resize-y leading-relaxed"
            ></textarea>
          {:else if field.kind === 'path'}
            <div class="flex gap-2">
              <input
                value={values.cwd as string}
                placeholder={field.placeholder ?? ''}
                oninput={(e) =>
                  (values = { ...values, cwd: (e.currentTarget as HTMLInputElement).value })}
                class="min-w-0 flex-1 rounded border border-line-soft bg-field px-3 py-2 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
              />
              <button
                type="button"
                onclick={pickCwd}
                title="Pick folder"
                class="flex-none rounded border border-line-soft bg-field px-2.5 py-2 text-ink-2 transition-colors hover:bg-card-hi hover:text-ink"
              >
                <svg
                  class="h-4 w-4"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <path
                    d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"
                  />
                </svg>
              </button>
            </div>
          {:else}
            <input
              value={values[field.key] as string}
              placeholder={field.placeholder ?? ''}
              oninput={(e) =>
                (values = { ...values, [field.key]: (e.currentTarget as HTMLInputElement).value })}
              class="{fieldClass} {field.kind === 'url' ? 'font-mono text-xs' : ''}"
            />
          {/if}

          {#if set && field.hint}
            <p class="mt-1.5 text-xs leading-relaxed text-ink-3">{field.hint}</p>
          {/if}
        {/if}
      </div>
    {/each}

    <div class="h-px bg-line-soft"></div>

    <div class="flex items-center gap-2">
      {#if confirmDelete}
        <span class="text-sm text-ink-2">Delete this template?</span>
        <button
          type="button"
          onclick={remove}
          class="rounded border border-danger/40 bg-danger/10 px-3 py-1.5 text-sm font-medium text-danger transition-colors hover:bg-danger/20"
        >
          Delete
        </button>
        <button
          type="button"
          onclick={() => (confirmDelete = false)}
          class="rounded px-3 py-1.5 text-sm text-ink-2 transition-colors hover:bg-white/5 hover:text-ink"
        >
          Cancel
        </button>
      {:else}
        <button
          type="button"
          onclick={() => (confirmDelete = true)}
          class="rounded border border-line-soft px-3 py-1.5 text-sm text-ink-3 transition-colors hover:border-danger/40 hover:text-danger"
        >
          Delete template
        </button>
      {/if}
    </div>
  </div>
</section>
