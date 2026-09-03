<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg, type Status } from '../lib/api'
  import { claudeIconVisible } from '../lib/claudeIntegration'
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
  let herdrActive = $state(false)
  let claudeActive = $state(false)
  let guiSettings = $state<Awaited<ReturnType<typeof api.getSettings>> | null>(null)
  /** True while the dialog is open and form fields have been seeded from props. */
  let dialogInitialized = false

  const showClaude = $derived(
    guiSettings
      ? claudeIconVisible({ cwd, human_only: humanOnly, status: 'pending' }, guiSettings)
      : false
  )

  const canShowStartClaude = $derived(
    !!guiSettings &&
      showClaude &&
      guiSettings.herdr.enabled &&
      guiSettings.claude.enabled &&
      herdrActive &&
      claudeActive
  )

  async function refreshIntegrationStatus() {
    herdrActive = false
    claudeActive = false
    guiSettings = null
    if (humanOnly) return
    try {
      const settings = await api.getSettings()
      guiSettings = settings
      if (!claudeIconVisible({ cwd, human_only: humanOnly, status: 'pending' }, settings)) {
        return
      }
      if (settings.herdr.enabled) {
        herdrActive = await api.checkBinary(settings.herdr.binary)
      }
      if (settings.claude.enabled) {
        claudeActive = await api.checkBinary(settings.claude.binary)
      }
    } catch {
      herdrActive = false
      claudeActive = false
    }
  }

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
    humanOnly = defaultHumanOnly
    includeInReport = defaultIncludeInReport
    dialogInitialized = true
    pickerOpen = openWithTemplatePicker
    refreshIntegrationStatus()
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

  $effect(() => {
    if (!open || !dialogInitialized) return
    void cwd
    void humanOnly
    refreshIntegrationStatus()
  })

  const canStartWithClaude = $derived(
    !!title.trim() && !submitting && canShowStartClaude
  )

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

  async function submitAndStart(e: Event) {
    e.preventDefault()
    if (!canStartWithClaude) return
    submitting = true
    try {
      const task = await api.create({
        title,
        description,
        status,
        parentId: parentId || undefined,
        cwd: cwd.trim() || undefined,
        humanOnly,
        includeInReport,
        slackThread: slackThread.trim() || undefined
      })
      await api.openHerdrTicket(task.id)
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
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-4 backdrop-blur-[2px]"
  >
    <form
      in:fly={{ y: 12, duration: 150 }}
      onclick={(e) => e.stopPropagation()}
      onsubmit={submit}
      class="flex max-h-[92vh] w-full max-w-md flex-col rounded-lg border border-line bg-col shadow-2xl"
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
          class="rounded p-1.5 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
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
            class="rounded p-1.5 leading-none transition-colors hover:bg-white/5 hover:text-ink
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
          class="rounded p-1.5 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
        >
          ✕
        </button>
      </div>

      <div class="flex min-h-0 flex-col gap-3.5 overflow-y-auto p-5">
        <label class="block">
          <span class="micro mb-1.5">Title <em class="not-italic text-danger">*</em></span>
          <input
            autofocus
            bind:this={titleEl}
            bind:value={title}
            onkeydown={(e) => e.key === 'Enter' && title.trim() && (e.currentTarget as HTMLInputElement).form?.requestSubmit()}
            placeholder="What needs doing?"
            class="w-full rounded border border-line-soft bg-field px-3 py-2 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
          />
        </label>

        <label class="block">
          <span class="micro mb-1.5">Description</span>
          <textarea
            bind:value={description}
            rows="3"
            placeholder="Optional notes…"
            class="w-full resize-y rounded border border-line-soft bg-field px-3 py-2 text-sm leading-relaxed text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
          ></textarea>
        </label>

        <div class="block">
          <span class="micro mb-1.5">Working directory</span>
          <div class="flex gap-2">
            <input
              bind:value={cwd}
              placeholder="Optional project path…"
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
            class="w-full rounded border border-line-soft bg-field px-3 py-2 font-mono text-xs text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
          />
          {#if slackThread.trim()}
            <p class="mt-1.5 text-xs leading-relaxed text-ink-3">
              Linked Slack thread:
              <a href={slackThread.trim()} target="_blank" rel="noopener noreferrer" class="text-accent hover:underline">{slackThread.trim()}</a>
            </p>
          {/if}
        </label>

        <label class="flex cursor-pointer items-center gap-2.5">
          <input
            type="checkbox"
            bind:checked={humanOnly}
            class="h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
          />
          <span class="text-sm text-ink-2">Human only <span class="text-ink-3">(agents skip this task)</span></span>
        </label>

        <label class="flex cursor-pointer items-center gap-2.5">
          <input
            type="checkbox"
            bind:checked={includeInReport}
            class="h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
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
          class="rounded px-3 py-1.5 text-sm text-ink-2 transition-colors hover:bg-white/5 hover:text-ink"
        >
          Cancel
        </button>
        {#if canShowStartClaude}
          <button
            type="button"
            onclick={submitAndStart}
            disabled={!canStartWithClaude}
            title="Create task and start Claude in Herdr"
            class="flex items-center gap-1.5 rounded border border-[#d97757]/40 bg-[#d97757]/10 px-3 py-1.5 text-sm font-medium text-[#e88a6a] shadow-sm transition-colors hover:bg-[#d97757]/20 disabled:cursor-not-allowed disabled:opacity-40"
          >
            <svg
              class="h-3.5 w-3.5 shrink-0"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 100 100"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                d="m19.6 66.5 19.7-11 .3-1-.3-.5h-1l-3.3-.2-11.2-.3L14 53l-9.5-.5-2.4-.5L0 49l.2-1.5 2-1.3 2.9.2 6.3.5 9.5.6 6.9.4L38 49.1h1.6l.2-.7-.5-.4-.4-.4L29 41l-10.6-7-5.6-4.1-3-2-1.5-2-.6-4.2 2.7-3 3.7.3.9.2 3.7 2.9 8 6.1L37 36l1.5 1.2.6-.4.1-.3-.7-1.1L33 25l-6-10.4-2.7-4.3-.7-2.6c-.3-1-.4-2-.4-3l3-4.2L28 0l4.2.6L33.8 2l2.6 6 4.1 9.3L47 29.9l2 3.8 1 3.4.3 1h.7v-.5l.5-7.2 1-8.7 1-11.2.3-3.2 1.6-3.8 3-2L61 2.6l2 2.9-.3 1.8-1.1 7.7L59 27.1l-1.5 8.2h.9l1-1.1 4.1-5.4 6.9-8.6 3-3.5L77 13l2.3-1.8h4.3l3.1 4.7-1.4 4.9-4.4 5.6-3.7 4.7-5.3 7.1-3.2 5.7.3.4h.7l12-2.6 6.4-1.1 7.6-1.3 3.5 1.6.4 1.6-1.4 3.4-8.2 2-9.6 2-14.3 3.3-.2.1.2.3 6.4.6 2.8.2h6.8l12.6 1 3.3 2 1.9 2.7-.3 2-5.1 2.6-6.8-1.6-16-3.8-5.4-1.3h-.8v.4l4.6 4.5 8.3 7.5L89 80.1l.5 2.4-1.3 2-1.4-.2-9.2-7-3.6-3-8-6.8h-.5v.7l1.8 2.7 9.8 14.7.5 4.5-.7 1.4-2.6 1-2.7-.6-5.8-8-6-9-4.7-8.2-.5.4-2.9 30.2-1.3 1.5-3 1.2-2.5-2-1.4-3 1.4-6.2 1.6-8 1.3-6.4 1.2-7.9.7-2.6v-.2H49L43 72l-9 12.3-7.2 7.6-1.7.7-3-1.5.3-2.8L24 86l10-12.8 6-7.9 4-4.6-.1-.5h-.3L17.2 77.4l-4.7.6-2-2 .2-3 1-1 8-5.5Z"
              />
            </svg>
            Start task
          </button>
        {/if}
        <button
          type="submit"
          disabled={!title.trim() || submitting}
          class="btn-primary rounded bg-accent px-4 py-1.5 text-sm font-medium text-accent-ink shadow-sm transition-colors hover:bg-accent-hi disabled:cursor-not-allowed disabled:opacity-40"
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
