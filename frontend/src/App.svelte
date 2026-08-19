<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import Board from './components/Board.svelte'
  import FilterBar from './components/FilterBar.svelte'
  import List from './components/List.svelte'
  import TaskDetail from './components/TaskDetail.svelte'
  import NewTaskDialog from './components/NewTaskDialog.svelte'
  import { api, errMsg, type Status } from './lib/api'

  // In the Wails webview window.runtime exists; in a plain browser (fe-dev) it does not.
  const inWails = typeof window !== 'undefined' && !!(window as any).runtime

  // Board is the default view (05-gui-spec.md); choice persists across launches.
  const storedView = (() => { try { return localStorage.getItem('mhtodo.view') } catch { return null } })()
  let view = $state<'board' | 'list'>(storedView === 'list' ? 'list' : 'board')

  let tasks = $state<any[]>([])
  let loading = $state(true)
  let toast: { msg: string; kind: 'error' | 'info' } | null = $state(null)
  let selectedId: string | null = $state(null)
  let dialogOpen = $state(false)
  // '' = default (pending); a board column's + button presets its status.
  let dialogInitialStatus = $state<Status | ''>('')
  let dbPath = $state('')

  // Filter state — mirrors core.ListFilter (parity with CLI `list` flags).
  let status = $state<Status | ''>('') // '' = all
  let search = $state('')
  let sort = $state<'created' | 'updated' | 'status' | 'progress' | 'title'>('updated')
  let ascending = $state(false)

  const selectedTask = $derived(tasks.find((t) => t.id === selectedId) ?? null)

  function showToast(msg: string, kind: 'error' | 'info' = 'error') {
    const t = { msg, kind }
    toast = t
    setTimeout(() => {
      if (toast === t) toast = null
    }, 5000)
  }

  function setView(v: 'board' | 'list') {
    if (view === v) return
    view = v
    try { localStorage.setItem('mhtodo.view', v) } catch { /* non-Wails, ignore */ }
    load() // board and list fetch with different filters
  }

  async function load() {
    try {
      // Board always shows all four columns (done included), sorted by recency;
      // the status chips/sort controls only apply to the list view.
      const filter =
        view === 'board'
          ? { search, sort: 'updated' as const, ascending: false }
          : { status, search, sort, ascending }
      tasks = await api.list(filter)
    } catch (e) {
      showToast(errMsg(e))
    } finally {
      loading = false
    }
  }

  // Single refresh path: Go emits "tasks:changed" after every local mutation;
  // the external watcher (internal/sync) emits the same event for CLI-side writes.
  let unbindChanged: (() => void) | undefined
  let unbindTrayNewTask: (() => void) | undefined

  function onKeydown(e: KeyboardEvent) {
    const el = e.target as HTMLElement | null
    const typing = !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable)
    if (e.key === 'Escape') {
      if (dialogOpen) dialogOpen = false
      else selectedId = null
      return
    }
    // Ctrl+Q → real quit. Window close hides to tray; this is the in-window exit.
    if ((e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey && (e.key === 'q' || e.key === 'Q')) {
      e.preventDefault()
      api.quit()
      return
    }
    if (typing || e.metaKey || e.ctrlKey || e.altKey) return
    switch (e.key) {
      case '/':
        e.preventDefault()
        document.getElementById('task-search')?.focus()
        break
      case 'n':
        e.preventDefault()
        dialogInitialStatus = ''
        dialogOpen = true
        break
      // b/l switch between board and list views.
      case 'b':
        setView('board')
        break
      case 'l':
        setView('list')
        break
      // 1..4 toggle a status filter; pressing the active one again clears it.
      case '1':
        status = status === 'pending' ? '' : 'pending'
        load()
        break
      case '2':
        status = status === 'wip' ? '' : 'wip'
        load()
        break
      case '3':
        status = status === 'waiting' ? '' : 'waiting'
        load()
        break
      case '4':
        status = status === 'done' ? '' : 'done'
        load()
        break
    }
  }

  onMount(async () => {
    if (!inWails) {
      loading = false
      showToast('Running outside Wails — API unavailable (use `make dev`)')
      return
    }
    const { EventsOn } = await import('../wailsjs/runtime/runtime.js')
    dbPath = await api.dbPath()
    unbindChanged = EventsOn('tasks:changed', () => load())
    // Tray "New Task": show the window (Go side) and open the create dialog here.
    unbindTrayNewTask = EventsOn('tray:new-task', () => { dialogInitialStatus = ''; dialogOpen = true })
    window.addEventListener('keydown', onKeydown)
    await load()
  })

  onDestroy(() => {
    unbindChanged?.()
    unbindTrayNewTask?.()
    window.removeEventListener('keydown', onKeydown)
  })
