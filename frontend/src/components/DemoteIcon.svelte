<script lang="ts">
  let {
    class: className = 'h-3.5 w-3.5',
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
  stroke-linejoin="round"
  aria-hidden="true"
>
  {#if title}<title>{title}</title>{/if}
  <polyline points="15 10 20 15 15 20" />
  <path d="M4 4v7a4 4 0 0 0 4 4h12" />
</svg>
