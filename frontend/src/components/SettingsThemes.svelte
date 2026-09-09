<script lang="ts">
  import { onDestroy } from 'svelte'
  import { api, errMsg } from '../lib/api'
  import {
    THEME_CATEGORIES,
    THEME_FIELDS,
    applyThemeTokens,
    colorInputValue,
    formatPx,
    parsePx,
    type Theme
  } from '../lib/themes'

  let {
    theme,
    onSaved,
    onDeleted,
    onDuplicated,
    onActivated,
    onStatus,
    onError
  }: {
    theme: Theme
    onSaved: (t: Theme) => void
    onDeleted: (id: string) => void
    onDuplicated: (t: Theme) => void
    onActivated: (t: Theme) => void
    onStatus: (s: 'idle' | 'saving' | 'dirty') => void
    onError: (msg: string) => void
  } = $props()

  // svelte-ignore state_referenced_locally
  let name = $state(theme.name)
  // svelte-ignore state_referenced_locally
  let tokens = $state<Record<string, string>>({ ...theme.tokens })
  let confirmDelete = $state(false)
  let confirmReset = $state(false)
  let nameError = $state('')
  let deleted = false
  let busy = $state(false)

  let persistTimer: ReturnType<typeof setTimeout> | undefined
  let inFlight: Promise<void> = Promise.resolve()
  // svelte-ignore state_referenced_locally
  let lastSaved = $state(snapshot(theme.name, theme.tokens))

  function snapshot(n: string, t: Record<string, string>) {
    return JSON.stringify({ n, t })
  }

  $effect(() => {
    const snap = snapshot(name, tokens)
    if (snap === lastSaved) {
      onStatus('idle')
      return
    }
    onStatus('dirty')
    clearTimeout(persistTimer)
    persistTimer = setTimeout(() => void queueSave(), 400)
    return () => clearTimeout(persistTimer)
  })

  onDestroy(() => {
    clearTimeout(persistTimer)
    if (!deleted && snapshot(name, tokens) !== lastSaved) void queueSave()
  })

  function queueSave(): Promise<void> {
    const n = name
    const t = $state.snapshot(tokens) as Record<string, string>
    inFlight = inFlight.then(() => save(n, t))
    return inFlight
  }

  async function save(n: string, t: Record<string, string>) {
    const trimmed = n.trim()
    if (!trimmed) {
      nameError = 'Name is required'
      onStatus('dirty')
      return
    }
    nameError = ''
    onStatus('saving')
    try {
      const saved = await api.updateTheme(theme.id, trimmed, t)
      lastSaved = snapshot(saved.name, saved.tokens)
      name = saved.name
      tokens = { ...saved.tokens }
      onSaved(saved)
      if (saved.active) applyThemeTokens(saved.tokens)
      onStatus('idle')
    } catch (err) {
      const msg = errMsg(err)
      if (/already exists/i.test(msg)) nameError = msg
      onError(msg)
      onStatus('dirty')
    }
  }

  export async function flush() {
    clearTimeout(persistTimer)
    if (!deleted && snapshot(name, tokens) !== lastSaved) await queueSave()
    else await inFlight
  }

  function setToken(key: string, value: string) {
    tokens = { ...tokens, [key]: value }
  }

  async function activate() {
    if (busy || theme.active) return
    busy = true
    try {
      await flush()
      const t = await api.activateTheme(theme.id)
      applyThemeTokens(t.tokens)
      onActivated(t)
    } catch (err) {
      onError(errMsg(err))
    } finally {
      busy = false
    }
  }

  async function duplicate() {
    if (busy) return
    busy = true
    try {
      await flush()
      const t = await api.duplicateTheme(theme.id)
      onDuplicated(t)
    } catch (err) {
      onError(errMsg(err))
    } finally {
      busy = false
    }
  }

  async function reset() {
    if (busy || !theme.builtin_key) return
    busy = true
    try {
      const t = await api.resetTheme(theme.id)
      name = t.name
      tokens = { ...t.tokens }
      lastSaved = snapshot(t.name, t.tokens)
      confirmReset = false
      onSaved(t)
      if (t.active) applyThemeTokens(t.tokens)
      onStatus('idle')
    } catch (err) {
      onError(errMsg(err))
    } finally {
      busy = false
    }
  }

  async function remove() {
    if (busy || theme.builtin_key) return
    busy = true
    try {
      deleted = true
      await api.deleteTheme(theme.id)
      onDeleted(theme.id)
    } catch (err) {
      deleted = false
      onError(errMsg(err))
    } finally {
      busy = false
    }
  }
</script>

