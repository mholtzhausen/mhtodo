<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import FilterBar from './components/FilterBar.svelte'
  import List from './components/List.svelte'
  import TaskDetail from './components/TaskDetail.svelte'
  import NewTaskDialog from './components/NewTaskDialog.svelte'
  import { api, errMsg, type Status } from './lib/api'

  // In the Wails webview window.runtime exists; in a plain browser (fe-dev) it does not.
  const inWails = typeof window !== 'undefined' && !!(window as any).runtime

  let tasks = $state<any[]>([])
  let loading = $state(true)
  let toast: string | null = $state(null)
  let selectedId: string | null = $state(null)
  let dialogOpen = $state(false)
  let dbPath = $state('')

  // Filter state — mirrors core.ListFilter (parity with CLI `list` flags).
  let status = $state<Status | ''>('') // '' = all
  let search = $state('')
  let sort = $state<'created' | 'updated' | 'status' | 'progress' | 'title'>('updated')
  let ascending = $state(false)

  const selectedTask = $derived(tasks.find((t) => t.id === selectedId) ?? null)

  function showToast(msg: string) {
    toast = msg
    setTimeout(() => {
      if (toast === msg) toast = null
    }, 5000)
  }

  async function load() {
    try {
      tasks = await api.list({ status, search, sort, ascending })
    } catch (e) {
      showToast(errMsg(e))
    } finally {
      loading = false
    }
  }

  // Single refresh path: Go emits "tasks:changed" after every local mutation;
  // M4's external watcher emits the same event for CLI-side writes.
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
        dialogOpen = true
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
    unbindTrayNewTask = EventsOn('tray:new-task', () => { dialogOpen = true })
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
    <div class="flex-1"></div>
    <button
      onclick={() => (dialogOpen = true)}
      class="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-indigo-500"
    >
      + New task <kbd>n</kbd>
    </button>
  </header>

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

  <main class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
    {#if loading}
      <p class="text-sm text-zinc-500">Loading…</p>
    {:else}
      <List tasks={tasks} selectedId={selectedId} onSelect={(id: string) => (selectedId = id)} />
    {/if}
  </main>

  <footer class="flex items-center gap-4 border-t border-zinc-800 px-5 py-2 text-xs text-zinc-500">
    <span class="truncate font-mono">{dbPath}</span>
    <div class="flex-1"></div>
    <span><kbd>/</kbd> search · <kbd>n</kbd> new · <kbd>esc</kbd> close · <kbd>1–4</kbd> filter · <kbd>ctrl+q</kbd> quit</span>
  </footer>

  {#if selectedTask}
    {#key selectedTask.id}
      <TaskDetail task={selectedTask} onClose={() => (selectedId = null)} onError={showToast} />
    {/key}
  {/if}

  <NewTaskDialog open={dialogOpen} onClose={() => (dialogOpen = false)} onError={showToast} />

  {#if toast}
    <div class="fixed bottom-10 left-1/2 z-[60] -translate-x-1/2 rounded-lg border border-zinc-700 bg-zinc-900 px-4 py-2 text-sm text-zinc-200 shadow-xl">
      {toast}
    </div>
  {/if}
</div>
