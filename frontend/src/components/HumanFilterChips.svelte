<script lang="ts">
  import HumanIcon from './HumanIcon.svelte'
  import type { HumanFilter } from '../lib/humanFilter'

  let {
    value,
    onChange
  }: {
    value: HumanFilter
    onChange: (v: HumanFilter) => void
  } = $props()

  const chips: { value: HumanFilter; label: string; title: string }[] = [
    { value: 'all', label: 'All', title: 'Show all tasks' },
    { value: 'exclude', label: 'Agents', title: 'Hide human-only tasks' },
    { value: 'only', label: 'Human', title: 'Show human-only tasks' }
  ]
</script>

<div
  class="flex gap-1 rounded border border-line-soft bg-field p-1 shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)]"
  role="group"
  aria-label="Human-only filter"
>
  {#each chips as chip (chip.value)}
    <button
      type="button"
      title={chip.title}
      aria-pressed={value === chip.value}
      onclick={() => onChange(chip.value)}
      class="flex items-center gap-1 rounded-[3px] border border-transparent px-2.5 py-1 text-xs font-medium transition-colors hover:bg-white/5 hover:text-ink
        {value === chip.value ? 'border-accent/60 bg-accent/15 text-accent-hi' : 'text-ink-2'}"
    >
      {#if chip.value === 'only'}
        <HumanIcon class="h-3 w-3" />
      {/if}
      {chip.label}
    </button>
  {/each}
</div>
