<script lang="ts">
  import { onMount } from 'svelte'
  import { api, errMsg, type Status } from '../lib/api'
  import {
    STATUS_OPTIONS,
    toggleStatusInOrder,
    type GUISettings,
    type TaskStatusId
  } from '../lib/settings'
  import {
    STATUS_COLOR_TOKEN,
    applyThemeTokens,
    colorInputValue,
    SLATE_FACTORY_TOKENS,
    type Theme
  } from '../lib/themes'

  let {
    statusId,
    settings = $bindable(),
    activeTheme,
    onThemeSaved,
    onError
  }: {
    statusId: TaskStatusId
    settings: GUISettings
    activeTheme: Theme | null
    onThemeSaved: (t: Theme) => void
    onError: (msg: string) => void
  } = $props()

  const label = $derived(STATUS_OPTIONS.find((s) => s.id === statusId)?.label ?? statusId)
  const tokenKey = $derived(STATUS_COLOR_TOKEN[statusId] ?? '')
  const visible = $derived(settings.visible_statuses.includes(statusId))
  const onlyVisible = $derived(settings.visible_statuses.length <= 1 && visible)

  let counts = $state<Record<string, number>>({})
  let colorValue = $state('')
  let migrateTo = $state<TaskStatusId>('pending')
  let hidePrompt = $state(false)
  let migrating = $state(false)
  let colorBusy = $state(false)

  const ticketCount = $derived(counts[statusId] ?? 0)

  const migrateTargets = $derived(
    STATUS_OPTIONS.filter(
      (s) => s.id !== statusId && settings.visible_statuses.includes(s.id)
    )
  )

  onMount(() => {
    void refreshCounts()
  })

  $effect(() => {
    // Re-read color when lane or active theme changes.
    void statusId
    const theme = activeTheme
    const key = STATUS_COLOR_TOKEN[statusId]
    if (!key) {
      colorValue = ''
      return
    }
    colorValue =
      theme?.tokens?.[key] ?? SLATE_FACTORY_TOKENS[key] ?? '#000000'
  })

  $effect(() => {
    // Keep migrate target valid when visibility list changes.
    const targets = migrateTargets
    if (targets.length === 0) return
    if (!targets.some((t) => t.id === migrateTo)) {
      migrateTo = targets[0].id
    }
  })

  async function refreshCounts() {
    try {
      counts = await api.countByStatus()
    } catch (err) {
      onError(errMsg(err))
    }
  }

  function onVisibleChange(on: boolean) {
    if (on) {
      settings.visible_statuses = toggleStatusInOrder(settings.visible_statuses, statusId, true)
      hidePrompt = false
      return
    }
    if (onlyVisible) return
    if (ticketCount > 0) {
      hidePrompt = true
      return
    }
    applyHide(false)
  }

  function applyHide(migrate: boolean) {
    if (migrate && migrateTargets.length > 0) {
      void doMigrateThenHide(migrateTo)
      return
    }
    settings.visible_statuses = toggleStatusInOrder(settings.visible_statuses, statusId, false)
    // Drop hidden lane from tray/panel attention lists.
    settings.notifications.tray_label_statuses = settings.notifications.tray_label_statuses.filter(
      (s) => s !== statusId
    )
    settings.notifications.tray_menu_statuses = settings.notifications.tray_menu_statuses.filter(
      (s) => s !== statusId
    )
    settings.notifications.panel_attn_statuses = settings.notifications.panel_attn_statuses.filter(
      (s) => s !== statusId
    )
    hidePrompt = false
  }

  async function doMigrateThenHide(to: TaskStatusId) {
    migrating = true
    try {
      await api.migrateStatus(statusId as Status, to as Status)
      await refreshCounts()
      applyHide(false)
    } catch (err) {
      onError(errMsg(err))
    } finally {
      migrating = false
    }
  }

  async function migrateOnly() {
    if (migrateTargets.length === 0) return
    migrating = true
    try {
      await api.migrateStatus(statusId as Status, migrateTo as Status)
      await refreshCounts()
    } catch (err) {
      onError(errMsg(err))
    } finally {
      migrating = false
    }
  }

  let colorTimer: ReturnType<typeof setTimeout> | undefined

  function setColor(value: string) {
    colorValue = value
    clearTimeout(colorTimer)
    colorTimer = setTimeout(() => void saveColor(value), 350)
  }

  async function saveColor(value: string) {
    if (!activeTheme || !tokenKey) return
    colorBusy = true
    try {
      const tokens = { ...activeTheme.tokens, [tokenKey]: value }
      const saved = await api.updateTheme(activeTheme.id, activeTheme.name, tokens)
      onThemeSaved(saved)
      if (saved.active) applyThemeTokens(saved.tokens)
    } catch (err) {
      onError(errMsg(err))
    } finally {
      colorBusy = false
    }
  }
</script>

