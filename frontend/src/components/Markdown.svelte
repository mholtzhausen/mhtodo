<script lang="ts">
  import { renderMarkdown } from '../lib/markdown'
  import { openExternalUrl } from '../lib/openExternal'

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

  /** Keep http(s) links out of the Wails webview — open in the system browser. */
  function onLinkClick(e: MouseEvent) {
    const el = e.target
    if (!(el instanceof Element)) return
    const a = el.closest('a')
    if (!(a instanceof HTMLAnchorElement)) return
    const href = a.getAttribute('href')?.trim()
    if (!href) return
    let parsed: URL
    try {
      parsed = new URL(href, window.location.href)
    } catch {
      return
    }
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return
    e.preventDefault()
    e.stopPropagation()
    void openExternalUrl(parsed.href)
  }
</script>

{#if html}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="md {className}" onclick={onLinkClick}>{@html html}</div>
{:else if empty}
  <div class="md md-empty {className}">{empty}</div>
{/if}
