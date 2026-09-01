<script lang="ts">
  let {
    class: className = 'h-4 w-4',
    title = 'Human only'
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
  <circle cx="12" cy="8" r="4" />
  <path d="M5.5 20a6.5 6.5 0 0 1 13 0" />
</svg>
