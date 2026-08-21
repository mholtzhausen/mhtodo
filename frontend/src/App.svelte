<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import Board from './components/Board.svelte'
  import FilterBar from './components/FilterBar.svelte'
  import List from './components/List.svelte'
  import TaskDetail from './components/TaskDetail.svelte'
  import NewTaskDialog from './components/NewTaskDialog.svelte'
  import ConfirmDialog from './components/ConfirmDialog.svelte'
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
  // Task awaiting delete confirmation (Delete key or the drawer's button).
  let confirmTask = $state<any | null>(null)
  let deleting = $state(false)
  let dbPath = $state('')

  // Filter state — mirrors core.ListFilter (parity with CLI `list` flags).
  // 'archived' is a separate mode: default/all views hide archived tasks; only
  // this filter shows them (v0.2). Mutually exclusive with the status chips.
  let status = $state<Status | '' | 'archived'>('') // '' = all
  let search = $state('')
  let sort = $state<'created' | 'updated' | 'status' | 'progress' | 'title'>('updated')
  let ascending = $state(false)

  const selectedTask = $derived(tasks.find((t) => t.id === selectedId) ?? null)

  // All toasts auto-dismiss after 3s by default; individual call sites may pass
  // a different lifetime when a message needs more (or less) time.
  function showToast(msg: string, kind: 'error' | 'info' = 'error', ms = 3000) {
    const t = { msg, kind }
    toast = t
    setTimeout(() => {
      if (toast === t) toast = null
    }, ms)
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
          ? { search, sort: 'updated' as const, ascending: false } // board hides archived automatically (Go side)
          : status === 'archived'
            ? { archived: true, search, sort, ascending }
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

  // One code path for both delete triggers (drawer button + Delete key).
  function requestDelete(t: any) {
    if (deleting || confirmTask) return
    confirmTask = t
  }

  async function doDelete() {
    const t = confirmTask
    if (!t || deleting) return
    deleting = true
    try {
      await api.remove(t.id)
      selectedId = null // drawer closes; the tasks:changed refetch drops the row
    } catch (e) {
      showToast(errMsg(e))
    } finally {
      confirmTask = null
      deleting = false
    }
  }

  function onKeydown(e: KeyboardEvent) {
    const el = e.target as HTMLElement | null
    const typing = !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable)
    if (e.key === 'Escape') {
      if (confirmTask) confirmTask = null
      else if (dialogOpen) dialogOpen = false
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
      // Delete key on a selected task → confirmation dialog, then deletion.
      case 'Delete':
        if (!dialogOpen && !confirmTask && selectedTask) {
          e.preventDefault()
          requestDelete(selectedTask)
        }
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
      // 5 toggles the archived view (list only; mutually exclusive with 1–4).
      case '5':
        if (view === 'list') {
          status = status === 'archived' ? '' : 'archived'
          load()
        }
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
  <header class="flex h-[52px] flex-none items-center gap-4 border-b border-line-soft bg-chrome px-5">
    <div class="flex items-center gap-2.5">
      <span
        class="grid h-[22px] w-[22px] flex-none place-items-center rounded-[5px] bg-accent text-[12px] font-bold text-accent-ink"
      >
        M
      </span>
      <h1 class="text-[15px] font-semibold tracking-tight text-ink">mhtodo</h1>
    </div>

    <nav class="flex items-stretch gap-1" role="tablist" aria-label="View">
      {#each [['board', 'Board'], ['list', 'List']] as [v, label] (v)}
        <button
          role="tab"
          aria-selected={view === v}
          onclick={() => setView(v as 'board' | 'list')}
          class="relative px-3 text-[13px] font-medium transition-colors
            {view === v ? 'text-ink' : 'text-ink-3 hover:text-ink-2'}"
        >
          {label}
          {#if view === v}<span class="absolute inset-x-2.5 -bottom-px h-0.5 rounded-full bg-accent"></span>{/if}
        </button>
      {/each}
    </nav>

    <div class="flex-1"></div>
    <button
      onclick={() => { dialogInitialStatus = ''; dialogOpen = true }}
      class="btn-primary flex items-center gap-2 rounded bg-accent px-3 py-1.5 text-sm font-medium text-accent-ink shadow-sm transition-colors hover:bg-accent-hi"
    >
      <span class="font-semibold leading-none">+</span>
      New task <kbd>n</kbd>
    </button>
  </header>

  {#if view === 'list'}
    <FilterBar
      {status}
      {search}
      {sort}
      {ascending}
      onStatusChange={(s: Status | '' | 'archived') => { status = s; load() }}
      onSearchInput={(v: string) => { search = v; load() }}
      onSortChange={(f: 'created' | 'updated' | 'status' | 'progress' | 'title') => { sort = f; load() }}
      onToggleAsc={() => { ascending = !ascending; load() }}
    />
  {:else}
    <div class="flex flex-none items-center gap-3 border-b border-line-soft px-5 py-2.5">
      <label
        class="flex h-8 w-64 cursor-text items-center gap-2 rounded border border-line-soft bg-field px-2.5 shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] transition-colors focus-within:border-accent focus-within:ring-2 focus-within:ring-accent/25"
      >
        <svg
          class="h-3.5 w-3.5 flex-none text-ink-3"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          aria-hidden="true"
        >
          <circle cx="11" cy="11" r="7" />
          <path d="m20 20-3.5-3.5" />
        </svg>
        <input
          id="task-search"
          type="search"
          placeholder="Search… ( / )"
          value={search}
          oninput={(e) => { search = e.currentTarget.value; load() }}
          class="min-w-0 flex-1 bg-transparent text-[13px] text-ink outline-none placeholder:text-ink-3"
        />
      </label>
      <div class="flex-1"></div>
      <span class="font-mono text-[11px] text-ink-3">
        {tasks.length} {tasks.length === 1 ? 'task' : 'tasks'}
      </span>
    </div>
  {/if}

  <!-- board view scrolls per-column; list view scrolls the whole pane -->
  <main class="min-h-0 flex-1 {view === 'board' ? '' : 'overflow-y-auto'} px-5 py-4">
    {#if loading}
      <p class="text-sm text-ink-3">Loading…</p>
    {:else if view === 'board'}
      <Board
        tasks={tasks}
        {search}
        selectedId={selectedId}
        onSelect={(id: string) => (selectedId = id)}
        onQuickAdd={(s: Status) => { dialogInitialStatus = s; dialogOpen = true }}
        onArchived={(n: number) => showToast(`Archived ${n} task${n === 1 ? '' : 's'}`, 'info')}
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

  <footer class="flex h-9 flex-none items-center gap-4 border-t border-line-soft bg-chrome px-5 text-xs text-ink-3">
    <span class="truncate font-mono text-[11px]">{dbPath}</span>
    <div class="flex-1"></div>
    <span class="flex-none whitespace-nowrap"
      ><kbd>/</kbd> search · <kbd>n</kbd> new · <kbd>b</kbd>/<kbd>l</kbd> view
      {#if view === 'list'}· <kbd>1–4</kbd> filter · <kbd>5</kbd> archived{/if} · <kbd>del</kbd> delete ·
      <kbd>esc</kbd> close · <kbd>ctrl+q</kbd> quit</span
    >
  </footer>

  {#if selectedTask}
    {#key selectedTask.id}
      <TaskDetail
        task={selectedTask}
        onClose={() => (selectedId = null)}
        onError={showToast}
        onDelete={(t: any) => requestDelete(t)}
      />
    {/key}
  {/if}

  <NewTaskDialog
    open={dialogOpen}
    initialStatus={dialogInitialStatus || 'pending'}
    onClose={() => (dialogOpen = false)}
    onError={showToast}
  />

  <ConfirmDialog
    open={confirmTask !== null}
    title="Delete task?"
    message={confirmTask ? `"${confirmTask.title}" will be permanently removed.` : ''}
    confirmLabel="Delete"
    onCancel={() => (confirmTask = null)}
    onConfirm={doDelete}
  />

  {#if toast}
    <div
      role="alert"
      class="fixed bottom-10 left-1/2 z-[60] -translate-x-1/2 rounded border px-4 py-2 text-sm shadow-xl
        {toast.kind === 'error'
          ? 'border-danger/50 bg-card-hi text-danger'
          : 'border-line bg-card-hi text-ink'}"
    >
      {toast.msg}
    </div>
  {/if}
</div>
