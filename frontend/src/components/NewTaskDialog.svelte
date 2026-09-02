<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg, type Status } from '../lib/api'
  import StatusPicker from './StatusPicker.svelte'
  import ProgressControl from './ProgressControl.svelte'

  let {
    open,
    initialStatus = 'pending',
    parentId = '',
    defaultCwd = '',
    defaultHumanOnly = false,
    onClose,
    onError
  }: {
    open: boolean
    initialStatus?: Status
    parentId?: string
    defaultCwd?: string
    defaultHumanOnly?: boolean
    onClose: () => void
    onError?: (msg: string) => void
  } = $props()

  let title = $state('')
  let description = $state('')
  let status = $state<Status>('pending')
  let cwd = $state('')
  let humanOnly = $state(false)

  $effect(() => {
    if (open) {
      status = initialStatus
      cwd = defaultCwd
      humanOnly = defaultHumanOnly
    }
  })
  let progress = $state(0)
  let submitting = $state(false)

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
        progress,
        parentId: parentId || undefined,
        cwd: cwd.trim() || undefined,
        humanOnly
      })
      title = ''
      description = ''
      status = 'pending'
      cwd = ''
      humanOnly = false
      progress = 0
      onClose()
    } catch (err) {
      onError?.(errMsg(err))
    } finally {
      submitting = false
    }
  }

  function resetAndClose() {
    title = ''
    description = ''
    status = 'pending'
    cwd = ''
    humanOnly = false
    progress = 0
    onClose()
  }
</script>

{#if open}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-4 backdrop-blur-[2px]"
    onclick={resetAndClose}
  >
    <form
      in:fly={{ y: 12, duration: 150 }}
      onclick={(e) => e.stopPropagation()}
      onsubmit={submit}
      class="flex max-h-[92vh] w-full max-w-md flex-col rounded-lg border border-line bg-col shadow-2xl"
    >
      <div class="flex flex-none items-center gap-2.5 border-b border-line-soft px-5 py-3.5">
        <h2 class="flex-1 text-base font-semibold text-ink">
          {parentId ? 'New sub-task' : 'New task'}
        </h2>
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

        <label class="flex cursor-pointer items-center gap-2.5">
          <input
            type="checkbox"
            bind:checked={humanOnly}
            class="h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
          />
          <span class="text-sm text-ink-2">Human only <span class="text-ink-3">(agents skip this task)</span></span>
        </label>

        <div>
          <span class="micro mb-1.5">Status</span>
          <StatusPicker value={status} onPick={(s) => (status = s)} />
        </div>

        <div>
          <span class="micro mb-1.5">Progress</span>
          <ProgressControl value={progress} onCommit={(p) => (progress = p)} />
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
{/if}
