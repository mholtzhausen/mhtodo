<script lang="ts">
  import { fly } from 'svelte/transition'
  import { api, errMsg } from '../lib/api'
  import {
    defaultSettings,
    DEFAULT_CLAUDE_TICKET_PROMPT,
    DEFAULT_HERDR_SPACE_NAME,
    type ClaudeConfig,
    type GUISettings,
    type HerdrConfig
  } from '../lib/settings'
  import ClearableField from './ClearableField.svelte'

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

  let claudeFound = $state(false)
  let herdrFound = $state(false)

  type SettingsPage = 'general' | 'integrations'
  let activePage = $state<SettingsPage>('general')

  const pages: { id: SettingsPage; label: string }[] = [
    { id: 'general', label: 'General' },
    { id: 'integrations', label: 'Integrations' }
  ]

  let persistTimer: ReturnType<typeof setTimeout> | undefined

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
    await flushPersist()
    onClose()
  }

  async function refreshBinaryStatus() {
    if (!open) return
    try {
      const [c, h] = await Promise.all([
        api.checkBinary(settings.claude.binary),
        api.checkBinary(settings.herdr.binary)
      ])
      claudeFound = c
      herdrFound = h
    } catch {
      claudeFound = false
      herdrFound = false
    }
  }

  $effect(() => {
    if (!open) {
      ready = false
      activePage = 'general'
      return
    }
    loading = true
    ready = false
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

  $effect(() => {
    if (!open || !ready || loading) return
    const { default_cwd, default_human_only, archive_done_subtasks, claude, herdr } = settings
    void default_cwd
    void default_human_only
    void archive_done_subtasks
    void claude.enabled
    void claude.binary
    void claude.env_start
    void claude.ticket_prompt
    void herdr.enabled
    void herdr.binary
    void herdr.env_start
    void herdr.space_name
    schedulePersist()
  })

  $effect(() => {
    if (!open) return
    settings.claude.binary
    settings.herdr.binary
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

  function patchIntegration(key: 'claude', patch: Partial<ClaudeConfig>): void
  function patchIntegration(key: 'herdr', patch: Partial<HerdrConfig>): void
  function patchIntegration(
    key: 'claude' | 'herdr',
    patch: Partial<ClaudeConfig> | Partial<HerdrConfig>
  ) {
    settings = {
      ...settings,
      [key]: { ...settings[key], ...patch }
    }
  }
</script>

{#if open}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-4 backdrop-blur-[2px]"
    onclick={handleClose}
  >
    <div
      in:fly={{ y: 12, duration: 150 }}
      onclick={(e) => e.stopPropagation()}
      class="flex h-[min(80vh,800px)] w-full max-w-3xl flex-col rounded-lg border border-line bg-col shadow-2xl"
    >
      <div class="flex flex-none items-center gap-2.5 border-b border-line-soft px-5 py-3.5">
        <h2 class="flex-1 text-base font-semibold text-ink">Settings</h2>
        <button
          type="button"
          onclick={handleClose}
          title="Close (esc)"
          class="rounded p-1.5 leading-none text-ink-3 transition-colors hover:bg-white/5 hover:text-ink"
        >
          ✕
        </button>
      </div>

      <div class="flex min-h-0 flex-1 overflow-hidden">
        <nav
          class="flex w-44 flex-none flex-col gap-0.5 overflow-y-auto border-r border-line-soft p-3"
          aria-label="Settings sections"
        >
          {#each pages as page (page.id)}
            <button
              type="button"
              onclick={() => (activePage = page.id)}
              class="rounded px-3 py-2 text-left text-[13px] font-medium transition-colors
                {activePage === page.id
                ? 'bg-accent/15 text-ink'
                : 'text-ink-3 hover:bg-white/5 hover:text-ink-2'}"
            >
              {page.label}
            </button>
          {/each}
        </nav>

        <div class="min-h-0 flex-1 overflow-y-auto p-5">
          {#if loading}
            <p class="text-sm text-ink-3">Loading…</p>
          {:else if activePage === 'general'}
            <section>
              <h3 class="mb-4 text-sm font-semibold text-ink">General</h3>
              <div class="flex flex-col gap-3.5">
              <div class="block">
                <span class="micro mb-1.5">Default working directory for new tasks</span>
                <div class="flex gap-2">
                  <input
                    bind:value={settings.default_cwd}
                    placeholder="Optional project path…"
                    class="min-w-0 flex-1 rounded border border-line-soft bg-field px-3 py-2 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
                  />
                  <button
                    type="button"
                    onclick={pickDefaultCwd}
                    title="Pick folder"
                    class="flex-none rounded border border-line-soft bg-field px-2.5 py-2 text-ink-2 transition-colors hover:bg-card-hi hover:text-ink"
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

              <label class="flex cursor-pointer items-center gap-2.5">
                <input
                  type="checkbox"
                  bind:checked={settings.default_human_only}
                  class="h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
                />
                <span class="text-sm text-ink-2"
                  >Human only by default <span class="text-ink-3">(agents skip new tasks)</span></span
                >
              </label>

              <label class="flex cursor-pointer items-start gap-2.5">
                <input
                  type="checkbox"
                  bind:checked={settings.archive_done_subtasks}
                  class="mt-0.5 h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
                />
                <span class="flex flex-col gap-0.5">
                  <span class="text-sm text-ink-2">Archive done subtasks</span>
                  <span class="text-xs italic text-ink-3/75"
                    >When archiving from the Done column, include subtasks (default: root tasks only)</span
                  >
                </span>
              </label>
              </div>
            </section>
          {:else if activePage === 'integrations'}
            <section>
              <h3 class="mb-4 text-sm font-semibold text-ink">Integrations</h3>
              <div class="flex flex-col gap-4">
              <div class="rounded border border-line-soft bg-field/30 p-3.5">
                <label class="flex cursor-pointer items-center gap-2.5">
                  <input
                    type="checkbox"
                    checked={settings.claude.enabled}
                    onchange={(e) =>
                      patchIntegration('claude', {
                        enabled: (e.currentTarget as HTMLInputElement).checked
                      })}
                    class="h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
                  />
                  <span class="text-sm font-medium text-ink">Claude</span>
                </label>
                <div class="mt-3 flex flex-col gap-2.5 pl-6">
                  <label class="block">
                    <span class="micro mb-1">Binary</span>
                    <div class="flex items-center gap-2">
                      <input
                        value={settings.claude.binary}
                        oninput={(e) =>
                          patchIntegration('claude', {
                            binary: (e.currentTarget as HTMLInputElement).value
                          })}
                        placeholder="/usr/bin/claude"
                        class="min-w-0 flex-1 rounded border border-line-soft bg-field px-3 py-1.5 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
                      />
                      <span
                        title={claudeFound ? 'Binary found' : 'Binary not found'}
                        class="h-2.5 w-2.5 shrink-0 rounded-full {claudeFound
                          ? 'bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.55)]'
                          : 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.45)]'}"
                        aria-label={claudeFound ? 'Binary found' : 'Binary not found'}
                      ></span>
                    </div>
                  </label>
                  <label class="block">
                    <span class="micro mb-1">Env start string</span>
                    <ClearableField
                      value={settings.claude.env_start}
                      placeholder="e.g. ANTHROPIC_API_KEY=… command args"
                      onChange={(env_start) => patchIntegration('claude', { env_start })}
                    />
                  </label>
                  <label class="block">
                    <span class="mb-1.5 flex flex-col gap-0.5">
                      <span class="micro">Ticket prompt</span>
                      <span class="text-xs italic text-ink-3/75">
                        Sent to Claude in a new Herdr tab. Use <code
                          class="font-mono not-italic text-ink-3/90">{'{{todo-hash}}'}</code
                        > for the short task ID.
                      </span>
                    </span>
                    <ClearableField
                      multiline
                      rows={4}
                      mono
                      value={settings.claude.ticket_prompt}
                      placeholder={DEFAULT_CLAUDE_TICKET_PROMPT}
                      onChange={(ticket_prompt) => patchIntegration('claude', { ticket_prompt })}
                    />
                  </label>
                  <label class="flex cursor-pointer items-start gap-2.5">
                    <input
                      type="checkbox"
                      checked={settings.claude.require_cwd}
                      onchange={(e) =>
                        patchIntegration('claude', {
                          require_cwd: (e.currentTarget as HTMLInputElement).checked
                        })}
                      class="mt-0.5 h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
                    />
                    <span class="flex flex-col gap-0.5">
                      <span class="text-sm text-ink-2">Require working directory</span>
                      <span class="text-xs italic text-ink-3/75">Hide Claude button when task has no cwd</span>
                    </span>
                  </label>
                  <label class="flex cursor-pointer items-start gap-2.5">
                    <input
                      type="checkbox"
                      checked={settings.claude.close_tab_on_done}
                      onchange={(e) =>
                        patchIntegration('claude', {
                          close_tab_on_done: (e.currentTarget as HTMLInputElement).checked
                        })}
                      class="mt-0.5 h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
                    />
                    <span class="flex flex-col gap-0.5">
                      <span class="text-sm text-ink-2">Close Herdr tab when done</span>
                      <span class="text-xs italic text-ink-3/75">Remove Claude ticket tab when task moves to done</span>
                    </span>
                  </label>
                </div>
              </div>

              <div class="rounded border border-line-soft bg-field/30 p-3.5">
                <label class="flex cursor-pointer items-center gap-2.5">
                  <input
                    type="checkbox"
                    checked={settings.herdr.enabled}
                    onchange={(e) =>
                      patchIntegration('herdr', {
                        enabled: (e.currentTarget as HTMLInputElement).checked
                      })}
                    class="h-4 w-4 rounded border-line-soft bg-field text-accent focus:ring-accent/25"
                  />
                  <span class="text-sm font-medium text-ink">Herdr</span>
                </label>
                <div class="mt-3 flex flex-col gap-2.5 pl-6">
                  <label class="block">
                    <span class="micro mb-1">Binary</span>
                    <div class="flex items-center gap-2">
                      <input
                        value={settings.herdr.binary}
                        oninput={(e) =>
                          patchIntegration('herdr', {
                            binary: (e.currentTarget as HTMLInputElement).value
                          })}
                        placeholder="/usr/local/bin/herdr"
                        class="min-w-0 flex-1 rounded border border-line-soft bg-field px-3 py-1.5 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
                      />
                      <span
                        title={herdrFound ? 'Binary found' : 'Binary not found'}
                        class="h-2.5 w-2.5 shrink-0 rounded-full {herdrFound
                          ? 'bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.55)]'
                          : 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.45)]'}"
                        aria-label={herdrFound ? 'Binary found' : 'Binary not found'}
                      ></span>
                    </div>
                  </label>
                  <label class="block">
                    <span class="micro mb-1">Env start string</span>
                    <ClearableField
                      value={settings.herdr.env_start}
                      placeholder="e.g. HERDR_CONFIG=… command args"
                      onChange={(env_start) => patchIntegration('herdr', { env_start })}
                    />
                  </label>
                  <label class="block">
                    <span class="micro mb-1">Space name</span>
                    <ClearableField
                      value={settings.herdr.space_name}
                      placeholder={DEFAULT_HERDR_SPACE_NAME}
                      onChange={(space_name) => patchIntegration('herdr', { space_name })}
                    />
                  </label>
                </div>
              </div>
              </div>
            </section>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}
