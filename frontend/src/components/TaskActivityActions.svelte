<script lang="ts">
  import { api, errMsg } from '../lib/api'
  import { claudeIconVisible } from '../lib/claudeIntegration'
  import HumanIcon from './HumanIcon.svelte'

  let {
    task,
    onError,
    onToast
  }: {
    task: {
      id: string
      status?: string
      cwd?: string
      human_only?: boolean
      include_in_report?: boolean
      archived_at?: string | null
    }
    onError?: (msg: string) => void
    onToast?: (msg: string, kind?: 'error' | 'info') => void
  } = $props()

  const inWails = typeof window !== 'undefined' && !!(window as any).runtime

  let copying = $state(false)
  let copied = $state(false)
  let openingClaude = $state(false)
  let togglingHuman = $state(false)
  let togglingReport = $state(false)
  let archiving = $state(false)
  let claudeActive = $state(false)
  let guiSettings = $state<Awaited<ReturnType<typeof api.getSettings>> | null>(null)
  let copyTimer: ReturnType<typeof setTimeout> | undefined

  const showClaude = $derived(
    guiSettings ? claudeIconVisible(task, guiSettings) : false
  )
  const includeInReport = $derived(task.include_in_report !== false)
  const canArchive = $derived(task.status === 'done' && !task.archived_at)

  function reportError(msg: string) {
    onError?.(msg)
  }

  function toast(msg: string, kind: 'error' | 'info' = 'info') {
    onToast?.(msg, kind)
  }

  $effect(() => {
    void task.id
    void task.cwd
    void task.human_only
    void task.status
    copied = false
    claudeActive = false
    guiSettings = null
    ;(async () => {
      try {
        const settings = await api.getSettings()
        guiSettings = settings
        if (!claudeIconVisible(task, settings)) return
        if (settings.claude.enabled) {
          claudeActive = await api.checkBinary(settings.claude.binary)
        }
      } catch {
        claudeActive = false
      }
    })()
  })

  function stop(e: MouseEvent) {
    e.stopPropagation()
    e.preventDefault()
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
      const { ClipboardSetText } = await import('../../wailsjs/runtime/runtime.js')
      const ok = await ClipboardSetText(md)
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

  async function openClaude(e: MouseEvent) {
    stop(e)
    if (openingClaude || !showClaude) return
    openingClaude = true
    try {
      await api.openHerdrTicket(task.id)
    } catch (err) {
      reportError(errMsg(err))
    } finally {
      openingClaude = false
    }
  }

  async function toggleHumanOnly(e: MouseEvent) {
    stop(e)
    if (togglingHuman) return
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
    if (togglingReport) return
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
    'rounded p-1 transition-all hover:bg-white/8 disabled:cursor-default disabled:opacity-40'
  const toggleBtn =
    'rounded p-0.5 transition-all disabled:cursor-default disabled:opacity-40'
  const toggleOff = 'text-ink-3/45 hover:bg-white/8 hover:text-ink-2'
  const toggleOn =
    'bg-accent/25 text-accent-hi shadow-[inset_0_1px_2px_rgba(0,0,0,0.4),inset_0_0_0_1px_rgba(255,255,255,0.08)] ring-1 ring-accent/40'

  function toggleClass(on: boolean) {
    return `${toggleBtn} ${on ? toggleOn : toggleOff}`
  }
</script>

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

  {#if showClaude}
    <button
      type="button"
      onclick={openClaude}
      disabled={openingClaude}
      title={claudeActive ? 'Open in Herdr with Claude' : 'Open in Herdr'}
      aria-label={claudeActive ? 'Open in Herdr with Claude' : 'Open in Herdr'}
      class="{actionBtn}
        {claudeActive ? 'text-[#d97757] hover:text-[#e88a6a]' : 'text-ink-3 hover:text-accent-hi'}"
    >
      {#if claudeActive}
        <svg class="h-3 w-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="currentColor" aria-hidden="true">
          <path
            d="m19.6 66.5 19.7-11 .3-1-.3-.5h-1l-3.3-.2-11.2-.3L14 53l-9.5-.5-2.4-.5L0 49l.2-1.5 2-1.3 2.9.2 6.3.5 9.5.6 6.9.4L38 49.1h1.6l.2-.7-.5-.4-.4-.4L29 41l-10.6-7-5.6-4.1-3-2-1.5-2-.6-4.2 2.7-3 3.7.3.9.2 3.7 2.9 8 6.1L37 36l1.5 1.2.6-.4.1-.3-.7-1.1L33 25l-6-10.4-2.7-4.3-.7-2.6c-.3-1-.4-2-.4-3l3-4.2L28 0l4.2.6L33.8 2l2.6 6 4.1 9.3L47 29.9l2 3.8 1 3.4.3 1h.7v-.5l.5-7.2 1-8.7 1-11.2.3-3.2 1.6-3.8 3-2L61 2.6l2 2.9-.3 1.8-1.1 7.7L59 27.1l-1.5 8.2h.9l1-1.1 4.1-5.4 6.9-8.6 3-3.5L77 13l2.3-1.8h4.3l3.1 4.7-1.4 4.9-4.4 5.6-3.7 4.7-5.3 7.1-3.2 5.7.3.4h.7l12-2.6 6.4-1.1 7.6-1.3 3.5 1.6.4 1.6-1.4 3.4-8.2 2-9.6 2-14.3 3.3-.2.1.2.3 6.4.6 2.8.2h6.8l12.6 1 3.3 2 1.9 2.7-.3 2-5.1 2.6-6.8-1.6-16-3.8-5.4-1.3h-.8v.4l4.6 4.5 8.3 7.5L89 80.1l.5 2.4-1.3 2-1.4-.2-9.2-7-3.6-3-8-6.8h-.5v.7l1.8 2.7 9.8 14.7.5 4.5-.7 1.4-2.6 1-2.7-.6-5.8-8-6-9-4.7-8.2-.5.4-2.9 30.2-1.3 1.5-3 1.2-2.5-2-1.4-3 1.4-6.2 1.6-8 1.3-6.4 1.2-7.9.7-2.6v-.2H49L43 72l-9 12.3-7.2 7.6-1.7.7-3-1.5.3-2.8L24 86l10-12.8 6-7.9 4-4.6-.1-.5h-.3L17.2 77.4l-4.7.6-2-2 .2-3 1-1 8-5.5Z"
          />
        </svg>
      {:else}
        <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <rect x="2" y="3" width="20" height="14" rx="2" />
          <path d="M8 21h8" stroke-linecap="round" />
          <path d="M12 17v4" stroke-linecap="round" />
          <path d="m7 8 3 3-3 3" stroke-linecap="round" stroke-linejoin="round" />
          <path d="M13 14h4" stroke-linecap="round" />
        </svg>
      {/if}
    </button>
  {/if}

  <div class="ml-0.5 flex items-center gap-1">
  <button
    type="button"
    onclick={toggleHumanOnly}
    disabled={togglingHuman}
    title={task.human_only ? 'Human only (click to allow agents)' : 'Mark human-only (agents skip)'}
    aria-label={task.human_only ? 'Clear human-only' : 'Mark human-only'}
    aria-pressed={!!task.human_only}
    class={toggleClass(!!task.human_only)}
  >
    <HumanIcon class="h-3 w-3" />
  </button>

  <button
    type="button"
    onclick={toggleIncludeInReport}
    disabled={togglingReport}
    title={includeInReport ? 'Included in Slack report (click to exclude)' : 'Excluded from Slack report (click to include)'}
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
