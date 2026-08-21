<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg, type Status } from '../lib/api'
  import { absShort } from '../lib/format'
  import StatusPicker from './StatusPicker.svelte'
  import ProgressControl from './ProgressControl.svelte'

  // onDelete is owned by App so the Delete key and this button share one
  // confirm → delete flow.
  let {
    task,
    onClose,
    onError,
    onDelete
  }: { task: any; onClose: () => void; onError: (msg: string) => void; onDelete: (task: any) => void } = $props()

  // Local edit state, initialized once per task (App keys the drawer by id).
  let title = $state(task.title)
  let description = $state(task.description)
  let progress = $state(task.progress)

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

  // v0.2 unarchive → the task goes back to pending with progress reset on the
  // Go side; sync the local slider so it doesn't write a stale value later.
  async function unarchive() {
    try {
      const t = await api.unarchive(task.id)
      progress = t.progress
    } catch (e) {
      onError(errMsg(e))
    }
  }

</script>

<aside
  in:fly={{ x: 40, duration: 150 }}
  class="fixed inset-y-0 right-0 z-40 flex w-[420px] max-w-full flex-col border-l border-line bg-canvas shadow-2xl"
>
  <div class="flex flex-none items-center gap-3 border-b border-line-soft bg-chrome px-4 py-2.5">
    <span class="min-w-0 flex-1 truncate font-mono text-xs text-ink-3">{task.id}</span>
    <button
      onclick={onClose}
      title="Close (esc)"
      class="rounded p-1.5 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
    >
      ✕
    </button>
  </div>

  <div class="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-5 py-4">
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
        rows="5"
        placeholder="Notes, links, context…"
        class="w-full resize-y rounded border border-line-soft bg-field px-3 py-2 text-sm leading-relaxed text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
      ></textarea>
    </label>

    <div>
      <span class="micro mb-1.5">Progress</span>
      <ProgressControl value={progress} onCommit={(p) => saveProgress(p)} />
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

  <div class="flex-none flex-col gap-2 border-t border-line-soft bg-chrome px-5 py-3">
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
