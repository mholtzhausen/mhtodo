<script lang="ts">
  // Three-state boolean for template presets: null (not part of the template),
  // false (explicitly off) and true (explicitly on).
  //
  // A native checkbox cannot express this cleanly — `indeterminate` reads as
  // "partially true" rather than "not set", renders in the WebKitGTK default
  // style instead of the accent-tinted style used elsewhere in the dialog, and
  // clicking it jumps straight to true with no way back. This is a
  // role="checkbox" button with aria-checked="mixed" for the unset state,
  // sized to match the native checkboxes it sits alongside.
  let {
    value,
    label,
    /** What the field resolves to when the template leaves it unset. */
    fallback,
    hint = '',
    onChange
  }: {
    value: boolean | null
    label: string
    fallback: boolean
    hint?: string
    onChange: (v: boolean | null) => void
  } = $props()

  // unset -> on -> off -> unset. The common intent (turn it on) is one click,
  // and returning to unset is never more than two.
  function cycle() {
    if (value === null) onChange(true)
    else if (value === true) onChange(false)
    else onChange(null)
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault()
      cycle()
    }
  }

  const state = $derived(value === null ? 'unset' : value ? 'on' : 'off')
  const stateLabel = $derived(
    value === null ? `default (${fallback ? 'on' : 'off'})` : value ? 'on' : 'off'
  )
  const boxClass = $derived(
    state === 'on'
      ? 'border-accent bg-accent text-accent-ink'
      : state === 'off'
        ? 'border-line-soft bg-field'
        : 'border-dashed border-line-soft bg-field/40 opacity-60'
  )
</script>

<div class="flex items-start gap-2.5">
  <button
    type="button"
    role="checkbox"
    aria-checked={value === null ? 'mixed' : value}
    aria-label="{label}: {stateLabel}"
    onclick={cycle}
    onkeydown={onKeydown}
    title="Click to cycle: default → on → off"
    class="mt-[2px] grid h-4 w-4 flex-none place-items-center rounded border transition-colors focus:outline-none focus:ring-2 focus:ring-accent/25 {boxClass}"
  >
    {#if state === 'on'}
      <svg
        class="h-3 w-3"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="3"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M20 6 9 17l-5-5" />
      </svg>
    {/if}
  </button>

  <div class="min-w-0 flex-1 leading-tight">
    <button
      type="button"
      onclick={cycle}
      class="text-left text-sm text-ink-2 transition-colors hover:text-ink"
    >
      {label}
      <!-- The state is always named in text, so the three states never depend
           on reading the glyph alone. -->
      <span
        class="ml-1 text-xs {state === 'unset' ? 'text-ink-3' : 'text-accent'}"
      >— {stateLabel}</span>
    </button>
    {#if hint}
      <p class="mt-0.5 text-xs text-ink-3">{hint}</p>
    {/if}
  </div>
</div>
