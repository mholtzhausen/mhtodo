<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg, type Status } from '../lib/api'
  import { focusOnOpen } from '../lib/focusFirstField'
  import { openExternalUrl } from '../lib/openExternal'
  import { applyTemplate, type TaskTemplate } from '../lib/templates'
  import StatusPicker from './StatusPicker.svelte'
  import TemplatePicker from './TemplatePicker.svelte'
  import SaveAsTemplateDialog from './SaveAsTemplateDialog.svelte'

  let {
    open,
    initialStatus = 'pending',
    parentId = '',
    defaultCwd = '',
    defaultHumanOnly = false,
    defaultIncludeInReport = true,
    defaultSlackThread = '',
    /** Open with the template picker already showing and its filter focused. */
    openWithTemplatePicker = false,
    onClose,
    onError,
    onNotify
  }: {
    open: boolean
    initialStatus?: Status
    parentId?: string
    defaultCwd?: string
    defaultHumanOnly?: boolean
    defaultIncludeInReport?: boolean
    defaultSlackThread?: string
    openWithTemplatePicker?: boolean
    onClose: () => void
    onError?: (msg: string) => void
    onNotify?: (msg: string) => void
  } = $props()

  let title = $state('')
  let description = $state('')
  let status = $state<Status>('pending')
  let cwd = $state('')
  let slackThread = $state('')
  let humanOnly = $state(false)
  let includeInReport = $state(true)

  let pickerOpen = $state(false)
  let saveAsOpen = $state(false)
  let titleEl = $state<HTMLInputElement | null>(null)
  /** Prefix contributed by the last applied template, so switching swaps it. */
  let appliedPrefix = $state('')
  let appliedTemplateName = $state('')

  let submitting = $state(false)
  /** True while the dialog is open and form fields have been seeded from props. */
  let dialogInitialized = false

  $effect(() => {
    if (!open) {
      dialogInitialized = false
      pickerOpen = false
      saveAsOpen = false
      return
    }
    if (dialogInitialized) return
    status = initialStatus
    cwd = defaultCwd
    slackThread = defaultSlackThread
    humanOnly = defaultHumanOnly
    includeInReport = defaultIncludeInReport
    dialogInitialized = true
    pickerOpen = openWithTemplatePicker
  })

  function onTemplatePicked(t: TaskTemplate) {
    const { values, titlePrefix } = applyTemplate(
      t,
      { title, description, status, cwd, slackThread, humanOnly, includeInReport },
      appliedPrefix
    )
    title = values.title
    description = values.description
    status = values.status
    cwd = values.cwd
    slackThread = values.slackThread
    humanOnly = values.humanOnly
    includeInReport = values.includeInReport
    appliedPrefix = titlePrefix
    appliedTemplateName = t.name
    pickerOpen = false

    // Focus the title with the caret after the prefix so the user types the
    // rest of the title straight away.
    requestAnimationFrame(() => {
      titleEl?.focus()
      const at = Math.min(titlePrefix.length, title.length)
      titleEl?.setSelectionRange(at, at)
    })
  }

  async function pickCwd() {
    try {
      const path = await api.pickDirectory(cwd.trim())
      if (path) cwd = path
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  async function submit(e: Event) {
    e.preventDefault()
    if (!title.trim() || submitting) return
    submitting = true
    try {
      await api.create({
        title,
        description,
        status,
        parentId: parentId || undefined,
        cwd: cwd.trim() || undefined,
        humanOnly,
        includeInReport,
        slackThread: slackThread.trim() || undefined
      })
      resetForm()
      onClose()
    } catch (err) {
      onError?.(errMsg(err))
    } finally {
      submitting = false
    }
  }

  function resetForm() {
    title = ''
    description = ''
    status = 'pending'
    cwd = ''
    slackThread = ''
    humanOnly = false
    includeInReport = true
    appliedPrefix = ''
    appliedTemplateName = ''
    pickerOpen = false
    saveAsOpen = false
  }

  function resetAndClose() {
    resetForm()
    onClose()
  }
</script>

{#if open}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-4"
  >
    <form
      use:focusOnOpen
      in:fly={{ y: 8, duration: 80 }}
      onsubmit={submit}
      class="flex max-h-[92vh] w-full max-w-md flex-col rounded-panel border border-line bg-col shadow-md"
    >
      <div class="relative flex flex-none items-center gap-1 border-b border-line-soft px-5 py-3.5">
        <h2 class="flex-1 text-base font-semibold text-ink">
          {parentId ? 'New sub-task' : 'New task'}
          {#if appliedTemplateName}
            <span class="ml-1.5 text-xs font-normal text-ink-3">from {appliedTemplateName}</span>
          {/if}
        </h2>

        <button
          type="button"
          onclick={() => (saveAsOpen = true)}
          title="Save these fields as a template"
          aria-label="Save as template"
          class="rounded-control p-1.5 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
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
            <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
            <path d="M17 21v-8H7v8" />
            <path d="M7 3v5h8" />
          </svg>
        </button>

        <!-- The toggle lives inside the picker's marker element so the picker's
             outside-click handler does not treat the very click that opened it
             as a click-away and close it again. -->
        <div class="relative" data-template-picker>
          <button
            type="button"
            onclick={() => (pickerOpen = !pickerOpen)}
            title="Apply a template"
            aria-label="Apply a template"
            aria-expanded={pickerOpen}
            class="rounded-control p-1.5 leading-none transition-colors hover:bg-white/5 hover:text-ink
              {pickerOpen ? 'bg-white/5 text-accent' : 'text-ink-3'}"
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
              <rect x="3" y="3" width="18" height="18" rx="2" />
              <path d="M3 9h18" />
              <path d="M9 21V9" />
            </svg>
          </button>

          <TemplatePicker
            open={pickerOpen}
            onPick={onTemplatePicked}
            onClose={() => (pickerOpen = false)}
            {onError}
          />
        </div>

        <button
          type="button"
          onclick={resetAndClose}
          title="Close (esc)"
          class="rounded-control p-1.5 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
        >
          ✕
        </button>
      </div>

      <div class="flex min-h-0 flex-col gap-gap-lg overflow-y-auto p-5">
        <label class="block">
          <span class="micro mb-1.5">Title <em class="not-italic text-danger">*</em></span>
          <input
            data-focus-primary
            bind:this={titleEl}
            bind:value={title}
            onkeydown={(e) => e.key === 'Enter' && title.trim() && (e.currentTarget as HTMLInputElement).form?.requestSubmit()}
            placeholder="What needs doing?"
            class="w-full rounded-control border border-line-soft bg-field px-3 py-2 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
          />
        </label>

        <label class="block">
          <span class="micro mb-1.5">Description</span>
          <textarea
            bind:value={description}
            rows="3"
            placeholder="Optional notes…"
            class="w-full resize-y rounded-control border border-line-soft bg-field px-3 py-2 text-sm leading-relaxed text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
          ></textarea>
        </label>

        <div class="block">
          <span class="micro mb-1.5">Working directory</span>
          <div class="flex gap-2">
            <input
              bind:value={cwd}
              placeholder="Optional project path…"
              class="min-w-0 flex-1 rounded-control border border-line-soft bg-field px-3 py-2 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
            />
            <button
              type="button"
              onclick={pickCwd}
              title="Pick folder"
              class="flex-none rounded-control border border-line-soft bg-field px-2.5 py-2 text-ink-2 transition-colors hover:bg-card-hi hover:text-ink"
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
                <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z" />
              </svg>
            </button>
          </div>
        </div>

        <label class="block">
          <span class="micro mb-1.5">Slack thread</span>
          <input
            bind:value={slackThread}
            placeholder="https://… (optional Slack thread link)"
            class="w-full rounded-control border border-line-soft bg-field px-3 py-2 font-mono text-xs text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
          />
          {#if slackThread.trim()}
            <p class="mt-1.5 text-xs leading-relaxed text-ink-3">
              Linked Slack thread:
              <a
                href={slackThread.trim()}
                target="_blank"
                rel="noopener noreferrer"
                class="text-accent hover:underline"
                onclick={(e) => {
                  e.preventDefault()
                  void openExternalUrl(slackThread.trim())
                }}>{slackThread.trim()}</a
              >
            </p>
          {/if}
        </label>

        <label class="flex cursor-pointer items-center gap-gap-md">
          <input
            type="checkbox"
            bind:checked={humanOnly}
            class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
          />
          <span class="text-sm text-ink-2">Human only <span class="text-ink-3">(agents skip this task)</span></span>
        </label>

        <label class="flex cursor-pointer items-center gap-gap-md">
          <input
            type="checkbox"
            bind:checked={includeInReport}
            class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
          />
          <span class="text-sm text-ink-2">Include in Slack report <span class="text-ink-3">(board summary copy)</span></span>
        </label>

        <div>
          <span class="micro mb-1.5">Status</span>
          <StatusPicker value={status} onPick={(s) => (status = s)} />
        </div>
      </div>

      <div class="flex flex-none items-center justify-end gap-2 border-t border-line-soft px-5 py-3">
        <button
          type="button"
          onclick={resetAndClose}
          class="rounded-control px-3 py-1.5 text-sm text-ink-2 transition-colors hover:bg-white/5 hover:text-ink"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={!title.trim() || submitting}
          class="btn-primary rounded-control bg-accent px-4 py-1.5 text-sm font-medium text-accent-ink shadow-sm transition-colors hover:bg-accent-hi disabled:cursor-not-allowed disabled:opacity-40"
        >
          Create {parentId ? 'sub-task' : 'task'}
        </button>
      </div>
    </form>
  </div>

  <SaveAsTemplateDialog
    open={saveAsOpen}
    source={{ title, description, status, cwd, slackThread, humanOnly, includeInReport }}
    onClose={() => (saveAsOpen = false)}
    onSaved={(n) => onNotify?.(`Template “${n}” saved`)}
    {onError}
  />
{/if}
