<script lang="ts">
  // Click-to-edit field for the Focus view lanes. Shows the value (markdown
  // when `markdown`), swaps to an input/textarea on click, saves on blur or
  // Enter (single-line) / Ctrl+Enter (multi-line), Esc cancels. The draft is
  // local while editing so live `tasks:changed` reloads don't clobber it.
  import { tick } from 'svelte'
  import Markdown from './Markdown.svelte'

  let {
    value,
    onSave,
    markdown = false,
    placeholder = '',
    empty = '',
    class: className = '',
    inputClass = '',
    disabled = false
  }: {
    value: string | null | undefined
    onSave: (v: string) => void | Promise<void>
    markdown?: boolean
    placeholder?: string
    /** Shown in view mode when the value is empty. */
    empty?: string
    class?: string
    inputClass?: string
    disabled?: boolean
  } = $props()

  let editing = $state(false)
  let draft = $state('')
  let inputEl = $state<HTMLInputElement | HTMLTextAreaElement | null>(null)

  const current = $derived(value ?? '')

  /** Grow textarea with content up to 400px, then scroll. */
  function fit() {
    if (!(inputEl instanceof HTMLTextAreaElement)) return
    inputEl.style.height = 'auto'
    inputEl.style.height = `${Math.min(inputEl.scrollHeight, 400)}px`
  }

  async function start() {
    if (disabled || editing) return
    draft = current
    editing = true
    await tick()
    fit()
    inputEl?.focus()
    if (inputEl instanceof HTMLInputElement) inputEl.select()
  }

  function commit() {
    if (!editing) return
    editing = false
    const v = markdown ? draft : draft.trim()
    if (!markdown && !v) return // title-like fields must stay non-empty
    if (v === current) return
    void onSave(v)
  }

  function cancel() {
    editing = false
    draft = current
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      cancel()
      return
    }
    if (e.key === 'Enter' && (!markdown || e.ctrlKey || e.metaKey)) {
      e.preventDefault()
      commit()
    }
  }
</script>

{#if editing}
  {#if markdown}
    <textarea
      bind:this={inputEl}
      bind:value={draft}
      oninput={fit}
      onblur={commit}
      onkeydown={onKeydown}
      rows="3"
      {placeholder}
      title="Ctrl+Enter or click away to save · Esc to cancel"
      class="w-full resize-none rounded-control border border-accent bg-field px-2 py-1.5 leading-relaxed text-ink placeholder:text-ink-3 focus:outline-none focus:ring-2 focus:ring-accent/25 {inputClass}"
    ></textarea>
  {:else}
    <input
      bind:this={inputEl}
      bind:value={draft}
      onblur={commit}
      onkeydown={onKeydown}
      {placeholder}
      class="w-full rounded-control border border-accent bg-field px-2 py-1 text-ink placeholder:text-ink-3 focus:outline-none focus:ring-2 focus:ring-accent/25 {inputClass}"
    />
  {/if}
{:else}
  <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
  <div
    role={disabled ? undefined : 'button'}
    tabindex={disabled ? undefined : 0}
    title={disabled ? undefined : 'Click to edit'}
    class="-mx-1 rounded-control px-1 {disabled
      ? ''
      : 'cursor-text transition-colors hover:bg-white/5'} {className}"
    onclick={(e) => {
      if ((e.target as HTMLElement).closest('a')) return
      void start()
    }}
    onkeydown={(e) => {
      if (e.target !== e.currentTarget) return
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        void start()
      }
    }}
  >
    {#if markdown}
      {#if current.trim()}
        <Markdown source={current} />
      {:else}
        <p class="italic text-ink-3/70">{empty}</p>
      {/if}
    {:else}
      {current || empty}
    {/if}
  </div>
{/if}
