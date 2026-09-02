<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg, type Activity, type Status } from '../lib/api'
  import { absShort, relTime, shortId } from '../lib/format'
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

  let title = $state(task.title)
  let description = $state(task.description)
  let progress = $state(task.progress)
  let cwd = $state(task.cwd ?? '')
  let humanOnly = $state(!!task.human_only)

  let editingDesc = $state(false)
  let descEl = $state<HTMLTextAreaElement | null>(null)

  let activities = $state<Activity[]>([])
  let actText = $state('')
  let commentText = $state('')
  let posting = $state(false)
  let copiedId = $state(false)
  let copyTimer: ReturnType<typeof setTimeout> | undefined

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
    loadActivity()
  })

  // Keep local fields in sync with live task updates; don't clobber an in-progress edit.
  $effect(() => {
    title = task.title
    progress = task.progress
    cwd = task.cwd ?? ''
    humanOnly = !!task.human_only
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
      const path = await api.pickDirectory()
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

  async function toggleHumanOnly() {
    if (humanOnly === !!task.human_only) return
    try {
      await api.update(task.id, { humanOnly })
    } catch (e) {
      humanOnly = !!task.human_only
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
        : 'flex max-h-[90vh] w-[90vw] max-w-5xl flex-col overflow-hidden rounded-lg border border-line bg-canvas shadow-2xl'
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
    <button
      onclick={onClose}
      title="Close (esc)"
      class="rounded p-1.5 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
    >
      ✕
    </button>
  </div>

  <div class="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-5 py-4">
    {#if task.parent_id}
      <div>
        <span class="micro mb-1.5">Parent task</span>
        {#if mode === 'modal'}
          <p class="truncate text-sm font-medium text-ink-2" title={parentTitle ?? task.parent_id}>
            {parentTitle ?? task.parent_id}
          </p>
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

    <label class="flex cursor-pointer items-center gap-2.5">
      <input
        type="checkbox"
        bind:checked={humanOnly}
        onchange={toggleHumanOnly}
        class="h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
      />
      <span class="text-sm text-ink-2">Human only <span class="text-ink-3">(agents skip this task)</span></span>
    </label>

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
            // Keep markdown links clickable.
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
        <div
          class="md-scroll rounded border border-accent/25 bg-accent/10 px-3 py-2 text-sm leading-relaxed text-ink-2"
        >
          <Markdown source={task.feedback} />
        </div>
      </div>
    {/if}

    <div>
      <span class="micro mb-1.5">Progress</span>
      <ProgressControl value={progress} onCommit={(p) => saveProgress(p)} />
    </div>

    {#if !task.parent_id}
      <button
        type="button"
        onclick={() => onAddSubtask(task.id)}
        class="rounded border border-line-soft bg-field px-3 py-2 text-sm text-ink-2 transition-colors hover:bg-card-hi hover:text-ink"
      >
        + Add sub-task
      </button>
    {/if}

    <div class="border-t border-line-soft pt-3">
      <span class="micro mb-2">Activity</span>
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
    </div>

    <dl class="space-y-1.5 border-t border-line-soft pt-3 text-xs text-ink-3">
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
  </div>

  <div class="flex flex-none flex-col gap-2 border-t border-line-soft bg-chrome px-5 py-3">
    {#if task.archived_at}
      <button
        onclick={unarchive}
        title="Restore to pending (progress resets to 0)"
        class="w-full rounded border border-accent/50 bg-accent/10 px-3 py-2 text-sm font-medium text-accent-hi transition-colors hover:bg-accent/20"
      >
        Unarchive → pending
      </button>
    {/if}
    <button
      onclick={() => onDelete(task)}
      title="Delete (del)"
      class="w-full rounded border border-danger/40 bg-danger/10 px-3 py-2 text-sm font-medium text-danger transition-colors hover:bg-danger/15"
    >
      Delete task <kbd>del</kbd>
    </button>
  </div>
</aside>
