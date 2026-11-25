<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg, type Task } from '../lib/api'
  import { shortId, STATUS_LABELS } from '../lib/format'
  import { fuzzyMatchIndices, fuzzyScore } from '../lib/fuzzy'
  import PriorityIcon from './PriorityIcon.svelte'

  let {
    open,
    task = null,
    busy = false,
    onPick,
    onClose,
    onError
  }: {
    open: boolean
    task?: any | null
    busy?: boolean
    onPick: (parent: Task) => void | Promise<void>
    onClose: () => void
    onError?: (msg: string) => void
  } = $props()

  const statusDot: Record<string, string> = {
    icebox: 'bg-st-icebox',
    pending: 'bg-st-pending',
    wip: 'bg-st-wip',
    waiting: 'bg-st-waiting',
    review: 'bg-st-review',
    pr: 'bg-st-pr',
    done: 'bg-st-done'
  }

  // Active pipeline tasks surface first when the query is empty; icebox and done follow.
  const emptyStatusRank: Record<string, number> = {
    wip: 0,
    review: 1,
    pr: 2,
    waiting: 3,
    pending: 4,
    icebox: 5,
    done: 6
  }

  let candidates = $state<Task[]>([])
  let loading = $state(false)
  let query = $state('')
  let cursor = $state(0)
  let searchEl = $state<HTMLInputElement | null>(null)
  let listEl = $state<HTMLDivElement | null>(null)

  // Reload all non-archived top-level tasks whenever the modal opens so board
  // filters/search never hide a valid target parent.
  $effect(() => {
    if (!open || !task) {
      query = ''
      cursor = 0
      return
    }
    const selfId = task.id
    const currentParentId = task.parent_id ?? ''
    loading = true
    api
      .list({
        rootsOnly: true,
        includeDone: true,
        includeHumanOnly: true,
        sort: 'board'
      })
      .then((all) => {
        candidates = all
          .filter((t) => t.id !== selfId && t.id !== currentParentId)
          .sort((a, b) => {
            const ra = emptyStatusRank[a.status] ?? 9
            const rb = emptyStatusRank[b.status] ?? 9
            if (ra !== rb) return ra - rb
            return 0
          })
      })
      .catch((e) => onError?.(errMsg(e)))
      .finally(() => {
        loading = false
      })
  })

  $effect(() => {
    if (open && searchEl) {
      requestAnimationFrame(() => searchEl?.focus())
    }
  })

  interface RankedCandidate {
    task: Task
    score: number
    highlights: Set<number>
  }

  function scoreCandidate(q: string, t: Task): RankedCandidate | null {
    const titleScore = fuzzyScore(q, t.title)
    if (titleScore >= 0) {
      const idx = fuzzyMatchIndices(q, t.title) ?? []
      return { task: t, score: titleScore + 1000, highlights: new Set(idx) }
    }

    // Multi-word out-of-order fuzzy match against title (e.g. "bug auth").
    const tokens = q.split(/\s+/).filter(Boolean)
    if (tokens.length > 1) {
      let total = 0
      const merged = new Set<number>()
      let ok = true
      for (const tok of tokens) {
        const s = fuzzyScore(tok, t.title)
        if (s < 0) {
          ok = false
          break
        }
        total += s
        for (const i of fuzzyMatchIndices(tok, t.title) ?? []) merged.add(i)
      }
      if (ok) {
        return { task: t, score: total + 500, highlights: merged }
      }
    }

    // Fallback: match against short ID, full UUID (dashes ignored), or git repo.
    const cleanQ = q.replace(/-/g, '').toLowerCase()
    const cleanID = t.id.replace(/-/g, '').toLowerCase()
    if (cleanQ.length >= 2 && cleanID.includes(cleanQ)) {
      return { task: t, score: 400, highlights: new Set() }
    }
    if (t.git_repo) {
      const repoScore = fuzzyScore(q, t.git_repo)
      if (repoScore >= 0) {
        return { task: t, score: repoScore + 100, highlights: new Set() }
      }
    }
    return null
  }

  const ranked = $derived.by<RankedCandidate[]>(() => {
    const q = query.trim()
    if (!q) {
      return candidates.map((t) => ({ task: t, score: 0, highlights: new Set<number>() }))
    }
    const out: RankedCandidate[] = []
    for (const t of candidates) {
      const hit = scoreCandidate(q, t)
      if (hit) out.push(hit)
    }
    out.sort((a, b) => b.score - a.score)
    return out
  })

  // Keep cursor clamped as query narrows or expands.
  $effect(() => {
    if (cursor > ranked.length - 1) cursor = Math.max(0, ranked.length - 1)
  })

  // Reset cursor to top match whenever the search query changes.
  $effect(() => {
    void query
    cursor = 0
  })

  function move(delta: number) {
    if (!ranked.length) return
    cursor = (cursor + delta + ranked.length) % ranked.length
    listEl?.querySelectorAll('[data-row]')[cursor]?.scrollIntoView({ block: 'nearest' })
  }

  function confirmSelection(row?: RankedCandidate) {
    if (busy || !row) return
    void onPick(row.task)
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      if (!busy) onClose()
      return
    }
    if (e.key === 'ArrowDown' || (e.ctrlKey && e.key === 'n')) {
      e.preventDefault()
      move(1)
      return
    }
    if (e.key === 'ArrowUp' || (e.ctrlKey && e.key === 'p')) {
      e.preventDefault()
      move(-1)
      return
    }
    if (e.key === 'Enter') {
      e.preventDefault()
      confirmSelection(ranked[cursor])
      return
    }
  }

  /** Split a title into highlighted/plain runs for clean fuzzy-match rendering. */
  function titleSegments(title: string, highlights: Set<number>): { text: string; hit: boolean }[] {
    if (highlights.size === 0) return [{ text: title, hit: false }]
    const chars = Array.from(title)
    const segs: { text: string; hit: boolean }[] = []
    for (let i = 0; i < chars.length; i++) {
      const hit = highlights.has(i)
      const last = segs[segs.length - 1]
      if (last && last.hit === hit) last.text += chars[i]
      else segs.push({ text: chars[i], hit })
    }
    return segs
  }
