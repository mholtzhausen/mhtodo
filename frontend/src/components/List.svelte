<script lang="ts">
  import { shortId, relTime } from '../lib/format'

  let {
    tasks,
    hasFilters,
    selectedId,
    onSelect
  }: { tasks: any[]; hasFilters: boolean; selectedId: string | null; onSelect: (id: string) => void } = $props()

  const badgeClass: Record<string, string> = {
    pending: 'bg-zinc-700/50 text-zinc-300',
    wip: 'bg-indigo-500/15 text-indigo-300',
    waiting: 'bg-amber-500/15 text-amber-300',
    done: 'bg-emerald-500/15 text-emerald-300'
  }
</script>

{#if tasks.length === 0}
  <div class="flex flex-col items-center gap-2 py-24 text-center">
    {#if hasFilters}
      <p class="text-sm text-zinc-500">No tasks match.</p>
      <p class="text-xs text-zinc-600">Clear the filters above to see everything.</p>
    {:else}
      <p class="text-sm text-zinc-500">No tasks yet.</p>
      <p class="text-xs text-zinc-600">Press <kbd>n</kbd> or use a column’s + button on the board to create one.</p>
    {/if}
  </div>
{:else}
  <table class="w-full border-separate border-spacing-y-1 text-sm">
    <thead>
      <tr class="text-left text-xs uppercase tracking-wide text-zinc-600">
        <th class="px-3 py-2 font-medium">ID</th>
        <th class="px-3 py-2 font-medium">Status</th>
        <th class="w-40 px-3 py-2 font-medium">Progress</th>
        <th class="px-3 py-2 font-medium">Title</th>
        <th class="w-28 px-3 py-2 text-right font-medium">Updated</th>
      </tr>
    </thead>
    <tbody>
      {#each tasks as t (t.id)}
        <tr
          role="button"
          tabindex="0"
          onclick={() => onSelect(t.id)}
          onkeydown={(e) => e.key === 'Enter' && onSelect(t.id)}
          class="cursor-pointer transition-colors
            {selectedId === t.id ? 'bg-indigo-500/10' : 'hover:bg-zinc-900'}"
        >
          <td class="rounded-l-lg bg-zinc-900/40 px-3 py-2.5 font-mono text-xs text-zinc-500">{shortId(t.id)}</td>
          <td class="bg-zinc-900/40 px-3 py-2.5">
            <span class="rounded-full px-2 py-0.5 text-xs font-medium {badgeClass[t.status] ?? badgeClass.pending}">
              {t.status}
            </span>
          </td>
          <td class="bg-zinc-900/40 px-3 py-2.5">
            <div class="flex items-center gap-2">
              <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-zinc-800">
                <div
                  class="h-full rounded-full bg-indigo-500 transition-all duration-150"
                  style="width: {t.progress}%"
                ></div>
              </div>
              <span class="w-9 text-right font-mono text-xs text-zinc-500">{t.progress}%</span>
            </div>
          </td>
          <td class="max-w-0 truncate bg-zinc-900/40 px-3 py-2.5 text-zinc-200" title={t.title}>{t.title}</td>
          <td class="rounded-r-lg bg-zinc-900/40 px-3 py-2.5 text-right text-xs text-zinc-500">{relTime(t.updated_at)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
