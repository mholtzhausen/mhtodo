<script lang="ts">
  let {
    class: className = 'h-4 w-4',
    filled = false,
    title = ''
  }: {
    class?: string
    filled?: boolean
    title?: string
  } = $props()
</script>

<svg
  class={className}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width="2"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
>
  {#if title}<title>{title}</title>{/if}
  <circle cx="12" cy="12" r="9" />
  <circle cx="12" cy="12" r="5" />
  <circle cx="12" cy="12" r="1.5" fill={filled ? 'currentColor' : 'none'} />
  {#if filled}<circle cx="12" cy="12" r="2.5" fill="currentColor" stroke="none" />{/if}
</svg>
