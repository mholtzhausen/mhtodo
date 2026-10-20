<script lang="ts">
  import { relTime, STATUS_LABELS } from '../lib/format'
  import { api, errMsg, type Activity, type Status } from '../lib/api'
  import { sortSubtasksByCreated } from '../lib/boardOrder'
  import { FOCUS_MAX } from '../lib/focus'
  import type { GUISettings } from '../lib/settings'
  import Markdown from './Markdown.svelte'
  import ProgressControl from './ProgressControl.svelte'
  import TaskActivityActions from './TaskActivityActions.svelte'
  import FocusIcon from './FocusIcon.svelte'
  import InlineEdit from './InlineEdit.svelte'
  import StatusPicker from './StatusPicker.svelte'

  let {
    tasks,
    activities = [],
    focusIds,
    selectedId,
    settings = null,
    zedBinaryOk = false,
    onSelect,
    onUnpin,
    onMove,
    onGoToBoard,
    onError,
    onToast
  }: {
    tasks: any[]
    activities?: Activity[]
    focusIds: string[]
    selectedId: string | null
    settings?: GUISettings | null
    zedBinaryOk?: boolean
    onSelect: (id: string) => void
    onUnpin: (id: string) => void
    onMove: (id: string, beforeId: string | null) => void
    onGoToBoard: () => void
    onError?: (msg: string) => void
    onToast?: (msg: string, kind?: 'error' | 'info') => void
  } = $props()

  const statusDot: Record<string, string> = {
    icebox: 'bg-st-icebox',
    pending: 'bg-st-pending',
    wip: 'bg-st-wip',
    waiting: 'bg-st-waiting',
    review: 'bg-st-review',
    pr: 'bg-st-pr',
    done: 'bg-st-done'
  }
  const statusEdge: Record<string, string> = {
    icebox: 'border-t-st-icebox',
    pending: 'border-t-st-pending',
    wip: 'border-t-st-wip',
    waiting: 'border-t-st-waiting',
    review: 'border-t-st-review',
    pr: 'border-t-st-pr',
    done: 'border-t-st-done'
  }

  const LANE_MIN = 260
  const LANE_GAP = 12

  const pinned = $derived.by(() => {
    const byId = new Map(tasks.map((t) => [t.id, t]))
    return focusIds.map((id) => byId.get(id)).filter((t): t is any => !!t)
  })

  const childrenOf = $derived.by(() => {
    const m: Record<string, any[]> = {}
    for (const t of tasks) {
      if (!t.parent_id) continue
      ;(m[t.parent_id] ??= []).push(t)
    }
    for (const pid of Object.keys(m)) m[pid] = sortSubtasksByCreated(m[pid])
    return m
  })

  // Newest first (core order); a lane shows its task's and its sub-tasks' entries.
  const activityOf = (t: any, kids: any[]): Activity[] => {
    const ids = new Set([t.id, ...kids.map((c) => c.id)])
    return activities.filter((a) => ids.has(a.task_id))
  }
  const titleOf = (id: string) => tasks.find((x) => x.id === id)?.title ?? id.slice(0, 8)

  let expanded = $state<Record<string, boolean>>({})
  const toggleExpanded = (id: string) => (expanded = { ...expanded, [id]: !expanded[id] })

  const parentOf = (t: any): any | null =>
    t.parent_id ? (tasks.find((x) => x.id === t.parent_id) ?? null) : null

  const gridStyle = $derived(
    `grid-template-columns: repeat(${pinned.length}, minmax(${LANE_MIN}px, 1fr)); grid-template-rows: minmax(0, 1fr); min-width: ${
      pinned.length * LANE_MIN + Math.max(0, pinned.length - 1) * LANE_GAP
    }px`
  )

  let draggingId = $state<string | null>(null)
  let overId = $state<string | null>(null)

  function onDragStart(e: DragEvent, id: string) {
    draggingId = id
    e.dataTransfer?.setData('text/plain', id)
    if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
  }

  function onDragOver(e: DragEvent, id: string) {
    if (!draggingId) return
    e.preventDefault()
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
    overId = id
  }

  function onDrop(e: DragEvent, targetId: string) {
    e.preventDefault()
    const from = draggingId
    draggingId = null
    overId = null
    if (!from || from === targetId) return
    const ids = pinned.map((t) => t.id)
    const fromIdx = ids.indexOf(from)
    const toIdx = ids.indexOf(targetId)
    if (fromIdx < 0 || toIdx < 0) return
    ids.splice(fromIdx, 1)
    ids.splice(toIdx, 0, from)
    onMove(from, ids[toIdx + 1] ?? null)
  }

  function onDragEnd() {
    draggingId = null
    overId = null
  }

  async function markDone(t: any) {
    try {
      await api.setStatus(t.id, 'done')
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  async function saveProgress(t: any, p: number) {
    try {
      await api.update(t.id, { progress: p })
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  async function saveField(t: any, patch: { title?: string; description?: string; feedback?: string }) {
    try {
      await api.update(t.id, patch)
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  async function setStatus(t: any, s: Status) {
    if (s === t.status) return
    try {
      await api.setStatus(t.id, s)
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  // Per-lane drafts for the quick-add sub-task and activity forms.
  let subDraft = $state<Record<string, string>>({})
  let actDraft = $state<Record<string, string>>({})
  let commentDraft = $state<Record<string, string>>({})
  let posting = $state<Record<string, boolean>>({})

  // Seeds cwd, Slack thread and human-only from the parent; not in the report
  // (same defaults as the new-task dialog's sub-task mode).
  async function addSubtask(t: any) {
    const title = (subDraft[t.id] ?? '').trim()
    if (!title || posting[t.id]) return
    posting = { ...posting, [t.id]: true }
    try {
      await api.create({
        title,
        parentId: t.id,
        status: 'pending',
        cwd: t.cwd ?? '',
        slackThread: t.slack_thread || undefined,
        humanOnly: !!t.human_only,
        includeInReport: false
      })
      subDraft = { ...subDraft, [t.id]: '' }
    } catch (err) {
      onError?.(errMsg(err))
    } finally {
      posting = { ...posting, [t.id]: false }
    }
  }

  async function postActivity(t: any) {
    const activity = (actDraft[t.id] ?? '').trim()
    const comment = (commentDraft[t.id] ?? '').trim()
    if ((!activity && !comment) || posting[t.id]) return
    posting = { ...posting, [t.id]: true }
    try {
      await api.addActivity(t.id, { activity, comment })
      actDraft = { ...actDraft, [t.id]: '' }
      commentDraft = { ...commentDraft, [t.id]: '' }
    } catch (err) {
      onError?.(errMsg(err))
    } finally {
      posting = { ...posting, [t.id]: false }
    }
  }

  const sectionLabel = 'mb-1 text-[10px] font-semibold uppercase tracking-[0.07em] text-ink-3'
  const fieldClass =
    'w-full rounded-control border border-line-soft bg-field px-2 py-1 text-[12px] text-ink placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25'
</script>

{#if pinned.length === 0}
  <div class="flex h-full flex-col items-center justify-center gap-3 text-center">
    <FocusIcon class="h-8 w-8 text-ink-3/50" />
    <p class="text-sm text-ink-3">Nothing in focus.</p>
    <p class="max-w-sm text-xs text-ink-3/70">
      Pin up to {FOCUS_MAX} tickets with the focus button on a board card or in the task detail
      header. Done tickets leave focus automatically.
    </p>
    <button
      type="button"
      onclick={onGoToBoard}
      class="mt-1 rounded-control border border-line-soft px-3 py-1.5 text-xs text-ink-2 transition-colors hover:bg-white/5 hover:text-ink"
    >
      Go to Board
    </button>
  </div>
{:else}
  <div class="h-full min-h-0 overflow-x-auto">
    <div class="grid h-full gap-3" style={gridStyle}>
      {#each pinned as t (t.id)}
        {@const kids = childrenOf[t.id] ?? []}
        {@const acts = activityOf(t, kids)}
        {@const parent = parentOf(t)}
        {@const isSub = !!t.parent_id}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <section
          data-task-id={t.id}
          ondragover={(e) => onDragOver(e, t.id)}
          ondrop={(e) => onDrop(e, t.id)}
          class="flex min-h-0 flex-col overflow-hidden rounded-card border border-t-2 shadow-sm
            {statusEdge[t.status] ?? statusEdge.pending}
            {isSub ? 'border-l-2 border-l-accent/60' : ''}
            {overId === t.id && draggingId && draggingId !== t.id
              ? 'border-accent/60 bg-accent/5'
              : selectedId === t.id
                ? 'border-line-soft bg-accent/10'
                : isSub
                  ? 'border-line-soft bg-field'
                  : 'border-line-soft bg-card'}
            {draggingId === t.id ? 'opacity-60' : ''}"
        >
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <header
            draggable="true"
            ondragstart={(e) => onDragStart(e, t.id)}
            ondragend={onDragEnd}
            class="flex flex-none cursor-grab items-center gap-2 border-b border-line-soft px-3 py-2"
            title="Drag to reorder focus"
          >
            <span class="h-2 w-2 flex-none rounded-full {statusDot[t.status] ?? statusDot.pending}"></span>
            <span class="text-[11px] font-semibold uppercase tracking-[0.07em] text-ink-2">
              {STATUS_LABELS[t.status] ?? t.status}
            </span>
            <span class="text-[10.5px] text-ink-3">{relTime(t.updated_at)}</span>
            <div class="flex-1"></div>
            <button
              type="button"
              onclick={() => onSelect(t.id)}
              title="Open task details"
              aria-label={`Open “${t.title}” details`}
              class="rounded-control p-0.5 text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
            >
              <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M15 3h6v6" />
                <path d="M9 21H3v-6" />
                <path d="M21 3l-7 7" />
                <path d="M3 21l7-7" />
              </svg>
            </button>
            <button
              type="button"
              onclick={() => void markDone(t)}
              title="Mark done (leaves focus)"
              aria-label={`Mark “${t.title}” done`}
              class="flex h-4 w-4 items-center justify-center rounded-chip border border-ink-3/50 text-transparent transition-colors hover:border-st-done hover:bg-st-done/20 hover:text-st-done"
            >
              <svg class="h-3 w-3" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
                <path d="M3.5 8.5 6.5 11.5 12.5 4.5" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </button>
            <button
              type="button"
              onclick={() => onUnpin(t.id)}
              title="Remove from focus"
              aria-label={`Remove “${t.title}” from focus`}
              class="rounded-control p-0.5 text-accent transition-colors hover:bg-white/5 hover:text-ink-3"
            >
              <FocusIcon class="h-3.5 w-3.5" filled />
            </button>
          </header>

          <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto px-3 py-3">
            <div>
              {#if parent}
                <button
                  type="button"
                  onclick={() => onSelect(t.parent_id)}
                  class="mb-1.5 flex max-w-full items-center gap-1.5 rounded-control border border-line-soft bg-card/60 px-2 py-1 text-left text-[11px] text-ink-3 hover:border-accent/40 hover:text-ink-2"
                  title={parent.focus_rank != null
                    ? 'Parent is pinned too — its lane returns when this sub-task leaves focus'
                    : 'Open parent ticket'}
                >
                  <span class="flex-none text-[10px] uppercase tracking-[0.06em] text-ink-3/70">Parent</span>
                  <span class="min-w-0 flex-1 truncate">{parent.title}</span>
                  {#if parent.focus_rank != null}
                    <FocusIcon class="h-3 w-3 flex-none text-accent/70" filled />
                  {/if}
                </button>
              {/if}
              <InlineEdit
                value={t.title}
                onSave={(v) => saveField(t, { title: v })}
                class="text-[15px] font-semibold leading-snug text-ink"
                inputClass="text-[15px] font-semibold"
              />
            </div>

            <StatusPicker
              value={t.status}
              visibleStatuses={settings?.visible_statuses}
              onPick={(s) => void setStatus(t, s)}
            />

            <ProgressControl value={t.progress} onCommit={(p) => void saveProgress(t, p)} />

            <div>
              <p class={sectionLabel}>Description</p>
              <InlineEdit
                markdown
                value={t.description}
                onSave={(v) => saveField(t, { description: v })}
                empty="Add a description…"
                placeholder="Notes, links, context… (markdown)"
                class="py-0.5 text-[13px] text-ink-2"
                inputClass="text-[13px]"
              />
            </div>

            <div class="rounded-control border border-line-soft bg-field/40 px-2.5 py-2">
              <p class={sectionLabel}>Feedback</p>
              <InlineEdit
                markdown
                value={t.feedback}
                onSave={(v) => saveField(t, { feedback: v })}
                empty="No feedback yet."
                placeholder="Feedback… (markdown)"
                class="text-[12.5px] text-ink-2"
                inputClass="text-[12.5px]"
              />
            </div>

            {#if !isSub}
              <div>
                <p class={sectionLabel}>
                  Sub-tasks{#if kids.length > 0}&nbsp;· {kids.filter((c) => c.status === 'done').length}/{kids.length}{/if}
                </p>
                <ul class="space-y-1">
                  {#each kids as c (c.id)}
                    {@const open = !!expanded[c.id]}
                    <li
                      class="rounded-control border {open
                        ? 'border-line-soft bg-field/40'
                        : 'border-transparent'}"
                    >
                      <button
                        type="button"
                        onclick={() => toggleExpanded(c.id)}
                        aria-expanded={open}
                        class="flex w-full items-center gap-2 rounded-control px-1.5 py-1 text-left hover:bg-white/5
                          {selectedId === c.id ? 'bg-accent/10' : ''}
                          {c.status === 'done' && !open ? 'opacity-60' : ''}"
                      >
                        <svg
                          class="h-3 w-3 flex-none text-ink-3 transition-transform {open ? 'rotate-90' : ''}"
                          viewBox="0 0 16 16"
                          fill="none"
                          stroke="currentColor"
                          stroke-width="2"
                          aria-hidden="true"
                        >
                          <path d="M6 4l4 4-4 4" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                        <span
                          class="h-1.5 w-1.5 flex-none rounded-full {statusDot[c.status] ?? statusDot.pending}"
                          title={STATUS_LABELS[c.status] ?? c.status}
                        ></span>
                        <span
                          class="min-w-0 flex-1 truncate text-[12px] text-ink-2
                            {c.status === 'done' ? 'line-through' : ''}">{c.title}</span
                        >
                        <span class="font-mono text-[10px] text-ink-3">{c.progress}%</span>
                      </button>
                      {#if open}
                        <div class="flex flex-col gap-2 px-2 pb-2 pt-1">
                          <div class="flex items-center gap-2 text-[10.5px] text-ink-3">
                            <span>{STATUS_LABELS[c.status] ?? c.status}</span>
                            <span>·</span>
                            <span>{relTime(c.updated_at)}</span>
                            <div class="flex-1"></div>
                            <button
                              type="button"
                              onclick={() => onSelect(c.id)}
                              class="rounded-control px-1.5 py-0.5 text-accent hover:bg-white/5 hover:text-accent-hi"
                            >
                              Open
                            </button>
                          </div>
                          <InlineEdit
                            value={c.title}
                            onSave={(v) => saveField(c, { title: v })}
                            class="text-[12.5px] font-medium text-ink"
                            inputClass="text-[12.5px]"
                          />
                          <StatusPicker
                            value={c.status}
                            visibleStatuses={settings?.visible_statuses}
                            onPick={(s) => void setStatus(c, s)}
                          />
                          <ProgressControl value={c.progress} onCommit={(p) => void saveProgress(c, p)} />
                          <InlineEdit
                            markdown
                            value={c.description}
                            onSave={(v) => saveField(c, { description: v })}
                            empty="Add a description…"
                            placeholder="Notes, links, context… (markdown)"
                            class="py-0.5 text-[12.5px] text-ink-2"
                            inputClass="text-[12.5px]"
                          />
                          <div class="rounded-control border border-line-soft bg-card/50 px-2 py-1.5">
                            <p class={sectionLabel}>Feedback</p>
                            <InlineEdit
                              markdown
                              value={c.feedback}
                              onSave={(v) => saveField(c, { feedback: v })}
                              empty="No feedback yet."
                              placeholder="Feedback… (markdown)"
                              class="text-[12px] text-ink-2"
                              inputClass="text-[12px]"
                            />
                          </div>
                        </div>
                      {/if}
                    </li>
                  {/each}
                </ul>
                <form
                  class="mt-1.5"
                  onsubmit={(e) => {
                    e.preventDefault()
                    void addSubtask(t)
                  }}
                >
                  <input
                    bind:value={subDraft[t.id]}
                    disabled={posting[t.id]}
                    placeholder="+ Add sub-task (Enter)"
                    onkeydown={(e) => {
                      if (e.key === 'Escape') {
                        e.stopPropagation()
                        subDraft = { ...subDraft, [t.id]: '' }
                        ;(e.currentTarget as HTMLInputElement).blur()
                      }
                    }}
                    class={fieldClass}
                  />
                </form>
              </div>
            {/if}

            <div>
              <p class={sectionLabel}>
                Activity{#if acts.length > 0}&nbsp;· {acts.length}{/if}
              </p>
              <form
                class="mb-2 space-y-1.5"
                onsubmit={(e) => {
                  e.preventDefault()
                  void postActivity(t)
                }}
              >
                <input
                  bind:value={actDraft[t.id]}
                  disabled={posting[t.id]}
                  placeholder="Activity summary… (Enter to post)"
                  onkeydown={(e) => {
                    if (e.key === 'Escape') {
                      e.stopPropagation()
                      ;(e.currentTarget as HTMLInputElement).blur()
                    }
                  }}
                  class={fieldClass}
                />
                {#if (actDraft[t.id] ?? '').trim() || (commentDraft[t.id] ?? '').trim()}
                  <textarea
                    bind:value={commentDraft[t.id]}
                    disabled={posting[t.id]}
                    rows="2"
                    placeholder="Optional comment… (markdown, Ctrl+Enter to post)"
                    onkeydown={(e) => {
                      if (e.key === 'Escape') {
                        e.stopPropagation()
                        ;(e.currentTarget as HTMLTextAreaElement).blur()
                      } else if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
                        e.preventDefault()
                        void postActivity(t)
                      }
                    }}
                    class="{fieldClass} resize-y"
                  ></textarea>
                  <div class="flex justify-end">
                    <button
                      type="submit"
                      disabled={posting[t.id]}
                      class="rounded-control bg-accent px-2.5 py-1 text-[11px] font-medium text-accent-ink disabled:opacity-40"
                    >
                      Post
                    </button>
                  </div>
                {/if}
              </form>
              {#if acts.length > 0}
                <ul class="space-y-2 border-l border-line-soft pl-2.5">
                  {#each acts as a (a.id)}
                    <li class="text-[12px] leading-snug">
                      <div class="flex items-center gap-1.5 text-[10.5px] text-ink-3">
                        <span title={a.created_at}>{relTime(a.created_at)}</span>
                        {#if a.task_id !== t.id}
                          <span>·</span>
                          <button
                            type="button"
                            onclick={() => onSelect(a.task_id)}
                            class="min-w-0 truncate hover:text-ink-2"
                          >
                            {titleOf(a.task_id)}
                          </button>
                        {/if}
                      </div>
                      {#if a.activity}
                        <span class="font-semibold text-ink">{a.activity}</span>
                      {/if}
                      {#if a.comment}
                        <Markdown source={a.comment} class="text-[12px] text-ink-2" />
                      {/if}
                    </li>
                  {/each}
                </ul>
              {/if}
            </div>
          </div>

          <div class="flex flex-none justify-end border-t border-line-soft px-2 py-1.5">
            <TaskActivityActions task={t} {settings} {zedBinaryOk} {onError} {onToast} />
          </div>
        </section>
      {/each}
    </div>
  </div>
{/if}
