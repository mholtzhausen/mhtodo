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

  // Column order is the workflow order; dot/bar colors double as the status
  // legend so cards don't need a redundant badge.
  const COLUMNS: { status: Status; label: string; dot: string; bar: string }[] = [
    { status: 'pending', label: 'Pending', dot: 'bg-zinc-400', bar: 'bg-indigo-500' },
    { status: 'wip', label: 'In progress', dot: 'bg-indigo-400', bar: 'bg-indigo-500' },
    { status: 'waiting', label: 'Waiting', dot: 'bg-amber-400', bar: 'bg-amber-500' },
    { status: 'done', label: 'Done', dot: 'bg-emerald-400', bar: 'bg-emerald-500' }
  ]

  const byStatus = $derived.by(() => {
    const m: Record<string, any[]> = {}
    for (const c of COLUMNS) m[c.status] = []
    for (const t of tasks) (m[t.status] ??= []).push(t)
    return m
  })

  const columnBar = $derived(Object.fromEntries(COLUMNS.map((c) => [c.status, c.bar])))

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
    if (dropTarget === col && !(sec?.contains(e.relatedTarget as Node | null))) dropTarget = ''
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
      <p class="text-sm text-zinc-500">No tasks match “{search}”.</p>
      <p class="text-xs text-zinc-600">Clear the search above to see everything.</p>
    {:else}
      <p class="text-sm text-zinc-500">No tasks yet.</p>
      <p class="text-xs text-zinc-600">Press <kbd>n</kbd> or use a column’s + button to create one.</p>
    {/if}
  </div>
{:else}
  <div class="grid h-full grid-cols-4 gap-3">
    {#each COLUMNS as col (col.status)}
      <section
        ondragover={(e) => onColumnDragOver(e, col.status)}
        ondragleave={(e) => onColumnDragLeave(e, col.status)}
        ondrop={(e) => onColumnDrop(e, col.status)}
        class="flex min-h-0 flex-col rounded-xl border transition-colors duration-150
          {dropTarget === col.status ? 'border-indigo-500/60 bg-indigo-500/[0.04]' : 'border-zinc-800/70 bg-zinc-900/30'}"
      >
        <header class="flex items-center gap-2 px-3 py-2.5">
          <span class="h-2 w-2 rounded-full {col.dot}"></span>
          <h2 class="text-xs font-semibold uppercase tracking-wide text-zinc-400">{col.label}</h2>
          <span class="rounded-full bg-zinc-800 px-1.5 py-0.5 font-mono text-[10px] leading-none text-zinc-400">
            {byStatus[col.status].length}
          </span>
          <div class="flex-1"></div>
          <button
            title={`New ${col.label.toLowerCase()} task`}
            onclick={() => onQuickAdd(col.status)}
            class="rounded-md px-1.5 text-sm leading-none text-zinc-500 transition-colors hover:bg-zinc-800 hover:text-indigo-300"
          >
            +
          </button>
        </header>

        <div class="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto px-2 pb-2">
          {#if byStatus[col.status].length === 0}
            <p class="rounded-lg border border-dashed border-zinc-800/80 py-4 text-center text-[11px] text-zinc-700">
              no tasks
            </p>
          {:else}
            {#each byStatus[col.status] as t (t.id)}
              <button
                draggable="true"
                onclick={() => onCardClick(t)}
                ondragstart={(e) => onCardDragStart(e, t)}
                ondragend={onCardDragEnd}
                class="w-full cursor-grab rounded-lg border p-2.5 text-left transition-all duration-150
                  {draggingId === t.id ? 'cursor-grabbing rotate-1 opacity-40' : ''}
                  {selectedId === t.id
                    ? 'border-indigo-500/60 bg-zinc-900 ring-1 ring-indigo-500/40'
                    : 'border-zinc-800 bg-zinc-900/70 hover:-translate-y-px hover:border-zinc-700 hover:bg-zinc-900 hover:shadow-lg'}"
              >
                <p class="mb-2 line-clamp-2 text-sm font-medium leading-snug text-zinc-200">{t.title}</p>
                <div class="flex items-center gap-2">
                  <div class="h-1 flex-1 overflow-hidden rounded-full bg-zinc-800">
                    <div
                      class="h-full rounded-full {columnBar[t.status] ?? 'bg-indigo-500'} transition-all duration-150"
                      style="width: {t.progress}%"
                    ></div>
                  </div>
                  <span class="w-8 text-right font-mono text-[10px] text-zinc-600">{t.progress}%</span>
                </div>
                <p class="mt-1.5 text-[10px] text-zinc-600">{relTime(t.updated_at)}</p>
              </button>
            {/each}
          {/if}
        </div>
      </section>
    {/each}
  </div>
{/if}
