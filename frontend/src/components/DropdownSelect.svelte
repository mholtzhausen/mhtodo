<script lang="ts" generics="T extends string">
  import { tick } from 'svelte'

  export type DropdownOption<T extends string> = {
    value: T
    label: string
    title?: string
    /** Extra classes for the trigger when this option is selected. */
    activeClass?: string
  }

  let {
    value,
    options,
    onChange,
    ariaLabel,
    minWidth = '7.5rem',
    disabled = false
  }: {
    value: T
    options: DropdownOption<T>[]
    onChange: (v: T) => void
    ariaLabel: string
    minWidth?: string
    disabled?: boolean
  } = $props()

  let open = $state(false)
  let rootEl = $state<HTMLDivElement | null>(null)
  let triggerEl = $state<HTMLButtonElement | null>(null)
  let listId = `dropdown-${Math.random().toString(36).slice(2, 9)}`

  const selected = $derived(options.find((o) => o.value === value) ?? options[0])

  function close() {
    open = false
  }

  function pick(v: T) {
    if (v !== value) onChange(v)
    close()
    triggerEl?.focus()
  }

  function onTriggerKeydown(e: KeyboardEvent) {
    if (disabled) return
    if (e.key === 'Escape') {
      e.preventDefault()
      close()
      return
    }
    if (e.key === 'ArrowDown' || e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      open = true
      tick().then(() => {
        rootEl?.querySelector<HTMLElement>('[data-option-selected="true"]')?.focus()
      })
    }
  }

  function onOptionKeydown(e: KeyboardEvent, idx: number) {
    if (e.key === 'Escape') {
      e.preventDefault()
      close()
      triggerEl?.focus()
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      focusOption(idx + 1)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      focusOption(idx - 1)
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      pick(options[idx].value)
    }
  }

  function focusOption(idx: number) {
    const wrapped = ((idx % options.length) + options.length) % options.length
    rootEl?.querySelectorAll<HTMLElement>('[data-dropdown-option]')[wrapped]?.focus()
  }

  function onWindowPointerDown(e: PointerEvent) {
    if (!open || !rootEl) return
    if (!rootEl.contains(e.target as Node)) close()
  }

  function onWindowKeydown(e: KeyboardEvent) {
    if (open && e.key === 'Escape') {
      e.preventDefault()
      close()
      triggerEl?.focus()
    }
  }
</script>

<svelte:window onpointerdown={onWindowPointerDown} onkeydown={onWindowKeydown} />

<div class="relative" bind:this={rootEl}>
  <button
    type="button"
    bind:this={triggerEl}
    {disabled}
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-controls={listId}
    aria-label={ariaLabel}
    onclick={() => !disabled && (open = !open)}
    onkeydown={onTriggerKeydown}
    style:min-width={minWidth}
    class="flex h-8 items-center justify-between gap-2 rounded border border-line-soft bg-field px-2.5 text-left text-xs font-medium shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] transition-colors
      hover:bg-card-hi disabled:cursor-not-allowed disabled:opacity-50
      {open ? 'border-accent ring-2 ring-accent/25' : ''}
      {selected?.activeClass ?? 'text-ink'}"
  >
    <span class="truncate">{selected?.label ?? '—'}</span>
    <svg
      class="h-3.5 w-3.5 shrink-0 text-ink-3 transition-transform {open ? 'rotate-180' : ''}"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <path d="m6 9 6 6 6-6" />
    </svg>
  </button>

  {#if open}
    <ul
      id={listId}
      role="listbox"
      aria-label={ariaLabel}
      class="absolute left-0 top-[calc(100%+4px)] z-50 max-h-64 min-w-full overflow-y-auto rounded border border-line bg-col py-1 shadow-xl"
    >
      {#each options as opt, idx (opt.value)}
        <li role="presentation">
          <button
            type="button"
            role="option"
            data-dropdown-option
            data-option-selected={value === opt.value ? 'true' : undefined}
            aria-selected={value === opt.value}
            title={opt.title}
            onclick={() => pick(opt.value)}
            onkeydown={(e) => onOptionKeydown(e, idx)}
            class="flex w-full items-center px-2.5 py-1.5 text-left text-xs font-medium transition-colors hover:bg-white/5
              {value === opt.value ? (opt.activeClass ?? 'bg-accent/15 text-accent-hi') : 'text-ink-2'}"
          >
            {opt.label}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>