</script>

{#if open && task}
  <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
  <div
    class="fixed inset-0 z-[55] flex items-start justify-center bg-black/55 px-4 pt-[min(14vh,110px)] pb-4"
    onclick={() => !busy && onClose()}
  >
    <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
    <div
      in:fly={{ y: -8, duration: 90 }}
      role="dialog"
      aria-modal="true"
      aria-label={task.parent_id ? 'Move sub-task to another parent' : 'Make sub-task of another task'}
      onkeydown={onKeydown}
      onclick={(e) => e.stopPropagation()}
      class="relative flex max-h-[76vh] w-full max-w-lg flex-col overflow-hidden rounded-panel border border-line bg-col shadow-md"
    >
      <div
        class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-accent/50 to-transparent"
        aria-hidden="true"
      ></div>

      <div class="flex flex-none items-start gap-2 border-b border-line-soft px-4 py-3">
        <div class="min-w-0 flex-1">
          <h2 class="text-sm font-semibold text-ink">
            {task.parent_id ? 'Move sub-task to another parent' : 'Make sub-task of…'}
          </h2>
          <p class="mt-1 flex items-center gap-1.5 text-xs text-ink-3">
            <span
              class="h-1.5 w-1.5 flex-none rounded-full {statusDot[task.status] ?? statusDot.pending}"
            ></span>
            <span class="truncate text-ink-2">{task.title}</span>
            <span class="flex-none font-mono text-[10px] text-ink-3">{shortId(task.id)}</span>
          </p>
        </div>
        <button
          type="button"
          disabled={busy}
          onclick={onClose}
          title="Close (esc)"
          class="rounded-control p-1 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink disabled:opacity-40"
        >
          ✕
        </button>
      </div>

      <div class="flex-none border-b border-line-soft p-3">
        <div class="relative">
          <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-ink-3">
            <svg
              class="h-4 w-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <circle cx="11" cy="11" r="7" />
              <path d="m21 21-4.3-4.3" />
            </svg>
          </span>
          <input
            bind:this={searchEl}
            bind:value={query}
            disabled={busy}
            placeholder="Fuzzy search parent task by title, ID, or repo…"
            aria-label="Search parent task"
            class="w-full rounded-control border border-line-soft bg-field py-2 pl-9 pr-8 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
          />
          {#if query}
            <button
              type="button"
              onclick={() => {
                query = ''
                searchEl?.focus()
              }}
              title="Clear search"
              aria-label="Clear search"
              class="absolute right-2 top-1/2 -translate-y-1/2 rounded-control p-1 text-xs leading-none text-ink-3 hover:bg-white/5 hover:text-ink"
            >
              ✕
            </button>
          {/if}
        </div>
      </div>

      <div
        bind:this={listEl}
        class="min-h-0 flex-1 overflow-y-auto p-1.5"
        role="listbox"
        aria-label="Eligible parent tasks"
        tabindex="-1"
      >
        {#if loading}
          <p class="px-3 py-6 text-center text-sm text-ink-3">Loading tasks…</p>
        {:else if !candidates.length}
          <p class="px-3 py-6 text-center text-sm text-ink-3">
            No other top-level tasks available as a parent.
          </p>
        {:else if !ranked.length}
          <p class="px-3 py-6 text-center text-sm text-ink-3">
            No task matches “{query.trim()}”.
          </p>
        {:else}
          {#each ranked as row, i (row.task.id)}
            {@const t = row.task}
            <button
              type="button"
              data-row
              role="option"
              disabled={busy}
              aria-selected={i === cursor}
              onclick={() => confirmSelection(row)}
              onmouseenter={() => (cursor = i)}
              class="flex w-full items-center gap-2.5 rounded-control border px-2.5 py-2 text-left transition-colors disabled:opacity-50
                {i === cursor
                  ? 'border-accent/40 bg-accent/15'
                  : 'border-transparent hover:bg-white/5'}"
            >
              <span
                class="h-2 w-2 flex-none rounded-full {statusDot[t.status] ?? statusDot.pending}"
                title={STATUS_LABELS[t.status] ?? t.status}
              ></span>
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm text-ink">
                  {#each titleSegments(t.title, row.highlights) as seg}
                    {#if seg.hit}<mark class="bg-transparent font-semibold text-accent-hi">{seg.text}</mark>{:else}{seg.text}{/if}
                  {/each}
                </p>
                {#if t.git_repo}
                  <p class="mt-0.5 truncate font-mono text-[10.5px] text-ink-3">{t.git_repo}</p>
                {/if}
              </div>
              {#if t.priority && t.priority !== 'none'}
                <PriorityIcon priority={t.priority} class="h-3.5 w-3.5 flex-none" />
              {/if}
              <span
                class="flex-none rounded-chip border border-line-soft bg-field/70 px-1.5 py-0.5 text-[10px] text-ink-3"
              >
                {STATUS_LABELS[t.status] ?? t.status}
              </span>
              <span class="flex-none font-mono text-[10px] text-ink-3/80">
                {shortId(t.id)}
              </span>
            </button>
          {/each}
        {/if}
      </div>

      <div class="flex flex-none items-center justify-between gap-2 border-t border-line-soft px-3.5 py-2 text-[11px] text-ink-3">
        <span><kbd>↑</kbd><kbd>↓</kbd> navigate · <kbd>↵</kbd> select · <kbd>esc</kbd> cancel</span>
        <span class="font-mono text-[10.5px]">
          {#if query.trim()}
            {ranked.length}/{candidates.length}
          {:else}
            {candidates.length} {candidates.length === 1 ? 'task' : 'tasks'}
          {/if}
        </span>
      </div>
    </div>
  </div>
{/if}
