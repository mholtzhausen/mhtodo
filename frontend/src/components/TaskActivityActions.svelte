<script lang="ts">
  import { onMount } from 'svelte'
  import { api, errMsg } from '../lib/api'
  import { shortId } from '../lib/format'
  import { type GUISettings } from '../lib/settings'
  import { openExternalUrl } from '../lib/openExternal'
  import { parsePRURLs, prURLLabel } from '../lib/prUrl'
  import HumanIcon from './HumanIcon.svelte'
  import PullRequestIcon from './PullRequestIcon.svelte'
  import SlackIcon from './SlackIcon.svelte'
  import ZedIcon from './ZedIcon.svelte'

  let {
    task,
    settings = null,
    zedBinaryOk = false,
    onError,
    onToast
  }: {
    task: {
      id: string
      title?: string
      status?: string
      cwd?: string
      human_only?: boolean
      include_in_report?: boolean
      archived_at?: string | null
      slack_thread?: string
      pr_url?: string
    }
    settings?: GUISettings | null
    zedBinaryOk?: boolean
    onError?: (msg: string) => void
    onToast?: (msg: string, kind?: 'error' | 'info') => void
  } = $props()

  const inWails = typeof window !== 'undefined' && !!(window as any).runtime

  let copying = $state(false)
  let copied = $state(false)
  let copyingRef = $state(false)
  let copiedRef = $state(false)
  let openingSlack = $state(false)
  let openingPR = $state(false)
  let prMenuOpen = $state(false)
  let prMenuRoot = $state<HTMLDivElement | null>(null)
  let prMenuBtn = $state<HTMLButtonElement | null>(null)
  let prMenuStyle = $state('')
  let openingZed = $state(false)
  let togglingHuman = $state(false)
  let togglingReport = $state(false)
  let archiving = $state(false)
  let zedCommand = $state('')
  let zedCommandLoading = $state(false)
  let copyTimer: ReturnType<typeof setTimeout> | undefined
  let copyRefTimer: ReturnType<typeof setTimeout> | undefined

  const showZed = $derived(
    !!settings?.zed.enabled && zedBinaryOk && !!(task.cwd ?? '').trim()
  )
  const zedTitle = $derived(
    `${zedCommand || 'Open in Zed'} — also copies ticket reference`
  )
  const slackURL = $derived((task.slack_thread ?? '').trim())
  const prURLs = $derived(parsePRURLs(task.pr_url))
  const includeInReport = $derived(task.include_in_report !== false)
  const canArchive = $derived(task.status === 'done' && !task.archived_at)
  const isArchived = $derived(!!task.archived_at)

  function reportError(msg: string) {
    onError?.(msg)
  }

  function toast(msg: string, kind: 'error' | 'info' = 'info') {
    onToast?.(msg, kind)
  }

  function ticketReference(): string {
    const id = shortId(task.id)
    const status = (task.status ?? 'pending').trim() || 'pending'
    const title = (task.title ?? '').trim() || '(untitled)'
    return [
      '',
      '---',
      `MHTodo Ticket: mhtodo://task/${id}`,
      `Status: ${status}`,
      `Title: ${title}`,
      'Instructions: (only if you start working on this task) ',
      ` - record your current claude session against the ticket: \`mhtodo edit ${id} --session <claude-session-uuid>\``,
      ' - remember to update subtasks and activity comments if you do work on this',
      ' - before starting: be on a clean `master` (or `main` if that is the default trunk); if the working tree is dirty, stop and clear it up with the user first; then create a new branch for this ticket and only then start work',
      '---',
      ''
    ].join('\n')
  }

  async function ensureZedCommand() {
    if (!showZed || zedCommand || zedCommandLoading) return
    zedCommandLoading = true
    try {
      zedCommand = await api.zedTicketCommand(task.id)
    } catch {
      zedCommand = ''
    } finally {
      zedCommandLoading = false
    }
  }

  function stop(e: MouseEvent) {
    e.stopPropagation()
    e.preventDefault()
  }

  async function clipboardSet(text: string): Promise<boolean> {
    const { ClipboardSetText } = await import('../../wailsjs/runtime/runtime.js')
    return ClipboardSetText(text)
  }

  async function copyMarkdown(e: MouseEvent) {
    stop(e)
    if (copying) return
    if (!inWails) {
      reportError('Running outside Wails — copy unavailable')
      return
    }
    copying = true
    try {
      const md = await api.taskMarkdownReport(task.id)
      const ok = await clipboardSet(md)
      if (ok) {
        copied = true
        toast('Task report copied', 'info')
        clearTimeout(copyTimer)
        copyTimer = setTimeout(() => {
          copied = false
        }, 1500)
      } else {
        reportError('Could not copy to clipboard')
      }
    } catch (err) {
      reportError(errMsg(err))
    } finally {
      copying = false
    }
  }

  async function copyTicketReference(): Promise<boolean> {
    if (!inWails) return false
    const ok = await clipboardSet(ticketReference())
    if (ok) {
      copiedRef = true
      clearTimeout(copyRefTimer)
      copyRefTimer = setTimeout(() => {
        copiedRef = false
      }, 1500)
    }
    return ok
  }

  async function copyTicketRef(e: MouseEvent) {
    stop(e)
    if (copyingRef) return
    if (!inWails) {
      reportError('Running outside Wails — copy unavailable')
      return
    }
    copyingRef = true
    try {
      const ok = await copyTicketReference()
      if (ok) {
        toast('Ticket reference copied', 'info')
      } else {
        reportError('Could not copy to clipboard')
      }
    } catch (err) {
      reportError(errMsg(err))
    } finally {
      copyingRef = false
    }
  }

  async function openSlack(e: MouseEvent) {
    stop(e)
    if (openingSlack || !slackURL) return
    openingSlack = true
    try {
      const ok = await openExternalUrl(slackURL)
      if (!ok) reportError('Invalid Slack thread URL')
    } finally {
      openingSlack = false
    }
  }

  function positionPRMenu() {
    const btn = prMenuBtn
    if (!btn) {
      prMenuStyle = ''
      return
    }
    const r = btn.getBoundingClientRect()
    const gap = 4
    const menuW = 224 // ~min-w-[14rem]
    // Prefer below the button; flip above if near the bottom of the viewport.
    const spaceBelow = window.innerHeight - r.bottom
    const openUp = spaceBelow < 160 && r.top > spaceBelow
    const top = openUp ? undefined : r.bottom + gap
    const bottom = openUp ? window.innerHeight - r.top + gap : undefined
    // Right-align to the button; clamp so it stays on-screen.
    let right = window.innerWidth - r.right
    right = Math.max(8, Math.min(right, window.innerWidth - menuW - 8))
    const parts = [`right:${right}px`, `width:${menuW}px`]
    if (top != null) parts.push(`top:${top}px`)
    if (bottom != null) parts.push(`bottom:${bottom}px`)
    prMenuStyle = parts.join(';')
  }

  async function openPR(e: MouseEvent) {
    stop(e)
    if (openingPR || prURLs.length === 0) return
    if (prURLs.length > 1) {
      if (!prMenuOpen) positionPRMenu()
      prMenuOpen = !prMenuOpen
      return
    }
    openingPR = true
    try {
      const ok = await openExternalUrl(prURLs[0])
      if (!ok) reportError('Invalid pull request URL')
    } finally {
      openingPR = false
    }
  }

  async function openPRChoice(e: MouseEvent, url: string) {
    stop(e)
    prMenuOpen = false
    if (openingPR) return
    openingPR = true
    try {
      const ok = await openExternalUrl(url)
      if (!ok) reportError('Invalid pull request URL')
    } finally {
      openingPR = false
    }
  }

  function onPRMenuPointerDown(e: PointerEvent) {
    if (!prMenuOpen || !prMenuRoot) return
    if (!prMenuRoot.contains(e.target as Node)) prMenuOpen = false
  }

  function onPRMenuKeydown(e: KeyboardEvent) {
    if (prMenuOpen && e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      prMenuOpen = false
    }
  }

  function onPRMenuScrollOrResize() {
    if (prMenuOpen) prMenuOpen = false
  }

  // Lane overflow scrolls don't bubble to window — capture so the fixed menu closes.
  onMount(() => {
    const close = () => {
      if (prMenuOpen) prMenuOpen = false
    }
    document.addEventListener('scroll', close, true)
    return () => document.removeEventListener('scroll', close, true)
  })

  async function openZed(e: MouseEvent) {
    stop(e)
    if (openingZed || !showZed) return
    openingZed = true
    try {
      await ensureZedCommand()
      // Copy first so the handoff block is ready before Zed (or tray-hide) takes focus.
      let refCopied = false
      try {
        refCopied = await copyTicketReference()
      } catch {
        refCopied = false
      }
      await api.openZedTicket(task.id)
      if (refCopied) {
        toast('Ticket reference copied', 'info')
      }
    } catch (err) {
      reportError(errMsg(err))
    } finally {
      openingZed = false
    }
  }

  async function toggleHumanOnly(e: MouseEvent) {
    stop(e)
    if (togglingHuman || isArchived) return
    togglingHuman = true
    const next = !task.human_only
    try {
      await api.update(task.id, { humanOnly: next })
    } catch (err) {
      reportError(errMsg(err))
    } finally {
      togglingHuman = false
    }
  }

  async function toggleIncludeInReport(e: MouseEvent) {
    stop(e)
    if (togglingReport || isArchived) return
    togglingReport = true
    const next = !includeInReport
    try {
      await api.update(task.id, { includeInReport: next })
    } catch (err) {
      reportError(errMsg(err))
    } finally {
      togglingReport = false
    }
  }

  async function archiveTask(e: MouseEvent) {
    stop(e)
    if (archiving || !canArchive) return
    archiving = true
    try {
      await api.archive(task.id)
      toast('Task archived', 'info')
    } catch (err) {
      reportError(errMsg(err))
    } finally {
      archiving = false
    }
  }

  const actionBtn =
    'rounded-control p-1 hover:bg-white/8 disabled:cursor-default disabled:opacity-40'
  const toggleBtn =
    'rounded-control p-0.5 disabled:cursor-default disabled:opacity-40'
  const toggleOff = 'text-ink-3/45 hover:bg-white/8 hover:text-ink-2'
  const toggleOn =
    'bg-accent/25 text-accent-hi shadow-[inset_0_1px_2px_rgba(0,0,0,0.4),inset_0_0_0_1px_rgba(255,255,255,0.08)] ring-1 ring-accent/40'

  function toggleClass(on: boolean) {
    return `${toggleBtn} ${on ? toggleOn : toggleOff}`
  }
