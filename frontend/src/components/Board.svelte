<script lang="ts">
  import { relTime } from '../lib/format'
  import { api, errMsg, type Status } from '../lib/api'

  let {
    tasks,
    search,
    selectedId,
    onSelect,
    onQuickAdd,
    onError
  }: {
    tasks: any[]
    search: string
    selectedId: string | null
    onSelect: (id: string) => void
    onQuickAdd: (s: Status) => void
    onError?: (msg: string) => void
  } = $props()

  // Column order is the workflow order; per-status colors double as the board
  // legend (dot, progress bar, card left edge) so cards need no redundant badge.
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
    { status: 'done', label: 'Done', dot: 'bg-st-done', bar: 'bg-st-done', edge: 'border-l-st-done' }
  ]

  const byStatus = $derived.by(() => {
    const m: Record<string, any[]> = {}
    for (const c of COLUMNS) m[c.status] = []
    for (const t of tasks) (m[t.status] ??= []).push(t)
    return m
  })

  // --- Native HTML5 drag & drop: dropping a card on another column calls
  // api.setStatus; Go emits tasks:changed and App refetches through its single
  // refresh path. We never mutate local task state here.
  let draggingId = $state<string | null>(null)
  let dragFrom = $state<Status | ''>('')
  let dropTarget = $state<Status | ''>('')
  // A completed drag can be followed by a stray click on the card; swallow it
  // so a drag never opens the detail drawer.
  let suppressClick = false

  function onCardDragStart(e: DragEvent, t: any) {
    draggingId = t.id
    dragFrom = t.status
    e.dataTransfer?.setData('text/plain', t.id) // required to start a drag in some engines
    if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
  }

  function onCardDragEnd() {
    suppressClick = true
    setTimeout(() => (suppressClick = false), 0)
    draggingId = null
    dragFrom = ''
    dropTarget = ''
  }

  function onColumnDragOver(e: DragEvent, col: Status) {
    if (!draggingId || dragFrom === col) return // own column is a no-op
    e.preventDefault() // allow the drop
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
    dropTarget = col
  }

  function onColumnDragLeave(e: DragEvent, col: Status) {
    const sec = e.currentTarget as HTMLElement | null
    if (dropTarget === col.status && !(sec?.contains(e.relatedTarget as Node))) dropTarget = ''
  }

  async function onColumnDrop(e: DragEvent, col: Status) {
    e.preventDefault()
    const id = (draggingId ?? e.dataTransfer?.getData('text/plain')) || null
    dropTarget = ''
    if (!id || dragFrom === col) return // same column → no API call
    try {
      await api.setStatus(id, col)
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  function onCardClick(t: any) {
    if (suppressClick) return
    onSelect(t.id)
  }
</script>

{#if tasks.length === 0}
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
  <div class="grid h-full grid-cols-4 gap-3">
    {#each COLUMNS as col (col.status)}
      <section
        ondragover={(e) => onColumnDragOver(e, col.status)}
        ondragleave={(e) => onColumnDragLeave(e, col.status)}
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
            {byStatus[col.status].length}
          </span>
          <div class="flex-1"></div>
          <button
            title={`New ${col.label.toLowerCase()} task`}
            onclick={() => onQuickAdd(col.status)}
            class="rounded p-1 text-sm leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-accent"
          >
            +
          </button>
        </header>

        <div class="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto px-2 py-2.5">
          {#if byStatus[col.status].length === 0}
            <p class="rounded border border-dashed border-line px-2 py-4 text-center text-[11px] text-ink-3">
              no tasks
            </p>
          {:else}
            {#each byStatus[col.status] as t (t.id)}
              <button
                draggable="true"
                onclick={() => onCardClick(t)}
                ondragstart={(e) => onCardDragStart(e, t)}
                ondragend={onCardDragEnd}
                class="w-full cursor-grab rounded-md border border-l-2 p-2.5 text-left shadow-sm transition-all duration-150
                  {col.edge}
                  {draggingId === t.id ? 'cursor-grabbing rotate-2 opacity-40' : ''}
                  {selectedId === t.id
                    ? 'border-accent bg-card-hi ring-1 ring-accent/70'
                    : 'border-line-soft bg-card hover:-translate-y-px hover:bg-card-hi hover:shadow-lg'}"
              >
                <p class="mb-2 line-clamp-2 text-[13.5px] font-medium leading-snug text-ink">{t.title}</p>
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
            {/each}
          {/if}
        </div>
      </section>
    {/each}
  </div>
{/if}
