<script lang="ts">
  import { relTime, absList, STATUS_LABELS } from '../lib/format'
  import type { Status } from '../lib/api'

  let {
    tasks,
    hasFilters,
    selectedId,
    showSubtasks,
    onSelect
  }: {
    tasks: any[]
    hasFilters: boolean
    selectedId: string | null
    showSubtasks: boolean
    onSelect: (id: string) => void
  } = $props()

  const badge: Record<Status, string> = {
    pending: 'border-st-pending/50 bg-st-pending/15 text-st-pending',
    wip: 'border-st-wip/60 bg-st-wip/20 text-st-wip',
    waiting: 'border-st-waiting/50 bg-st-waiting/15 text-st-waiting',
    review: 'border-st-review/50 bg-st-review/15 text-st-review',
    done: 'border-st-done/50 bg-st-done/15 text-st-done'
  }
  const bar: Record<Status, string> = {
    pending: 'bg-st-pending',
    wip: 'bg-st-wip',
    waiting: 'bg-st-waiting',
    review: 'bg-st-review',
    done: 'bg-st-done'
  }

  const rows = $derived.by(() => {
    const roots = tasks.filter((t) => !t.parent_id)
    const childrenOf = (pid: string) => tasks.filter((t) => t.parent_id === pid)
    const out: { task: any; depth: number }[] = []
    if (!showSubtasks) {
      for (const t of roots) out.push({ task: t, depth: 0 })
      return out
    }
    const rootIds = new Set(roots.map((t) => t.id))
    for (const t of roots) {
      out.push({ task: t, depth: 0 })
      for (const c of childrenOf(t.id)) out.push({ task: c, depth: 1 })
    }
    for (const t of tasks) {
      if (t.parent_id && !rootIds.has(t.parent_id) && !roots.some((r) => r.id === t.id)) {
        if (!out.some((r) => r.task.id === t.id)) out.push({ task: t, depth: 0 })
      }
    }
    return out
  })
</script>

{#if rows.length === 0}
  <div class="flex h-full flex-col items-center justify-center gap-2 text-center">
    {#if hasFilters}
      <p class="text-sm text-ink-3">No tasks match.</p>
      <p class="text-xs text-ink-3/70">Clear the filters above to see everything.</p>
    {:else}
      <p class="text-sm text-ink-3">No tasks yet.</p>
      <p class="text-xs text-ink-3/70">Press <kbd>n</kbd> or use a column’s + button on the board to create one.</p>
    {/if}
  </div>
{:else}
  <div class="flex h-full flex-col overflow-hidden rounded-md border border-line-soft bg-col shadow-sm">
    <div class="flex flex-none items-center gap-3 px-3 py-2 text-[11px] uppercase tracking-[0.07em] text-ink-3">
      <span class="w-28 flex-none font-medium">Status</span>
      <span class="min-w-0 flex-1 font-medium">Title</span>
      <span class="w-36 flex-none text-right font-medium">Updated</span>
    </div>
    <div class="min-h-0 flex-1 space-y-1 overflow-y-auto p-2 pt-0">
      {#each rows as { task: t, depth } (t.id)}
        <button
          type="button"
          onclick={() => onSelect(t.id)}
          class="flex items-center gap-3 rounded px-3 py-2.5 text-left transition-colors
            {depth > 0
              ? 'ml-5 w-[calc(100%-1.25rem)] border-l-2 border-line'
              : 'w-full'}
            {selectedId === t.id ? 'bg-accent/10' : 'bg-white/[0.03] hover:bg-white/[0.06]'}"
        >
          <div class="w-28 flex-none">
            <div class="flex flex-col gap-1.5">
              <span
                class="inline-flex w-fit items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium
                  {badge[t.status] ?? badge.pending}"
              >
                <span class="h-1.5 w-1.5 rounded-full {bar[t.status] ?? bar.pending}"></span>
                {STATUS_LABELS[t.status] ?? t.status}
              </span>
              <div class="h-[3px] overflow-hidden rounded-full bg-white/10">
                <div
                  class="h-full rounded-full {bar[t.status] ?? bar.pending} transition-all duration-150"
                  style="width: {t.progress}%"
                ></div>
              </div>
            </div>
          </div>
          <span
            class="min-w-0 flex-1 truncate text-sm {depth > 0 ? 'text-ink-2' : 'text-ink'}"
            title={t.title}
          >
            {t.title}
          </span>
          <div class="flex w-36 flex-none flex-col items-end gap-0.5">
            <span class="text-xs text-ink-2">{relTime(t.updated_at)}</span>
            <span class="font-mono text-[10px] text-ink-3">{absList(t.updated_at)}</span>
          </div>
        </button>
      {/each}
    </div>
  </div>
{/if}
