<script lang="ts">
  import type { Status } from '../lib/api'
  import type { TaskStatusId } from '../lib/settings'

  let {
    value,
    onPick,
    disabled = false,
    /** When set, only these statuses are pickable (current value always shown). */
    visibleStatuses
  }: {
    value: Status
    onPick: (s: Status) => void
    disabled?: boolean
    visibleStatuses?: readonly TaskStatusId[] | readonly Status[] | null
  } = $props()

  const ALL_OPTIONS: { s: Status; label: string; short: string; active: string; dot: string }[] = [
    {
      s: 'icebox',
      label: 'Icebox',
      short: 'Ice',
      active: 'border-st-icebox/60 bg-st-icebox/15 text-st-icebox',
      dot: 'bg-st-icebox'
    },
    {
      s: 'pending',
      label: 'Pending',
      short: 'Pend',
      active: 'border-st-pending/60 bg-st-pending/15 text-st-pending',
      dot: 'bg-st-pending'
    },
    {
      s: 'wip',
      label: 'Wip',
      short: 'Wip',
      active: 'border-st-wip/70 bg-st-wip/20 text-st-wip',
      dot: 'bg-st-wip'
    },
    {
      s: 'waiting',
      label: 'Waiting',
      short: 'Wait',
      active: 'border-st-waiting/60 bg-st-waiting/15 text-st-waiting',
      dot: 'bg-st-waiting'
    },
    {
      s: 'review',
      label: 'Review',
      short: 'Rev',
      active: 'border-st-review/60 bg-st-review/15 text-st-review',
      dot: 'bg-st-review'
    },
    {
      s: 'pr',
      label: 'PR',
      short: 'PR',
      active: 'border-st-pr/60 bg-st-pr/15 text-st-pr',
      dot: 'bg-st-pr'
    },
    {
      s: 'done',
      label: 'Done',
      short: 'Done',
      active: 'border-st-done/60 bg-st-done/15 text-st-done',
      dot: 'bg-st-done'
    }
  ]

  const allowSet = $derived(
    visibleStatuses?.length ? new Set<string>(visibleStatuses) : null
  )

  const OPTIONS = $derived(
    allowSet
      ? ALL_OPTIONS.filter((o) => allowSet.has(o.s) || o.s === value)
      : ALL_OPTIONS
  )

  const colsClass = $derived(
    OPTIONS.length <= 4
      ? 'grid-cols-2 @[360px]:grid-cols-4'
      : OPTIONS.length <= 6
        ? 'grid-cols-3 @[360px]:grid-cols-6'
        : 'grid-cols-4 @[420px]:grid-cols-7'
  )

  function canPick(s: Status): boolean {
    if (disabled) return false
    if (!allowSet) return true
    return allowSet.has(s)
  }
</script>

<div class="@container">
  <div
    role="radiogroup"
    aria-label="Status"
    class="grid gap-1 rounded-control border border-line-soft bg-field p-1 shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)]
      {colsClass} {disabled ? 'opacity-55' : ''}"
  >
    {#each OPTIONS as o (o.s)}
      {@const pickable = canPick(o.s)}
      <button
        type="button"
        role="radio"
        aria-checked={value === o.s}
        aria-disabled={disabled || !pickable}
        disabled={disabled}
        title={pickable
          ? o.label
          : `${o.label} (lane hidden — re-enable in Settings → Statuses)`}
        onclick={() => {
          if (!pickable) return
          onPick(o.s)
        }}
        class="flex items-center justify-center gap-1 rounded-chip border border-transparent px-0.5 py-[7px] text-[11px] font-medium text-ink-2
          {pickable && !disabled ? 'hover:bg-white/5 hover:text-ink' : 'cursor-default'}
          {value === o.s ? o.active : ''}
          {!pickable ? 'opacity-70' : ''}"
      >
        <span class="h-[7px] w-[7px] flex-none rounded-full {o.dot}"></span>
        <span class="@[360px]:hidden">{o.short}</span>
        <span class="hidden @[360px]:inline">{o.label}</span>
      </button>
    {/each}
  </div>
</div>
