<script lang="ts">
  import { absList, absShort, relTime, shortId, STATUS_LABELS } from '../lib/format'
  import type { Activity, Task } from '../lib/api'
  import Markdown from './Markdown.svelte'

  let {
    activities,
    tasks,
    search = '',
    selectedTaskIds,
    onToggleTask,
    onSelectTask,
    onClearFilter
  }: {
    activities: Activity[]
    tasks: Task[]
    search?: string
    selectedTaskIds: string[]
    onToggleTask: (id: string) => void
    onSelectTask: (id: string) => void
    onClearFilter: () => void
  } = $props()

  let filterOpen = $state(false)
  let filterRoot = $state<HTMLDivElement | null>(null)

  /** Mouse-following ticket tooltip (null when not hovering a title link). */
  let tip = $state<{
    x: number
    y: number
    task: Task
  } | null>(null)

  let tipEl = $state<HTMLDivElement | null>(null)
  /** Measured tip box; used so flip-above doesn't leave a huge gap. */
  let tipH = $state(0)

  const TIP_W = 320
  const TIP_GAP = 8 // cursor ↔ tip edge (same whether above or below)

  const taskById = $derived.by(() => {
    const m: Record<string, Task> = {}
    for (const t of tasks) m[t.id] = t
    return m
  })

  const searchQ = $derived(search.trim().toLowerCase())

  function taskMatchesSearch(t: Task | undefined): boolean {
    if (!searchQ) return true
    if (!t) return false
    return (
      t.title.toLowerCase().includes(searchQ) ||
      t.id.toLowerCase().includes(searchQ) ||
      shortId(t.id).toLowerCase().includes(searchQ) ||
      (t.description ?? '').toLowerCase().includes(searchQ)
    )
  }

  $effect(() => {
    if (!tip || !tipEl) {
      tipH = 0
      return
    }
    // Re-measure whenever the hovered task (content) changes.
    void tip.task.id
    void tip.task.description
    void tip.task.feedback
    tipH = tipEl.offsetHeight
  })

  function tipStyle(t: NonNullable<typeof tip>): string {
    const vw = typeof window !== 'undefined' ? window.innerWidth : 1280
    const vh = typeof window !== 'undefined' ? window.innerHeight : 800
    // Prefer measured height; fall back to a tight estimate until first paint.
    const h = tipH > 0 ? tipH : 72
    let left = t.x + TIP_GAP
    let top = t.y + TIP_GAP
    if (left + TIP_W + 8 > vw) left = t.x - TIP_W - TIP_GAP
    if (left < 8) left = 8
    if (top + h + TIP_GAP > vh) top = t.y - h - TIP_GAP
    if (top < 8) top = 8
    return `left:${left}px;top:${top}px;width:${TIP_W}px`
  }

  function showTip(e: MouseEvent, task: Task | undefined) {
    if (!task) return
    tip = { x: e.clientX, y: e.clientY, task }
  }

  function moveTip(e: MouseEvent) {
    if (!tip) return
    tip = { ...tip, x: e.clientX, y: e.clientY }
  }

  function hideTip() {
    tip = null
  }

  function onWindowPointerDown(e: PointerEvent) {
    if (!filterOpen || !filterRoot) return
    if (!filterRoot.contains(e.target as Node)) filterOpen = false
  }

  function onWindowKeydown(e: KeyboardEvent) {
    if (filterOpen && e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      filterOpen = false
    }
  }

  const ticketOptions = $derived(
    [...tasks]
      .filter((t) => taskMatchesSearch(t))
      .sort((a, b) => a.title.localeCompare(b.title))
  )

  const filtered = $derived.by(() => {
    let list = activities
    if (selectedTaskIds.length > 0) {
      list = list.filter((a) => selectedTaskIds.includes(a.task_id))
    }
    if (searchQ) {
      list = list.filter((a) => {
        const task = taskById[a.task_id]
        if (taskMatchesSearch(task)) return true
        return (
          (a.activity ?? '').toLowerCase().includes(searchQ) ||
          (a.comment ?? '').toLowerCase().includes(searchQ)
        )
      })
    }
    return list
  })
</script>

<svelte:window onpointerdown={onWindowPointerDown} onkeydown={onWindowKeydown} />