</script>

<div class="flex h-full flex-col">
  <header class="flex items-center gap-4 border-b border-zinc-800 px-5 py-3">
    <h1 class="text-lg font-semibold tracking-tight text-zinc-100">mhtodo</h1>

    <div class="flex gap-1 rounded-lg bg-zinc-900 p-1" role="tablist" aria-label="View">
      {#each [['board', 'Board'], ['list', 'List']] as [v, label] (v)}
        <button
          role="tab"
          aria-selected={view === v}
          onclick={() => setView(v as 'board' | 'list')}
          class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors
            {view === v ? 'bg-zinc-700 text-zinc-100' : 'text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200'}"
        >
          {label}
        </button>
      {/each}
    </div>

    <div class="flex-1"></div>
    <button
      onclick={() => { dialogInitialStatus = ''; dialogOpen = true }}
      class="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-indigo-500"
    >
      + New task <kbd>n</kbd>
    </button>
  </header>

  {#if view === 'list'}
    <FilterBar
      {status}
      {search}
      {sort}
      {ascending}
      onStatusChange={(s: Status | '') => { status = s; load() }}
      onSearchInput={(v: string) => { search = v; load() }}
      onSortChange={(f: 'created' | 'updated' | 'status' | 'progress' | 'title') => { sort = f; load() }}
      onToggleAsc={() => { ascending = !ascending; load() }}
    />
  {:else}
    <div class="flex items-center gap-3 border-b border-zinc-800 px-5 py-3">
      <input
        id="task-search"
        type="search"
        placeholder="Search… ( / )"
        value={search}
        oninput={(e) => { search = e.currentTarget.value; load() }}
        class="w-56 rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-sm text-zinc-200 placeholder:text-zinc-600 focus:border-indigo-500 focus:outline-none"
      />
    </div>
  {/if}

  <!-- board view scrolls per-column; list view scrolls the whole pane -->
  <main class="min-h-0 flex-1 {view === 'board' ? '' : 'overflow-y-auto'} px-5 py-4">
    {#if loading}
      <p class="text-sm text-zinc-500">Loading…</p>
    {:else if view === 'board'}
      <Board
        tasks={tasks}
        {search}
        selectedId={selectedId}
        onSelect={(id: string) => (selectedId = id)}
        onQuickAdd={(s: Status) => { dialogInitialStatus = s; dialogOpen = true }}
        onError={showToast}
      />
    {:else}
      <List
        tasks={tasks}
        hasFilters={status !== '' || search.trim() !== ''}
        selectedId={selectedId}
        onSelect={(id: string) => (selectedId = id)}
      />
    {/if}
  </main>

  <footer class="flex items-center gap-4 border-t border-zinc-800 px-5 py-2 text-xs text-zinc-500">
    <span class="truncate font-mono">{dbPath}</span>
    <div class="flex-1"></div>
    <span><kbd>/</kbd> search · <kbd>n</kbd> new · <kbd>b</kbd>/<kbd>l</kbd> view {#if view === 'list'}· <kbd>1–4</kbd> filter{/if} · <kbd>esc</kbd> close · <kbd>ctrl+q</kbd> quit</span>
  </footer>

  {#if selectedTask}
    {#key selectedTask.id}
      <TaskDetail task={selectedTask} onClose={() => (selectedId = null)} onError={showToast} />
    {/key}
  {/if}

  <NewTaskDialog
    open={dialogOpen}
    initialStatus={dialogInitialStatus || 'pending'}
    onClose={() => (dialogOpen = false)}
    onError={showToast}
  />

  {#if toast}
    <div
      role="alert"
      class="fixed bottom-10 left-1/2 z-[60] -translate-x-1/2 rounded-lg border px-4 py-2 text-sm shadow-xl
        {toast.kind === 'error'
          ? 'border-rose-900/70 bg-rose-950/90 text-rose-200'
          : 'border-zinc-700 bg-zinc-900 text-zinc-200'}"
    >
      {toast.msg}
    </div>
  {/if}
</div>