<section>
  <h3 class="mb-1 text-sm font-semibold text-ink">{label}</h3>
  <p class="mb-4 text-xs text-ink-3">
    Lane visibility and color for the active theme
    {#if activeTheme}
      (<span class="text-ink-2">{activeTheme.name}</span>).
    {:else}
      (no active theme).
    {/if}
  </p>

  <div class="flex flex-col gap-gap-lg">
    <label class="flex cursor-pointer items-start gap-gap-md {onlyVisible ? 'opacity-60' : ''}">
      <input
        type="checkbox"
        checked={visible}
        disabled={onlyVisible && visible}
        onchange={(e) => onVisibleChange(e.currentTarget.checked)}
        class="mt-0.5 h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
      />
      <span class="flex flex-col gap-0.5">
        <span class="text-sm text-ink-2">Visible on board</span>
        <span class="text-xs italic text-ink-3/75">
          {#if onlyVisible}
            At least one lane must stay visible.
          {:else}
            When hidden, this lane is omitted from the board and status picker.
          {/if}
        </span>
      </span>
    </label>

    {#if hidePrompt}
      <div class="rounded-control border border-line-soft bg-field/60 p-3">
        <p class="text-sm text-ink">
          {ticketCount} ticket{ticketCount === 1 ? '' : 's'} currently in {label}. Leave them in this
          hidden lane, or migrate them first.
        </p>
        {#if migrateTargets.length > 0}
          <label class="mt-3 block">
            <span class="micro mb-1.5">Migrate to</span>
            <select
              bind:value={migrateTo}
              class="w-full rounded-control border border-line-soft bg-field px-3 py-2 text-sm text-ink focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
            >
              {#each migrateTargets as t (t.id)}
                <option value={t.id}>{t.label}</option>
              {/each}
            </select>
          </label>
        {/if}
        <div class="mt-3 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            disabled={migrating}
            onclick={() => {
              hidePrompt = false
            }}
            class="rounded-control px-3 py-1.5 text-sm text-ink-2 hover:bg-white/5 hover:text-ink disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="button"
            disabled={migrating}
            onclick={() => applyHide(false)}
            class="rounded-control border border-line-soft px-3 py-1.5 text-sm text-ink-2 hover:bg-white/5 hover:text-ink disabled:opacity-50"
          >
            Leave tickets
          </button>
          {#if migrateTargets.length > 0}
            <button
              type="button"
              disabled={migrating}
              onclick={() => applyHide(true)}
              class="rounded-control bg-accent px-3 py-1.5 text-sm font-medium text-accent-ink hover:bg-accent-hi disabled:opacity-50"
            >
              {migrating ? 'Migrating…' : 'Migrate & hide'}
            </button>
          {/if}
        </div>
      </div>
    {/if}

    <div>
      <span class="micro mb-1.5">Lane color</span>
      <div class="flex flex-wrap items-center gap-3">
        <input
          type="color"
          value={colorInputValue(colorValue)}
          disabled={!activeTheme || colorBusy}
          oninput={(e) => setColor((e.currentTarget as HTMLInputElement).value)}
          class="h-9 w-12 cursor-pointer rounded-control border border-line-soft bg-field p-0.5 disabled:opacity-50"
          title="{label} color"
        />
        <input
          type="text"
          value={colorValue}
          disabled={!activeTheme || colorBusy}
          oninput={(e) => setColor((e.currentTarget as HTMLInputElement).value)}
          class="min-w-0 flex-1 rounded-control border border-line-soft bg-field px-3 py-1.5 font-mono text-xs text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25 disabled:opacity-50"
          placeholder="#rrggbb"
        />
      </div>
      <p class="mt-1 text-xs italic text-ink-3/75">
        Stored on the active theme. Switch themes to use a different palette.
      </p>
    </div>

    <div class="border-t border-line-soft pt-4">
      <p class="text-sm text-ink-2">
        Tickets in this lane:
        <span class="font-mono text-ink">{ticketCount}</span>
      </p>
      {#if !visible && ticketCount > 0 && migrateTargets.length > 0}
        <div class="mt-3 flex flex-wrap items-end gap-2">
          <label class="block min-w-[10rem] flex-1">
            <span class="micro mb-1.5">Migrate remaining to</span>
            <select
              bind:value={migrateTo}
              class="w-full rounded-control border border-line-soft bg-field px-3 py-2 text-sm text-ink focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
            >
              {#each migrateTargets as t (t.id)}
                <option value={t.id}>{t.label}</option>
              {/each}
            </select>
          </label>
          <button
            type="button"
            disabled={migrating}
            onclick={() => void migrateOnly()}
            class="rounded-control border border-line-soft px-3 py-2 text-sm text-ink-2 hover:bg-white/5 hover:text-ink disabled:opacity-50"
          >
            {migrating ? 'Migrating…' : 'Migrate'}
          </button>
        </div>
      {/if}
    </div>
  </div>
</section>
