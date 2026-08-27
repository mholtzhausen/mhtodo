<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg, type Activity, type Status } from '../lib/api'
  import { absShort, relTime } from '../lib/format'
  import StatusPicker from './StatusPicker.svelte'
  import ProgressControl from './ProgressControl.svelte'

  let {
    task,
    parentTitle = null,
    pinned,
    onClose,
    onError,
    onDelete,
    onTogglePin,
    onAddSubtask,
    onSelectParent
  }: {
    task: any
    parentTitle?: string | null
    pinned: boolean
    onClose: () => void
    onError: (msg: string) => void
    onDelete: (task: any) => void
    onTogglePin: () => void
    onAddSubtask: (parentId: string) => void
    onSelectParent?: (parentId: string) => void
  } = $props()

  let title = $state(task.title)
  let description = $state(task.description)
  let progress = $state(task.progress)

  let activities = $state<Activity[]>([])
  let actText = $state('')
  let commentText = $state('')
  let posting = $state(false)

  async function loadActivity() {
    try {
      activities = await api.listActivity({ taskIds: [task.id] })
    } catch (e) {
      onError(errMsg(e))
    }
  }

  $effect(() => {
    void task.id
    loadActivity()
  })

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

  const shellClass = pinned
    ? 'flex h-full w-[420px] flex-none flex-col border-l border-line bg-canvas'
    : 'fixed inset-y-0 right-0 z-40 flex w-[420px] max-w-full flex-col border-l border-line bg-canvas shadow-2xl'
</script>

<aside in:fly={{ x: 40, duration: 150 }} class={shellClass}>
  <div class="flex flex-none items-center justify-end gap-2 border-b border-line-soft bg-chrome px-4 py-2.5">
    <button
      onclick={onTogglePin}
      title={pinned ? 'Unpin detail' : 'Pin detail to the right'}
      class="rounded px-2 py-1 text-xs font-medium transition-colors
        {pinned ? 'bg-accent/20 text-accent-hi' : 'text-ink-3 hover:bg-white/5 hover:text-ink'}"
    >
      {pinned ? 'Pinned' : 'Pin'}
    </button>
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
        <button
          type="button"
          onclick={() => onSelectParent?.(task.parent_id)}
          class="block w-full truncate text-left text-sm font-medium text-accent-hi hover:underline"
          title={parentTitle ?? task.parent_id}
        >
          {parentTitle ?? 'Open parent'}
        </button>
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

    <label class="block">
      <span class="micro mb-1.5">Description</span>
      <textarea
        bind:value={description}
        onblur={saveDescription}
        rows="4"
        placeholder="Notes, links, context…"
        class="w-full resize-y rounded border border-line-soft bg-field px-3 py-2 text-sm leading-relaxed text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
      ></textarea>
    </label>

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
          placeholder="Optional comment…"
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
            {#if a.comment}<p class="text-xs text-ink-2">{a.comment}</p>{/if}
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
