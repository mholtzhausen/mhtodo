<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg } from '../lib/api'
  import {
    defaultSettings,
    STATUS_OPTIONS,
    toggleStatusInOrder,
    type GUISettings,
    type IntegrationConfig,
    type TaskStatusId
  } from '../lib/settings'
  import { emptyValues, type TaskTemplate } from '../lib/templates'
  import { SLATE_FACTORY_TOKENS, type Theme } from '../lib/themes'
  import ClearableField from './ClearableField.svelte'
  import SettingsStatuses from './SettingsStatuses.svelte'
  import SettingsTemplates from './SettingsTemplates.svelte'
  import SettingsThemes from './SettingsThemes.svelte'
  import { scheduleFocusFirstField } from '../lib/focusFirstField'

  let {
    open,
    onClose,
    onSaved,
    onError
  }: {
    open: boolean
    onClose: () => void
    onSaved?: (s: GUISettings) => void
    onError?: (msg: string) => void
  } = $props()

  let settings = $state<GUISettings>(defaultSettings())
  let loading = $state(false)
  let ready = $state(false)
  let persisting = $state(false)
  let lastSaved = $state('')

  let zedFound = $state(false)

  type SettingsPage =
    | 'general'
    | 'notifications'
    | 'integrations'
    | 'statuses'
    | 'templates'
    | 'themes'
  type NotificationsSub = 'tray' | 'panel'
  let activePage = $state<SettingsPage>('general')
  let notificationsSub = $state<NotificationsSub>('tray')
  let statusesSub = $state<TaskStatusId>('icebox')

  const pages: { id: SettingsPage; label: string }[] = [
    { id: 'general', label: 'General' },
    { id: 'notifications', label: 'Notifications' },
    { id: 'integrations', label: 'Integrations' },
    { id: 'statuses', label: 'Statuses / Lanes' },
    { id: 'templates', label: 'Task Templates' },
    { id: 'themes', label: 'Themes' }
  ]
  const pagesWithSubnav = new Set<SettingsPage>([
    'notifications',
    'statuses',
    'templates',
    'themes'
  ])

  const activeThemeForStatuses = $derived(themes.find((t) => t.active) ?? null)

  // --- task templates ---
  // Templates persist through their own bound methods, not the whole-settings
  // save path, so they carry a separate save-status signal that is merged into
  // the shared header indicator below.
  let templates = $state<TaskTemplate[]>([])
  let activeTemplateId = $state('')
  let templateStatus = $state<'idle' | 'saving' | 'dirty'>('idle')
  let creatingTemplate = $state(false)
  let templateEditor = $state<{ flush: () => Promise<void> } | null>(null)

  const activeTemplate = $derived(templates.find((t) => t.id === activeTemplateId) ?? null)

  // --- themes ---
  let themes = $state<Theme[]>([])
  let activeThemeId = $state('')
  let themeStatus = $state<'idle' | 'saving' | 'dirty'>('idle')
  let creatingTheme = $state(false)
  let themeEditor = $state<{ flush: () => Promise<void> } | null>(null)

  const activeTheme = $derived(themes.find((t) => t.id === activeThemeId) ?? null)

  let persistTimer: ReturnType<typeof setTimeout> | undefined
  /** Content pane — focus first field here once settings finish loading. */
  let bodyEl = $state<HTMLElement | null>(null)
  let focusedOpen = false

  const dirty = $derived(
    (ready && !loading && snapshot(settings) !== lastSaved) ||
      templateStatus === 'dirty' ||
      themeStatus === 'dirty'
  )
  const saveStatus = $derived(
    persisting || templateStatus === 'saving' || themeStatus === 'saving'
      ? 'Saving…'
      : dirty
        ? 'Unsaved'
        : ready && !loading
          ? 'Saved'
          : ''
  )

  async function loadTemplates(selectId?: string) {
    try {
      templates = await api.listTemplates()
      if (selectId) activeTemplateId = selectId
      else if (!templates.some((t) => t.id === activeTemplateId)) {
        activeTemplateId = templates[0]?.id ?? ''
      }
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  async function createTemplate() {
    if (creatingTemplate) return
    creatingTemplate = true
    try {
      // Names are unique, so a fresh one is suffixed until it does not collide.
      const base = 'New template'
      let name = base
      let n = 2
      while (templates.some((t) => t.name.toLowerCase() === name.toLowerCase())) {
        name = `${base} ${n++}`
      }
      const created = await api.createTemplate(name, emptyValues())
      templates = [...templates, created].sort((a, b) =>
        a.name.localeCompare(b.name, undefined, { sensitivity: 'base' })
      )
      activeTemplateId = created.id
      activePage = 'templates'
    } catch (err) {
      onError?.(errMsg(err))
    } finally {
      creatingTemplate = false
    }
  }

  function onTemplateSaved(saved: TaskTemplate) {
    templates = templates
      .map((t) => (t.id === saved.id ? saved : t))
      .sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }))
  }

  function onTemplateDeleted(id: string) {
    templates = templates.filter((t) => t.id !== id)
    templateStatus = 'idle'
    if (activeTemplateId === id) activeTemplateId = templates[0]?.id ?? ''
  }

  async function loadThemes(selectId?: string) {
    try {
      themes = await api.listThemes()
      if (selectId) activeThemeId = selectId
      else if (!themes.some((t) => t.id === activeThemeId)) {
        const active = themes.find((t) => t.active)
        activeThemeId = active?.id ?? themes[0]?.id ?? ''
      }
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  async function createTheme() {
    if (creatingTheme) return
    creatingTheme = true
    try {
      const base = 'New theme'
      let name = base
      let n = 2
      while (themes.some((t) => t.name.toLowerCase() === name.toLowerCase())) {
        name = `${base} ${n++}`
      }
      const created = await api.createTheme(name, { ...SLATE_FACTORY_TOKENS })
      themes = [...themes, created].sort((a, b) =>
        a.name.localeCompare(b.name, undefined, { sensitivity: 'base' })
      )
      activeThemeId = created.id
      activePage = 'themes'
    } catch (err) {
      onError?.(errMsg(err))
    } finally {
      creatingTheme = false
    }
  }

  function onThemeSaved(saved: Theme) {
    themes = themes
      .map((t) =>
        t.id === saved.id ? { ...saved, active: saved.active || t.active } : t
      )
      .sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }))
  }

  function onThemeDeleted(id: string) {
    themes = themes.filter((t) => t.id !== id)
    themeStatus = 'idle'
    if (activeThemeId === id) activeThemeId = themes[0]?.id ?? ''
  }

  function onThemeDuplicated(created: Theme) {
    themes = [...themes, created].sort((a, b) =>
      a.name.localeCompare(b.name, undefined, { sensitivity: 'base' })
    )
    activeThemeId = created.id
    activePage = 'themes'
  }

  function onThemeActivated(activated: Theme) {
    themes = themes.map((t) => ({
      ...t,
      active: t.id === activated.id,
      ...(t.id === activated.id ? activated : {})
    }))
  }

  function snapshot(s: GUISettings) {
    return JSON.stringify(s)
  }

  async function persist(force = false) {
    if (persisting) return
    if (!force && (loading || !ready)) return
    const snap = snapshot(settings)
    if (snap === lastSaved) return
    persisting = true
    try {
      await api.setSettings(settings)
      lastSaved = snap
      onSaved?.(settings)
    } catch (err) {
      onError?.(errMsg(err))
    } finally {
      persisting = false
    }
  }

  function schedulePersist() {
    clearTimeout(persistTimer)
    persistTimer = setTimeout(() => {
      void persist(false)
    }, 350)
  }

  async function flushPersist() {
    clearTimeout(persistTimer)
    await persist(true)
  }

  async function handleClose() {
    // Templates/themes save through their own path, so pending writes must land
    // before the dialog goes away.
    await Promise.all([
      flushPersist(),
      templateEditor?.flush() ?? Promise.resolve(),
      themeEditor?.flush() ?? Promise.resolve()
    ])
    onClose()
  }

  async function refreshBinaryStatus() {
    if (!open) return
    try {
      zedFound = await api.checkBinary(settings.zed.binary)
    } catch {
      zedFound = false
    }
  }

  $effect(() => {
    if (!open) {
      ready = false
      activePage = 'general'
      notificationsSub = 'tray'
      templateStatus = 'idle'
      themeStatus = 'idle'
      focusedOpen = false
      return
    }
    loading = true
    ready = false
    void loadTemplates()
    void loadThemes()
    api
      .getSettings()
      .then((s) => {
        settings = s
        lastSaved = snapshot(s)
        return refreshBinaryStatus()
      })
      .catch((e) => onError?.(errMsg(e)))
      .finally(() => {
        loading = false
        ready = true
      })

    return () => {
      clearTimeout(persistTimer)
      void persist(true)
    }
  })

  // After settings load into the General pane, land focus on the first text field
  // (default cwd) — not the nav or close chrome.
  $effect(() => {
    if (!open || !ready || loading || focusedOpen) return
    focusedOpen = true
    scheduleFocusFirstField(bodyEl)
  })

  $effect(() => {
    if (!open || !ready || loading) return
    const {
      default_cwd,
      default_human_only,
      default_include_in_report,
      archive_done_subtasks,
      start_hidden,
      notifications,
      zed
    } = settings
    void default_cwd
    void default_human_only
    void default_include_in_report
    void archive_done_subtasks
    void start_hidden
    void notifications.tray_label_statuses
    void notifications.tray_menu_statuses
    void notifications.max_items_per_status
    void notifications.notify_send_wip
    void notifications.notify_send_waiting
    void notifications.notify_send_review
    void notifications.notify_send_done
    void notifications.panel_attn_statuses
    void notifications.panel_attn_interval_sec
    void notifications.panel_attn_intensity
    void zed.enabled
    void zed.binary
    void zed.env_start
    schedulePersist()
  })

  $effect(() => {
    if (!open) return
    settings.zed.binary
    const t = setTimeout(() => {
      void refreshBinaryStatus()
    }, 200)
    return () => clearTimeout(t)
  })

  async function pickDefaultCwd() {
    try {
      const path = await api.pickDirectory(settings.default_cwd.trim())
      if (path) settings.default_cwd = path
    } catch (err) {
      onError?.(errMsg(err))
    }
  }

  function patchIntegration(key: 'zed', patch: Partial<IntegrationConfig>): void {
    settings = {
      ...settings,
      [key]: { ...settings[key], ...patch }
    }
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-4"
    onclick={handleClose}
  >
    <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
    <div
      in:fly={{ y: 8, duration: 80 }}
      onclick={(e) => e.stopPropagation()}
      class="flex h-[min(80vh,800px)] w-full max-w-3xl flex-col rounded-panel border border-line bg-col shadow-md"
    >
      <div class="flex flex-none items-center gap-gap-md border-b border-line-soft px-5 py-3.5">
        <h2 class="flex-1 text-base font-semibold text-ink">Settings</h2>
        {#if saveStatus}
          <span
            class="text-[11px] {persisting || dirty ? 'text-ink-3' : 'text-ink-3/70'}"
            aria-live="polite"
          >{saveStatus}</span>
        {/if}
        <button
          type="button"
          onclick={handleClose}
          title="Close (esc)"
          class="rounded-control p-1.5 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
        >
          ✕
        </button>
      </div>

      <div class="@container flex min-h-0 flex-1 overflow-hidden">
        <nav
          class="flex max-h-14 w-full flex-none flex-row gap-0.5 overflow-x-auto border-b border-line-soft p-2
            @[560px]:max-h-none @[560px]:w-44 @[560px]:flex-col @[560px]:overflow-y-auto @[560px]:border-b-0 @[560px]:border-r @[560px]:p-3"
          aria-label="Settings sections"
        >
          {#each pages as page (page.id)}
            {@const hasSubnav = pagesWithSubnav.has(page.id)}
            {@const subnavOpen = hasSubnav && activePage === page.id}
            <button
              type="button"
              onclick={() => {
                activePage = page.id
                if (page.id === 'notifications') notificationsSub = 'tray'
              }}
              class="inline-flex items-center gap-1.5 whitespace-nowrap rounded-control px-3 py-2 text-left text-[13px] font-medium transition-colors
                {activePage === page.id
                ? 'bg-accent/15 text-ink'
                : 'text-ink-3 hover:bg-white/5 hover:text-ink-2'}"
            >
              {#if hasSubnav}
                <svg
                  class="h-3.5 w-3.5 flex-none opacity-80"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  {#if subnavOpen}
                    <path
                      d="M6 14l1.45-2.9A2 2 0 0 1 9.24 10H20a2 2 0 0 1 1.94 2.5l-1.55 6a2 2 0 0 1-1.94 1.5H4a2 2 0 0 1-2-2V5c0-1.1.9-2 2-2h3.93a2 2 0 0 1 1.66.9l.82 1.2a2 2 0 0 0 1.66.9H18a2 2 0 0 1 2 2v2"
                    />
                  {:else}
                    <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z" />
                  {/if}
                </svg>
              {:else}
                <span class="h-3.5 w-3.5 flex-none" aria-hidden="true"></span>
              {/if}
              {page.label}
            </button>

            {#if page.id === 'notifications' && activePage === 'notifications'}
              <div class="mb-1 hidden flex-col gap-0.5 pl-3 @[560px]:flex">
                <button
                  type="button"
                  onclick={() => {
                    activePage = 'notifications'
                    notificationsSub = 'panel'
                  }}
                  class="truncate rounded-control px-3 py-1.5 text-left text-[12px] transition-colors
                    {activePage === 'notifications' && notificationsSub === 'panel'
                    ? 'bg-accent/10 text-ink'
                    : 'text-ink-3 hover:bg-white/5 hover:text-ink-2'}"
                >
                  Panel Notifications
                </button>
              </div>
            {/if}

            {#if page.id === 'statuses' && activePage === 'statuses'}
              <div class="mb-1 hidden flex-col gap-0.5 pl-3 @[560px]:flex">
                {#each STATUS_OPTIONS as opt (opt.id)}
                  <button
                    type="button"
                    onclick={() => {
                      activePage = 'statuses'
                      statusesSub = opt.id
                    }}
                    class="truncate rounded-control px-3 py-1.5 text-left text-[12px] transition-colors
                      {statusesSub === opt.id
                      ? 'bg-accent/10 text-ink'
                      : 'text-ink-3 hover:bg-white/5 hover:text-ink-2'}"
                  >
                    {opt.label}
                    {#if !settings.visible_statuses.includes(opt.id)}
                      <span class="text-ink-3"> (hidden)</span>
                    {/if}
                  </button>
                {/each}
              </div>
            {/if}

            <!-- Task Templates second nav tier: shown only while this section is active. -->
            {#if page.id === 'templates' && activePage === 'templates'}
              <div class="mb-1 hidden flex-col gap-0.5 pl-3 @[560px]:flex">
                {#each templates as tpl (tpl.id)}
                  <button
                    type="button"
                    onclick={() => {
                      activePage = 'templates'
                      activeTemplateId = tpl.id
                    }}
                    title={tpl.name}
                    class="truncate rounded-control px-3 py-1.5 text-left text-[12px] transition-colors
                      {activePage === 'templates' && activeTemplateId === tpl.id
                      ? 'bg-accent/10 text-ink'
                      : 'text-ink-3 hover:bg-white/5 hover:text-ink-2'}"
                  >
                    {tpl.name}
                  </button>
                {/each}
                <button
                  type="button"
                  onclick={createTemplate}
                  disabled={creatingTemplate}
                  class="rounded-control px-3 py-1.5 text-left text-[12px] text-ink-3 transition-colors hover:bg-white/5 hover:text-accent disabled:opacity-50"
                >
                  + New template
                </button>
              </div>
            {/if}

            {#if page.id === 'themes' && activePage === 'themes'}
              <div class="mb-1 hidden flex-col gap-0.5 pl-3 @[560px]:flex">
                {#each themes as th (th.id)}
                  <button
                    type="button"
                    onclick={() => {
                      activePage = 'themes'
                      activeThemeId = th.id
                    }}
                    title={th.name}
                    class="inline-flex w-full items-center gap-1 truncate rounded-control px-3 py-1.5 text-left text-[12px] transition-colors
                      {activePage === 'themes' && activeThemeId === th.id
                      ? 'bg-accent/10 text-ink'
                      : 'text-ink-3 hover:bg-white/5 hover:text-ink-2'}"
                  >
                    <span
                      class="w-2.5 flex-none text-center text-[10px] leading-none"
                      aria-hidden="true"
                    >{th.active ? '●' : ''}</span>
                    <span class="truncate">{th.name}</span>
                  </button>
                {/each}
                <button
                  type="button"
                  onclick={createTheme}
                  disabled={creatingTheme}
                  class="rounded-control px-3 py-1.5 text-left text-[12px] text-ink-3 transition-colors hover:bg-white/5 hover:text-accent disabled:opacity-50"
                >
                  + New theme
                </button>
              </div>
            {/if}
          {/each}
        </nav>

        <div class="min-h-0 flex-1 overflow-y-auto p-5" bind:this={bodyEl}>
          {#if loading}
            <p class="text-sm text-ink-3">Loading…</p>
          {:else if activePage === 'general'}
            <section>
              <h3 class="mb-4 text-sm font-semibold text-ink">General</h3>
              <div class="flex flex-col gap-gap-lg">
              <div class="block">
                <span class="micro mb-1.5">Default working directory for new tasks</span>
                <div class="flex gap-2">
                  <input
                    data-focus-primary
                    bind:value={settings.default_cwd}
                    placeholder="Optional project path…"
                    class="min-w-0 flex-1 rounded-control border border-line-soft bg-field px-3 py-2 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
                  />
                  <button
                    type="button"
                    onclick={pickDefaultCwd}
                    title="Pick folder"
                    class="flex-none rounded-control border border-line-soft bg-field px-2.5 py-2 text-ink-2 transition-colors hover:bg-card-hi hover:text-ink"
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
                      <path
                        d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"
                      />
                    </svg>
                  </button>
                </div>
              </div>

              <label class="flex cursor-pointer items-center gap-gap-md">
                <input
                  type="checkbox"
                  bind:checked={settings.default_human_only}
                  class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                />
                <span class="text-sm text-ink-2"
                  >Human only by default <span class="text-ink-3">(agents skip new tasks)</span></span
                >
              </label>

              <label class="flex cursor-pointer items-center gap-gap-md">
                <input
                  type="checkbox"
                  bind:checked={settings.default_include_in_report}
                  class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                />
                <span class="text-sm text-ink-2"
                  >Include in Slack report by default
                  <span class="text-ink-3">(board summary copy)</span></span
                >
              </label>

              <label class="flex cursor-pointer items-start gap-gap-md">
                <input
                  type="checkbox"
                  bind:checked={settings.archive_done_subtasks}
                  class="mt-0.5 h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                />
                <span class="flex flex-col gap-0.5">
                  <span class="text-sm text-ink-2">Archive done subtasks</span>
                  <span class="text-xs italic text-ink-3/75"
                    >When archiving from the Done column, include subtasks (default: root tasks only)</span
                  >
                </span>
              </label>

              <label class="flex cursor-pointer items-start gap-gap-md">
                <input
                  type="checkbox"
                  bind:checked={settings.start_hidden}
                  class="mt-0.5 h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                />
                <span class="flex flex-col gap-0.5">
                  <span class="text-sm text-ink-2">Start hidden in system tray</span>
                  <span class="text-xs italic text-ink-3/75"
                    >When off, the window is shown on launch; when on, only the tray icon appears until you open it. Takes effect on next launch.</span
                  >
                </span>
              </label>
              </div>
            </section>
          {:else if activePage === 'notifications' && notificationsSub === 'panel'}
            <section>
              <h3 class="mb-2 text-sm font-semibold text-ink">Panel Notifications</h3>
              <p class="mb-4 text-xs text-ink-3">
                Choose which board columns highlight occupancy while collapsed. Empty lanes never
                pulse. Frequency and intensity control how often and how strongly the background wash
                appears.
              </p>
              <div class="flex flex-col gap-gap-lg">
                <div>
                  <p class="mb-1 text-sm text-ink-2">Highlight while collapsed</p>
                  <p class="mb-2 text-xs italic text-ink-3/75">
                    Checked statuses pulse when the column is collapsed and has root tasks.
                  </p>
                  <div class="flex flex-col gap-2">
                    {#each STATUS_OPTIONS as opt}
                      <label class="flex cursor-pointer items-center gap-gap-md">
                        <input
                          type="checkbox"
                          checked={settings.notifications.panel_attn_statuses.includes(opt.id)}
                          onchange={(e) => {
                            settings.notifications.panel_attn_statuses = toggleStatusInOrder(
                              settings.notifications.panel_attn_statuses,
                              opt.id,
                              e.currentTarget.checked
                            )
                          }}
                          class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                        />
                        <span class="text-sm text-ink-2">{opt.label}</span>
                      </label>
                    {/each}
                  </div>
                </div>

                <label class="block">
                  <span class="micro mb-1.5">Pulse every (seconds)</span>
                  <input
                    type="number"
                    min="1"
                    max="30"
                    bind:value={settings.notifications.panel_attn_interval_sec}
                    class="w-24 rounded-control border border-line-soft bg-field px-3 py-2 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
                  />
                  <p class="mt-1 text-xs italic text-ink-3/75">Default 5. Range 1–30.</p>
                </label>

                <div>
                  <div class="mb-1.5 flex items-baseline justify-between gap-2">
                    <span class="micro">Intensity</span>
                    <span class="font-mono text-[11px] text-ink-3"
                      >{settings.notifications.panel_attn_intensity}</span
                    >
                  </div>
                  <input
                    type="range"
                    min="1"
                    max="100"
                    bind:value={settings.notifications.panel_attn_intensity}
                    class="w-full accent-[var(--color-accent)]"
                  />
                  <p class="mt-1 text-xs italic text-ink-3/75">
                    How aggressive the background pulse color is (gentle → strong). Default 40.
                  </p>
                </div>
              </div>
            </section>
          {:else if activePage === 'notifications'}
            <section>
              <h3 class="mb-4 text-sm font-semibold text-ink">Notifications</h3>
              <div class="flex flex-col gap-gap-lg">
                <div>
                  <p class="mb-1 text-sm text-ink-2">Tray label statuses</p>
                  <p class="mb-2 text-xs italic text-ink-3/75">
                    When any of these have root tasks, the tray title shows counts (e.g. “2 waiting, 1
                    review”). Otherwise it falls back to the open-task count.
                  </p>
                  <div class="flex flex-col gap-2">
                    {#each STATUS_OPTIONS as opt}
                      <label class="flex cursor-pointer items-center gap-gap-md">
                        <input
                          type="checkbox"
                          checked={settings.notifications.tray_label_statuses.includes(opt.id)}
                          onchange={(e) => {
                            settings.notifications.tray_label_statuses = toggleStatusInOrder(
                              settings.notifications.tray_label_statuses,
                              opt.id,
                              e.currentTarget.checked
                            )
                          }}
                          class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                        />
                        <span class="text-sm text-ink-2">{opt.label}</span>
                      </label>
                    {/each}
                  </div>
                </div>

                <div>
                  <p class="mb-1 text-sm text-ink-2">Tray menu statuses</p>
                  <p class="mb-2 text-xs italic text-ink-3/75">
                    Status submenus under the tray icon; click a task to open mhtodo and select it.
                  </p>
                  <div class="flex flex-col gap-2">
                    {#each STATUS_OPTIONS as opt}
                      <label class="flex cursor-pointer items-center gap-gap-md">
                        <input
                          type="checkbox"
                          checked={settings.notifications.tray_menu_statuses.includes(opt.id)}
                          onchange={(e) => {
                            settings.notifications.tray_menu_statuses = toggleStatusInOrder(
                              settings.notifications.tray_menu_statuses,
                              opt.id,
                              e.currentTarget.checked
                            )
                          }}
                          class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                        />
                        <span class="text-sm text-ink-2">{opt.label}</span>
                      </label>
                    {/each}
                  </div>
                </div>

                <label class="block">
                  <span class="micro mb-1.5">Max tasks per status submenu</span>
                  <input
                    type="number"
                    min="1"
                    max="20"
                    bind:value={settings.notifications.max_items_per_status}
                    class="w-24 rounded-control border border-line-soft bg-field px-3 py-2 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
                  />
                </label>

                <div class="border-t border-line-soft pt-4">
                  <p class="mb-2 text-sm font-medium text-ink-2">Desktop notifications (notify-send)</p>
                  <div class="flex flex-col gap-2">
                    <label class="flex cursor-pointer items-center gap-gap-md">
                      <input
                        type="checkbox"
                        bind:checked={settings.notifications.notify_send_wip}
                        class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                      />
                      <span class="text-sm text-ink-2"
                        >Notify on →in progress <span class="text-ink-3">(→wip)</span></span
                      >
                    </label>
                    <label class="flex cursor-pointer items-center gap-gap-md">
                      <input
                        type="checkbox"
                        bind:checked={settings.notifications.notify_send_waiting}
                        class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                      />
                      <span class="text-sm text-ink-2">Notify on →waiting</span>
                    </label>
                    <label class="flex cursor-pointer items-center gap-gap-md">
                      <input
                        type="checkbox"
                        bind:checked={settings.notifications.notify_send_review}
                        class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                      />
                      <span class="text-sm text-ink-2">Notify on →review</span>
                    </label>
                    <label class="flex cursor-pointer items-center gap-gap-md">
                      <input
                        type="checkbox"
                        bind:checked={settings.notifications.notify_send_done}
                        class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                      />
                      <span class="text-sm text-ink-2">Notify on →done</span>
                    </label>
                  </div>
                </div>
              </div>
            </section>
          {:else if activePage === 'integrations'}
            <section>
              <h3 class="mb-4 text-sm font-semibold text-ink">Integrations</h3>
              <div class="flex flex-col gap-4">
              <div class="rounded-control border border-line-soft bg-field/30 p-3.5">
                <label class="flex cursor-pointer items-center gap-gap-md">
                  <input
                    type="checkbox"
                    checked={settings.zed.enabled}
                    onchange={(e) =>
                      patchIntegration('zed', {
                        enabled: (e.currentTarget as HTMLInputElement).checked
                      })}
                    class="h-4 w-4 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
                  />
                  <span class="text-sm font-medium text-ink">Zed</span>
                </label>
                <div class="mt-3 flex flex-col gap-gap-md pl-6">
                  <label class="block">
                    <span class="micro mb-1">Binary</span>
                    <div class="flex items-center gap-2">
                      <input
                        value={settings.zed.binary}
                        oninput={(e) =>
                          patchIntegration('zed', {
                            binary: (e.currentTarget as HTMLInputElement).value
                          })}
                        placeholder="/usr/bin/zed"
                        class="min-w-0 flex-1 rounded-control border border-line-soft bg-field px-3 py-1.5 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
                      />
                      <span
                        title={zedFound ? 'Binary found' : 'Binary not found'}
                        class="h-2.5 w-2.5 shrink-0 rounded-full {zedFound
                          ? 'bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.55)]'
                          : 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.45)]'}"
                        aria-label={zedFound ? 'Binary found' : 'Binary not found'}
                      ></span>
                    </div>
                  </label>
                  <label class="block">
                    <span class="micro mb-1">Env start string</span>
                    <ClearableField
                      value={settings.zed.env_start}
                      placeholder="e.g. extra KEY=VAL tokens before path"
                      onChange={(env_start) => patchIntegration('zed', { env_start })}
                    />
                  </label>
                  <p class="text-xs italic text-ink-3/75">
                    Shown on cards when enabled, binary found, and the task has a working directory.
                    Opens Zed at the task working folder and copies the ticket reference.
                  </p>
                </div>
              </div>
              </div>
            </section>
          {:else if activePage === 'statuses'}
            {#key statusesSub}
              <SettingsStatuses
                statusId={statusesSub}
                bind:settings
                activeTheme={activeThemeForStatuses}
                onThemeSaved={onThemeSaved}
                onError={(m) => onError?.(m)}
              />
            {/key}
          {:else if activePage === 'templates'}
            {#if activeTemplate}
              {#key activeTemplate.id}
                <SettingsTemplates
                  bind:this={templateEditor}
                  template={activeTemplate}
                  defaultHumanOnly={settings.default_human_only}
                  defaultIncludeInReport={settings.default_include_in_report}
                  onSaved={onTemplateSaved}
                  onDeleted={onTemplateDeleted}
                  onStatus={(s) => (templateStatus = s)}
                  onError={(m) => onError?.(m)}
                />
              {/key}
            {:else}
              <section>
                <h3 class="mb-2 text-sm font-semibold text-ink">Task Templates</h3>
                <p class="mb-4 text-sm leading-relaxed text-ink-3">
                  A template pre-fills the new-task form. Only the fields you set are applied —
                  anything you leave out keeps its normal default.
                </p>
                <button
                  type="button"
                  onclick={createTemplate}
                  disabled={creatingTemplate}
                  class="btn-primary rounded-control bg-accent px-3 py-1.5 text-sm font-medium text-accent-ink shadow-sm transition-colors hover:bg-accent-hi disabled:opacity-50"
                >
                  + New template
                </button>
              </section>
            {/if}
          {:else if activePage === 'themes'}
            {#if activeTheme}
              {#key activeTheme.id}
                <SettingsThemes
                  bind:this={themeEditor}
                  theme={activeTheme}
                  onSaved={onThemeSaved}
                  onDeleted={onThemeDeleted}
                  onDuplicated={onThemeDuplicated}
                  onActivated={onThemeActivated}
                  onStatus={(s) => (themeStatus = s)}
                  onError={(m) => onError?.(m)}
                />
              {/key}
            {:else}
              <section>
                <h3 class="mb-2 text-sm font-semibold text-ink">Themes</h3>
                <p class="mb-4 text-sm leading-relaxed text-ink-3">
                  Customize colors, radii, and spacing. Lane status colors live under Statuses /
                  Lanes. Slate, Paper, and Ember ship built-in; Duplicate any theme to start a custom
                  one.
                </p>
                <button
                  type="button"
                  onclick={createTheme}
                  disabled={creatingTheme}
                  class="btn-primary rounded-control bg-accent px-3 py-1.5 text-sm font-medium text-accent-ink shadow-sm transition-colors hover:bg-accent-hi disabled:opacity-50"
                >
                  + New theme
                </button>
              </section>
            {/if}
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}
