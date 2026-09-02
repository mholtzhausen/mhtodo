<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { fly } from 'svelte/transition'
  import Board from './components/Board.svelte'
  import FilterBar from './components/FilterBar.svelte'
  import List from './components/List.svelte'
  import ActivityView from './components/ActivityView.svelte'
  import TaskDetail from './components/TaskDetail.svelte'
  import NewTaskDialog from './components/NewTaskDialog.svelte'
  import SettingsDialog from './components/SettingsDialog.svelte'
  import ConfirmDialog from './components/ConfirmDialog.svelte'
  import { api, errMsg, type Activity, type GUISettings, type Status } from './lib/api'
  import { defaultSettings } from './lib/settings'
  import { boardAdjacentTaskId } from './lib/boardOrder'
  import { applyHumanFilter, loadHumanFilter, type HumanFilter } from './lib/humanFilter'

  const inWails = typeof window !== 'undefined' && !!(window as any).runtime

  type View = 'board' | 'list' | 'activity'
  type DetailMode = 'pinned' | 'floating' | 'modal'

  function loadDetailMode(): DetailMode {
    try {
      const stored = localStorage.getItem('mhtodo.detailMode')
      if (stored === 'pinned' || stored === 'floating' || stored === 'modal') return stored
      const legacy = localStorage.getItem('mhtodo.detailPinned')
      if (legacy === 'true') return 'pinned'
    } catch {
      /* ignore */
    }
    return 'floating'
  }

  const DETAIL_PANEL_WIDTH_DEFAULT = 420
  const DETAIL_PANEL_WIDTH_MIN = 280

  function maxDetailPanelWidth() {
    return Math.min(960, Math.floor(window.innerWidth * 0.75))
  }

  function loadDetailPanelWidth(): number {
    try {
      for (const key of ['mhtodo.detailPanelWidth', 'mhtodo.detailPinnedWidth']) {
        const stored = localStorage.getItem(key)
        if (stored) {
          const n = parseInt(stored, 10)
          if (!Number.isNaN(n)) {
            return Math.max(DETAIL_PANEL_WIDTH_MIN, Math.min(maxDetailPanelWidth(), n))
          }
        }
      }
    } catch {
      /* ignore */
    }
    return DETAIL_PANEL_WIDTH_DEFAULT
  }

  function clampDetailPanelWidth(w: number) {
    return Math.max(DETAIL_PANEL_WIDTH_MIN, Math.min(maxDetailPanelWidth(), w))
  }
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

  let detailMode = $state<DetailMode>(loadDetailMode())
  let detailPanelWidth = $state(loadDetailPanelWidth())
  let resizingDetail = $state(false)
  let alwaysOnTop = $state(false)

  let tasks = $state<any[]>([])
  let activities = $state<Activity[]>([])
  let activityFilterIds = $state<string[]>([])
  let loading = $state(true)
  const TOAST_MS = 3000
  let toast: { id: number; msg: string; kind: 'error' | 'info' } | null = $state(null)
  let toastSeq = 0
  let toastTimer: ReturnType<typeof setTimeout> | undefined
  let selectedId: string | null = $state(null)
  let dialogOpen = $state(false)
  let dialogInitialStatus = $state<Status | ''>('')
  let dialogParentId = $state('')
  let settingsOpen = $state(false)
  let guiSettings = $state<GUISettings>(defaultSettings())
  let confirmTask = $state<any | null>(null)
  let confirmMsg = $state('')
  let deleting = $state(false)
  let dbPath = $state('')

  let status = $state<Status | '' | 'archived'>('')
  let search = $state('')
  let sort = $state<'board' | 'created' | 'updated' | 'status' | 'progress' | 'title'>('board')
  let ascending = $state(false)
  let humanFilter = $state<HumanFilter>(loadHumanFilter())

  const displayTasks = $derived(applyHumanFilter(tasks, humanFilter))
  const displayRootCount = $derived(displayTasks.filter((t) => !t.parent_id).length)

  const selectedTask = $derived(tasks.find((t) => t.id === selectedId) ?? null)
  const selectedParentTitle = $derived(
    selectedTask?.parent_id
      ? (tasks.find((t) => t.id === selectedTask.parent_id)?.title ?? null)
      : null
  )

  function showToast(msg: string, kind: 'error' | 'info' = 'error', ms = TOAST_MS) {
    const id = ++toastSeq
    toast = { id, msg, kind }
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => {
      if (toast?.id === id) toast = null
    }, ms)
  }

  async function copySlackReport() {
    if (!inWails) {
      showToast('Running outside Wails — API unavailable (use `make dev`)')
      return
    }
    try {
      const report = await api.slackReport()
      const { ClipboardSetText } = await import('../wailsjs/runtime/runtime.js')
      const ok = await ClipboardSetText(report)
      if (ok) {
        showToast('Slack report copied', 'info')
      } else {
        showToast('Could not copy to clipboard')
      }
    } catch (e) {
      showToast(errMsg(e))
    }
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

  function setDetailMode(mode: DetailMode) {
    if (detailMode === mode) return
    detailMode = mode
    try {
      localStorage.setItem('mhtodo.detailMode', mode)
    } catch {
      /* ignore */
    }
  }

  function selectTask(id: string) {
    if (detailMode === 'modal' && selectedId) return
    selectedId = id
  }

  function navigateBoardTask(dir: -1 | 1) {
    if (detailMode !== 'modal' || !selectedId || view !== 'board') return
    const nextId = boardAdjacentTaskId(displayTasks, showSubtasks, selectedId, dir)
    if (nextId) selectedId = nextId
  }

  function setHumanFilter(v: HumanFilter) {
    if (humanFilter === v) return
    humanFilter = v
    try {
      localStorage.setItem('mhtodo.humanFilter', v)
    } catch {
      /* ignore */
    }
  }

  function persistDetailPanelWidth() {
    try {
      localStorage.setItem('mhtodo.detailPanelWidth', String(detailPanelWidth))
    } catch {
      /* ignore */
    }
  }

  function startDetailResize(e: PointerEvent) {
    e.preventDefault()
    e.stopPropagation()
    const startX = e.clientX
    const startWidth = detailPanelWidth
    resizingDetail = true
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'

    function onMove(ev: PointerEvent | MouseEvent) {
      if ('buttons' in ev && ev.buttons === 0) return
      detailPanelWidth = clampDetailPanelWidth(startWidth + startX - ev.clientX)
    }

    function onUp() {
      resizingDetail = false
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
      document.removeEventListener('pointermove', onMove, true)
      document.removeEventListener('pointerup', onUp, true)
      document.removeEventListener('pointercancel', onUp, true)
      document.removeEventListener('mousemove', onMove, true)
      document.removeEventListener('mouseup', onUp, true)
      persistDetailPanelWidth()
    }

    document.addEventListener('pointermove', onMove, true)
    document.addEventListener('pointerup', onUp, true)
    document.addEventListener('pointercancel', onUp, true)
    document.addEventListener('mousemove', onMove, true)
    document.addEventListener('mouseup', onUp, true)
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
        // Include done so ticket filter + hover tooltips cover the full non-archived set.
        const [t, a] = await Promise.all([
          api.list({ sort: 'title', ascending: true, includeDone: true }),
          api.listActivity({})
        ])
        tasks = t
        activities = a
      } else {
        const filter =
          view === 'board'
            ? status === 'archived'
              ? { archived: true, search, sort: 'board' as const, ascending: false }
              : { status, search, sort: 'board' as const, ascending: false }
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
      if (document.querySelector('[aria-haspopup="listbox"][aria-expanded="true"]')) return
      e.preventDefault()
      if (confirmTask) confirmTask = null
      else if (settingsOpen) settingsOpen = false
      else if (dialogOpen) {
        dialogOpen = false
        dialogParentId = ''
      } else if (selectedId && detailMode !== 'pinned') selectedId = null
      else api.hideWindow()
      return
    }
    if ((e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey && (e.key === 'q' || e.key === 'Q')) {
      e.preventDefault()
      api.quit()
      return
    }
    if (
      (e.key === 'ArrowLeft' || e.key === 'ArrowRight') &&
      detailMode === 'modal' &&
      selectedId &&
      view === 'board' &&
      !dialogOpen &&
      !confirmTask &&
      !typing &&
      !e.metaKey &&
      !e.ctrlKey &&
      !e.altKey
    ) {
      e.preventDefault()
      navigateBoardTask(e.key === 'ArrowLeft' ? -1 : 1)
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

  function onWindowResize() {
    const clamped = clampDetailPanelWidth(detailPanelWidth)
    if (clamped !== detailPanelWidth) {
      detailPanelWidth = clamped
      persistDetailPanelWidth()
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
    try {
      guiSettings = await api.getSettings()
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
    window.addEventListener('resize', onWindowResize)
    await load()
  })

  onDestroy(() => {
    clearTimeout(toastTimer)
    unbindChanged?.()
    unbindTrayNewTask?.()
    window.removeEventListener('keydown', onKeydown)
    window.removeEventListener('resize', onWindowResize)
    if (resizingDetail) {
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
  })
</script>

<div class="flex h-full flex-col {resizingDetail ? 'select-none' : ''}">
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
      onclick={() => (settingsOpen = true)}
      title="Settings"
      class="grid h-8 w-8 place-items-center rounded border border-line-soft text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
    >
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
        <path
          d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"
        />
        <circle cx="12" cy="12" r="3" />
      </svg>
    </button>

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
      {humanFilter}
      {showSubtasks}
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
      onHumanFilterChange={setHumanFilter}
      onToggleSubtasks={toggleSubtasks}
      onCopySlackReport={copySlackReport}
    />
  {:else if view === 'board'}
    <FilterBar
      {status}
      {search}
      sort="board"
      ascending={false}
      {humanFilter}
      {showSubtasks}
      showSort={false}
      taskCount={displayRootCount}
      onStatusChange={(s: Status | '' | 'archived') => {
        status = s
        load()
      }}
      onSearchInput={(v: string) => {
        search = v
        load()
      }}
      onSortChange={() => {}}
      onToggleAsc={() => {}}
      onHumanFilterChange={setHumanFilter}
      onToggleSubtasks={toggleSubtasks}
      onCopySlackReport={copySlackReport}
    />
  {/if}

  <div class="flex min-h-0 flex-1">
    <main
      class="min-h-0 flex-1 {view === 'board' ? '' : 'overflow-y-auto'} px-5 py-4"
    >
      {#if loading}
        <p class="text-sm text-ink-3">Loading…</p>
      {:else if view === 'board'}
        <Board
          tasks={displayTasks}
          {search}
          selectedId={selectedId}
          {showSubtasks}
          archiveDoneSubtasks={guiSettings.archive_done_subtasks}
          onSelect={selectTask}
          onQuickAdd={(s: Status) => {
            dialogInitialStatus = s
            dialogParentId = ''
            dialogOpen = true
          }}
          onArchived={(n: number) => showToast(`Archived ${n} task${n === 1 ? '' : 's'}`, 'info')}
          onError={showToast}
          onToast={showToast}
        />
      {:else if view === 'list'}
        <List
          tasks={displayTasks}
          hasFilters={status !== '' || search.trim() !== '' || humanFilter !== 'all'}
          selectedId={selectedId}
          {showSubtasks}
          onSelect={selectTask}
          onError={showToast}
          onToast={showToast}
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
          onSelectTask={selectTask}
          onClearFilter={() => (activityFilterIds = [])}
        />
      {/if}
    </main>

    {#if selectedTask && detailMode === 'pinned'}
      {#key selectedTask.id}
        <TaskDetail
          task={selectedTask}
          parentTitle={selectedParentTitle}
          mode="pinned"
          width={detailPanelWidth}
          resizing={resizingDetail}
          onResizeStart={startDetailResize}
          onClose={() => (selectedId = null)}
          onError={showToast}
          onDelete={(t: any) => requestDelete(t)}
          onSetMode={setDetailMode}
          onSelectParent={selectTask}
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
      ><kbd>/</kbd> search · <kbd>n</kbd> new · <kbd>b</kbd>/<kbd>l</kbd>/<kbd>a</kbd> view
      {#if view === 'list'}· <kbd>1–5</kbd> filter · <kbd>6</kbd> archived{/if} · <kbd>del</kbd> delete ·
      <kbd>esc</kbd> dismiss/hide · <kbd>ctrl+shift+alt+t</kbd> toggle · <kbd>ctrl+q</kbd> quit</span
    >
  </footer>

  {#if selectedTask && detailMode === 'floating'}
    {#key selectedTask.id}
      <TaskDetail
        task={selectedTask}
        parentTitle={selectedParentTitle}
        mode="floating"
        width={detailPanelWidth}
        resizing={resizingDetail}
        onResizeStart={startDetailResize}
        onClose={() => (selectedId = null)}
        onError={showToast}
        onDelete={(t: any) => requestDelete(t)}
        onSetMode={setDetailMode}
        onSelectParent={selectTask}
        onAddSubtask={(pid) => {
          dialogParentId = pid
          dialogInitialStatus = 'pending'
          dialogOpen = true
        }}
      />
    {/key}
  {/if}

  {#if selectedTask && detailMode === 'modal'}
    {#key selectedTask.id}
      <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-4 backdrop-blur-[2px]"
        onclick={() => (selectedId = null)}
      >
        <TaskDetail
          task={selectedTask}
          parentTitle={selectedParentTitle}
          mode="modal"
          onClose={() => (selectedId = null)}
          onError={showToast}
          onDelete={(t: any) => requestDelete(t)}
          onSetMode={setDetailMode}
          onAddSubtask={(pid) => {
            dialogParentId = pid
            dialogInitialStatus = 'pending'
            dialogOpen = true
          }}
        />
      </div>
    {/key}
  {/if}

  <NewTaskDialog
    open={dialogOpen}
    initialStatus={dialogInitialStatus || 'pending'}
    parentId={dialogParentId}
    defaultCwd={guiSettings.default_cwd}
    defaultHumanOnly={guiSettings.default_human_only}
    onClose={() => {
      dialogOpen = false
      dialogParentId = ''
    }}
    onError={showToast}
  />

  <SettingsDialog
    open={settingsOpen}
    onClose={() => (settingsOpen = false)}
    onSaved={(s) => (guiSettings = s)}
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
      in:fly={{ y: 8, duration: 150 }}
      out:fly={{ y: 8, duration: 150 }}
      class="fixed bottom-10 left-1/2 z-[60] -translate-x-1/2 rounded border px-4 py-2 text-sm shadow-xl
        {toast.kind === 'error'
          ? 'border-danger/50 bg-card-hi text-danger'
          : 'border-line bg-card-hi text-ink'}"
    >
      {toast.msg}
    </div>
  {/if}
</div>
