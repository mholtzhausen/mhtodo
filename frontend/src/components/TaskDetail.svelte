<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg, type Activity, type Status, type Task } from '../lib/api'
  import { claudeIconVisible } from '../lib/claudeIntegration'
  import { absShort, relTime, shortId, STATUS_LABELS } from '../lib/format'
  import StatusPicker from './StatusPicker.svelte'
  import ProgressControl from './ProgressControl.svelte'
  import Markdown from './Markdown.svelte'

  type DetailMode = 'pinned' | 'floating' | 'modal'

  let {
    task,
    parentTitle = null,
    mode,
    width = 420,
    resizing = false,
    onResizeStart,
    onClose,
    onError,
    onDelete,
    onSetMode,
    onAddSubtask,
    onSelectParent
  }: {
    task: any
    parentTitle?: string | null
    mode: DetailMode
    width?: number
    resizing?: boolean
    onResizeStart?: (e: PointerEvent) => void
    onClose: () => void
    onError: (msg: string) => void
    onDelete: (task: any) => void
    onSetMode: (mode: DetailMode) => void
    onAddSubtask: (parentId: string) => void
    onSelectParent?: (parentId: string) => void
  } = $props()

  const modeOptions: { value: DetailMode; label: string; title: string }[] = [
    { value: 'pinned', label: 'Pin', title: 'Pin detail to the right' },
    { value: 'floating', label: 'Float', title: 'Floating panel on the right' },
    { value: 'modal', label: 'Modal', title: 'Full modal overlay' }
  ]

  type ModalSection = 'task' | 'subtasks' | 'activity'
  let activeSection = $state<ModalSection>('task')

  const modalSections = $derived.by(() => {
    const sections: { id: ModalSection; label: string }[] = [
      { id: 'task', label: 'Task' }
    ]
    if (!task.parent_id) {
      sections.push({ id: 'subtasks', label: 'Sub-tasks' })
    }
    sections.push({ id: 'activity', label: 'Activity' })
    return sections
  })

  const statusDot: Record<string, string> = {
    pending: 'bg-st-pending',
    wip: 'bg-st-wip',
    waiting: 'bg-st-waiting',
    review: 'bg-st-review',
    done: 'bg-st-done'
  }

  let title = $state(task.title)
  let description = $state(task.description)
  let progress = $state(task.progress)
  let cwd = $state(task.cwd ?? '')
  let slackThread = $state(task.slack_thread ?? '')
  let humanOnly = $state(!!task.human_only)
  let includeInReport = $state(task.include_in_report !== false)

  let editingDesc = $state(false)
  let descEl = $state<HTMLTextAreaElement | null>(null)

  let activities = $state<Activity[]>([])
  let subtasks = $state<Task[]>([])
  let actText = $state('')
  let commentText = $state('')
  let posting = $state(false)
  let copiedId = $state(false)
  let copyTimer: ReturnType<typeof setTimeout> | undefined
  let herdrActive = $state(false)
  let claudeActive = $state(false)
  let herdrOpening = $state(false)
  let guiSettings = $state<Awaited<ReturnType<typeof api.getSettings>> | null>(null)

  const showClaude = $derived(
    guiSettings ? claudeIconVisible({ ...task, cwd }, guiSettings) : false
  )

  const canOpenClaude = $derived(
    !!guiSettings &&
      showClaude &&
      guiSettings.herdr.enabled &&
      guiSettings.claude.enabled &&
      herdrActive &&
      claudeActive
  )

  async function refreshHerdrStatus() {
    herdrActive = false
    claudeActive = false
    guiSettings = null
    if (humanOnly || task.status === 'done') return
    try {
      const settings = await api.getSettings()
      guiSettings = settings
      if (!claudeIconVisible({ ...task, cwd }, settings)) return
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

  async function openHerdrTicket() {
    if (herdrOpening || !canOpenClaude) return
    herdrOpening = true
    try {
      await api.openHerdrTicket(task.id)
    } catch (e) {
      onError(errMsg(e))
    } finally {
      herdrOpening = false
    }
  }

  async function copyShortId() {
    const code = shortId(task.id)
    try {
      await navigator.clipboard.writeText(code)
      copiedId = true
      clearTimeout(copyTimer)
      copyTimer = setTimeout(() => {
        copiedId = false
      }, 1500)
    } catch {
      onError('Could not copy to clipboard')
    }
  }

  async function loadSubtasks() {
    if (task.parent_id) {
      subtasks = []
      return
    }
    try {
      const all = await api.list({ includeDone: true, includeHumanOnly: true })
      subtasks = all.filter((t) => t.parent_id === task.id)
    } catch (e) {
      onError(errMsg(e))
    }
  }

  async function loadActivity() {
    try {
      activities = await api.listActivity({ taskIds: [task.id] })
    } catch (e) {
      onError(errMsg(e))
    }
  }

  $effect(() => {
    void task.id
    editingDesc = false
    copiedId = false
    activeSection = 'task'
    loadActivity()
    loadSubtasks()
    void refreshHerdrStatus()
  })

  $effect(() => {
    if (task.parent_id && activeSection === 'subtasks') {
      activeSection = 'task'
    }
  })

  $effect(() => {
    void humanOnly
    void cwd
    void task.status
    void refreshHerdrStatus()
  })

  // Keep local fields in sync with live task updates; don't clobber an in-progress edit.
  $effect(() => {
    title = task.title
    progress = task.progress
    cwd = task.cwd ?? ''
    slackThread = task.slack_thread ?? ''
    humanOnly = !!task.human_only
    includeInReport = task.include_in_report !== false
    if (!editingDesc) {
      description = task.description ?? ''
    }
  })

  /** Grow textarea with content up to 500px, then scroll. */
  function fitTextarea(el: HTMLTextAreaElement | null) {
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, 500)}px`
  }

  function startEditDesc() {
    editingDesc = true
    queueMicrotask(() => {
      fitTextarea(descEl)
      descEl?.focus()
    })
  }

  async function saveTitle() {
    const v = title.trim()
    if (!v || v === task.title) return
    try {
      await api.update(task.id, { title: v })
    } catch (e) {
      onError(errMsg(e))
    }
  }

  async function saveDescription() {
    editingDesc = false
    if (description === task.description) return
    try {
      await api.update(task.id, { description })
    } catch (e) {
      onError(errMsg(e))
    }
  }

  async function setStatus(s: Status) {
    if (s === task.status) return
    try {
      await api.setStatus(task.id, s)
    } catch (e) {
      onError(errMsg(e))
    }
  }

  async function saveProgress(p: number) {
    p = Math.max(0, Math.min(100, Number(p) || 0))
    if (p === task.progress) return
    try {
      await api.update(task.id, { progress: p })
    } catch (e) {
      onError(errMsg(e))
    }
  }

  async function pickCwd() {
    try {
      const path = await api.pickDirectory(cwd.trim())
      if (path && path !== cwd) {
        cwd = path
        await api.update(task.id, { cwd: path })
      }
    } catch (e) {
      onError(errMsg(e))
    }
  }

  async function saveCwd() {
    const v = cwd.trim()
    if (v === (task.cwd ?? '')) return
    try {
      await api.update(task.id, { cwd: v })
    } catch (e) {
      onError(errMsg(e))
    }
  }

  async function saveSlackThread() {
    const v = slackThread.trim()
    if (v === (task.slack_thread ?? '')) return
    try {
      await api.update(task.id, { slackThread: v })
    } catch (e) {
      onError(errMsg(e))
    }
  }

  async function toggleHumanOnly() {
    if (humanOnly === !!task.human_only) return
    try {
      await api.update(task.id, { humanOnly })
    } catch (e) {
      humanOnly = !!task.human_only
      onError(errMsg(e))
    }
  }

  async function toggleIncludeInReport() {
    const taskIncluded = task.include_in_report !== false
    if (includeInReport === taskIncluded) return
    try {
      await api.update(task.id, { includeInReport })
    } catch (e) {
      includeInReport = taskIncluded
      onError(errMsg(e))
    }
  }

  async function unarchive() {
    try {
      const t = await api.unarchive(task.id)
      progress = t.progress
    } catch (e) {
      onError(errMsg(e))
    }
  }

  async function postActivity(e: Event) {
    e.preventDefault()
    if (posting || (!actText.trim() && !commentText.trim())) return
    posting = true
    try {
      await api.addActivity(task.id, { activity: actText, comment: commentText })
      actText = ''
      commentText = ''
      await loadActivity()
    } catch (err) {
      onError(errMsg(err))
    } finally {
      posting = false
    }
  }

  const resizable = $derived(mode === 'pinned' || mode === 'floating')

  const shellClass = $derived(
    mode === 'pinned'
      ? 'relative flex h-full flex-none flex-col border-l border-line bg-canvas'
      : mode === 'floating'
        ? 'fixed inset-y-0 right-0 z-40 flex flex-none flex-col border-l border-line bg-canvas shadow-2xl'
        : 'flex h-[min(85vh,860px)] w-full max-w-4xl flex-col overflow-hidden rounded-lg border border-line bg-canvas shadow-2xl'
  )

  const enterTransition = $derived(
    mode === 'modal' ? { y: 12, duration: 150 } : { x: 40, duration: 150 }
  )
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<aside
  in:fly={enterTransition}
  class={shellClass}
  style:width={resizable ? `${width}px` : undefined}
  style:max-width={resizable ? '100%' : undefined}
  onclick={(e) => e.stopPropagation()}
>
  {#if resizable && onResizeStart}
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize detail pane"
      class="absolute inset-y-0 left-0 z-30 w-1.5 -translate-x-1/2 cursor-col-resize touch-none
        {resizing ? 'bg-accent/50' : 'bg-transparent hover:bg-accent/30'}"
      onpointerdown={onResizeStart}
    ></div>
  {/if}
  <div class="flex flex-none items-center justify-end gap-2 border-b border-line-soft bg-chrome px-4 py-2.5">
    <div
      class="mr-auto flex rounded border border-line-soft p-0.5"
      role="group"
      aria-label="Detail display mode"
    >
      {#each modeOptions as opt (opt.value)}
        <button
          type="button"
          onclick={() => onSetMode(opt.value)}
          title={opt.title}
          aria-pressed={mode === opt.value}
          class="rounded px-2 py-0.5 text-xs font-medium transition-colors
            {mode === opt.value ? 'bg-accent/20 text-accent-hi' : 'text-ink-3 hover:bg-white/5 hover:text-ink'}"
        >
          {opt.label}
        </button>
      {/each}
    </div>
    <div class="flex items-center gap-1">
      <div
        class="flex items-center gap-0.5 rounded border border-line-soft bg-field/40 pl-1.5 font-mono text-[10px] leading-none text-ink-3"
        title={task.id}
      >
        <span class="py-1">{shortId(task.id)}</span>
        <button
          type="button"
          onclick={copyShortId}
          title={copiedId ? 'Copied' : 'Copy short ID'}
          aria-label={copiedId ? 'Copied short ID' : 'Copy short ID'}
          class="rounded p-1 text-ink-3 transition-colors hover:bg-white/5 hover:text-ink-2"
        >
          {#if copiedId}
            <svg
              class="h-3 w-3"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <path d="M20 6 9 17l-5-5" />
            </svg>
          {:else}
            <svg
              class="h-3 w-3"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
            </svg>
          {/if}
        </button>
      </div>
      {#if canOpenClaude}
        <button
          type="button"
          onclick={openHerdrTicket}
          disabled={herdrOpening}
          title={claudeActive ? 'Open in Herdr with Claude' : 'Open in Herdr'}
          aria-label={claudeActive ? 'Open in Herdr with Claude' : 'Open in Herdr'}
          class="rounded border border-line-soft bg-field/40 p-1 transition-colors hover:bg-white/5 disabled:opacity-40
            {claudeActive
            ? 'text-[#d97757] hover:text-[#e88a6a]'
            : 'text-ink-3 hover:text-accent-hi'}"
        >
          {#if claudeActive}
            <svg
              class="h-3 w-3"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 100 100"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                d="m19.6 66.5 19.7-11 .3-1-.3-.5h-1l-3.3-.2-11.2-.3L14 53l-9.5-.5-2.4-.5L0 49l.2-1.5 2-1.3 2.9.2 6.3.5 9.5.6 6.9.4L38 49.1h1.6l.2-.7-.5-.4-.4-.4L29 41l-10.6-7-5.6-4.1-3-2-1.5-2-.6-4.2 2.7-3 3.7.3.9.2 3.7 2.9 8 6.1L37 36l1.5 1.2.6-.4.1-.3-.7-1.1L33 25l-6-10.4-2.7-4.3-.7-2.6c-.3-1-.4-2-.4-3l3-4.2L28 0l4.2.6L33.8 2l2.6 6 4.1 9.3L47 29.9l2 3.8 1 3.4.3 1h.7v-.5l.5-7.2 1-8.7 1-11.2.3-3.2 1.6-3.8 3-2L61 2.6l2 2.9-.3 1.8-1.1 7.7L59 27.1l-1.5 8.2h.9l1-1.1 4.1-5.4 6.9-8.6 3-3.5L77 13l2.3-1.8h4.3l3.1 4.7-1.4 4.9-4.4 5.6-3.7 4.7-5.3 7.1-3.2 5.7.3.4h.7l12-2.6 6.4-1.1 7.6-1.3 3.5 1.6.4 1.6-1.4 3.4-8.2 2-9.6 2-14.3 3.3-.2.1.2.3 6.4.6 2.8.2h6.8l12.6 1 3.3 2 1.9 2.7-.3 2-5.1 2.6-6.8-1.6-16-3.8-5.4-1.3h-.8v.4l4.6 4.5 8.3 7.5L89 80.1l.5 2.4-1.3 2-1.4-.2-9.2-7-3.6-3-8-6.8h-.5v.7l1.8 2.7 9.8 14.7.5 4.5-.7 1.4-2.6 1-2.7-.6-5.8-8-6-9-4.7-8.2-.5.4-2.9 30.2-1.3 1.5-3 1.2-2.5-2-1.4-3 1.4-6.2 1.6-8 1.3-6.4 1.2-7.9.7-2.6v-.2H49L43 72l-9 12.3-7.2 7.6-1.7.7-3-1.5.3-2.8L24 86l10-12.8 6-7.9 4-4.6-.1-.5h-.3L17.2 77.4l-4.7.6-2-2 .2-3 1-1 8-5.5Z"
              />
            </svg>
          {:else}
            <svg
              class="h-3 w-3"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <rect x="2" y="3" width="20" height="14" rx="2" />
              <path d="M8 21h8" />
              <path d="M12 17v4" />
              <path d="m7 8 3 3-3 3" />
              <path d="M13 14h4" />
            </svg>
          {/if}
        </button>
      {/if}
    </div>
    <button
      type="button"
      onclick={() => onDelete(task)}
      title="Delete task (del)"
      aria-label="Delete task"
      class="rounded p-1.5 text-ink-3 transition-colors hover:bg-danger/15 hover:text-danger"
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
    <button
      onclick={onClose}
      title="Close (esc)"
      class="rounded p-1.5 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
    >
      ✕
    </button>
  </div>

  {#snippet overviewSection()}
    {#if task.parent_id}
      <div>
        <span class="micro mb-1.5">Parent task</span>
        {#if mode === 'modal'}
          {#if onSelectParent}
            <button
              type="button"
              onclick={() => onSelectParent?.(task.parent_id)}
              class="block w-full truncate text-left text-sm font-medium text-accent-hi hover:underline"
              title={parentTitle ?? task.parent_id}
            >
              {parentTitle ?? 'Open parent'}
            </button>
          {:else}
            <p class="truncate text-sm font-medium text-ink-2" title={parentTitle ?? task.parent_id}>
              {parentTitle ?? task.parent_id}
            </p>
          {/if}
        {:else}
          <button
            type="button"
            onclick={() => onSelectParent?.(task.parent_id)}
            class="block w-full truncate text-left text-sm font-medium text-accent-hi hover:underline"
            title={parentTitle ?? task.parent_id}
          >
            {parentTitle ?? 'Open parent'}
          </button>
        {/if}
      </div>
    {/if}

    <label class="block">
      <span class="micro mb-1.5">Title</span>
      <input
        bind:value={title}
        onblur={saveTitle}
        onkeydown={(e) => e.key === 'Enter' && (e.currentTarget as HTMLInputElement).blur()}
        class="w-full rounded border border-line-soft bg-field px-3 py-2 text-sm font-medium text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
      />
    </label>

    <div>
      <span class="micro mb-1.5">Status</span>
      <StatusPicker value={task.status} onPick={(s) => setStatus(s)} />
    </div>

    <div>
      <span class="micro mb-1.5">Progress</span>
      <ProgressControl value={progress} onCommit={(p) => saveProgress(p)} />
    </div>
  {/snippet}

  {#snippet contentSection()}
    <div class="block">
      <span class="micro mb-1.5">Description</span>
      {#if editingDesc}
        <textarea
          bind:this={descEl}
          bind:value={description}
          oninput={() => fitTextarea(descEl)}
          onblur={saveDescription}
          onkeydown={(e) => {
            if (e.key === 'Escape') {
              e.stopPropagation()
              description = task.description ?? ''
              editingDesc = false
            }
          }}
          rows="3"
          placeholder="Notes, links, context… (markdown)"
          class="ta-autogrow w-full rounded border border-line-soft bg-field px-3 py-2 text-sm leading-relaxed text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
        ></textarea>
      {:else}
        <div
          role="button"
          tabindex="0"
          class="md-scroll w-full cursor-text rounded border border-line-soft bg-field px-3 py-2 text-left text-sm leading-relaxed text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] transition-colors hover:border-line hover:bg-card-hi/40"
          title="Click to edit"
          onclick={(e) => {
            if ((e.target as HTMLElement).closest('a')) return
            startEditDesc()
          }}
          onkeydown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              startEditDesc()
            }
          }}
        >
          <Markdown source={description} empty="Notes, links, context…" class="text-ink-2" />
        </div>
      {/if}
    </div>

    {#if task.feedback}
      <div>
        <span class="micro mb-1.5">Feedback</span>
        <p class="mb-1.5 text-[11px] leading-snug text-ink-3">
          Agent/CLI-authored (<code class="font-mono text-ink-3/90">mhtodo edit --feedback</code>); read-only here.
        </p>
        <div
          class="md-scroll rounded border border-accent/25 bg-accent/10 px-3 py-2 text-sm leading-relaxed text-ink-2"
        >
          <Markdown source={task.feedback} />
        </div>
      </div>
    {/if}
  {/snippet}

  {#snippet contextSection()}
    <div class="block">
      <span class="micro mb-1.5">Working directory</span>
      <div class="flex gap-2">
        <input
          bind:value={cwd}
          onblur={saveCwd}
          placeholder="Optional project path…"
          class="min-w-0 flex-1 rounded border border-line-soft bg-field px-3 py-2 font-mono text-xs text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
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
        onblur={saveSlackThread}
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
        onchange={toggleHumanOnly}
        class="h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
      />
      <span class="text-sm text-ink-2">Human only <span class="text-ink-3">(agents skip this task)</span></span>
    </label>

    <label class="flex cursor-pointer items-center gap-2.5">
      <input
        type="checkbox"
        bind:checked={includeInReport}
        onchange={toggleIncludeInReport}
        class="h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
      />
      <span class="text-sm text-ink-2">Include in Slack report <span class="text-ink-3">(board summary copy)</span></span>
    </label>
  {/snippet}

  {#snippet subtasksSection()}
    <div class="flex flex-col gap-3">
      <button
        type="button"
        onclick={() => onAddSubtask(task.id)}
        class="self-start rounded border border-line-soft bg-field px-3 py-2 text-sm text-ink-2 transition-colors hover:bg-card-hi hover:text-ink"
      >
        + Add sub-task
      </button>
    <ul class="space-y-1">
      {#each subtasks as st (st.id)}
        <li>
          <button
            type="button"
            onclick={() => onSelectParent?.(st.id)}
            class="flex w-full items-center gap-2.5 rounded border border-line-soft bg-field/50 px-3 py-2 text-left transition-colors hover:bg-card-hi"
          >
            <span
              class="h-2 w-2 flex-none rounded-full {statusDot[st.status] ?? statusDot.pending}"
              title={STATUS_LABELS[st.status] ?? st.status}
            ></span>
            <span class="min-w-0 flex-1 truncate text-sm text-ink">{st.title}</span>
            <span class="font-mono text-[11px] text-ink-3">{st.progress}%</span>
          </button>
        </li>
      {:else}
        <li class="text-sm text-ink-3">No sub-tasks yet.</li>
      {/each}
    </ul>
    </div>
  {/snippet}

  {#snippet taskSection()}
    {@render overviewSection()}
    {@render contentSection()}
    {@render contextSection()}
    <div class="border-t border-line-soft pt-4">
      {@render infoSection()}
    </div>
  {/snippet}

  {#snippet activitySection()}
    <form onsubmit={postActivity} class="mb-3 space-y-2">
      <input
        bind:value={actText}
        placeholder="Activity summary…"
        class="w-full rounded border border-line-soft bg-field px-3 py-1.5 text-sm text-ink placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
      />
      <textarea
        bind:value={commentText}
        rows="2"
        placeholder="Optional comment… (markdown)"
        class="w-full resize-y rounded border border-line-soft bg-field px-3 py-1.5 text-sm text-ink placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
      ></textarea>
      <button
        type="submit"
        disabled={posting || (!actText.trim() && !commentText.trim())}
        class="rounded bg-accent px-3 py-1.5 text-xs font-medium text-accent-ink disabled:opacity-40"
      >
        Post
      </button>
    </form>
    <ul class="space-y-2">
      {#each activities as a (a.id)}
        <li class="rounded border border-line-soft bg-field/50 px-2.5 py-2">
          <p class="mb-0.5 text-[10px] text-ink-3">{relTime(a.created_at)}</p>
          {#if a.activity}<p class="text-xs text-ink">{a.activity}</p>{/if}
          {#if a.comment}
            <Markdown source={a.comment} class="mt-0.5 text-xs text-ink-2" />
          {/if}
        </li>
      {:else}
        <li class="text-xs text-ink-3">No activity yet.</li>
      {/each}
    </ul>
  {/snippet}

  {#snippet infoSection()}
    <dl class="space-y-1.5 text-xs text-ink-3">
      <div class="flex justify-between">
        <dt>Created</dt>
        <dd class="font-mono text-[11px] text-ink-2">{absShort(task.created_at)}</dd>
      </div>
      <div class="flex justify-between">
        <dt>Updated</dt>
        <dd class="font-mono text-[11px] text-ink-2">{absShort(task.updated_at)}</dd>
      </div>
      <div class="flex justify-between">
        <dt>Completed</dt>
        <dd class="font-mono text-[11px] text-ink-2">{task.completed_at ? absShort(task.completed_at) : '—'}</dd>
      </div>
      {#if task.archived_at}
        <div class="flex justify-between">
          <dt>Archived</dt>
          <dd class="font-mono text-[11px] text-ink-2">{absShort(task.archived_at)}</dd>
        </div>
      {/if}
    </dl>
  {/snippet}

  {#snippet actionSection()}
    {#if task.archived_at}
      <button
        onclick={unarchive}
        title="Restore to pending (progress resets to 0)"
        class="w-full rounded border border-accent/50 bg-accent/10 px-3 py-2 text-sm font-medium text-accent-hi transition-colors hover:bg-accent/20"
      >
        Unarchive → pending
      </button>
    {/if}
  {/snippet}

  {#if mode === 'modal'}
    <div class="flex min-h-0 flex-1 overflow-hidden">
      <nav
        class="flex w-44 flex-none flex-col gap-0.5 overflow-y-auto border-r border-line-soft p-3"
        aria-label="Task sections"
      >
        {#each modalSections as section (section.id)}
          <button
            type="button"
            onclick={() => {
              activeSection = section.id
              if (section.id === 'subtasks') void loadSubtasks()
            }}
            class="rounded px-3 py-2 text-left text-[13px] font-medium transition-colors
              {activeSection === section.id
              ? 'bg-accent/15 text-ink'
              : 'text-ink-3 hover:bg-white/5 hover:text-ink-2'}"
          >
            {section.label}
            {#if section.id === 'activity' && activities.length > 0}
              <span class="ml-1.5 text-[11px] text-ink-3">({activities.length})</span>
            {:else if section.id === 'subtasks' && subtasks.length > 0}
              <span class="ml-1.5 text-[11px] text-ink-3">({subtasks.length})</span>
            {/if}
          </button>
        {/each}
      </nav>

      <div class="min-h-0 flex-1 overflow-y-auto p-5">
        <div class="flex flex-col gap-5">
          {#if activeSection === 'task'}
            {@render taskSection()}
          {:else if activeSection === 'subtasks'}
            {@render subtasksSection()}
          {:else}
            {@render activitySection()}
          {/if}
        </div>
      </div>
    </div>

    {#if task.archived_at}
      <div class="flex flex-none flex-col gap-2 border-t border-line-soft bg-chrome px-5 py-3">
        {@render actionSection()}
      </div>
    {/if}
  {:else}
    <div class="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-5 py-4">
      {@render overviewSection()}
      {@render contextSection()}
      {@render contentSection()}
      {#if !task.parent_id}
        <div class="border-t border-line-soft pt-3">
          <span class="micro mb-2">Sub-tasks</span>
          {@render subtasksSection()}
        </div>
      {/if}
      <div class="border-t border-line-soft pt-3">
        <span class="micro mb-2">Activity</span>
        {@render activitySection()}
      </div>
      {@render infoSection()}
    </div>

    {#if task.archived_at}
      <div class="flex flex-none flex-col gap-2 border-t border-line-soft bg-chrome px-5 py-3">
        {@render actionSection()}
      </div>
    {/if}
  {/if}
</aside>
