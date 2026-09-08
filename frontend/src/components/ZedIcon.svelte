<script lang="ts">
  let {
    class: className = 'h-3 w-3',
    title = 'Zed'
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
  <path d="M4 4h7l9 16H13L4 4Z" />
  <path d="M13 4h7v4h-5" />
</svg>
