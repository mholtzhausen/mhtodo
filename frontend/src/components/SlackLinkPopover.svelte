<script lang="ts">
  import { tick } from 'svelte'
  import { api, errMsg, type SlackLink, type SlackLinkInfo } from '../lib/api'
  import { portal } from '../lib/portal'
  import { slackLinkLabel } from '../lib/slack'
  import SlackIcon from './SlackIcon.svelte'

  // "+" trigger and a portalled add form. Paste-and-go: the URL field is
  // focused (prefilled + selected from the clipboard when it already holds a
  // new Slack link), recognition is live, Enter saves from either field and
  // errors stay inline — the popover never closes on a bad link.
  let {
    existing = [],
    onAdd,
    disabled = false,
    compact = false
  }: {
    existing?: SlackLink[]
    /** Persist the link; throw to keep the popover open with the message inline. */
    onAdd: (link: SlackLink) => Promise<void>
    disabled?: boolean
    compact?: boolean
  } = $props()

  const inWails = typeof window !== 'undefined' && !!(window as any).runtime

  let open = $state(false)
  let triggerEl = $state<HTMLButtonElement | null>(null)
  let popEl = $state<HTMLDivElement | null>(null)
  let urlEl = $state<HTMLInputElement | null>(null)
  let descEl = $state<HTMLInputElement | null>(null)
  let pos = $state({ top: 0, left: 0, ready: false })
  let url = $state('')
  let desc = $state('')
  let info = $state<SlackLinkInfo | null>(null)
  let error = $state('')
  let saving = $state(false)
  let describeSeq = 0
  let describeTimer: ReturnType<typeof setTimeout> | undefined
  const formId = `slack-add-${Math.random().toString(36).slice(2, 9)}`

  const key = (u: string) => u.trim().replace(/#.*$/, '').replace(/\/+(\?|$)/, '$1').toLowerCase()
  const duplicate = $derived(
    url.trim() ? existing.find((l) => key(l.url) === key(url)) ?? null : null
  )

  async function describe(raw: string): Promise<SlackLinkInfo> {
    if (!inWails) {
      const ok = /^(https?:\/\/([a-z0-9-]+\.)*slack\.com\/|slack:\/\/)/i.test(raw.trim())
      return ok
        ? { valid: true, kind: 'link', workspace: '', label: slackLinkLabel(raw), error: '' }
        : { valid: false, kind: '', workspace: '', label: '', error: 'not a Slack link' }
    }
    return api.describeSlackLink(raw)
  }

  function scheduleDescribe() {
    error = ''
    clearTimeout(describeTimer)
    const raw = url
    if (!raw.trim()) {
      info = null
      return
    }
    describeTimer = setTimeout(async () => {
      const seq = ++describeSeq
      try {
        const res = await describe(raw)
        if (seq === describeSeq) info = res
      } catch {
        /* recognition is advisory; save still validates */
      }
    }, 90)
  }

  async function clipboardSlackURL(): Promise<string> {
    if (!inWails) return ''
    try {
      const { ClipboardGetText } = await import('../../wailsjs/runtime/runtime.js')
      const text = ((await ClipboardGetText()) ?? '').trim()
      if (!text || /\s/.test(text) || text.length > 2000) return ''
      const res = await api.describeSlackLink(text)
      if (!res.valid || existing.some((l) => key(l.url) === key(text))) return ''
      return text
    } catch {
      return ''
    }
  }

  async function openPop() {
    if (disabled) return
    url = ''
    desc = ''
    info = null
    error = ''
    open = true
    pos = { top: 0, left: 0, ready: false }
    await tick()
    place()
    urlEl?.focus()
    const clip = await clipboardSlackURL()
    if (clip && open && !url) {
      url = clip
      scheduleDescribe()
      await tick()
      urlEl?.select()
    }
  }

  function close(refocus = true) {
    if (!open) return
    open = false
    clearTimeout(describeTimer)
    if (refocus) triggerEl?.focus()
  }

  function place() {
    if (!triggerEl || !popEl) return
    const r = triggerEl.getBoundingClientRect()
    const w = popEl.offsetWidth
    const h = popEl.offsetHeight
    const m = 8
    const vw = window.innerWidth
    const vh = window.innerHeight
    let top = r.bottom + 4
    if (top + h > vh - m && r.top - 4 - h >= m) top = r.top - 4 - h
    top = Math.max(m, Math.min(top, vh - h - m))
    let left = r.left + r.width / 2 > vw / 2 ? r.right - w : r.left
    left = Math.max(m, Math.min(left, vw - w - m))
    pos = { top, left, ready: true }
  }

  async function save() {
    if (saving) return
    const u = url.trim()
    if (!u) {
      error = 'Paste a Slack link first'
      urlEl?.focus()
      return
    }
    const res = info && url === u ? info : await describe(u).catch(() => null)
    if (res && !res.valid) {
      error = `Not a Slack link — ${res.error || 'want https://….slack.com/… or slack://…'}`
      urlEl?.focus()
      return
    }
    saving = true
    try {
      await onAdd({ url: u, description: desc.trim() })
      close()
    } catch (e) {
      error = errMsg(e)
    } finally {
      saving = false
    }
  }

  function onUrlPaste() {
    // After a paste, hop to the description so "paste, type, Enter" flows.
    setTimeout(async () => {
      scheduleDescribe()
      if (url.trim() && !desc) {
        await tick()
        descEl?.focus()
      }
    }, 0)
  }

  function onPopKeydown(e: KeyboardEvent) {
    // Keep App's global shortcuts (Esc-to-hide, arrows, letters) out.
    e.stopPropagation()
    if (e.key === 'Escape') {
      e.preventDefault()
      close()
    } else if (e.key === 'Enter' && (e.target as HTMLElement).tagName === 'INPUT') {
      e.preventDefault()
      void save()
    }
  }

  function onWindowPointerDown(e: PointerEvent) {
    if (!open) return
    const t = e.target as Node
    if (popEl?.contains(t) || triggerEl?.contains(t)) return
    close(false)
  }

  function onResize() {
    if (open) place()
  }

  function onScroll(e: Event) {
    if (!open || popEl?.contains(e.target as Node)) return
    close(false)
  }

  const KIND_ICON_LABEL: Record<string, string> = {
    thread: 'Thread',
    channel: 'Channel',
    dm: 'Direct message',
    group: 'Group DM',
    workspace: 'Workspace',
    link: 'Slack link'
  }
</script>

<svelte:window onpointerdown={onWindowPointerDown} onresize={onResize} onscrollcapture={onScroll} />

<button
  type="button"
  bind:this={triggerEl}
  {disabled}
  aria-haspopup="dialog"
  aria-expanded={open}
  aria-controls={open ? formId : undefined}
  aria-label="Add Slack conversation"
  title="Add Slack conversation"
  onmousedown={(e) => e.stopPropagation()}
  onclick={(e) => {
    e.preventDefault()
    e.stopPropagation()
    if (open) close()
    else void openPop()
  }}
  class="grid shrink-0 place-items-center rounded-chip border border-line-soft text-ink-3 transition-colors hover:bg-card-hi hover:text-ink disabled:cursor-default disabled:opacity-40
    {compact ? 'h-5 w-5' : 'h-6 w-6'} {open ? 'border-accent/60 bg-accent/15 text-accent-hi' : ''}"
>
  <svg class={compact ? 'h-3 w-3' : 'h-3.5 w-3.5'} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.25" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg>
</button>

{#if open}
  <div
    use:portal
    bind:this={popEl}
    id={formId}
    role="dialog"
    aria-label="Add Slack conversation"
    tabindex="-1"
    onkeydown={onPopKeydown}
    onmousedown={(e) => e.stopPropagation()}
    onclick={(e) => e.stopPropagation()}
    style:top="{pos.top}px"
    style:left="{pos.left}px"
    style:visibility={pos.ready ? 'visible' : 'hidden'}
    class="fixed z-[100] w-80 rounded-control border border-line bg-col p-3 text-ink shadow-lg"
  >
    <div class="mb-2 flex items-center gap-1.5 text-xs font-semibold text-ink-2">
      <SlackIcon class="h-3.5 w-3.5" title="" />
      Add Slack conversation
    </div>

    <input
      bind:this={urlEl}
      bind:value={url}
      oninput={scheduleDescribe}
      onpaste={onUrlPaste}
      placeholder="Paste a Slack link"
      spellcheck="false"
      autocomplete="off"
      aria-label="Slack link URL"
      aria-invalid={!!error}
      class="w-full rounded-control border bg-field px-2.5 py-1.5 font-mono text-[11px] text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:font-sans placeholder:text-ink-3 focus:outline-none focus:ring-2
        {error || (info && !info.valid) ? 'border-danger/70 focus:ring-danger/25' : 'border-line-soft focus:border-accent focus:ring-accent/25'}"
    />

    <div class="mt-1 min-h-[16px] text-[11px] leading-4" aria-live="polite">
      {#if error}
        <span class="text-danger">{error}</span>
      {:else if info && !info.valid}
        <span class="text-danger">Not a Slack link{info.error ? ` — ${info.error}` : ''}</span>
      {:else if info?.valid}
        <span class="text-ink-2">
          <span class="text-accent-hi">✓</span>
          {KIND_ICON_LABEL[info.kind] ?? 'Slack link'}{#if info.workspace}<span class="text-ink-3"> · {info.workspace}</span>{/if}
          {#if duplicate}<span class="text-ink-3"> — already linked{desc.trim() ? '; description will update' : ''}</span>{/if}
        </span>
      {:else}
        <span class="text-ink-3">Thread, channel or DM link — https://….slack.com/… or slack://…</span>
      {/if}
    </div>

    <input
      bind:this={descEl}
      bind:value={desc}
      oninput={() => (error = '')}
      placeholder="What's this conversation? (optional)"
      maxlength="200"
      aria-label="Description (optional)"
      class="mt-1.5 w-full rounded-control border border-line-soft bg-field px-2.5 py-1.5 text-xs text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
    />

    <div class="mt-2.5 flex items-center justify-between gap-2">
      <span class="text-[10px] text-ink-3"><kbd class="font-mono">Enter</kbd> save · <kbd class="font-mono">Esc</kbd> cancel</span>
      <div class="flex gap-1.5">
        <button
          type="button"
          onclick={() => close()}
          class="rounded-control px-2.5 py-1 text-xs text-ink-2 hover:bg-white/5"
        >Cancel</button>
        <button
          type="button"
          onclick={() => void save()}
          disabled={saving || !url.trim()}
          class="rounded-control bg-accent px-2.5 py-1 text-xs font-semibold text-accent-ink hover:opacity-90 disabled:opacity-40"
        >{duplicate ? 'Update' : 'Add'}</button>
      </div>
    </div>
  </div>
{/if}