<section>
  <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
    <div class="min-w-0 flex-1">
      <h3 class="text-sm font-semibold text-ink">Theme</h3>
      <p class="mt-1 text-sm leading-relaxed text-ink-3">
        Colors, radii, and spacing applied across the GUI. Changes autosave.
        {#if theme.builtin_key}
          <span class="text-ink-2"> Built-in ({theme.builtin_key}) — editable; Reset restores factory values.</span>
        {/if}
      </p>
    </div>
    <div class="flex flex-wrap items-center gap-2">
      {#if theme.active}
        <span class="rounded-control bg-accent/15 px-2.5 py-1 text-[11px] font-medium text-accent-hi"
          >Active</span
        >
      {:else}
        <button
          type="button"
          onclick={activate}
          disabled={busy}
          class="btn-primary rounded-control bg-accent px-3 py-1.5 text-sm font-medium text-accent-ink shadow-sm transition-colors hover:bg-accent-hi disabled:opacity-50"
        >
          Use theme
        </button>
      {/if}
      <button
        type="button"
        onclick={duplicate}
        disabled={busy}
        class="rounded-control border border-line-soft px-3 py-1.5 text-sm text-ink-2 transition-colors hover:bg-white/5 hover:text-ink disabled:opacity-50"
      >
        Duplicate
      </button>
    </div>
  </div>

  <div class="flex flex-col gap-5">
    <label class="block">
      <span class="micro mb-1.5">Name</span>
      <input
        bind:value={name}
        maxlength={80}
        class="w-full rounded-control border border-line-soft bg-field px-3 py-2 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25
          {nameError ? 'border-danger' : ''}"
      />
      {#if nameError}
        <p class="mt-1 text-xs text-danger">{nameError}</p>
      {/if}
    </label>

    <div class="h-px bg-line-soft"></div>

    {#each THEME_CATEGORIES as category (category)}
      <div>
        <h4 class="mb-3 text-xs font-semibold uppercase tracking-wide text-ink-3">{category}</h4>
        <div class="flex flex-col gap-3">
          {#each THEME_FIELDS.filter((f) => f.category === category) as field (field.key)}
            <div class="flex flex-wrap items-center gap-3">
              <span class="w-36 flex-none text-sm text-ink-2">{field.label}</span>
              {#if field.kind === 'color'}
                {#if field.allowRgba}
                  <input
                    type="text"
                    value={tokens[field.key] ?? ''}
                    oninput={(e) => setToken(field.key, (e.currentTarget as HTMLInputElement).value)}
                    class="min-w-0 flex-1 rounded-control border border-line-soft bg-field px-3 py-1.5 font-mono text-xs text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
                    placeholder="rgba(…) or #hex"
                  />
                {:else}
                  <input
                    type="color"
                    value={colorInputValue(tokens[field.key] ?? '#000000')}
                    oninput={(e) => setToken(field.key, (e.currentTarget as HTMLInputElement).value)}
                    class="h-9 w-12 cursor-pointer rounded-control border border-line-soft bg-field p-0.5"
                    title={field.label}
                  />
                  <input
                    type="text"
                    value={tokens[field.key] ?? ''}
                    oninput={(e) => setToken(field.key, (e.currentTarget as HTMLInputElement).value)}
                    class="min-w-0 flex-1 rounded-control border border-line-soft bg-field px-3 py-1.5 font-mono text-xs text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
                    placeholder="#rrggbb"
                  />
                {/if}
              {:else}
                <div class="flex items-center gap-2">
                  <input
                    type="number"
                    min={field.minPx ?? 0}
                    max={field.maxPx ?? 96}
                    value={parsePx(tokens[field.key] ?? '0')}
                    oninput={(e) => {
                      const n = Number((e.currentTarget as HTMLInputElement).value)
                      const clamped = Math.max(
                        field.minPx ?? 0,
                        Math.min(field.maxPx ?? 96, Number.isFinite(n) ? n : 0)
                      )
                      setToken(field.key, formatPx(clamped))
                    }}
                    class="w-24 rounded-control border border-line-soft bg-field px-3 py-1.5 text-sm text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25"
                  />
                  <span class="text-xs text-ink-3">px</span>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {/each}

    <div class="h-px bg-line-soft"></div>

    <div class="flex flex-wrap items-center gap-2">
      {#if theme.builtin_key}
        {#if confirmReset}
          <span class="text-sm text-ink-2">Reset to factory values?</span>
          <button
            type="button"
            onclick={reset}
            disabled={busy}
            class="rounded-control border border-accent/40 bg-accent/10 px-3 py-1.5 text-sm font-medium text-accent-hi transition-colors hover:bg-accent/20 disabled:opacity-50"
          >
            Reset
          </button>
          <button
            type="button"
            onclick={() => (confirmReset = false)}
            class="rounded-control px-3 py-1.5 text-sm text-ink-2 transition-colors hover:bg-white/5 hover:text-ink"
          >
            Cancel
          </button>
        {:else}
          <button
            type="button"
            onclick={() => (confirmReset = true)}
            class="rounded-control border border-line-soft px-3 py-1.5 text-sm text-ink-3 transition-colors hover:border-accent/40 hover:text-ink-2"
          >
            Reset to default
          </button>
        {/if}
      {:else if confirmDelete}
        <span class="text-sm text-ink-2">Delete this theme?</span>
        <button
          type="button"
          onclick={remove}
          disabled={busy}
          class="rounded-control border border-danger/40 bg-danger/10 px-3 py-1.5 text-sm font-medium text-danger transition-colors hover:bg-danger/20 disabled:opacity-50"
        >
          Delete
        </button>
        <button
          type="button"
          onclick={() => (confirmDelete = false)}
          class="rounded-control px-3 py-1.5 text-sm text-ink-2 transition-colors hover:bg-white/5 hover:text-ink"
        >
          Cancel
        </button>
      {:else}
        <button
          type="button"
          onclick={() => (confirmDelete = true)}
          class="rounded-control border border-line-soft px-3 py-1.5 text-sm text-ink-3 transition-colors hover:border-danger/40 hover:text-danger"
        >
          Delete theme
        </button>
      {/if}
    </div>
  </div>
</section>
