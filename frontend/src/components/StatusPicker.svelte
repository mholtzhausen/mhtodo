<script lang="ts">
  import type { Status } from '../lib/api'

  let { value, onPick }: { value: Status; onPick: (s: Status) => void } = $props()

  const OPTIONS: { s: Status; label: string; short: string; active: string; dot: string }[] = [
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
      s: 'done',
      label: 'Done',
      short: 'Done',
      active: 'border-st-done/60 bg-st-done/15 text-st-done',
      dot: 'bg-st-done'
    }
  ]
</script>

<div class="@container">
  <div
    role="radiogroup"
    aria-label="Status"
    class="grid grid-cols-3 gap-1 rounded-control border border-line-soft bg-field p-1 shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)]
      @[320px]:grid-cols-5"
  >
    {#each OPTIONS as o (o.s)}
      <button
        type="button"
        role="radio"
        aria-checked={value === o.s}
        title={o.label}
        onclick={() => onPick(o.s)}
        class="flex items-center justify-center gap-1 rounded-chip border border-transparent px-0.5 py-[7px] text-[11px] font-medium text-ink-2 hover:bg-white/5 hover:text-ink
          {value === o.s ? o.active : ''}"
      >
        <span class="h-[7px] w-[7px] flex-none rounded-full {o.dot}"></span>
        <span class="@[320px]:hidden">{o.short}</span>
        <span class="hidden @[320px]:inline">{o.label}</span>
      </button>
    {/each}
  </div>
</div>
