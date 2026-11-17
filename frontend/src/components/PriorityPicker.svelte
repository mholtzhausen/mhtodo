<script lang="ts">
  import { tick } from 'svelte'
  import {
    PRIORITIES,
    PRIORITY_LABELS,
    PRIORITY_TEXT,
    normPriority,
    type Priority
  } from '../lib/priority'
  import { portal } from '../lib/portal'
  import PriorityIcon from './PriorityIcon.svelte'

  // One themed popover for every priority surface.
  //  - mode "single": cards, Focus lanes, sub-tasks, archive, detail. Click /
  //    Enter / 0–4 applies immediately and closes.
  //  - mode "multi": the header filter. Toggles levels, stays open, has Clear.
  // The popover is portalled to <body> and flips/clamps so it never clips.
  let {
    mode = 'single',
    value = 'none',
    values = [],
    onPick,
    onChange,
    variant = 'icon',
    idleClass = '',
    iconClass = 'h-3.5 w-3.5',
    title: titleProp = '',
    disabled = false
  }: {
    mode?: 'single' | 'multi'
    value?: Priority | string
    values?: readonly Priority[]
    onPick?: (p: Priority) => void
    onChange?: (ps: Priority[]) => void
    /** icon: bare glyph (cards). field: glyph + label (detail). filter: header button with count. */
    variant?: 'icon' | 'field' | 'filter'
    /** Extra trigger classes when closed and priority is none (e.g. hover-reveal on cards). */
    idleClass?: string
    iconClass?: string
    title?: string
    disabled?: boolean
  } = $props()

  let open = $state(false)
  let triggerEl = $state<HTMLButtonElement | null>(null)
  let popEl = $state<HTMLDivElement | null>(null)
  let pos = $state({ top: 0, left: 0, ready: false })
  const listId = `prio-${Math.random().toString(36).slice(2, 9)}`

  const current = $derived(normPriority(value))
  const selected = $derived(new Set<Priority>(values))
  const filterActive = $derived(mode === 'multi' && values.length > 0)
  // Filter popover lists most-urgent first (scan order); single mode too, so
  // both read the same way.
  const ORDER = [...PRIORITIES].reverse() as Priority[]

  const triggerTitle = $derived(
    titleProp ||
      (mode === 'multi'
        ? filterActive
          ? `Priority filter: ${values.map((p) => PRIORITY_LABELS[p]).join(', ')}`
          : 'Filter by priority'
        : current === 'none'
          ? 'Set priority'
          : `Priority: ${PRIORITY_LABELS[current]}`)
  )

  async function openPop() {
    if (disabled) return
    open = true
    pos = { top: 0, left: 0, ready: false }
    await tick()
    place()
    const target =
      mode === 'single'
        ? popEl?.querySelector<HTMLElement>(`[data-prio="${current}"]`)
        : popEl?.querySelector<HTMLElement>('[data-prio]')
    target?.focus()
  }

  function close(refocus = true) {
    if (!open) return
    open = false
    if (refocus) triggerEl?.focus()
  }

  function place() {
    if (!triggerEl || !popEl) return
    const r = triggerEl.getBoundingClientRect()
    const w = popEl.offsetWidth
    const h = popEl.offsetHeight
    const m = 8
    const vw = window.innerWidth
    const vh = window.innerHeight
    let top = r.bottom + 4
    if (top + h > vh - m && r.top - 4 - h >= m) top = r.top - 4 - h
    top = Math.max(m, Math.min(top, vh - h - m))
    // Right-align to the trigger when it sits in the right half (card corner).
    let left = r.left + r.width / 2 > vw / 2 ? r.right - w : r.left
    left = Math.max(m, Math.min(left, vw - w - m))
    pos = { top, left, ready: true }
  }

  function apply(p: Priority) {
    if (mode === 'single') {
      if (p !== current) onPick?.(p)
      close()
    } else {
      const next = PRIORITIES.filter((q) => (q === p ? !selected.has(q) : selected.has(q)))
      onChange?.(next)
    }
  }

  function focusRow(delta: number, from: HTMLElement) {
    const rows = Array.from(popEl?.querySelectorAll<HTMLElement>('[data-row]') ?? [])
    const i = rows.indexOf(from)
    rows[(i + delta + rows.length) % rows.length]?.focus()
  }

  function onPopKeydown(e: KeyboardEvent) {
    const el = e.target as HTMLElement
    const handled = () => {
      e.preventDefault()
      e.stopPropagation()
    }
    if (e.key === 'Escape') {
      handled()
      close()
    } else if (e.key === 'ArrowDown') {
      handled()
      focusRow(1, el)
    } else if (e.key === 'ArrowUp') {
      handled()
      focusRow(-1, el)
    } else if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
      handled() // keep App's prev/next-task navigation out of the popover
    } else if (e.key === 'Tab') {
      close(false)
    } else if (/^[0-4]$/.test(e.key) && !e.ctrlKey && !e.metaKey && !e.altKey) {
      handled()
      const p = PRIORITIES[Number(e.key)]
      if (mode === 'multi') {
        apply(p)
        popEl?.querySelector<HTMLElement>(`[data-prio="${p}"]`)?.focus()
      } else apply(p)
    }
  }

  function onTriggerKeydown(e: KeyboardEvent) {
    if (open) return
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      e.stopPropagation()
      void openPop()
    }
  }

  function onWindowPointerDown(e: PointerEvent) {
    if (!open) return
    const t = e.target as Node
    if (popEl?.contains(t) || triggerEl?.contains(t)) return
    close(false)
  }

  function onScrollOrResize(e: Event) {
    if (!open) return
    if (e.type === 'scroll' && popEl?.contains(e.target as Node)) return
    close(false)
  }
