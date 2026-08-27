<script lang="ts">
  import { absList, relTime, shortId } from '../lib/format'
  import type { Activity, Task } from '../lib/api'

  let {
    activities,
    tasks,
    selectedTaskIds,
    onToggleTask,
    onSelectTask,
    onClearFilter
  }: {
    activities: Activity[]
    tasks: Task[]
    selectedTaskIds: string[]
    onToggleTask: (id: string) => void
    onSelectTask: (id: string) => void
    onClearFilter: () => void
  } = $props()

  let filterOpen = $state(false)

  const taskById = $derived.by(() => {
    const m: Record<string, Task> = {}
    for (const t of tasks) m[t.id] = t
    return m
  })

  /** Root title, or `parent > child` when the ticket is a sub-task. */
  function ticketLabel(task: Task | undefined, taskId: string): string {
    if (!task) return shortId(taskId)
    if (!task.parent_id) return task.title
    const parent = taskById[task.parent_id]
    const parentTitle = parent?.title ?? shortId(task.parent_id)
    return `${parentTitle} > ${task.title}`
  }

  // Ticket picker lists non-archived roots + children currently in `tasks`.
  const ticketOptions = $derived(
    [...tasks].sort((a, b) => a.title.localeCompare(b.title))
  )

  const filtered = $derived(
    selectedTaskIds.length === 0
      ? activities
      : activities.filter((a) => selectedTaskIds.includes(a.task_id))
  )
</script>

<div class="flex h-full flex-col gap-3">
  <div class="relative flex flex-none items-center gap-3">
    <button
      type="button"
      onclick={() => (filterOpen = !filterOpen)}
      class="rounded border border-line-soft bg-field px-3 py-1.5 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] hover:bg-card-hi"
    >
      Filter by ticket
      {#if selectedTaskIds.length > 0}
        <span class="ml-1 font-mono text-xs text-accent-hi">({selectedTaskIds.length})</span>
      {/if}
    </button>
    {#if selectedTaskIds.length > 0}
      <button type="button" onclick={onClearFilter} class="text-xs text-ink-3 hover:text-ink">
        Clear
      </button>
    {/if}
    <div class="flex-1"></div>
    <span class="font-mono text-[11px] text-ink-3">
      {filtered.length} {filtered.length === 1 ? 'entry' : 'entries'}
    </span>

    {#if filterOpen}
      <div
        class="absolute left-0 top-full z-20 mt-1 max-h-64 w-80 overflow-y-auto rounded border border-line bg-card p-2 shadow-xl"
      >
        {#each ticketOptions as t (t.id)}
          <label class="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm hover:bg-white/5">
            <input
              type="checkbox"
              checked={selectedTaskIds.includes(t.id)}
              onchange={() => onToggleTask(t.id)}
              class="rounded border-line"
            />
            <span class="min-w-0 flex-1 truncate text-ink">{t.title}</span>
            <span class="font-mono text-[10px] text-ink-3">{shortId(t.id)}</span>
          </label>
        {:else}
          <p class="px-2 py-3 text-xs text-ink-3">No tickets</p>
        {/each}
      </div>
    {/if}
  </div>

  <div class="min-h-0 flex-1 overflow-y-auto rounded-md border border-line-soft bg-col shadow-sm">
    {#if filtered.length === 0}
      <div class="flex h-40 flex-col items-center justify-center gap-1 text-center">
        <p class="text-sm text-ink-3">No activity yet.</p>
        <p class="text-xs text-ink-3/70">Agents can post with <code class="font-mono">mhtodo activity add</code>.</p>
      </div>
    {:else}
      <ul class="space-y-1.5 p-2">
        {#each filtered as a (a.id)}
          {@const task = taskById[a.task_id]}
          {@const parent = task?.parent_id ? taskById[task.parent_id] : undefined}
          <li class="rounded bg-white/[0.03] px-3 py-2.5 hover:bg-white/[0.06]">
            <div class="mb-1.5 flex gap-3">
              <div class="flex w-28 flex-none flex-col gap-0.5 pt-0.5">
                <span class="text-xs text-ink-2">{relTime(a.created_at)}</span>
                <span class="font-mono text-[10px] text-ink-3">{absList(a.created_at)}</span>
              </div>
              <div
                class="flex min-w-0 flex-1 items-baseline gap-1.5 truncate text-sm font-medium"
                title={ticketLabel(task, a.task_id)}
              >
                {#if task?.parent_id}
                  <button
                    type="button"
                    onclick={() => onSelectTask(task.parent_id!)}
                    class="max-w-[55%] shrink truncate text-left text-accent-hi hover:underline"
                  >
                    {parent?.title ?? shortId(task.parent_id)}
                  </button>
                  <span class="shrink-0 text-ink-3" aria-hidden="true">></span>
                  <button
                    type="button"
                    onclick={() => onSelectTask(a.task_id)}
                    class="min-w-0 truncate text-left text-accent-hi hover:underline"
                  >
                    {task.title}
                  </button>
                {:else}
                  <button
                    type="button"
                    onclick={() => onSelectTask(a.task_id)}
                    class="truncate text-left text-accent-hi hover:underline"
                  >
                    {task?.title ?? shortId(a.task_id)}
                  </button>
                {/if}
              </div>
            </div>
            {#if a.activity || a.comment}
              <div class="flex min-w-0 items-baseline gap-2">
                {#if a.activity}
                  <span
                    class="inline-flex max-w-[40%] shrink-0 truncate rounded-full border border-accent/40 bg-accent/15 px-2 py-0.5 text-xs font-medium text-accent-hi"
                    title={a.activity}
                  >
                    {a.activity}
                  </span>
                  {#if a.comment}
                    <span class="shrink-0 text-ink-3" aria-hidden="true">—</span>
                  {/if}
                {/if}
                {#if a.comment}
                  <span class="min-w-0 flex-1 truncate text-sm text-ink-2" title={a.comment}>{a.comment}</span>
                {/if}
              </div>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</div>
