<script lang="ts">
  import { shortId, relTime, STATUS_LABELS } from '../lib/format'
  import type { Status } from '../lib/api'

  let {
    tasks,
    hasFilters,
    selectedId,
    onSelect
  }: { tasks: any[]; hasFilters: boolean; selectedId: string | null; onSelect: (id: string) => void } = $props()

  // Status badges: status color on a soft tinted fill (Slate design system).
  const badge: Record<Status, string> = {
    pending: 'border-st-pending/50 bg-st-pending/15 text-st-pending',
    wip: 'border-st-wip/60 bg-st-wip/20 text-st-wip',
    waiting: 'border-st-waiting/50 bg-st-waiting/15 text-st-waiting',
    done: 'border-st-done/50 bg-st-done/15 text-st-done'
  }
  const bar: Record<Status, string> = {
    pending: 'bg-st-pending',
    wip: 'bg-st-wip',
    waiting: 'bg-st-waiting',
    done: 'bg-st-done'
  }
</script>

{#if tasks.length === 0}
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
    <div class="min-h-0 flex-1 overflow-y-auto p-2">
      <table class="w-full border-separate border-spacing-y-1 text-sm">
        <thead>
          <tr class="text-left text-[11px] uppercase tracking-[0.07em] text-ink-3">
            <th class="px-3 py-2 font-medium">ID</th>
            <th class="px-3 py-2 font-medium">Status</th>
            <th class="w-40 px-3 py-2 font-medium">Progress</th>
            <th class="px-3 py-2 font-medium">Title</th>
            <th class="w-28 px-3 py-2 text-right font-medium">Updated</th>
          </tr>
        </thead>
        <tbody>
          {#each tasks as t (t.id)}
            {@const cell =
              selectedId === t.id ? 'bg-accent/10' : 'bg-white/[0.03] hover:bg-white/[0.06]'}
            <tr
              role="button"
              tabindex="0"
              onclick={() => onSelect(t.id)}
              onkeydown={(e) => e.key === 'Enter' && onSelect(t.id)}
              class="cursor-pointer"
            >
              <td class="rounded-l px-3 py-2.5 font-mono text-xs text-ink-3 transition-colors {cell}">
                {shortId(t.id)}
              </td>
              <td class="px-3 py-2.5 transition-colors {cell}">
                <span
                  class="inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium
                    {badge[t.status] ?? badge.pending}"
                >
                  <span class="h-1.5 w-1.5 rounded-full {bar[t.status] ?? bar.pending}"></span>
                  {STATUS_LABELS[t.status] ?? t.status}
                </span>
              </td>
              <td class="px-3 py-2.5 transition-colors {cell}">
                <div class="flex items-center gap-2">
                  <div class="h-[3px] flex-1 overflow-hidden rounded-full bg-white/10">
                    <div
                      class="h-full rounded-full {bar[t.status] ?? bar.pending} transition-all duration-150"
                      style="width: {t.progress}%"
                    ></div>
                  </div>
                  <span class="w-9 text-right font-mono text-[11px] text-ink-3">{t.progress}%</span>
                </div>
              </td>
              <td class="max-w-0 truncate px-3 py-2.5 text-ink transition-colors {cell}" title={t.title}>
                {t.title}
              </td>
              <td class="rounded-r px-3 py-2.5 text-right text-xs text-ink-3 transition-colors {cell}">
                {relTime(t.updated_at)}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
{/if}
