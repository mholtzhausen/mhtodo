<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import Board from './components/Board.svelte'
  import FilterBar from './components/FilterBar.svelte'
  import List from './components/List.svelte'
  import ActivityView from './components/ActivityView.svelte'
  import TaskDetail from './components/TaskDetail.svelte'
  import NewTaskDialog from './components/NewTaskDialog.svelte'
  import ConfirmDialog from './components/ConfirmDialog.svelte'
  import { api, errMsg, type Activity, type Status } from './lib/api'

  const inWails = typeof window !== 'undefined' && !!(window as any).runtime

  type View = 'board' | 'list' | 'activity'
  const storedView = (() => {
    try {
      return localStorage.getItem('mhtodo.view')
    } catch {
      return null
    }
  })()
  let view = $state<View>(
    storedView === 'list' || storedView === 'activity' ? storedView : 'board'
  )

  const storedShowSub = (() => {
    try {
      return localStorage.getItem('mhtodo.showSubtasks')
    } catch {
      return null
    }
  })()
  let showSubtasks = $state(storedShowSub !== 'false')

  const storedPin = (() => {
    try {
      return localStorage.getItem('mhtodo.detailPinned')
    } catch {
      return null
    }
  })()
  let detailPinned = $state(storedPin === 'true')
  let alwaysOnTop = $state(false)

  let tasks = $state<any[]>([])
  let activities = $state<Activity[]>([])
  let activityFilterIds = $state<string[]>([])
  let loading = $state(true)
  let toast: { msg: string; kind: 'error' | 'info' } | null = $state(null)
  let selectedId: string | null = $state(null)
  let dialogOpen = $state(false)
  let dialogInitialStatus = $state<Status | ''>('')
  let dialogParentId = $state('')
  let confirmTask = $state<any | null>(null)
  let confirmMsg = $state('')
  let deleting = $state(false)
  let dbPath = $state('')

  let status = $state<Status | '' | 'archived'>('')
  let search = $state('')
  let sort = $state<'created' | 'updated' | 'status' | 'progress' | 'title'>('updated')
  let ascending = $state(false)

  const selectedTask = $derived(tasks.find((t) => t.id === selectedId) ?? null)
  const selectedParentTitle = $derived(
    selectedTask?.parent_id
      ? (tasks.find((t) => t.id === selectedTask.parent_id)?.title ?? null)
      : null
  )

  function showToast(msg: string, kind: 'error' | 'info' = 'error', ms = 3000) {
    const t = { msg, kind }
    toast = t
    setTimeout(() => {
      if (toast === t) toast = null
    }, ms)
  }

  function setView(v: View) {
    if (view === v) return
    view = v
    try {
      localStorage.setItem('mhtodo.view', v)
    } catch {
      /* ignore */
    }
    load()
  }

  function toggleSubtasks() {
    showSubtasks = !showSubtasks
    try {
      localStorage.setItem('mhtodo.showSubtasks', showSubtasks ? 'true' : 'false')
    } catch {
      /* ignore */
    }
  }

  function togglePin() {
    detailPinned = !detailPinned
    try {
      localStorage.setItem('mhtodo.detailPinned', detailPinned ? 'true' : 'false')
    } catch {
      /* ignore */
    }
  }

  async function toggleAlwaysOnTop() {
    const next = !alwaysOnTop
    alwaysOnTop = next
    try {
      await api.setAlwaysOnTop(next)
    } catch (e) {
      alwaysOnTop = !next
      showToast(errMsg(e))
    }
  }

  async function load() {
    try {
      if (view === 'activity') {
        // Include done so the ticket filter covers the full non-archived set.
        const [t, a] = await Promise.all([
          api.list({ sort: 'title', ascending: true }),
          api.listActivity({})
        ])
        tasks = t
        activities = a
      } else {
        const filter =
          view === 'board'
            ? { search, sort: 'updated' as const, ascending: false }
            : status === 'archived'
              ? { archived: true, search, sort, ascending }
              : { status, search, sort, ascending }
        tasks = await api.list(filter)
      }
    } catch (e) {
      showToast(errMsg(e))
    } finally {
      loading = false
    }
  }

  let unbindChanged: (() => void) | undefined
  let unbindTrayNewTask: (() => void) | undefined

  function requestDelete(t: any) {
    if (deleting || confirmTask) return
    confirmTask = t
    confirmMsg = `"${t.title}" will be permanently removed.`
    api
      .countChildren(t.id)
      .then((n) => {
        if (confirmTask?.id === t.id && n > 0) {
          confirmMsg = `"${t.title}" and its ${n} sub-task(s) will be permanently removed.`
        }
      })
      .catch(() => {})
  }

  async function doDelete() {
    const t = confirmTask
    if (!t || deleting) return
    deleting = true
    try {
      await api.remove(t.id)
      selectedId = null
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
      e.preventDefault()
      if (confirmTask) confirmTask = null
      else if (dialogOpen) {
        dialogOpen = false
        dialogParentId = ''
      } else if (selectedId && !detailPinned) selectedId = null
      else api.hideWindow()
      return
    }
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
        dialogParentId = ''
        dialogOpen = true
        break
      case 's':
        e.preventDefault()
        toggleSubtasks()
        break
      case 'Delete':
        if (!dialogOpen && !confirmTask && selectedTask) {
          e.preventDefault()
          requestDelete(selectedTask)
        }
        break
      case 'b':
        setView('board')
        break
      case 'l':
        setView('list')
        break
      case 'a':
        setView('activity')
        break
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
        status = status === 'review' ? '' : 'review'
        load()
        break
      case '5':
        status = status === 'done' ? '' : 'done'
        load()
        break
      case '6':
        if (view === 'list') {
          status = status === 'archived' ? '' : 'archived'
          load()
        }
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
    try {
      alwaysOnTop = await api.getAlwaysOnTop()
    } catch {
      /* ignore */
    }
    unbindChanged = EventsOn('tasks:changed', () => load())
    unbindTrayNewTask = EventsOn('tray:new-task', () => {
      dialogInitialStatus = ''
      dialogParentId = ''
      dialogOpen = true
    })
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
      {#each [['board', 'Board'], ['list', 'List'], ['activity', 'Activity']] as [v, label] (v)}
        <button
          role="tab"
          aria-selected={view === v}
          onclick={() => setView(v as View)}
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
      type="button"
      onclick={toggleAlwaysOnTop}
      title={alwaysOnTop ? 'Always on top (on)' : 'Always on top (off)'}
      aria-pressed={alwaysOnTop}
      class="grid h-8 w-8 place-items-center rounded border transition-colors
        {alwaysOnTop
          ? 'border-accent/50 bg-accent/15 text-accent-hi'
          : 'border-line-soft text-ink-3 hover:text-ink'}"
    >
      <svg
        class="h-4 w-4"
        viewBox="0 0 24 24"
        fill={alwaysOnTop ? 'currentColor' : 'none'}
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M12 17v5" />
        <path
          d="M9 10.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24V16a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V6a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1z"
        />
      </svg>
    </button>

    {#if view === 'board' || view === 'list'}
      <button
        type="button"
        onclick={toggleSubtasks}
        title="Show/hide sub-tasks (s)"
        class="rounded border px-2.5 py-1 text-xs font-medium transition-colors
          {showSubtasks
            ? 'border-accent/50 bg-accent/15 text-accent-hi'
            : 'border-line-soft text-ink-3 hover:text-ink'}"
      >
        Sub-tasks <kbd>s</kbd>
      </button>
    {/if}

    <button
      onclick={() => {
        dialogInitialStatus = ''
        dialogParentId = ''
        dialogOpen = true
      }}
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
      onStatusChange={(s: Status | '' | 'archived') => {
        status = s
        load()
      }}
      onSearchInput={(v: string) => {
        search = v
        load()
      }}
      onSortChange={(f: 'created' | 'updated' | 'status' | 'progress' | 'title') => {
        sort = f
        load()
      }}
      onToggleAsc={() => {
        ascending = !ascending
        load()
      }}
    />
  {:else if view === 'board'}
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
          oninput={(e) => {
            search = e.currentTarget.value
            load()
          }}
          class="min-w-0 flex-1 bg-transparent text-[13px] text-ink outline-none placeholder:text-ink-3"
        />
      </label>
      <div class="flex-1"></div>
      <span class="font-mono text-[11px] text-ink-3">
        {tasks.filter((t) => !t.parent_id).length} {tasks.filter((t) => !t.parent_id).length === 1
          ? 'task'
          : 'tasks'}
      </span>
    </div>
  {/if}

  <div class="flex min-h-0 flex-1">
    <main
      class="min-h-0 flex-1 {view === 'board' ? '' : 'overflow-y-auto'} px-5 py-4"
    >
      {#if loading}
        <p class="text-sm text-ink-3">Loading…</p>
      {:else if view === 'board'}
        <Board
          tasks={tasks}
          {search}
          selectedId={selectedId}
          {showSubtasks}
          onSelect={(id: string) => (selectedId = id)}
          onQuickAdd={(s: Status) => {
            dialogInitialStatus = s
            dialogParentId = ''
            dialogOpen = true
          }}
          onArchived={(n: number) => showToast(`Archived ${n} task${n === 1 ? '' : 's'}`, 'info')}
          onError={showToast}
        />
      {:else if view === 'list'}
        <List
          tasks={tasks}
          hasFilters={status !== '' || search.trim() !== ''}
          selectedId={selectedId}
          {showSubtasks}
          onSelect={(id: string) => (selectedId = id)}
        />
      {:else}
        <ActivityView
          {activities}
          {tasks}
          selectedTaskIds={activityFilterIds}
          onToggleTask={(id) => {
            activityFilterIds = activityFilterIds.includes(id)
              ? activityFilterIds.filter((x) => x !== id)
              : [...activityFilterIds, id]
          }}
          onSelectTask={(id) => (selectedId = id)}
          onClearFilter={() => (activityFilterIds = [])}
        />
      {/if}
    </main>

    {#if selectedTask && detailPinned}
      {#key selectedTask.id}
        <TaskDetail
          task={selectedTask}
          parentTitle={selectedParentTitle}
          pinned={true}
          onClose={() => (selectedId = null)}
          onError={showToast}
          onDelete={(t: any) => requestDelete(t)}
          onTogglePin={togglePin}
          onSelectParent={(id) => (selectedId = id)}
          onAddSubtask={(pid) => {
            dialogParentId = pid
            dialogInitialStatus = 'pending'
            dialogOpen = true
          }}
        />
      {/key}
    {/if}
  </div>

  <footer class="flex h-9 flex-none items-center gap-4 border-t border-line-soft bg-chrome px-5 text-xs text-ink-3">
    <span class="truncate font-mono text-[11px]">{dbPath}</span>
    <div class="flex-1"></div>
    <span class="flex-none whitespace-nowrap"
      ><kbd>/</kbd> search · <kbd>n</kbd> new · <kbd>b</kbd>/<kbd>l</kbd>/<kbd>a</kbd> view ·
      <kbd>s</kbd> sub-tasks
      {#if view === 'list'}· <kbd>1–5</kbd> filter · <kbd>6</kbd> archived{/if} · <kbd>del</kbd> delete ·
      <kbd>esc</kbd> dismiss/hide · <kbd>ctrl+shift+alt+t</kbd> toggle · <kbd>ctrl+q</kbd> quit</span
    >
  </footer>

  {#if selectedTask && !detailPinned}
    {#key selectedTask.id}
      <TaskDetail
        task={selectedTask}
        parentTitle={selectedParentTitle}
        pinned={false}
        onClose={() => (selectedId = null)}
        onError={showToast}
        onDelete={(t: any) => requestDelete(t)}
        onTogglePin={togglePin}
        onSelectParent={(id) => (selectedId = id)}
        onAddSubtask={(pid) => {
          dialogParentId = pid
          dialogInitialStatus = 'pending'
          dialogOpen = true
        }}
      />
    {/key}
  {/if}

  <NewTaskDialog
    open={dialogOpen}
    initialStatus={dialogInitialStatus || 'pending'}
    parentId={dialogParentId}
    onClose={() => {
      dialogOpen = false
      dialogParentId = ''
    }}
    onError={showToast}
  />

  <ConfirmDialog
    open={confirmTask !== null}
    title="Delete task?"
    message={confirmMsg}
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
