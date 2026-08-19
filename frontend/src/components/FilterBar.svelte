<script lang="ts">
  import { STATUS_LABELS } from '../lib/format'
  import type { Status } from '../lib/api'

  let {
    status,
    search,
    sort,
    ascending,
    onStatusChange,
    onSearchInput,
    onSortChange,
    onToggleAsc
  }: {
    status: string
    search: string
    sort: string
    ascending: boolean
    onStatusChange: (s: Status | '') => void
    onSearchInput: (v: string) => void
    onSortChange: (f: string) => void
    onToggleAsc: () => void
  } = $props()

  const chips: { value: Status | ''; label: string }[] = [
    { value: '', label: 'All' },
    { value: 'pending', label: STATUS_LABELS.pending },
    { value: 'wip', label: STATUS_LABELS.wip },
    { value: 'waiting', label: STATUS_LABELS.waiting },
    { value: 'done', label: STATUS_LABELS.done }
  ]

  const sortFields = ['updated', 'created', 'status', 'progress', 'title'] as const
</script>

<div class="flex flex-wrap items-center gap-3 border-b border-zinc-800 px-5 py-3">
  <div class="flex gap-1 rounded-lg bg-zinc-900 p-1">
    {#each chips as chip (chip.value)}
      <button
        onclick={() => onStatusChange(chip.value)}
        class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors
          {status === chip.value
            ? 'bg-indigo-600 text-white'
            : 'text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200'}"
      >
        {chip.label}
      </button>
    {/each}
  </div>

  <input
    id="task-search"
    type="search"
    placeholder="Search… ( / )"
    value={search}
    oninput={(e) => onSearchInput(e.currentTarget.value)}
    class="w-56 rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-sm text-zinc-200 placeholder:text-zinc-600 focus:border-indigo-500 focus:outline-none"
  />

  <div class="flex-1"></div>

  <label for="sort-field" class="text-xs text-zinc-500">Sort</label>
  <select
    id="sort-field"
    value={sort}
    onchange={(e) => onSortChange(e.currentTarget.value)}
    class="rounded-lg border border-zinc-800 bg-zinc-900 px-2 py-1.5 text-sm text-zinc-300 focus:border-indigo-500 focus:outline-none"
  >
    {#each sortFields as f (f)}
      <option value={f}>{f}</option>
    {/each}
  </select>
  <button
    title="Toggle sort direction ({ascending ? 'ascending' : 'descending'})"
    onclick={onToggleAsc}
    class="rounded-lg border border-zinc-800 bg-zinc-900 px-2.5 py-1.5 text-sm text-zinc-300 hover:bg-zinc-800"
  >
    {ascending ? '↑' : '↓'}
  </button>
</div>
