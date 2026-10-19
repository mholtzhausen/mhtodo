<script lang="ts">
  let {
    class: className = 'h-3 w-3',
    title = ''
  }: {
    class?: string
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
  aria-hidden="true"
>
  {#if title}<title>{title}</title>{/if}
  <path d="M12 3v18" />
  <path d="M3 12h18" />
  <path d="m5.6 5.6 12.8 12.8" />
  <path d="m18.4 5.6-12.8 12.8" />
</svg>
