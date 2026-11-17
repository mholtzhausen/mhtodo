<script lang="ts">
  import { tick } from 'svelte'
  import { api, errMsg, type SlackLink } from '../lib/api'
  import { openExternalUrl } from '../lib/openExternal'
  import { slackLinksOf, slackLinkText } from '../lib/slack'
  import SlackIcon from './SlackIcon.svelte'
  import SlackLinkPopover from './SlackLinkPopover.svelte'

  // Ordered Slack conversations on a ticket (detail modal + Focus lanes).
  // First row is the primary thread (mirrored to slack_thread). Rows show the
  // description first; inline edit, reorder (↑ on row 2 = make primary) and
  // remove all write the full list via SetSlackLinks.
  let {
    task,
    readonly = false,
    compact = false,
    showHeader = true,
    onError
  }: {
    task: { id: string; slack_links?: SlackLink[] | null; slack_thread?: string }
    readonly?: boolean
    compact?: boolean
    showHeader?: boolean
    onError?: (msg: string) => void
  } = $props()

  const links = $derived(slackLinksOf(task))
  let editing = $state<number | null>(null)
  let draft = $state('')
  let editEl = $state<HTMLInputElement | null>(null)
  let busy = $state(false)

  async function write(next: SlackLink[]) {
    if (busy) return
    busy = true
    try {
      await api.setSlackLinks(task.id, next)
    } catch (e) {
      onError?.(errMsg(e))
    } finally {
      busy = false
    }
  }

  async function add(link: SlackLink) {
    // Errors propagate so the popover shows them inline.
    await api.addSlackLink(task.id, link, false)
  }

  function move(i: number, d: -1 | 1) {
    const j = i + d
    if (j < 0 || j >= links.length) return
    const next = [...links]
    ;[next[i], next[j]] = [next[j], next[i]]
    void write(next)
  }

  function remove(i: number) {
    void write(links.filter((_, k) => k !== i))
  }

  async function startEdit(i: number) {
    editing = i
    draft = links[i].description
    await tick()
    editEl?.focus()
    editEl?.select()
  }

  function commitEdit() {
    if (editing === null) return
    const i = editing
    editing = null
    const d = draft.trim()
    if (d === links[i]?.description) return
    void write(links.map((l, k) => (k === i ? { ...l, description: d } : l)))
  }

  function onEditKeydown(e: KeyboardEvent) {
    e.stopPropagation()
    if (e.key === 'Enter') {
      e.preventDefault()
      commitEdit()
    } else if (e.key === 'Escape') {
      e.preventDefault()
      editing = null
    }
  }

  const iconBtn =
    'grid h-5 w-5 shrink-0 place-items-center rounded-chip text-ink-3 transition-colors hover:bg-white/8 hover:text-ink disabled:opacity-30 disabled:hover:bg-transparent'
</script>

<div class="min-w-0">
  {#if showHeader}
    <div class="mb-1.5 flex items-center gap-2">
      <span class={compact ? 'text-[10px] font-semibold uppercase tracking-[0.07em] text-ink-3' : 'micro'}>Slack{compact ? '' : ' conversations'}</span>
      {#if links.length > 0}<span class="font-mono text-[10px] text-ink-3">{links.length}</span>{/if}
      <span class="flex-1"></span>
      {#if !readonly}<SlackLinkPopover existing={links} onAdd={add} {compact} />{/if}
    </div>
  {/if}

  {#if links.length === 0}
    {#if showHeader && !compact}
      <p class="text-[11px] text-ink-3">
        {readonly ? 'No Slack conversations.' : 'None yet — use + to paste a thread, channel or DM link.'}
      </p>
    {/if}
  {:else}
    <ul class="divide-y divide-line-soft overflow-hidden rounded-control border border-line-soft bg-field/60" aria-label="Slack conversations">
      {#each links as link, i (link.url)}
        {@const text = slackLinkText(link)}
        <li class="group flex items-center gap-2 {compact ? 'px-2 py-1' : 'px-2.5 py-1.5'}">
          <SlackIcon class="h-3 w-3 shrink-0 {i === 0 ? '' : 'opacity-70'}" title="" />
          <div class="min-w-0 flex-1">
            {#if editing === i}
              <input
                bind:this={editEl}
                bind:value={draft}
                onkeydown={onEditKeydown}
                onblur={commitEdit}
                maxlength="200"
                placeholder="Description (optional)"
                aria-label="Edit description"
                class="w-full rounded-chip border border-accent bg-field px-1.5 py-0.5 text-xs text-ink focus:outline-none focus:ring-2 focus:ring-accent/25"
              />
            {:else}
              <a
                href={link.url}
                title={link.url}
                onclick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  void openExternalUrl(link.url)
                }}
                class="flex min-w-0 items-baseline gap-1.5 hover:underline"
              >
                <span class="truncate text-xs {link.description ? 'text-ink' : 'text-ink-2'}">{text.primary}</span>
                {#if text.secondary && !compact}
                  <span class="shrink-0 truncate text-[10px] text-ink-3">{text.secondary}</span>
                {/if}
              </a>
            {/if}
          </div>
          {#if i === 0 && links.length > 1}
            <span class="shrink-0 rounded-chip bg-accent/15 px-1 py-px text-[9px] font-semibold uppercase tracking-wide text-accent-hi" title="Primary thread (slack_thread)">primary</span>
          {/if}
          {#if !readonly && editing !== i}
            <div class="flex shrink-0 items-center opacity-0 transition-opacity focus-within:opacity-100 group-hover:opacity-100">
              <button type="button" class={iconBtn} disabled={busy} title="Edit description" aria-label="Edit description" onclick={() => void startEdit(i)}>
                <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 20h9" /><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" /></svg>
              </button>
              {#if links.length > 1}
                <button type="button" class={iconBtn} disabled={busy || i === 0} title={i === 1 ? 'Make primary' : 'Move up'} aria-label={i === 1 ? 'Make primary' : 'Move up'} onclick={() => move(i, -1)}>
                  <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m18 15-6-6-6 6" /></svg>
                </button>
                <button type="button" class={iconBtn} disabled={busy || i === links.length - 1} title="Move down" aria-label="Move down" onclick={() => move(i, 1)}>
                  <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
                </button>
              {/if}
              <button type="button" class="{iconBtn} hover:!text-danger" disabled={busy} title="Remove link" aria-label="Remove link" onclick={() => remove(i)}>
                <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M18 6 6 18M6 6l12 12" /></svg>
              </button>
            </div>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>