<div class="flex h-full flex-col gap-3">
  <div
    class="relative flex flex-none items-center gap-3"
    bind:this={filterRoot}
    data-ticket-filter
    data-open={filterOpen ? 'true' : 'false'}
  >
    <button
      type="button"
      onclick={() => (filterOpen = !filterOpen)}
      aria-expanded={filterOpen}
      class="rounded border border-line-soft bg-field px-3 py-1.5 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] hover:bg-card-hi"
    >
      Filter by ticket
      {#if selectedTaskIds.length > 0}
        <span class="ml-1 font-mono text-xs text-accent-hi">({selectedTaskIds.length})</span>
      {/if}
    </button>
    {#if selectedTaskIds.length > 0}
      <button type="button" onclick={onClearFilter} class="text-xs text-ink-3 hover:text-ink">
        Clear
      </button>
    {/if}
    <div class="flex-1"></div>
    <span class="font-mono text-[11px] text-ink-3">
      {filtered.length} {filtered.length === 1 ? 'entry' : 'entries'}
    </span>

    {#if filterOpen}
      <div
        class="absolute left-0 top-full z-20 mt-1 max-h-64 w-80 overflow-y-auto rounded border border-line bg-card p-2 shadow-xl"
      >
        {#each ticketOptions as t (t.id)}
          <label class="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm hover:bg-white/5">
            <input
              type="checkbox"
              checked={selectedTaskIds.includes(t.id)}
              onchange={() => onToggleTask(t.id)}
              class="rounded border-line"
            />
            <span class="min-w-0 flex-1 truncate text-ink">{t.title}</span>
            <span class="font-mono text-[10px] text-ink-3">{shortId(t.id)}</span>
          </label>
        {:else}
          <p class="px-2 py-3 text-xs text-ink-3">No tickets</p>
        {/each}
      </div>
    {/if}
  </div>

  <div class="min-h-0 flex-1 overflow-y-auto rounded-md border border-line-soft bg-col shadow-sm">
    {#if filtered.length === 0}
      <div class="flex h-40 flex-col items-center justify-center gap-1 text-center">
        <p class="text-sm text-ink-3">No activity yet.</p>
        <p class="text-xs text-ink-3/70">Agents can post with <code class="font-mono">mhtodo activity add</code>.</p>
      </div>
    {:else}
      <ul class="space-y-1.5 p-2">
        {#each filtered as a (a.id)}
          {@const task = taskById[a.task_id]}
          {@const parent = task?.parent_id ? taskById[task.parent_id] : undefined}
          <li class="rounded bg-white/[0.03] px-3 py-2.5 hover:bg-white/[0.06]">
            <div class="mb-1.5 flex gap-3">
              <div class="flex w-28 flex-none flex-col gap-0.5 pt-0.5">
                <span class="text-xs text-ink-2">{relTime(a.created_at)}</span>
                <span class="font-mono text-[10px] text-ink-3">{absList(a.created_at)}</span>
              </div>
              <div class="flex min-w-0 flex-1 items-baseline gap-1.5 truncate text-sm font-medium">
                {#if task?.parent_id}
                  <button
                    type="button"
                    onclick={() => onSelectTask(task.parent_id!)}
                    onmouseenter={(e) => showTip(e, parent)}
                    onmousemove={moveTip}
                    onmouseleave={hideTip}
                    class="max-w-[55%] shrink truncate text-left text-accent-hi hover:underline"
                  >
                    {parent?.title ?? shortId(task.parent_id)}
                  </button>
                  <span class="shrink-0 text-ink-3" aria-hidden="true">></span>
                  <button
                    type="button"
                    onclick={() => onSelectTask(a.task_id)}
                    onmouseenter={(e) => showTip(e, task)}
                    onmousemove={moveTip}
                    onmouseleave={hideTip}
                    class="min-w-0 truncate text-left text-accent-hi hover:underline"
                  >
                    {task.title}
                  </button>
                {:else}
                  <button
                    type="button"
                    onclick={() => onSelectTask(a.task_id)}
                    onmouseenter={(e) => showTip(e, task)}
                    onmousemove={moveTip}
                    onmouseleave={hideTip}
                    class="truncate text-left text-accent-hi hover:underline"
                  >
                    {task?.title ?? shortId(a.task_id)}
                  </button>
                {/if}
              </div>
            </div>
            {#if a.activity || a.comment}
              <div class="flex min-w-0 items-start gap-2">
                {#if a.activity}
                  <span
                    class="inline-block max-w-full shrink-0 break-words rounded-full border border-accent/40 bg-accent/15 px-2 py-0.5 text-xs font-medium leading-snug text-accent-hi"
                  >
                    {a.activity}
                  </span>
                  {#if a.comment}
                    <span class="shrink-0 self-baseline text-ink-3" aria-hidden="true">—</span>
                  {/if}
                {/if}
                {#if a.comment}
                  <div class="min-w-0 flex-1 text-sm text-ink-2">
                    <Markdown source={a.comment} />
                  </div>
                {/if}
              </div>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</div>

{#if tip}
  {@const t = tip.task}
  <div
    bind:this={tipEl}
    class="pointer-events-none fixed z-50 max-h-[min(360px,70vh)] overflow-hidden rounded-md border border-line bg-card shadow-2xl ring-1 ring-black/30"
    style={tipStyle(tip)}
    role="tooltip"
  >
    <div class="border-b border-line-soft bg-chrome px-3 py-2">
      <p class="font-mono text-[11px] leading-snug text-ink-2">
        <span class="text-accent-hi">{shortId(t.id)}</span>
        <span class="text-ink-3"> - </span>
        <span>{absShort(t.created_at)}</span>
        <span class="text-ink-3"> - </span>
        <span class="capitalize">{STATUS_LABELS[t.status] ?? t.status}</span>
        <span class="text-ink-3"> - </span>
        <span>{t.progress}%</span>
      </p>
    </div>
    <div class="max-h-[300px] overflow-y-auto px-3 py-2.5">
      {#if t.description?.trim()}
        <Markdown source={t.description} class="text-xs text-ink-2" />
      {:else}
        <p class="text-xs italic text-ink-3">No description</p>
      {/if}
      {#if t.feedback?.trim()}
        <hr class="my-2.5 border-0 border-t border-line-soft" />
        <Markdown source={t.feedback} class="text-xs text-accent-hi/90" />
      {/if}
    </div>
  </div>
{/if}
