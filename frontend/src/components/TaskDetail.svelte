<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg, type Status } from '../lib/api'
  import { absShort } from '../lib/format'

  let { task, onClose, onError }: { task: any; onClose: () => void; onError: (msg: string) => void } = $props()

  const STATUSES: Status[] = ['pending', 'wip', 'waiting', 'done']
  const activeClass: Record<string, string> = {
    pending: 'bg-zinc-600 text-white',
    wip: 'bg-indigo-600 text-white',
    waiting: 'bg-amber-600 text-white',
    done: 'bg-emerald-600 text-white'
  }

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

  async function saveProgress() {
    const p = Math.max(0, Math.min(100, Number(progress) || 0))
    if (p === task.progress) return
    try {
      await api.update(task.id, { progress: p })
    } catch (e) {
      onError(errMsg(e))
    }
  }

  async function remove() {
    if (!confirm(`Delete "${task.title}"? This cannot be undone.`)) return
    try {
      await api.remove(task.id)
      onClose()
    } catch (e) {
      onError(errMsg(e))
    }
  }
</script>

<aside
  in:fly={{ x: 40, duration: 150 }}
  class="fixed inset-y-0 right-0 z-40 flex w-[420px] max-w-full flex-col border-l border-zinc-800 bg-zinc-950 shadow-2xl"
>
  <div class="flex items-center gap-3 border-b border-zinc-800 px-5 py-3">
    <span class="font-mono text-xs text-zinc-600">{task.id}</span>
    <div class="flex-1"></div>
    <button onclick={onClose} title="Close (esc)" class="rounded-md p-1 text-zinc-500 hover:bg-zinc-800 hover:text-zinc-200">✕</button>
  </div>

  <div class="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-5 py-4">
    <label class="block">
      <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-600">Title</span>
      <input
        bind:value={title}
        onblur={saveTitle}
        onkeydown={(e) => e.key === 'Enter' && (e.currentTarget as HTMLInputElement).blur()}
        class="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm font-medium text-zinc-100 focus:border-indigo-500 focus:outline-none"
      />
    </label>

    <div>
      <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-600">Status</span>
      <div class="flex gap-1 rounded-lg bg-zinc-900 p-1">
        {#each STATUSES as s (s)}
          <button
            onclick={() => setStatus(s)}
            class="flex-1 rounded-md px-2 py-1.5 text-xs font-medium capitalize transition-colors
              {task.status === s ? activeClass[s] : 'text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200'}"
          >
            {s}
          </button>
        {/each}
      </div>
    </div>

    <label class="block">
      <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-600">Description</span>
      <textarea
        bind:value={description}
        onblur={saveDescription}
        rows="5"
        placeholder="Notes, links, context…"
        class="w-full resize-y rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 placeholder:text-zinc-600 focus:border-indigo-500 focus:outline-none"
      ></textarea>
    </label>

    <div>
      <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-600">Progress</span>
      <div class="flex items-center gap-3">
        <input
          type="range"
          min="0"
          max="100"
          step="5"
          bind:value={progress}
          onchange={() => saveProgress()}
          class="flex-1 accent-indigo-500"
        />
        <div class="flex items-center gap-1">
          <input
            type="number"
            min="0"
            max="100"
            bind:value={progress}
            onblur={() => saveProgress()}
            onkeydown={(e) => e.key === 'Enter' && (e.currentTarget as HTMLInputElement).blur()}
            class="w-16 rounded-lg border border-zinc-800 bg-zinc-900 px-2 py-1 text-right font-mono text-sm text-zinc-200 focus:border-indigo-500 focus:outline-none"
          />
          <span class="text-xs text-zinc-600">%</span>
        </div>
      </div>
    </div>

    <dl class="space-y-1 border-t border-zinc-800 pt-3 text-xs text-zinc-500">
      <div class="flex justify-between"><dt>Created</dt><dd>{absShort(task.created_at)}</dd></div>
      <div class="flex justify-between"><dt>Updated</dt><dd>{absShort(task.updated_at)}</dd></div>
      <div class="flex justify-between">
        <dt>Completed</dt>
        <dd>{task.completed_at ? absShort(task.completed_at) : '—'}</dd>
      </div>
    </dl>
  </div>

  <div class="border-t border-zinc-800 px-5 py-3">
    <button
      onclick={remove}
      class="rounded-lg border border-rose-900/60 bg-rose-950/40 px-3 py-1.5 text-sm font-medium text-rose-300 transition-colors hover:bg-rose-900/40"
    >
      Delete task
    </button>
  </div>
</aside>