</script>

<svelte:window
  onpointerdown={onWindowPointerDown}
  onresize={onScrollOrResize}
  onscrollcapture={onScrollOrResize}
/>

<button
  type="button"
  bind:this={triggerEl}
  {disabled}
  aria-haspopup="listbox"
  aria-expanded={open}
  aria-controls={open ? listId : undefined}
  aria-label={triggerTitle}
  title={triggerTitle}
  draggable="false"
  ondragstart={(e) => e.preventDefault()}
  onmousedown={(e) => e.stopPropagation()}
  onclick={(e) => {
    e.preventDefault()
    e.stopPropagation()
    if (open) close()
    else void openPop()
  }}
  onkeydown={onTriggerKeydown}
  class={variant === 'filter'
    ? `relative grid h-8 min-w-8 shrink-0 grid-flow-col place-items-center gap-1 rounded-control border px-1.5 transition-colors
       ${filterActive || open
         ? 'border-accent/50 bg-accent/15 text-accent-hi'
         : 'border-line-soft text-ink-3 hover:bg-card-hi hover:text-ink'}`
    : variant === 'field'
      ? `flex h-8 items-center gap-1.5 rounded-control border border-line-soft bg-field px-2.5 text-xs font-medium shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] transition-colors hover:bg-card-hi disabled:opacity-50
         ${open ? 'border-accent ring-2 ring-accent/25' : ''} ${PRIORITY_TEXT[current]}`
      : `flex shrink-0 items-center justify-center rounded-chip transition-opacity transition-colors disabled:cursor-default
         ${PRIORITY_TEXT[current]}
         ${current === 'none' && !open ? `${idleClass} hover:!opacity-100 hover:text-ink-2 focus-visible:!opacity-100` : ''}`}
>
  {#if variant === 'filter'}
    <PriorityIcon priority={filterActive ? values[values.length - 1] : 'high'} class="h-4 w-4" />
    {#if filterActive}
      <span class="font-mono text-[11px] leading-none">{values.length}</span>
    {/if}
  {:else}
    <PriorityIcon priority={current} class={iconClass} />
    {#if variant === 'field'}
      <span class={current === 'none' ? 'text-ink-3' : ''}>{PRIORITY_LABELS[current]}</span>
      <svg class="h-3.5 w-3.5 shrink-0 text-ink-3 transition-transform {open ? 'rotate-180' : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="m6 9 6 6 6-6" />
      </svg>
    {/if}
  {/if}
</button>

{#if open}
  <div
    use:portal
    bind:this={popEl}
    role="presentation"
    onkeydown={onPopKeydown}
    onmousedown={(e) => e.stopPropagation()}
    style:top="{pos.top}px"
    style:left="{pos.left}px"
    style:visibility={pos.ready ? 'visible' : 'hidden'}
    class="fixed z-[100] w-44 rounded-control border border-line bg-col py-1 text-ink shadow-lg"
  >
    <ul
      id={listId}
      role="listbox"
      aria-label={mode === 'multi' ? 'Filter by priority' : 'Priority'}
      aria-multiselectable={mode === 'multi'}
    >
      {#each ORDER as p (p)}
        {@const on = mode === 'single' ? current === p : selected.has(p)}
        <li role="presentation">
          <button
            type="button"
            role="option"
            data-row
            data-prio={p}
            aria-selected={on}
            onclick={(e) => {
              e.stopPropagation()
              apply(p)
            }}
            class="flex w-full items-center gap-2 px-2.5 py-1.5 text-left text-xs font-medium outline-none hover:bg-white/5 focus-visible:bg-white/10
              {on ? 'bg-accent/15 text-ink' : 'text-ink-2'}"
          >
            {#if mode === 'multi'}
              <span
                class="grid h-3.5 w-3.5 shrink-0 place-items-center rounded-chip border
                  {on ? 'border-accent bg-accent text-accent-ink' : 'border-ink-3/60'}"
                aria-hidden="true"
              >
                {#if on}
                  <svg class="h-2.5 w-2.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M3.5 8.5 6.5 11.5 12.5 4.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
                {/if}
              </span>
            {/if}
            <span class={PRIORITY_TEXT[p]}><PriorityIcon priority={p} class="h-3.5 w-3.5" /></span>
            <span class="flex-1">{p === 'none' && mode === 'single' ? 'None' : PRIORITY_LABELS[p]}</span>
            <kbd class="font-mono text-[10px] text-ink-3">{PRIORITIES.indexOf(p)}</kbd>
          </button>
        </li>
      {/each}
    </ul>
    {#if mode === 'multi'}
      <div class="mt-1 border-t border-line-soft px-1 pt-1">
        <button
          type="button"
          data-row
          disabled={!filterActive}
          onclick={(e) => {
            e.stopPropagation()
            onChange?.([])
            close()
          }}
          class="w-full rounded-chip px-1.5 py-1 text-left text-xs text-ink-2 outline-none hover:bg-white/5 focus-visible:bg-white/10 disabled:cursor-default disabled:opacity-40"
        >
          Clear filter
        </button>
      </div>
    {/if}
  </div>
{/if}
