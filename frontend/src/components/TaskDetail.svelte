<script lang="ts">
  import { api, errMsg, type Activity, type Status, type Task } from '../lib/api'
  import { absShort, relTime, shortId, STATUS_LABELS } from '../lib/format'
  import { openExternalUrl } from '../lib/openExternal'
  import StatusPicker from './StatusPicker.svelte'
  import ProgressControl from './ProgressControl.svelte'
  import Markdown from './Markdown.svelte'
  import SaveAsTemplateDialog from './SaveAsTemplateDialog.svelte'
  import { focusOnOpen } from '../lib/focusFirstField'

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
    onNotify,
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
    onNotify?: (msg: string) => void
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

  let saveAsOpen = $state(false)

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
    pr: 'bg-st-pr',
    done: 'bg-st-done'
  }

  // Draft fields: seed from prop, then keep local until save / $effect resync.
  // svelte-ignore state_referenced_locally
  let title = $state(task.title)
  // svelte-ignore state_referenced_locally
  let description = $state(task.description)
  // svelte-ignore state_referenced_locally
  let progress = $state(task.progress)
  // svelte-ignore state_referenced_locally
  let cwd = $state(task.cwd ?? '')
  // svelte-ignore state_referenced_locally
  let slackThread = $state(task.slack_thread ?? '')
  // svelte-ignore state_referenced_locally
  let prUrl = $state(task.pr_url ?? '')
  // svelte-ignore state_referenced_locally
  let todoSession = $state(task.todo_session ?? '')
  // svelte-ignore state_referenced_locally
  let humanOnly = $state(!!task.human_only)
  // svelte-ignore state_referenced_locally
  let includeInReport = $state(task.include_in_report !== false)

  let editingDesc = $state(false)
  let descEl = $state<HTMLTextAreaElement | null>(null)

  let activities = $state<Activity[]>([])
  let subtasks = $state<Task[]>([])
  let actText = $state('')
  let commentText = $state('')
  let posting = $state(false)
  let copiedKind = $state<'short' | 'full' | null>(null)
  let copyTimer: ReturnType<typeof setTimeout> | undefined

  async function copyTaskId(kind: 'short' | 'full') {
    const text = kind === 'short' ? shortId(task.id) : task.id
    try {
      await navigator.clipboard.writeText(text)
      copiedKind = kind
      clearTimeout(copyTimer)
      copyTimer = setTimeout(() => {
        copiedKind = null
      }, 1500)
    } catch {
      onError('Could not copy to clipboard')
    }
  }

  function onShortIdClick(e: MouseEvent) {
    if (!(e.ctrlKey || e.metaKey)) return
    e.preventDefault()
    void copyTaskId('full')
  }

  async function loadSubtasks() {
    if (task.parent_id) {
      subtasks = []
      return
    }
    try {
      subtasks = await api.list({
        parentId: task.id,
        includeDone: true,
        includeHumanOnly: true,
        sort: 'created',
        ascending: true
      })
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
    copiedKind = null
    activeSection = 'task'
    loadActivity()
    loadSubtasks()
  })

  $effect(() => {
    if (task.parent_id && activeSection === 'subtasks') {
      activeSection = 'task'
    }
  })

  // Keep local fields in sync with live task updates; don't clobber an in-progress edit.
  $effect(() => {
    title = task.title
    progress = task.progress
    cwd = task.cwd ?? ''
    slackThread = task.slack_thread ?? ''
    prUrl = task.pr_url ?? ''
    todoSession = task.todo_session ?? ''
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

  async function savePRUrl() {
    const v = prUrl.trim()
    if (v === (task.pr_url ?? '')) return
    try {
      await api.update(task.id, { prUrl: v })
    } catch (e) {
      onError(errMsg(e))
    }
  }

  async function saveTodoSession() {
    const v = todoSession.trim()
    if (v === (task.todo_session ?? '')) return
    try {
      await api.update(task.id, { todoSession: v })
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
        ? 'fixed inset-y-0 right-0 z-40 flex flex-none flex-col border-l border-line bg-canvas shadow-md'
        : 'flex h-[min(85vh,860px)] w-full max-w-4xl flex-col overflow-hidden rounded-panel border border-line bg-canvas shadow-md'
  )
</script>

<!-- Click sink so modal backdrop close does not fire while editing the pane. -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions, a11y_click_events_have_key_events -->
<aside
  use:focusOnOpen={mode === 'modal'}
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
      class="mr-auto flex rounded-control border border-line-soft p-0.5"
      role="group"
      aria-label="Detail display mode"
    >
      {#each modeOptions as opt (opt.value)}
        <button
          type="button"
          onclick={() => onSetMode(opt.value)}
          title={opt.title}
          aria-pressed={mode === opt.value}
          class="rounded-control px-2 py-0.5 text-xs font-medium transition-colors
            {mode === opt.value ? 'bg-accent/20 text-accent-hi' : 'text-ink-3 hover:bg-white/5 hover:text-ink'}"
        >
          {opt.label}
        </button>
      {/each}
    </div>
    <div class="flex items-center gap-1">
      <div
        class="flex items-center gap-0.5 rounded-control border border-line-soft bg-field/40 pl-1.5 font-mono text-[10px] leading-none text-ink-3"
      >
        <button
          type="button"
          onclick={onShortIdClick}
          title={
            copiedKind === 'full'
              ? 'Copied full UUID'
              : `${task.id} — Ctrl+click to copy full UUID`
          }
          aria-label={
            copiedKind === 'full'
              ? 'Copied full UUID'
              : `Task ID ${shortId(task.id)}. Ctrl+click to copy full UUID`
          }
          class="cursor-copy py-1 transition-colors hover:text-ink-2"
        >
          {shortId(task.id)}
        </button>
        <button
          type="button"
          onclick={() => void copyTaskId('short')}
          title={
            copiedKind === 'short'
              ? 'Copied short ID'
              : copiedKind === 'full'
                ? 'Copied full UUID'
                : 'Copy short ID'
          }
          aria-label={
            copiedKind === 'short'
              ? 'Copied short ID'
              : copiedKind === 'full'
                ? 'Copied full UUID'
                : 'Copy short ID'
          }
          class="rounded-control p-1 text-ink-3 transition-colors hover:bg-white/5 hover:text-ink-2"
        >
          {#if copiedKind}
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
    </div>
    <button
      type="button"
      onclick={() => (saveAsOpen = true)}
      title="Save this task's fields as a template"
      aria-label="Save as template"
      class="rounded-control p-1.5 text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
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
        <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
        <path d="M17 21v-8H7v8" />
        <path d="M7 3v5h8" />
      </svg>
    </button>
    <button
      type="button"
      onclick={() => onDelete(task)}
      title="Delete task (del)"
      aria-label="Delete task"
      class="rounded-control p-1.5 text-ink-3 transition-colors hover:bg-danger/15 hover:text-danger"
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
      class="rounded-control p-1.5 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
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
        data-focus-primary
        bind:value={title}
        onblur={saveTitle}
        onkeydown={(e) => e.key === 'Enter' && (e.currentTarget as HTMLInputElement).blur()}
        class="w-full rounded-control border border-line-soft bg-field px-3 py-2 text-sm font-medium text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
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
          class="ta-autogrow w-full rounded-control border border-line-soft bg-field px-3 py-2 text-sm leading-relaxed text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
        ></textarea>
      {:else}
        <div
          role="button"
          tabindex="0"
          class="md-scroll w-full cursor-text rounded-control border border-line-soft bg-field px-3 py-2 text-left text-sm leading-relaxed text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] transition-colors hover:border-line hover:bg-card-hi/40"
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
          class="md-scroll rounded-control border border-accent/25 bg-accent/10 px-3 py-2 text-sm leading-relaxed text-ink-2"
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
          class="min-w-0 flex-1 rounded-control border border-line-soft bg-field px-3 py-2 font-mono text-xs text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
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
      <span class="micro mb-1.5">Todo session</span>
      <input
        bind:value={todoSession}
        onblur={saveTodoSession}
        placeholder="Claude session UUID"
        class="w-full rounded-control border border-line-soft bg-field px-3 py-2 font-mono text-xs text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
      />
      <p class="mt-1 text-[11px] text-ink-3">
        Claude session UUID linked to this ticket (<code class="text-ink-2">edit --session</code>); empty
        clears. Setting a non-empty value also posts a Claude Session activity.
      </p>
    </label>

    <label class="block">
      <span class="micro mb-1.5">Slack thread</span>
      <input
        bind:value={slackThread}
        onblur={saveSlackThread}
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

    <label class="block">
      <span class="micro mb-1.5">Pull request</span>
      <input
        bind:value={prUrl}
        onblur={savePRUrl}
        placeholder="https://… (optional PR link)"
        class="w-full rounded-control border border-line-soft bg-field px-3 py-2 font-mono text-xs text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
      />
      <p class="mt-1 text-[11px] text-ink-3">
        Setting a URL advances the ticket to the Pull Request lane.
      </p>
      {#if prUrl.trim()}
        <p class="mt-1.5 text-xs leading-relaxed text-ink-3">
          Linked PR:
          <a
            href={prUrl.trim()}
            target="_blank"
            rel="noopener noreferrer"
            class="text-accent hover:underline"
            onclick={(e) => {
              e.preventDefault()
              void openExternalUrl(prUrl.trim())
            }}>{prUrl.trim()}</a
          >
        </p>
      {/if}
    </label>

    <label class="flex cursor-pointer items-center gap-gap-md">
      <input
        type="checkbox"
        bind:checked={humanOnly}
        onchange={toggleHumanOnly}
        class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
      />
      <span class="text-sm text-ink-2">Human only <span class="text-ink-3">(agents skip this task)</span></span>
    </label>

    <label class="flex cursor-pointer items-center gap-gap-md">
      <input
        type="checkbox"
        bind:checked={includeInReport}
        onchange={toggleIncludeInReport}
        class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
      />
      <span class="text-sm text-ink-2">Include in Slack report <span class="text-ink-3">(board summary copy)</span></span>
    </label>
  {/snippet}

  {#snippet subtasksSection()}
    <div class="flex flex-col gap-3">
      <button
        type="button"
        onclick={() => onAddSubtask(task.id)}
        class="self-start rounded-control border border-line-soft bg-field px-3 py-2 text-sm text-ink-2 transition-colors hover:bg-card-hi hover:text-ink"
      >
        + Add sub-task
      </button>
    <ul class="space-y-1">
      {#each subtasks as st (st.id)}
        <li>
          <button
            type="button"
            onclick={() => onSelectParent?.(st.id)}
            class="flex w-full items-center gap-gap-md rounded-control border border-line-soft bg-field/50 px-3 py-2 text-left transition-colors hover:bg-card-hi"
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
        class="w-full rounded-control border border-line-soft bg-field px-3 py-1.5 text-sm text-ink placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
      />
      <textarea
        bind:value={commentText}
        rows="2"
        placeholder="Optional comment… (markdown)"
        class="w-full resize-y rounded-control border border-line-soft bg-field px-3 py-1.5 text-sm text-ink placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
      ></textarea>
      <button
        type="submit"
        disabled={posting || (!actText.trim() && !commentText.trim())}
        class="rounded-control bg-accent px-3 py-1.5 text-xs font-medium text-accent-ink disabled:opacity-40"
      >
        Post
      </button>
    </form>
    <ul class="space-y-2">
      {#each activities as a (a.id)}
        <li class="rounded-control border border-line-soft bg-field/50 px-2.5 py-2">
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
        class="w-full rounded-control border border-accent/50 bg-accent/10 px-3 py-2 text-sm font-medium text-accent-hi transition-colors hover:bg-accent/20"
      >
        Unarchive → pending
      </button>
    {/if}
  {/snippet}

  {#if mode === 'modal'}
    <div class="@container flex min-h-0 flex-1 overflow-hidden">
      <nav
        class="flex max-h-14 w-full flex-none flex-row gap-0.5 overflow-x-auto border-b border-line-soft p-2
          @[560px]:max-h-none @[560px]:w-44 @[560px]:flex-col @[560px]:overflow-y-auto @[560px]:border-b-0 @[560px]:border-r @[560px]:p-3"
        aria-label="Task sections"
      >
        {#each modalSections as section (section.id)}
          <button
            type="button"
            onclick={() => {
              activeSection = section.id
              if (section.id === 'subtasks') void loadSubtasks()
            }}
            class="whitespace-nowrap rounded-control px-3 py-2 text-left text-[13px] font-medium transition-colors
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

<SaveAsTemplateDialog
  open={saveAsOpen}
  source={{
    title: task.title,
    description: task.description,
    status: task.status,
    cwd: task.cwd,
    slackThread: task.slack_thread,
    humanOnly: task.human_only,
    includeInReport: task.include_in_report
  }}
  onClose={() => (saveAsOpen = false)}
  onSaved={(n) => onNotify?.(`Template “${n}” saved`)}
  {onError}
/>
