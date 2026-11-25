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
  <path d="m18 9-6-6-6 6" />
  <path d="M12 3v14" />
  <path d="M5 21h14" />
</svg>
