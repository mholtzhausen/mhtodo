<script lang="ts">
  import { renderMarkdown } from '../lib/markdown'

  let {
    source = '',
    class: className = '',
    empty = ''
  }: {
    source?: string | null
    class?: string
    /** Shown when source is empty (plain text, not markdown). */
    empty?: string
  } = $props()

  const html = $derived(renderMarkdown(source))
</script>

{#if html}
  <div class="md {className}">{@html html}</div>
{:else if empty}
  <div class="md md-empty {className}">{empty}</div>
{/if}
