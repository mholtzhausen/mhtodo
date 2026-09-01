<script lang="ts">
  import { STATUS_LABELS } from '../lib/format'
  import type { Status } from '../lib/api'
  import HumanFilterChips from './HumanFilterChips.svelte'
  import type { HumanFilter } from '../lib/humanFilter'

  let {
    status,
    search,
    sort,
    ascending,
    humanFilter,
    onStatusChange,
    onSearchInput,
    onSortChange,
    onToggleAsc,
    onHumanFilterChange
  }: {
    // '' = all (hides archived); 'archived' = archive view; else a status chip.
    status: Status | '' | 'archived'
    search: string
    sort: string
    ascending: boolean
    humanFilter: HumanFilter
    onStatusChange: (s: Status | '' | 'archived') => void
    onSearchInput: (v: string) => void
    onSortChange: (f: string) => void
    onToggleAsc: () => void
    onHumanFilterChange: (v: HumanFilter) => void
  } = $props()

  // Archived is a neutral chip (no status color): it is a storage state, not
  // a workflow state. Single-select with the rest — picking any other chip
  // leaves the archive view and vice versa.
  const chips: { value: Status | '' | 'archived'; label: string; active: string }[] = [
    { value: '', label: 'All', active: 'border-accent/60 bg-accent/15 text-accent-hi' },
    {
      value: 'pending',
      label: STATUS_LABELS.pending,
      active: 'border-st-pending/60 bg-st-pending/15 text-st-pending'
    },
    { value: 'wip', label: STATUS_LABELS.wip, active: 'border-st-wip/70 bg-st-wip/20 text-st-wip' },
    {
      value: 'waiting',
      label: STATUS_LABELS.waiting,
      active: 'border-st-waiting/60 bg-st-waiting/15 text-st-waiting'
    },
    {
      value: 'review',
      label: STATUS_LABELS.review,
      active: 'border-st-review/60 bg-st-review/15 text-st-review'
    },
    { value: 'done', label: STATUS_LABELS.done, active: 'border-st-done/60 bg-st-done/15 text-st-done' },
    { value: 'archived', label: 'Archived', active: 'border-line bg-white/10 text-ink' }
  ]

  const sortFields = ['board', 'updated', 'created', 'status', 'progress', 'title'] as const
</script>

<div class="flex flex-wrap items-center gap-3 border-b border-line-soft px-5 py-3">
  <div
    class="flex gap-1 rounded border border-line-soft bg-field p-1 shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)]"
  >
    {#each chips as chip (chip.value)}
      <button
        onclick={() => onStatusChange(chip.value)}
        class="rounded-[3px] border border-transparent px-2.5 py-1 text-xs font-medium text-ink-2 transition-colors hover:bg-white/5 hover:text-ink
          {status === chip.value ? chip.active : ''}"
      >
        {chip.label}
      </button>
    {/each}
  </div>

  <HumanFilterChips value={humanFilter} onChange={onHumanFilterChange} />

  <input
    id="task-search"
    type="search"
    placeholder="Search… ( / )"
    value={search}
    oninput={(e) => onSearchInput(e.currentTarget.value)}
    class="w-56 rounded border border-line-soft bg-field px-3 py-1.5 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
  />

  <div class="flex-1"></div>

  <label for="sort-field" class="text-xs text-ink-3">Sort</label>
  <span class="relative">
    <select
      id="sort-field"
      value={sort}
      onchange={(e) => onSortChange(e.currentTarget.value)}
      class="appearance-none rounded border border-line-soft bg-field py-1.5 pl-2.5 pr-7 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
    >
      {#each sortFields as f (f)}
        <option value={f}>{f}</option>
      {/each}
    </select>
    <svg
      class="pointer-events-none absolute right-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-3"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <path d="m6 9 6 6 6-6" />
    </svg>
  </span>
  <button
    title="Toggle sort direction ({ascending ? 'ascending' : 'descending'})"
    onclick={onToggleAsc}
    class="rounded border border-line-soft bg-field px-2.5 py-1.5 text-sm text-ink transition-colors hover:bg-card-hi"
  >
    {ascending ? '↑' : '↓'}
  </button>
</div>
