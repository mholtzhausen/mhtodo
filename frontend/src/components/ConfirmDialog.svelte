<script lang="ts">
  import { fly } from 'svelte/transition'
  import { focusOnOpen } from '../lib/focusFirstField'

  // Small destructive-action confirmation (delete task). Same visual language
  // as NewTaskDialog; rendered above the detail drawer (z-40) and below toasts.
  let {
    open,
    title = 'Are you sure?',
    message = '',
    confirmLabel = 'Delete',
    onCancel,
    onConfirm
  }: {
    open: boolean
    title?: string
    message?: string
    confirmLabel?: string
    onCancel: () => void
    onConfirm: () => void
  } = $props()

  // Enter confirms (the Delete key already opened this dialog; a second,
  // deliberate keystroke is the confirmation step). preventDefault stops the
  // focused button's native activation from double-firing.
  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault()
      onConfirm()
    }
  }
</script>

{#if open}
  <div
    class="fixed inset-0 z-[55] flex items-center justify-center bg-black/55 p-4"
    onclick={onCancel}
  >
    <div
      use:focusOnOpen
      in:fly={{ y: 8, duration: 80 }}
      role="alertdialog"
      aria-modal="true"
      aria-label={title}
      onkeydown={onKeydown}
      class="w-full max-w-sm rounded-panel border border-line bg-col shadow-md"
    >
      <div class="px-5 pt-4">
        <h2 class="text-base font-semibold text-ink">{title}</h2>
        {#if message}<p class="mt-1.5 text-sm leading-relaxed text-ink-2">{message}</p>{/if}
      </div>

      <div class="flex items-center justify-end gap-2 border-t border-line-soft px-5 py-3">
        <button
          type="button"
          onclick={onCancel}
          class="rounded-control px-3 py-1.5 text-sm text-ink-2 transition-colors hover:bg-white/5 hover:text-ink"
        >
          Cancel
        </button>
        <button
          type="button"
          data-focus-primary
          onclick={onConfirm}
          class="rounded-control bg-danger px-4 py-1.5 text-sm font-medium text-[#38090f] shadow-sm transition-colors hover:bg-danger/85"
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}
