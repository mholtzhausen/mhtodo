<script lang="ts">
  // Shared status control for the detail drawer and the new-task dialog —
  // replaces the old native <select> (design: tmp/design/01-slate.html).
  // Four chips with status dots; the active chip takes its status color.
  import type { Status } from '../lib/api'

  let { value, onPick }: { value: Status; onPick: (s: Status) => void } = $props()

  const OPTIONS: { s: Status; label: string; active: string; dot: string }[] = [
    {
      s: 'pending',
      label: 'Pending',
      active: 'border-st-pending/60 bg-st-pending/15 text-st-pending',
      dot: 'bg-st-pending'
    },
    { s: 'wip', label: 'Wip', active: 'border-st-wip/70 bg-st-wip/20 text-st-wip', dot: 'bg-st-wip' },
    {
      s: 'waiting',
      label: 'Waiting',
      active: 'border-st-waiting/60 bg-st-waiting/15 text-st-waiting',
      dot: 'bg-st-waiting'
    },
    { s: 'done', label: 'Done', active: 'border-st-done/60 bg-st-done/15 text-st-done', dot: 'bg-st-done' }
  ]
</script>

<div
  role="radiogroup"
  aria-label="Status"
  class="grid grid-cols-4 gap-1 rounded border border-line-soft bg-field p-1 shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)]"
>
  {#each OPTIONS as o (o.s)}
    <button
      type="button"
      role="radio"
      aria-checked={value === o.s}
      onclick={() => onPick(o.s)}
      class="flex items-center justify-center gap-1.5 rounded-[3px] border border-transparent px-0.5 py-[7px] text-xs font-medium text-ink-2 transition-colors hover:bg-white/5 hover:text-ink
        {value === o.s ? o.active : ''}"
    >
      <span class="h-[7px] w-[7px] flex-none rounded-full {o.dot}"></span>
      {o.label}
    </button>
  {/each}
</div>
