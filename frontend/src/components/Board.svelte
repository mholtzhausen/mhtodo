<script lang="ts">
  import { relTime, STATUS_LABELS } from '../lib/format'
  import { api, errMsg, type Status } from '../lib/api'
  import HumanIcon from './HumanIcon.svelte'

  let {
    tasks,
    search,
    selectedId,
    showSubtasks,
    onSelect,
    onQuickAdd,
    onArchived,
    onError
  }: {
    tasks: any[]
    search: string
    selectedId: string | null
    showSubtasks: boolean
    onSelect: (id: string) => void
    onQuickAdd: (s: Status) => void
    onArchived?: (n: number) => void
    onError?: (msg: string) => void
  } = $props()

  const COLUMNS: { status: Status; label: string; dot: string; bar: string; edge: string }[] = [
    {
      status: 'pending',
      label: 'Pending',
      dot: 'bg-st-pending',
      bar: 'bg-st-pending',
      edge: 'border-l-st-pending'
    },
    { status: 'wip', label: 'In progress', dot: 'bg-st-wip', bar: 'bg-st-wip', edge: 'border-l-st-wip' },
    {
      status: 'waiting',
      label: 'Waiting',
      dot: 'bg-st-waiting',
      bar: 'bg-st-waiting',
      edge: 'border-l-st-waiting'
    },
    {
      status: 'review',
      label: 'Review',
      dot: 'bg-st-review',
      bar: 'bg-st-review',
      edge: 'border-l-st-review'
    },
    { status: 'done', label: 'Done', dot: 'bg-st-done', bar: 'bg-st-done', edge: 'border-l-st-done' }
  ]

  const childBar: Record<string, string> = {
    pending: 'bg-st-pending',
    wip: 'bg-st-wip',
    waiting: 'bg-st-waiting',
    review: 'bg-st-review',
    done: 'bg-st-done'
  }

  // Only roots occupy columns; children nest under their parent card.
  const byStatus = $derived.by(() => {
    const m: Record<string, any[]> = {}
    for (const c of COLUMNS) m[c.status] = []
    for (const t of tasks) {
      if (t.parent_id) continue
      ;(m[t.status] ??= []).push(t)
    }
    return m
  })

  const childrenOf = $derived.by(() => {
    const m: Record<string, any[]> = {}
    if (!showSubtasks) return m
    for (const t of tasks) {
      if (!t.parent_id) continue
      ;(m[t.parent_id] ??= []).push(t)
    }
    return m
  })

  let draggingId = $state<string | null>(null)
  let dragLifted = $state(false) // defer DOM removal until after dragstart completes
  let dragFrom = $state<Status | ''>('')
  let dropTarget = $state<Status | ''>('')
  let dropInsert = $state<{ status: Status; beforeId: string | null } | null>(null)
  let suppressClick = false

  // Hide native drag image — the in-column ghost shows placement instead.
  const emptyDragImage = typeof Image !== 'undefined' ? new Image() : null
  if (emptyDragImage) emptyDragImage.src = 'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7'

  const draggingTask = $derived(
    draggingId ? (tasks.find((t) => t.id === draggingId) ?? null) : null
  )

  /** Roots shown in a column; the dragged card is hidden once the drag session is active. */
  function columnRoots(col: Status): any[] {
    if (!dragLifted || !draggingId) return byStatus[col]
    return byStatus[col].filter((t) => t.id !== draggingId)
  }

  function clearDragState() {
    draggingId = null
    dragLifted = false
    dragFrom = ''
    dropTarget = ''
    dropInsert = null
  }

  function setDropInsert(col: Status, beforeId: string | null, crossColumn: boolean) {
    if (crossColumn) dropTarget = col
    else dropTarget = ''
    if (dropInsert?.status === col && dropInsert?.beforeId === beforeId) return
    dropInsert = { status: col, beforeId }
  }

  function liftDraggedCard() {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        if (draggingId) dragLifted = true
      })
    })
  }

  function onCardDragStart(e: DragEvent, t: any) {
    if (t.parent_id) {
      e.preventDefault()
      return
    }
    draggingId = t.id
    dragFrom = t.status
    dragLifted = false
    e.dataTransfer?.setData('text/plain', t.id)
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move'
      if (emptyDragImage) e.dataTransfer.setDragImage(emptyDragImage, 0, 0)
    }
    liftDraggedCard()
  }

  function onCardDragEnd() {
    suppressClick = true
    setTimeout(() => (suppressClick = false), 0)
    clearDragState()
  }

  function taskCardEl(lane: HTMLElement, id: string): HTMLElement | null {
    return lane.querySelector(`[data-task-id="${id}"]`)
  }

  /** Pointer-based insert; in-flow ghost reserves space (cards only — ghost is not a hit target). */
  function onLaneDragOver(e: DragEvent, col: Status) {
    if (!draggingId || !dragLifted) return
    e.preventDefault()
    e.stopPropagation()
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'

    const lane = e.currentTarget as HTMLElement
    const roots = columnRoots(col)
    const y = e.clientY
    const crossColumn = dragFrom !== col

    let beforeId: string | null = null
    for (const t of roots) {
      const el = taskCardEl(lane, t.id)
      if (!el) continue
      const rect = el.getBoundingClientRect()
      if (y < rect.top + rect.height / 2) {
        beforeId = t.id
        break
      }
    }

    setDropInsert(col, beforeId, crossColumn)
  }

  function onColumnDragOver(e: DragEvent, col: Status) {
    if (!draggingId) return
    e.preventDefault()
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  }

  async function reorderInLane(id: string, col: Status, beforeId: string | null) {
    try {
      await api.reorderTask(id, beforeId)
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  async function onLaneDrop(e: DragEvent, col: Status) {
    e.preventDefault()
    e.stopPropagation()
    const id = (draggingId ?? e.dataTransfer?.getData('text/plain')) || null
    const insert = dropInsert
    const from = dragFrom
    clearDragState()
    if (!id) return

    if (from === col) {
      await reorderInLane(id, col, insert?.status === col ? insert.beforeId : null)
      return
    }

    try {
      await api.setStatus(id, col)
      const beforeId = insert?.status === col ? insert.beforeId : null
      if (beforeId) {
        await api.reorderTask(id, beforeId)
      }
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  async function onColumnDrop(e: DragEvent, col: Status) {
    // Drops on header/chrome bubble here; lane body handles card-area drops.
    if ((e.target as HTMLElement | null)?.closest('[data-lane-body]')) return
    await onLaneDrop(e, col)
  }

  function onCardClick(t: any) {
    if (suppressClick) return
    onSelect(t.id)
  }

  function showGhostAt(col: Status, beforeId: string | null): boolean {
    return (
      dragLifted && dropInsert?.status === col && dropInsert.beforeId === beforeId && draggingTask !== null
    )
  }

  let archiving = $state(false)

  async function archiveAll() {
    if (archiving || byStatus.done.length === 0) return
    archiving = true
    try {
      const archived = await api.archiveDone()
      onArchived?.(archived.length)
    } catch (err) {
      onError?.(errMsg(err))
    } finally {
      archiving = false
    }
  }

  const rootCount = $derived(tasks.filter((t) => !t.parent_id).length)
</script>

{#snippet ghostCard(t: any, col: (typeof COLUMNS)[number])}
  <div
    class="ghost-card pointer-events-none rounded-md border border-dashed border-accent/70 border-l-2 bg-accent/10 shadow-[0_0_16px_rgba(123,140,255,0.25)] ring-2 ring-accent/40 {col.edge}"
    aria-hidden="true"
  >
    <div class="p-2.5 text-left">
      <p class="mb-2 line-clamp-2 text-[13.5px] font-medium leading-snug text-ink/90">{t.title}</p>
      <div class="flex items-center gap-2">
        <div class="h-[3px] flex-1 overflow-hidden rounded-full bg-white/10">
          <div class="h-full rounded-full {col.bar} opacity-80" style="width: {t.progress}%"></div>
        </div>
        <span class="font-mono text-[10.5px] text-ink-3/80">{t.progress}%</span>
        <span class="text-[10.5px] text-ink-3/80">{relTime(t.updated_at)}</span>
      </div>
    </div>
  </div>
{/snippet}

{#if rootCount === 0}
  <div class="flex h-full flex-col items-center justify-center gap-2 text-center">
    {#if search}
      <p class="text-sm text-ink-3">No tasks match “{search}”.</p>
      <p class="text-xs text-ink-3/70">Clear the search above to see everything.</p>
    {:else}
      <p class="text-sm text-ink-3">No tasks yet.</p>
      <p class="text-xs text-ink-3/70">Press <kbd>n</kbd> or use a column’s + button to create one.</p>
    {/if}
  </div>
{:else}
  <div class="grid h-full grid-cols-5 gap-3">
    {#each COLUMNS as col (col.status)}
      {@const roots = columnRoots(col.status)}
      <section
        ondragover={(e) => onColumnDragOver(e, col.status)}
        ondrop={(e) => onColumnDrop(e, col.status)}
        class="flex min-h-0 flex-col rounded-md border shadow-sm transition-colors duration-150
          {dropTarget === col.status
            ? 'border-accent/60 bg-accent/5'
            : 'border-line-soft bg-col'}"
      >
        <header class="flex flex-none items-center gap-2 px-3 py-2.5">
          <span class="h-2 w-2 flex-none rounded-full {col.dot}"></span>
          <h2 class="text-[11px] font-semibold uppercase tracking-[0.07em] text-ink-2">{col.label}</h2>
          <span
            class="rounded-[3px] border border-line-soft bg-white/5 px-1.5 py-[3px] font-mono text-[10px] leading-none text-ink-3"
          >
            {dragLifted && dragFrom === col.status && draggingId
              ? columnRoots(col.status).length
              : byStatus[col.status].length}
          </span>
          <div class="flex-1"></div>
          {#if col.status === 'done'}
            <button
              title="Archive all done tasks (reversible from List → Archived)"
              disabled={byStatus.done.length === 0 || archiving}
              onclick={archiveAll}
              class="rounded p-1 transition-colors hover:bg-white/5 hover:text-accent disabled:cursor-default disabled:opacity-30"
            >
              <svg
                class="h-3.5 w-3.5 {archiving ? 'animate-pulse text-accent' : 'text-ink-3'}"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
              >
                <rect x="3" y="4" width="18" height="5" rx="1" />
                <path d="M5 9v9a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V9" />
                <path d="M10 13h4" />
              </svg>
            </button>
          {/if}
          <button
            title={`New ${col.label.toLowerCase()} task`}
            onclick={() => onQuickAdd(col.status)}
            class="rounded p-1 text-sm leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-accent"
          >
            +
          </button>
        </header>

        <div
          data-lane-body
          class="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto px-2 py-2.5
            {draggingId ? 'board-dragging' : ''}"
          ondragover={(e) => onLaneDragOver(e, col.status)}
          ondrop={(e) => onLaneDrop(e, col.status)}
        >
          {#if roots.length === 0 && !showGhostAt(col.status, null)}
            <p class="rounded border border-dashed border-line px-2 py-4 text-center text-[11px] text-ink-3">
              no tasks
            </p>
          {:else}
            {#each roots as t (t.id)}
              {#if showGhostAt(col.status, t.id) && draggingTask}
                <div data-ghost-slot class="pointer-events-none">
                  {@render ghostCard(draggingTask, col)}
                </div>
              {/if}
              <div
                role="group"
                data-task-id={t.id}
                draggable="true"
                ondragstart={(e) => onCardDragStart(e, t)}
                ondragend={onCardDragEnd}
                class="relative rounded-md border border-l-2 shadow-sm transition-all duration-150 select-none
                  {col.edge}
                  {selectedId === t.id
                    ? 'border-accent bg-card-hi ring-1 ring-accent/70'
                    : 'border-line-soft bg-card hover:bg-card-hi hover:shadow-lg'}
                  {draggingId === t.id && !dragLifted ? 'cursor-grabbing opacity-60' : ''}"
              >
                {#if t.human_only}
                  <span
                    class="pointer-events-none absolute right-2 top-2 text-ink-2"
                    title="Human only — agents skip"
                  >
                    <HumanIcon class="h-3.5 w-3.5" />
                  </span>
                {/if}
                <button
                  type="button"
                  onclick={() => onCardClick(t)}
                  ondragstart={(e) => e.preventDefault()}
                  class="w-full cursor-grab p-2.5 text-left"
                >
                  <p class="mb-2 line-clamp-2 pr-5 text-[13.5px] font-medium leading-snug text-ink">{t.title}</p>
                  <div class="flex items-center gap-2">
                    <div class="h-[3px] flex-1 overflow-hidden rounded-full bg-white/10">
                      <div
                        class="h-full rounded-full {col.bar} transition-all duration-150"
                        style="width: {t.progress}%"
                      ></div>
                    </div>
                    <span class="font-mono text-[10.5px] text-ink-3">{t.progress}%</span>
                    <span class="text-[10.5px] text-ink-3">{relTime(t.updated_at)}</span>
                  </div>
                </button>
                {#if showSubtasks && (childrenOf[t.id]?.length ?? 0) > 0}
                  <ul class="space-y-1 border-t border-line-soft px-2 pb-2 pt-1.5">
                    {#each childrenOf[t.id] as c (c.id)}
                      <li>
                        <button
                          onclick={() => onSelect(c.id)}
                          class="flex w-full items-center gap-2 rounded border border-transparent px-1.5 py-1 text-left hover:border-line-soft hover:bg-white/5
                            {selectedId === c.id ? 'bg-accent/10' : ''}"
                        >
                          <span
                            class="h-1.5 w-1.5 flex-none rounded-full {childBar[c.status] ?? childBar.pending}"
                            title={STATUS_LABELS[c.status] ?? c.status}
                          ></span>
                          <span class="min-w-0 flex-1 truncate text-[11px] text-ink-2">{c.title}</span>
                          <span class="font-mono text-[10px] text-ink-3">{c.progress}%</span>
                        </button>
                      </li>
                    {/each}
                  </ul>
                {/if}
              </div>
            {/each}
            {#if showGhostAt(col.status, null) && draggingTask}
              <div data-ghost-slot class="pointer-events-none">
                {@render ghostCard(draggingTask, col)}
              </div>
            {/if}
          {/if}
        </div>
      </section>
    {/each}
  </div>
{/if}
