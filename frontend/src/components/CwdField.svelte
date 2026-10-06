<script lang="ts">
  import { onDestroy } from 'svelte'
  import { fly } from 'svelte/transition'
  import { api, errMsg } from '../lib/api'
  import {
    cwdSuggestionsLoading,
    cwdSuggestionsReady,
    getCwdCandidates,
    rankCwdCandidates,
    subscribeCwdSuggestions,
    type CwdCandidate
  } from '../lib/cwdSuggestions'

  let {
    value = $bindable(''),
    inputClass = '',
    placeholder = 'Optional project path…',
    onCommit,
    onError
  }: {
    value?: string
    /** Extra classes for the text input (mono/xs vs sm parity). */
    inputClass?: string
    placeholder?: string
    /** Called when the value should be persisted (blur / pick / folder). */
    onCommit?: (cwd: string) => void | Promise<void>
    onError?: (msg: string) => void
  } = $props()

  let open = $state(false)
  let cursor = $state(0)
  let listEl = $state<HTMLDivElement | null>(null)
  /** Prevent blur-commit from racing a mousedown pick. */
  let picking = $state(false)

  // Tick when the warm store rebuilds so ranked stays current.
  let storeTick = $state(0)
  const unsub = subscribeCwdSuggestions(() => {
    storeTick++
  })
  onDestroy(unsub)

  const ranked = $derived.by(() => {
    void storeTick
    return rankCwdCandidates(value, getCwdCandidates())
  })

  const ready = $derived.by(() => {
    void storeTick
    return cwdSuggestionsReady()
  })

  const loading = $derived.by(() => {
    void storeTick
    return cwdSuggestionsLoading()
  })

  $effect(() => {
    if (cursor > ranked.length - 1) cursor = Math.max(0, ranked.length - 1)
  })

  function openList() {
    open = true
    cursor = 0
  }

  function closeList() {
    open = false
    cursor = 0
  }

  async function commit(next: string) {
    value = next
    closeList()
    await onCommit?.(next)
  }

  function pick(row: CwdCandidate) {
    picking = true
    void commit(row.cwd).finally(() => {
      picking = false
    })
  }

  async function onBlur() {
    // Defer so option mousedown can set picking / apply before we commit.
    await Promise.resolve()
    if (picking) return
    closeList()
    await onCommit?.(value)
  }

  async function pickFolder() {
    try {
      const path = await api.pickDirectory(value.trim())
      if (path) await commit(path)
    } catch (e) {
      onError?.(errMsg(e))
    }
  }

  function move(delta: number) {
    if (!ranked.length) return
    cursor = (cursor + delta + ranked.length) % ranked.length
    listEl?.querySelectorAll('[data-row]')[cursor]?.scrollIntoView({ block: 'nearest' })
  }

  function onKeydown(e: KeyboardEvent) {
    if (!open) {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault()
        openList()
      }
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      closeList()
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      move(1)
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      move(-1)
      return
    }
    if (e.key === 'Enter') {
      const row = ranked[cursor]
      if (row) {
        e.preventDefault()
        pick(row)
      }
      // else let the form submit (new-task) with the typed path
    }
  }

  $effect(() => {
    if (!open) return
    const onDocumentClick = (e: MouseEvent) => {
      const target = e.target as HTMLElement | null
      if (!target?.closest('[data-cwd-field]')) closeList()
    }
    document.addEventListener('click', onDocumentClick)
    return () => document.removeEventListener('click', onDocumentClick)
  })
</script>

<div data-cwd-field class="relative min-w-0 flex-1">
  <div class="flex gap-2">
    <input
      bind:value
      {placeholder}
      onfocus={openList}
      oninput={() => {
        if (!open) openList()
        cursor = 0
      }}
      onblur={onBlur}
      onkeydown={onKeydown}
      autocomplete="off"
      spellcheck="false"
      role="combobox"
      aria-expanded={open}
      aria-autocomplete="list"
      aria-controls="cwd-typeahead-list"
      class="min-w-0 flex-1 rounded-control border border-line-soft bg-field px-3 py-2 text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25 {inputClass}"
    />
    <button
      type="button"
      onclick={pickFolder}
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

  {#if open}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      id="cwd-typeahead-list"
      in:fly={{ y: -4, duration: 80 }}
      class="absolute left-0 right-0 top-full z-30 mt-1.5 overflow-hidden rounded-panel border border-line bg-col shadow-md"
      role="listbox"
    >
      <div bind:this={listEl} class="max-h-72 overflow-y-auto p-1.5">
        {#if !ready && loading}
          <p class="px-2 py-3 text-sm text-ink-3">Loading paths…</p>
        {:else if !ranked.length}
          <p class="px-2 py-3 text-sm text-ink-3">
            {value.trim()
              ? `No matches for “${value.trim()}”. Keep typing a path, or use the folder button.`
              : 'No template or prior working directories yet.'}
          </p>
        {:else}
          {#each ranked as row, i (row.kind + ':' + row.id)}
            <button
              type="button"
              data-row
              role="option"
              aria-selected={i === cursor}
              onmousedown={(e) => {
                e.preventDefault()
                pick(row)
              }}
              onmouseenter={() => (cursor = i)}
              class="block w-full rounded-control px-2 py-1.5 text-left
                {i === cursor ? 'bg-accent/15' : 'hover:bg-white/5'}"
            >
              <span
                class="block truncate text-sm font-medium
                  {row.name ? 'text-ink' : 'invisible select-none'}"
                aria-hidden={!row.name}
              >{row.name || '\u00a0'}</span>
              <span class="mt-0.5 block truncate font-mono text-xs text-ink-2">{row.cwd}</span>
            </button>
          {/each}
        {/if}
      </div>
      <div class="border-t border-line-soft px-2.5 py-1.5 text-[10.5px] text-ink-3">
        <kbd>↑</kbd><kbd>↓</kbd> navigate · <kbd>↵</kbd> select · <kbd>esc</kbd> close
      </div>
    </div>
  {/if}
</div>
