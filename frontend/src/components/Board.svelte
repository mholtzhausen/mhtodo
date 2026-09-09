<script lang="ts">
  import { relTime, STATUS_LABELS } from '../lib/format'
  import { api, errMsg, type Status } from '../lib/api'
  import { sortSubtasksByCreated } from '../lib/boardOrder'
  import type { GUISettings } from '../lib/settings'
  import TaskActivityActions from './TaskActivityActions.svelte'

  let {
    tasks,
    search,
    selectedId,
    showSubtasks,
    statusFilter = '',
    humanFilterEmpty = false,
    archiveDoneSubtasks = false,
    settings = null,
    claudeBinaryOk = false,
    zedBinaryOk = false,
    onSelect,
    onQuickAdd,
    onArchived,
    onError,
    onToast
  }: {
    tasks: any[]
    search: string
    selectedId: string | null
    showSubtasks: boolean
    statusFilter?: Status | '' | 'archived'
    humanFilterEmpty?: boolean
    archiveDoneSubtasks?: boolean
    settings?: GUISettings | null
    claudeBinaryOk?: boolean
    zedBinaryOk?: boolean
    onSelect: (id: string) => void
    onQuickAdd: (s: Status) => void
    onArchived?: (n: number) => void
    onError?: (msg: string) => void
    onToast?: (msg: string, kind?: 'error' | 'info') => void
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

  const visibleColumns = $derived(
    statusFilter && statusFilter !== 'archived'
      ? COLUMNS.filter((c) => c.status === statusFilter)
      : COLUMNS
  )

  const childBar: Record<string, string> = {
    pending: 'bg-st-pending',
    wip: 'bg-st-wip',
    waiting: 'bg-st-waiting',
    review: 'bg-st-review',
    done: 'bg-st-done'
  }

  const COLLAPSED_COL_W = 28
  const EXPANDED_COL_MIN = 200
  const COL_GAP = 12
  const COLLAPSE_STATUSES: Status[] = ['pending', 'wip', 'waiting', 'review', 'done']
  const COLLAPSE_STORAGE_KEY = 'mhtodo.collapsedColumns'

  function loadCollapsedColumns(): Partial<Record<Status, boolean>> {
    try {
      const raw = localStorage.getItem(COLLAPSE_STORAGE_KEY)
      if (!raw) return {}
      const parsed = JSON.parse(raw)
      if (!Array.isArray(parsed)) return {}
      const out: Partial<Record<Status, boolean>> = {}
      for (const s of parsed) {
        if (COLLAPSE_STATUSES.includes(s as Status)) out[s as Status] = true
      }
      return out
    } catch {
      return {}
    }
  }

  function persistCollapsedColumns(state: Partial<Record<Status, boolean>>) {
    try {
      const ids = COLLAPSE_STATUSES.filter((s) => state[s])
      localStorage.setItem(COLLAPSE_STORAGE_KEY, JSON.stringify(ids))
    } catch {
      /* ignore quota / private mode */
    }
  }

  let collapsedColumns = $state<Partial<Record<Status, boolean>>>(loadCollapsedColumns())

  function isCollapsed(status: Status): boolean {
    return !!collapsedColumns[status]
  }

  function toggleCollapsed(status: Status) {
    collapsedColumns = { ...collapsedColumns, [status]: !collapsedColumns[status] }
    persistCollapsedColumns(collapsedColumns)
  }

  function onCollapsedActivate(status: Status) {
    if (suppressCollapseClick || draggingId) return
    toggleCollapsed(status)
  }

  const boardGridStyle = $derived.by(() => {
    const cols = visibleColumns
    if (cols.length === 0) return undefined
    const singleExpanded = cols.length === 1 && !isCollapsed(cols[0].status)
    if (singleExpanded) return undefined
    const parts = cols.map((c) =>
      isCollapsed(c.status) ? `${COLLAPSED_COL_W}px` : `minmax(${EXPANDED_COL_MIN}px, 1fr)`
    )
    let minW = Math.max(0, cols.length - 1) * COL_GAP
    for (const c of cols) {
      minW += isCollapsed(c.status) ? COLLAPSED_COL_W : EXPANDED_COL_MIN
    }
    return `grid-template-columns: ${parts.join(' ')}; min-width: ${minW}px`
  })

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
    for (const pid of Object.keys(m)) {
      m[pid] = sortSubtasksByCreated(m[pid])
    }
    return m
  })

  let draggingId = $state<string | null>(null)
  let dragLifted = $state(false) // defer DOM removal until after dragstart completes
  let dragFrom = $state<Status | ''>('')
  let dropTarget = $state<Status | ''>('')
  let dropInsert = $state<{ status: Status; beforeId: string | null } | null>(null)
  let suppressClick = false
  let suppressCollapseClick = false
  let dragOverRaf = 0
  let pendingDrop: { col: Status; beforeId: string | null; crossColumn: boolean } | null = null

  function armCollapseClickSuppress() {
    suppressCollapseClick = true
    setTimeout(() => (suppressCollapseClick = false), 50)
  }

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
    pendingDrop = null
    if (dragOverRaf) {
      cancelAnimationFrame(dragOverRaf)
      dragOverRaf = 0
    }
  }

  function setDropInsert(col: Status, beforeId: string | null, crossColumn: boolean) {
    if (crossColumn) dropTarget = col
    else dropTarget = ''
    if (dropInsert?.status === col && dropInsert?.beforeId === beforeId) return
    dropInsert = { status: col, beforeId }
  }

  function flushPendingDrop() {
    dragOverRaf = 0
    if (!pendingDrop) return
    const { col, beforeId, crossColumn } = pendingDrop
    pendingDrop = null
    setDropInsert(col, beforeId, crossColumn)
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
    armCollapseClickSuppress()
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

    pendingDrop = { col, beforeId, crossColumn }
    if (!dragOverRaf) {
      dragOverRaf = requestAnimationFrame(flushPendingDrop)
    }
  }

  function onColumnDragOver(e: DragEvent, col: Status) {
    if (!draggingId) return
    e.preventDefault()
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
    // Collapsed lanes have no card list — drop appends to the column.
    if (isCollapsed(col) && dragLifted) {
      pendingDrop = { col, beforeId: null, crossColumn: dragFrom !== col }
      if (!dragOverRaf) {
        dragOverRaf = requestAnimationFrame(flushPendingDrop)
      }
    }
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

  function onCardClick(t: any, el?: HTMLButtonElement) {
    if (suppressClick) return
    onSelect(t.id)
    el?.blur()
  }

  async function markDone(id: string, e: MouseEvent) {
    e.preventDefault()
    e.stopPropagation()
    const t = tasks.find((x) => x.id === id)
    if (!t || t.status === 'done') return
    try {
      await api.setStatus(id, 'done')
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  function showGhostAt(col: Status, beforeId: string | null): boolean {
    return (
      dragLifted && dropInsert?.status === col && dropInsert.beforeId === beforeId && draggingTask !== null
    )
  }

  const archivableDoneCount = $derived.by(() => {
    const done = tasks.filter((t) => t.status === 'done')
    if (archiveDoneSubtasks) return done.length
    return done.filter((t) => !t.parent_id).length
  })

  let archiving = $state(false)

  async function archiveAll() {
    if (archiving || archivableDoneCount === 0) return
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
    class="ghost-card pointer-events-none rounded-card border border-dashed border-accent/70 border-l-2 bg-accent/10 shadow-[0_0_16px_rgba(123,140,255,0.25)] ring-2 ring-accent/40 {col.edge}"
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
    {:else if humanFilterEmpty}
      <p class="text-sm text-ink-3">No tasks match the owner filter.</p>
      <p class="text-xs text-ink-3/70">Switch the filter to <strong class="font-medium text-ink-2">All tasks</strong> or <strong class="font-medium text-ink-2">Human</strong>.</p>
    {:else if statusFilter && statusFilter !== 'archived'}
      <p class="text-sm text-ink-3">No {STATUS_LABELS[statusFilter] ?? statusFilter} tasks.</p>
      <p class="text-xs text-ink-3/70">Clear the status filter above to see all columns.</p>
    {:else}
      <p class="text-sm text-ink-3">No tasks yet.</p>
      <p class="text-xs text-ink-3/70">Press <kbd>n</kbd> or use a column’s + button to create one.</p>
    {/if}
  </div>
{:else}
  {@const singleExpanded =
    visibleColumns.length === 1 && !isCollapsed(visibleColumns[0].status)}
  <div class="h-full min-h-0 {singleExpanded ? '' : 'overflow-x-auto'}">
  <div
    class="grid h-full min-w-0 gap-3 {singleExpanded ? 'grid-cols-1 max-w-md' : ''}"
    style={boardGridStyle}
  >
    {#each visibleColumns as col (col.status)}
      {@const roots = columnRoots(col.status)}
      {@const collapsed = isCollapsed(col.status)}
      {@const colCount =
        dragLifted && dragFrom === col.status && draggingId
          ? columnRoots(col.status).length
          : byStatus[col.status].length}
      <section
        ondragover={(e) => onColumnDragOver(e, col.status)}
        ondrop={(e) => onColumnDrop(e, col.status)}
        class="flex min-h-0 flex-col rounded-card border shadow-sm
          {dropTarget === col.status
            ? 'border-accent/60 bg-accent/5'
            : 'border-line-soft bg-col'}"
      >
        {#if collapsed}
          <button
            type="button"
            title={`Expand ${col.label} (${colCount})`}
            aria-label={`Expand ${col.label} column`}
            aria-expanded="false"
            onclick={() => onCollapsedActivate(col.status)}
            ondragover={(e) => onColumnDragOver(e, col.status)}
            ondrop={(e) => onColumnDrop(e, col.status)}
            class="flex h-full min-h-0 w-full flex-col items-center gap-1.5 px-0.5 py-2 text-ink-3 transition-colors hover:bg-white/5 hover:text-ink-2"
          >
            <span
              class="flex h-5 w-5 flex-none items-center justify-center rounded-control transition-colors"
              aria-hidden="true"
            >
              <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m9 18 6-6-6-6" />
              </svg>
            </span>
            <span class="h-1.5 w-1.5 flex-none rounded-full {col.dot}" aria-hidden="true"></span>
            <span
              class="flex-none overflow-hidden text-[10px] font-semibold uppercase tracking-[0.06em] text-ink-2 [writing-mode:vertical-rl] rotate-180"
            >
              {col.label}
            </span>
            <span
              class="flex-none rounded-[2px] border border-line-soft bg-white/5 px-0.5 py-px font-mono text-[9px] leading-none text-ink-3"
            >
              {colCount}
            </span>
          </button>
        {:else}
        <header class="flex flex-none items-center gap-1.5 px-2 py-2.5 sm:gap-2 sm:px-3">
          <button
            type="button"
            title={`Collapse ${col.label}`}
            aria-label={`Collapse ${col.label} column`}
            aria-expanded="true"
            onclick={() => toggleCollapsed(col.status)}
            class="rounded-control p-1 text-ink-3 transition-colors hover:bg-white/5 hover:text-accent"
          >
            <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="m15 18-6-6 6-6" />
            </svg>
          </button>
          <span class="h-2 w-2 flex-none rounded-full {col.dot}"></span>
          <h2 class="text-[11px] font-semibold uppercase tracking-[0.07em] text-ink-2">{col.label}</h2>
          <span
            class="rounded-chip border border-line-soft bg-white/5 px-1.5 py-[3px] font-mono text-[10px] leading-none text-ink-3"
          >
            {colCount}
          </span>
          <div class="flex-1"></div>
          {#if col.status === 'done'}
            <button
              title={archiveDoneSubtasks
                ? 'Archive all done tasks including subtasks (reversible from List → Archived)'
                : 'Archive done root tasks only (reversible from List → Archived)'}
              disabled={archivableDoneCount === 0 || archiving}
              onclick={archiveAll}
              class="rounded-control p-1 transition-colors hover:bg-white/5 hover:text-accent disabled:cursor-default disabled:opacity-30"
            >
              <svg
                class="h-3.5 w-3.5 {archiving ? 'text-accent' : 'text-ink-3'}"
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
            class="rounded-control p-1 text-sm leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-accent"
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
            <p class="rounded-control border border-dashed border-line px-2 py-4 text-center text-[11px] text-ink-3">
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
                class="relative rounded-card border border-l-2 border-line-soft shadow-sm select-none cursor-grab
                  {col.edge}
                  {selectedId === t.id
                    ? 'bg-accent/10'
                    : 'bg-card hover:bg-card-hi'}
                  {draggingId === t.id && !dragLifted ? 'cursor-grabbing opacity-60' : ''}"
                title="Drag to reorder within column or drop on another column to change status"
              >
                <button
                  type="button"
                  tabindex={t.status === 'done' ? -1 : 0}
                  disabled={t.status === 'done'}
                  title={t.status === 'done' ? 'Done' : 'Mark done'}
                  aria-label={t.status === 'done' ? 'Done' : `Mark “${t.title}” done`}
                  aria-pressed={t.status === 'done'}
                  onclick={(e) => void markDone(t.id, e)}
                  onmousedown={(e) => e.stopPropagation()}
                  ondragstart={(e) => e.preventDefault()}
                  class="absolute right-1.5 top-1.5 z-10 flex h-3.5 w-3.5 items-center justify-center rounded-chip border transition-colors
                    {t.status === 'done'
                      ? 'cursor-default border-st-done bg-st-done text-white'
                      : 'cursor-pointer border-ink-3/50 bg-card/80 text-transparent hover:border-st-done hover:bg-st-done/20 hover:text-st-done'}"
                >
                  <svg class="h-2.5 w-2.5" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
                    <path d="M3.5 8.5 6.5 11.5 12.5 4.5" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </button>
                <button
                  type="button"
                  onclick={(e) => onCardClick(t, e.currentTarget)}
                  ondragstart={(e) => e.preventDefault()}
                  class="w-full cursor-grab p-2.5 pr-6 text-left focus:outline-none"
                >
                  <p class="mb-2 line-clamp-2 text-[13.5px] font-medium leading-snug text-ink">{t.title}</p>
                  <div class="flex items-center gap-2">
                    <div class="h-[3px] flex-1 overflow-hidden rounded-full bg-white/10">
                      <div
                        class="h-full rounded-full {col.bar}"
                        style="width: {t.progress}%"
                      ></div>
                    </div>
                    <span class="font-mono text-[10.5px] text-ink-3">{t.progress}%</span>
                    <span class="text-[10.5px] text-ink-3">{relTime(t.updated_at)}</span>
                  </div>
                </button>
                {#if showSubtasks && (childrenOf[t.id]?.length ?? 0) > 0}
                  <ul class="space-y-1 border-t border-line-soft px-2 pt-1.5">
                    {#each childrenOf[t.id] as c (c.id)}
                      <li>
                        <button
                          onclick={() => onSelect(c.id)}
                          class="flex w-full items-center gap-2 rounded-control border border-transparent px-1.5 py-1 text-left hover:border-line-soft hover:bg-white/5
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
                <div class="flex justify-end px-2 pb-2 pt-1">
                  <TaskActivityActions
                    task={t}
                    {settings}
                    {claudeBinaryOk}
                    {zedBinaryOk}
                    {onError}
                    {onToast}
                  />
                </div>
              </div>
            {/each}
            {#if showGhostAt(col.status, null) && draggingTask}
              <div data-ghost-slot class="pointer-events-none">
                {@render ghostCard(draggingTask, col)}
              </div>
            {/if}
          {/if}
        </div>
        {/if}
      </section>
    {/each}
  </div>
  </div>
{/if}