</script>

<svelte:window
  onpointerdown={onPRMenuPointerDown}
  onkeydown={onPRMenuKeydown}
  onscroll={onPRMenuScrollOrResize}
  onresize={onPRMenuScrollOrResize}
/>

<div
  class="flex items-center gap-0.5"
  role="group"
  aria-label="Task actions"
  onpointerdown={(e) => e.stopPropagation()}
>
  <button
    type="button"
    onclick={copyMarkdown}
    disabled={copying}
    title={copied ? 'Copied' : 'Copy task markdown report'}
    aria-label={copied ? 'Copied task report' : 'Copy task markdown report'}
    class="{actionBtn} {copied ? toggleOn : 'text-ink-3 hover:text-ink'}"
  >
    {#if copied}
      <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <path d="M20 6 9 17l-5-5" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    {:else}
      <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z" stroke-linecap="round" stroke-linejoin="round" />
        <path d="M14 2v6h6" stroke-linecap="round" stroke-linejoin="round" />
        <path d="M8 13h8" stroke-linecap="round" />
        <path d="M8 17h5" stroke-linecap="round" />
      </svg>
    {/if}
  </button>

  <button
    type="button"
    onclick={copyTicketRef}
    disabled={copyingRef}
    title={copiedRef ? 'Copied' : 'Copy ticket reference'}
    aria-label={copiedRef ? 'Copied ticket reference' : 'Copy ticket reference'}
    class="{actionBtn} {copiedRef ? toggleOn : 'text-ink-3 hover:text-ink'}"
  >
    {#if copiedRef}
      <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <path d="M20 6 9 17l-5-5" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    {:else}
      <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" stroke-linecap="round" stroke-linejoin="round" />
        <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    {/if}
  </button>

  {#if slackURL}
    <button
      type="button"
      onclick={openSlack}
      disabled={openingSlack}
      title="Open Slack thread"
      aria-label="Open Slack thread"
      class="{actionBtn} text-ink-3 hover:text-ink"
    >
      <SlackIcon class="h-3 w-3" />
    </button>
  {/if}

  {#if prURLs.length > 0}
    <div class="relative" bind:this={prMenuRoot} data-pr-menu>
      <button
        type="button"
        bind:this={prMenuBtn}
        onclick={openPR}
        disabled={openingPR}
        title={prURLs.length > 1 ? 'Choose pull request' : 'Open pull request'}
        aria-label={prURLs.length > 1 ? 'Choose pull request' : 'Open pull request'}
        aria-expanded={prURLs.length > 1 ? prMenuOpen : undefined}
        aria-haspopup={prURLs.length > 1 ? 'menu' : undefined}
        class="{actionBtn} text-ink-3 hover:text-ink {prMenuOpen ? 'bg-white/8 text-ink' : ''}"
      >
        <PullRequestIcon class="h-3 w-3" title="" />
      </button>
      {#if prMenuOpen && prURLs.length > 1}
        <div
          role="menu"
          style={prMenuStyle}
          class="fixed z-50 max-h-[min(240px,40vh)] overflow-y-auto rounded-control border border-line bg-col py-0 shadow-md ring-1 ring-black/25"
        >
          {#each prURLs as url, i (url)}
            <button
              type="button"
              role="menuitem"
              title={url}
              onclick={(e) => void openPRChoice(e, url)}
              class="flex w-full items-center gap-2 truncate px-2.5 py-2 text-left font-mono text-[11px] text-ink-2 transition-colors hover:bg-accent/20 hover:text-ink
                {i > 0 ? 'border-t border-line-soft' : ''}"
            >
              <PullRequestIcon class="h-3 w-3 flex-none text-ink-3" title="" />
              <span class="min-w-0 truncate">{prURLLabel(url)}</span>
            </button>
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  {#if showZed}
    <button
      type="button"
      onclick={openZed}
      onmouseenter={() => void ensureZedCommand()}
      disabled={openingZed}
      title={zedTitle}
      aria-label="Open in Zed and copy ticket reference"
      class="{actionBtn} text-ink-3 hover:text-ink"
    >
      <ZedIcon class="h-3 w-3" title="" />
    </button>
  {/if}

  <div class="ml-0.5 flex items-center gap-1">
  <button
    type="button"
    onclick={toggleHumanOnly}
    disabled={togglingHuman || isArchived}
    title={isArchived
      ? 'Unarchive to change human-only'
      : task.human_only
        ? 'Human only (click to allow agents)'
        : 'Mark human-only (agents skip)'}
    aria-label={task.human_only ? 'Clear human-only' : 'Mark human-only'}
    aria-pressed={!!task.human_only}
    class={toggleClass(!!task.human_only)}
  >
    <HumanIcon class="h-3 w-3" />
  </button>

  <button
    type="button"
    onclick={toggleIncludeInReport}
    disabled={togglingReport || isArchived}
    title={isArchived
      ? 'Unarchive to change Slack report inclusion'
      : includeInReport
        ? 'Included in Slack report (click to exclude)'
        : 'Excluded from Slack report (click to include)'}
    aria-label={includeInReport ? 'Exclude from Slack report' : 'Include in Slack report'}
    aria-pressed={includeInReport}
    class={toggleClass(includeInReport)}
  >
    <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <path d="M8 6h13" stroke-linecap="round" />
      <path d="M8 12h13" stroke-linecap="round" />
      <path d="M8 18h13" stroke-linecap="round" />
      <path d="M3 6h.01" stroke-linecap="round" stroke-linejoin="round" />
      <path d="M3 12h.01" stroke-linecap="round" stroke-linejoin="round" />
      <path d="M3 18h.01" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
  </button>

  {#if canArchive}
    <button
      type="button"
      onclick={archiveTask}
      disabled={archiving}
      title="Archive this done task"
      aria-label="Archive task"
      class="{actionBtn} text-ink-3 hover:text-accent-hi"
    >
      <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <rect x="3" y="4" width="18" height="5" rx="1" />
        <path d="M5 9v9a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V9" stroke-linecap="round" stroke-linejoin="round" />
        <path d="M10 13h4" stroke-linecap="round" />
      </svg>
    </button>
  {/if}
  </div>
</div>
