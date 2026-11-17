<script lang="ts">
  import type { Priority } from '../lib/priority'

  // Signal-bar glyph: none = three faint outline bars, low/medium/high fill
  // 1/2/3 bars, urgent = filled badge with "!". Colour comes from currentColor.
  let { priority, class: cls = 'h-3.5 w-3.5' }: { priority: Priority; class?: string } = $props()

  const filled = $derived(
    priority === 'low' ? 1 : priority === 'medium' ? 2 : priority === 'high' ? 3 : 0
  )
  const BARS = [
    { x: 2.5, y: 9, h: 4.5 },
    { x: 6.75, y: 6, h: 7.5 },
    { x: 11, y: 3, h: 10.5 }
  ]
</script>

<svg class={cls} viewBox="0 0 16 16" aria-hidden="true">
  {#if priority === 'urgent'}
    <rect x="1.5" y="1.5" width="13" height="13" rx="3" fill="currentColor" />
    <path d="M8 4.5v4.25" stroke="var(--color-card)" stroke-width="2" stroke-linecap="round" />
    <circle cx="8" cy="11.4" r="1.1" fill="var(--color-card)" />
  {:else}
    {#each BARS as b, i (i)}
      <rect
        x={b.x}
        y={b.y}
        width="2.5"
        height={b.h}
        rx="0.75"
        fill="currentColor"
        opacity={i < filled ? 1 : priority === 'none' ? 0.55 : 0.25}
      />
    {/each}
  {/if}
</svg>
