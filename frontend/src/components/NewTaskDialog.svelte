<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg, type Status } from '../lib/api'

  let {
    open,
    initialStatus = 'pending',
    onClose,
    onError
  }: { open: boolean; initialStatus?: Status; onClose: () => void; onError: (msg: string) => void } = $props()

  const STATUSES: Status[] = ['pending', 'wip', 'waiting', 'done']

  let title = $state('')
  let description = $state('')
  let status = $state<Status>('pending')

  // Board column quick-add presets the status; applied each time the dialog opens.
  $effect(() => {
    if (open) status = initialStatus
  })
  let progress = $state(0)
  let submitting = $state(false)

  async function submit(e: Event) {
    e.preventDefault()
    if (!title.trim() || submitting) return
    submitting = true
    try {
      await api.create({ title, description, status, progress })
      title = ''
      description = ''
      status = 'pending'
      progress = 0
      onClose()
    } catch (err) {
      onError(errMsg(err))
    } finally {
      submitting = false
    }
  }

  function resetAndClose() {
    title = ''
    description = ''
    status = 'pending'
    progress = 0
    onClose()
  }
</script>

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onclick={resetAndClose}>
    <form
      in:fly={{ y: 12, duration: 150 }}
      onclick={(e) => e.stopPropagation()}
      onsubmit={submit}
      class="w-full max-w-md rounded-xl border border-zinc-800 bg-zinc-950 p-5 shadow-2xl"
    >
      <h2 class="mb-4 text-base font-semibold text-zinc-100">New task</h2>

      <label class="mb-3 block">
        <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-600">Title *</span>
        <input
          autofocus
          bind:value={title}
          onkeydown={(e) => e.key === 'Enter' && title.trim() && (e.currentTarget as HTMLInputElement).form?.requestSubmit()}
          placeholder="What needs doing?"
          class="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-indigo-500 focus:outline-none"
        />
      </label>

      <label class="mb-3 block">
        <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-600">Description</span>
        <textarea
          bind:value={description}
          rows="3"
          placeholder="Optional notes…"
          class="w-full resize-y rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 placeholder:text-zinc-600 focus:border-indigo-500 focus:outline-none"
        ></textarea>
      </label>

      <div class="mb-4 flex gap-3">
        <label class="flex-1 block">
          <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-600">Status</span>
          <select
            value={status}
            onchange={(e) => (status = e.currentTarget.value as Status)}
            class="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-2 py-2 text-sm capitalize text-zinc-200 focus:border-indigo-500 focus:outline-none"
          >
            {#each STATUSES as s (s)}
              <option value={s}>{s}</option>
            {/each}
          </select>
        </label>
        <label class="flex-1 block">
          <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-600">Progress</span>
          <input
            type="number"
            min="0"
            max="100"
            value={progress}
            onchange={(e) => (progress = Math.max(0, Math.min(100, Number(e.currentTarget.value) || 0)))}
            class="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm font-mono text-zinc-200 focus:border-indigo-500 focus:outline-none"
          />
        </label>
      </div>

      <div class="flex justify-end gap-2">
        <button type="button" onclick={resetAndClose} class="rounded-lg px-3 py-1.5 text-sm text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200">
          Cancel
        </button>
        <button
          type="submit"
          disabled={!title.trim() || submitting}
          class="rounded-lg bg-indigo-600 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-40"
        >
          Create task
        </button>
      </div>
    </form>
  </div>
{/if}
