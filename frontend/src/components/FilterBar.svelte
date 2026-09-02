<script lang="ts">
  import { STATUS_LABELS } from '../lib/format'
  import type { Status } from '../lib/api'
  import type { HumanFilter } from '../lib/humanFilter'
  import DropdownSelect from './DropdownSelect.svelte'

  let {
    status,
    search,
    sort,
    ascending,
    humanFilter,
    showSubtasks,
    showSort = true,
    taskCount,
    onStatusChange,
    onSearchInput,
    onSortChange,
    onToggleAsc,
    onHumanFilterChange,
    onToggleSubtasks
  }: {
    status: Status | '' | 'archived'
    search: string
    sort: string
    ascending: boolean
    humanFilter: HumanFilter
    showSubtasks: boolean
    showSort?: boolean
    taskCount?: number
    onStatusChange: (s: Status | '' | 'archived') => void
    onSearchInput: (v: string) => void
    onSortChange: (f: string) => void
    onToggleAsc: () => void
    onHumanFilterChange: (v: HumanFilter) => void
    onToggleSubtasks: () => void
  } = $props()

  const statusOptions: {
    value: Status | '' | 'archived'
    label: string
    activeClass: string
  }[] = [
    { value: '', label: 'All statuses', activeClass: 'text-accent-hi' },
    {
      value: 'pending',
      label: STATUS_LABELS.pending,
      activeClass: 'text-st-pending'
    },
    { value: 'wip', label: STATUS_LABELS.wip, activeClass: 'text-st-wip' },
    {
      value: 'waiting',
      label: STATUS_LABELS.waiting,
      activeClass: 'text-st-waiting'
    },
    {
      value: 'review',
      label: STATUS_LABELS.review,
      activeClass: 'text-st-review'
    },
    { value: 'done', label: STATUS_LABELS.done, activeClass: 'text-st-done' },
    { value: 'archived', label: 'Archived', activeClass: 'text-ink' }
  ]

  const humanOptions: { value: HumanFilter; label: string; title: string }[] = [
    { value: 'all', label: 'All tasks', title: 'Show all tasks' },
    { value: 'exclude', label: 'Agents', title: 'Hide human-only tasks' },
    { value: 'only', label: 'Human', title: 'Show human-only tasks' }
  ]

  const sortFields = ['board', 'updated', 'created', 'status', 'progress', 'title'] as const
  const sortLabels: Record<(typeof sortFields)[number], string> = {
    board: 'Board',
    updated: 'Updated',
    created: 'Created',
    status: 'Status',
    progress: 'Progress',
    title: 'Title'
  }
</script>

<div class="flex flex-wrap items-center gap-2 border-b border-line-soft px-5 py-2.5">
  <DropdownSelect
    value={status}
    options={statusOptions}
    ariaLabel="Filter by status"
    minWidth="8.5rem"
    onChange={onStatusChange}
  />

  <DropdownSelect
    value={humanFilter}
    options={humanOptions}
    ariaLabel="Filter by task owner"
    minWidth="6.5rem"
    onChange={onHumanFilterChange}
  />

  <label
    class="flex h-8 min-w-[12rem] flex-1 cursor-text items-center gap-2 rounded border border-line-soft bg-field px-2.5 shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] transition-colors focus-within:border-accent focus-within:ring-2 focus-within:ring-accent/25 sm:max-w-xs"
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
      oninput={(e) => onSearchInput(e.currentTarget.value)}
      class="min-w-0 flex-1 bg-transparent text-[13px] text-ink outline-none placeholder:text-ink-3"
    />
  </label>

  <button
    type="button"
    onclick={onToggleSubtasks}
    title={showSubtasks ? 'Hide sub-tasks' : 'Show sub-tasks'}
    aria-label={showSubtasks ? 'Hide sub-tasks' : 'Show sub-tasks'}
    aria-pressed={showSubtasks}
    class="grid h-8 w-8 shrink-0 place-items-center rounded border transition-colors
      {showSubtasks
        ? 'border-accent/50 bg-accent/15 text-accent-hi'
        : 'border-line-soft text-ink-3 hover:bg-card-hi hover:text-ink'}"
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
      <path d="M3 5v14" />
      <path d="M8 8h13" />
      <path d="M8 12h13" />
      <path d="M12 16h9" />
    </svg>
  </button>

  {#if showSort}
    <DropdownSelect
      value={sort}
      options={sortFields.map((f) => ({ value: f, label: sortLabels[f] }))}
      ariaLabel="Sort field"
      minWidth="6.5rem"
      onChange={onSortChange}
    />
    <button
      type="button"
      title="Toggle sort direction ({ascending ? 'ascending' : 'descending'})"
      onclick={onToggleAsc}
      class="grid h-8 w-8 shrink-0 place-items-center rounded border border-line-soft bg-field text-sm text-ink transition-colors hover:bg-card-hi"
    >
      {ascending ? '↑' : '↓'}
    </button>
  {:else if taskCount !== undefined}
    <span class="shrink-0 font-mono text-[11px] text-ink-3">
      {taskCount} {taskCount === 1 ? 'task' : 'tasks'}
    </span>
  {/if}
</div>
