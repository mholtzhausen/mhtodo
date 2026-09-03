<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg } from '../lib/api'
  import { chipLabel, setFields, type TaskTemplate } from '../lib/templates'

  let {
    open,
    onPick,
    onClose,
    onError
  }: {
    open: boolean
    onPick: (t: TaskTemplate) => void
    onClose: () => void
    onError?: (msg: string) => void
  } = $props()

  let templates = $state<TaskTemplate[]>([])
  let loading = $state(false)
  let query = $state('')
  let cursor = $state(0)
  let searchEl = $state<HTMLInputElement | null>(null)
  let listEl = $state<HTMLDivElement | null>(null)

  const filtered = $derived(
    query.trim()
      ? templates.filter((t) => t.name.toLowerCase().includes(query.trim().toLowerCase()))
      : templates
  )

  // Reload on every open: a template may have been added from the save-as
  // dialog or settings since this picker was last shown.
  $effect(() => {
    if (!open) {
      query = ''
      cursor = 0
      return
    }
    loading = true
    api
      .listTemplates()
      .then((t) => (templates = t))
      .catch((e) => onError?.(errMsg(e)))
      .finally(() => (loading = false))
  })

  // Keep the cursor inside the filtered list as the query narrows it.
  $effect(() => {
    if (cursor > filtered.length - 1) cursor = Math.max(0, filtered.length - 1)
  })

  $effect(() => {
    if (open && searchEl) searchEl.focus()
  })

  function move(delta: number) {
    if (!filtered.length) return
    cursor = (cursor + delta + filtered.length) % filtered.length
    listEl?.querySelectorAll('[data-row]')[cursor]?.scrollIntoView({ block: 'nearest' })
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      onClose()
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
      e.preventDefault()
      const t = filtered[cursor]
      if (t) onPick(t)
      return
    }
    // `/` refocuses the filter when the user has tabbed into the list. The
    // dialog's own global handler must not see it, hence stopPropagation.
    if (e.key === '/' && document.activeElement !== searchEl) {
      e.preventDefault()
      e.stopPropagation()
      searchEl?.focus()
    }
  }
</script>

<svelte:window
  onclick={(e) => {
    if (!open) return
    const target = e.target as HTMLElement
    if (!target.closest('[data-template-picker]')) onClose()
  }}
/>

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    data-template-picker
    in:fly={{ y: -6, duration: 120 }}
    onkeydown={onKeydown}
    class="absolute right-0 top-full z-30 mt-1.5 w-80 overflow-hidden rounded-lg border border-line bg-col shadow-2xl"
  >
    <div class="border-b border-line-soft p-2">
      <div class="relative">
        <span class="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-ink-3">
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
            <circle cx="11" cy="11" r="7" />
            <path d="m21 21-4.3-4.3" />
          </svg>
        </span>
        <input
          bind:this={searchEl}
          bind:value={query}
          placeholder="Filter templates…  /"
          aria-label="Filter templates"
          class="w-full rounded border border-line-soft bg-field py-1.5 pl-8 pr-3 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
        />
      </div>
    </div>

    <div bind:this={listEl} class="max-h-72 overflow-y-auto p-1.5" role="listbox" tabindex="-1">
      {#if loading}
        <p class="px-2 py-3 text-sm text-ink-3">Loading…</p>
      {:else if !templates.length}
        <p class="px-2 py-3 text-sm leading-relaxed text-ink-3">
          No templates yet. Create one in Settings → Task Templates, or save this form as a
          template.
        </p>
      {:else if !filtered.length}
        <p class="px-2 py-3 text-sm text-ink-3">No template matches “{query.trim()}”.</p>
      {:else}
        {#each filtered as tpl, i (tpl.id)}
          <button
            type="button"
            data-row
            role="option"
            aria-selected={i === cursor}
            onclick={() => onPick(tpl)}
            onmouseenter={() => (cursor = i)}
            class="block w-full rounded px-2 py-2 text-left transition-colors
              {i === cursor ? 'bg-accent/15' : 'hover:bg-white/5'}"
          >
            <span class="block truncate text-sm font-medium text-ink">{tpl.name}</span>
            {#if setFields(tpl).length}
              <span class="mt-1 flex flex-wrap gap-1">
                {#each setFields(tpl) as f (f.key)}
                  <span
                    class="rounded-full border border-line-soft bg-field px-1.5 py-px text-[10px] leading-[15px] text-ink-3"
                  >{chipLabel(tpl, f)}</span>
                {/each}
              </span>
            {:else}
              <span class="mt-0.5 block text-[11px] italic text-ink-3/70">No fields set</span>
            {/if}
          </button>
        {/each}
      {/if}
    </div>

    <div class="border-t border-line-soft px-2.5 py-1.5 text-[10.5px] text-ink-3">
      <kbd>↑</kbd><kbd>↓</kbd> navigate · <kbd>↵</kbd> apply · <kbd>esc</kbd> close
    </div>
  </div>
{/if}
