<script lang="ts">
  import { fly } from 'svelte/transition'
  import { focusOnOpen } from '../lib/focusFirstField'
  import type { InstallStatus } from '../lib/api'
  import ZedIcon from './ZedIcon.svelte'

  let {
    open,
    status = null,
    busy = false,
    refreshing = false,
    onCancel,
    onConfirm,
    onRefresh
  }: {
    open: boolean
    status?: InstallStatus | null
    busy?: boolean
    refreshing?: boolean
    onCancel: () => void
    onConfirm: (opts: {
      updateApp: boolean
      installService: boolean
      integrationZsh: boolean
      integrationBash: boolean
    }) => void
    onRefresh?: () => void
  } = $props()

  let updateApp = $state(true)
  let installService = $state(false)
  let integrationZsh = $state(false)
  let integrationBash = $state(false)
  let seeded = false

  $effect(() => {
    if (!open) {
      seeded = false
      return
    }
    if (seeded) return
    seeded = true
    updateApp = true
    installService = status ? !status.has_service : false
    integrationZsh = false
    integrationBash = false
  })

  const canConfirm = $derived(
    (updateApp || installService || integrationZsh || integrationBash) && !busy
  )

  /** "upgrade" when a newer release is known; otherwise "install". */
  const actionVerb = $derived(
    status && status.latest_version && !status.up_to_date ? 'upgrade' : 'install'
  )

  const currentLabel = $derived(
    status?.current_version?.trim() ? status.current_version.trim() : '…'
  )

  const targetVersion = $derived(status?.latest_version?.trim() || '')

  const statusBadge = $derived.by(() => {
    if (!status) return { label: 'Checking…', tone: 'muted' as const }
    if (!status.latest_version) return { label: 'No release data', tone: 'muted' as const }
    if (!status.up_to_date) return { label: 'Update available', tone: 'accent' as const }
    return { label: 'Up to date', tone: 'ok' as const }
  })

  const planBits = $derived.by(() => {
    const bits: string[] = []
    if (updateApp) bits.push(targetVersion ? `${actionVerb} ${targetVersion}` : actionVerb)
    if (installService) bits.push(status?.has_service ? 'refresh service' : 'install service')
    if (integrationZsh) bits.push('zsh')
    if (integrationBash) bits.push('bash')
    return bits
  })

  function submit() {
    if (!canConfirm) return
    onConfirm({ updateApp, installService, integrationZsh, integrationBash })
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault()
      submit()
    }
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
  <div
    class="fixed inset-0 z-[55] flex items-center justify-center bg-black/55 p-4"
    onclick={() => !busy && onCancel()}
  >
    <!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
    <div
      use:focusOnOpen
      in:fly={{ y: 8, duration: 80 }}
      role="alertdialog"
      aria-modal="true"
      aria-label="Install or upgrade"
      tabindex="-1"
      onkeydown={onKeydown}
      onclick={(e) => e.stopPropagation()}
      class="w-full max-w-md overflow-hidden rounded-panel border border-line bg-col shadow-md"
    >
      <!-- Version delta header -->
      <div class="relative border-b border-line-soft px-5 pb-4 pt-4">
        <div
          class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-accent/50 to-transparent"
          aria-hidden="true"
        ></div>
        <div class="flex items-start gap-3">
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <p class="micro !mb-0">Current build</p>
              {#if statusBadge.tone === 'accent'}
                <span
                  class="rounded-chip border border-accent/35 bg-accent/15 px-1.5 py-0.5 text-[10px] font-semibold tracking-wide text-accent-hi"
                >
                  {statusBadge.label}
                </span>
              {:else if statusBadge.tone === 'ok'}
                <span
                  class="rounded-chip border border-st-done/35 bg-st-done/10 px-1.5 py-0.5 text-[10px] font-semibold tracking-wide text-st-done"
                >
                  {statusBadge.label}
                </span>
              {:else}
                <span
                  class="rounded-chip border border-line-soft bg-field/60 px-1.5 py-0.5 text-[10px] font-semibold tracking-wide text-ink-3"
                >
                  {statusBadge.label}
                </span>
              {/if}
            </div>

            <div class="mt-2.5 flex flex-wrap items-center gap-2">
              <span
                class="inline-flex items-center rounded-control border border-line-soft bg-field px-2.5 py-1 font-mono text-sm text-ink-2"
              >
                {currentLabel}
              </span>
              {#if targetVersion}
                <svg
                  class="h-3.5 w-3.5 flex-none text-ink-3"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <path d="M5 12h14" />
                  <path d="m12 5 7 7-7 7" />
                </svg>
                <span
                  class="inline-flex items-center rounded-control border border-accent/40 bg-accent/10 px-2.5 py-1 font-mono text-sm font-semibold text-accent-hi"
                >
                  {targetVersion}
                </span>
              {/if}
            </div>

            {#if status?.install_path}
              <p class="mt-2 truncate font-mono text-[11px] text-ink-3" title={status.install_path}>
                {status.install_path}
              </p>
            {/if}
            {#if status?.message && !status.latest_version}
              <p class="mt-1.5 text-xs leading-relaxed text-ink-3">{status.message}</p>
            {/if}
          </div>

          <button
            type="button"
            onclick={() => onRefresh?.()}
            disabled={busy || refreshing}
            title={status?.fresh
              ? 'Refresh version check'
              : 'Refresh version check (cache stale or empty)'}
            aria-label="Refresh version check"
            class="grid h-8 w-8 flex-none place-items-center rounded-control border border-line-soft text-ink-3 transition-colors hover:border-line hover:bg-white/5 hover:text-ink disabled:opacity-40"
          >
            <svg
              class="h-3.5 w-3.5 {refreshing ? 'animate-spin' : ''}"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <path d="M21 12a9 9 0 1 1-2.6-6.2" />
              <polyline points="21 3 21 9 15 9" />
            </svg>
          </button>
        </div>
      </div>

      <div class="flex flex-col gap-gap-lg px-5 py-4">
        <!-- App action card -->
        <div>
          <p class="micro mb-2 capitalize">{actionVerb}</p>
          <div
            class="overflow-hidden rounded-card border transition-colors
              {updateApp
              ? 'border-accent/45 bg-accent/10 shadow-[inset_3px_0_0_0_var(--color-accent)]'
              : 'border-line-soft bg-field/40'}"
          >
            <label
              class="flex cursor-pointer items-start gap-3 px-3.5 py-3 transition-colors
                {updateApp ? '' : 'hover:bg-field/70'}"
            >
              <input
                type="checkbox"
                data-focus-primary
                bind:checked={updateApp}
                disabled={busy}
                class="peer sr-only"
              />
              <span
                class="mt-0.5 grid h-4 w-4 flex-none place-items-center rounded-control border transition-colors peer-focus-visible:ring-2 peer-focus-visible:ring-accent/40
                  {updateApp
                  ? 'border-accent bg-accent text-accent-ink'
                  : 'border-line-soft bg-field text-transparent'}"
                aria-hidden="true"
              >
                <svg class="h-2.5 w-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium capitalize text-ink">{actionVerb}</span>
                  {#if targetVersion}
                    <span class="font-mono text-sm text-ink-2">{targetVersion}</span>
                  {:else}
                    <span class="text-sm text-ink-3">…</span>
                  {/if}
                </div>
                <p class="mt-0.5 text-xs leading-relaxed text-ink-3">
                  Copy this build into <span class="font-mono">~/.local</span>
                  {#if status?.ephemeral}
                    <span class="text-ink-2"> · running from source / temp</span>
                  {/if}
                </p>
              </div>
              <svg
                class="mt-0.5 h-4 w-4 flex-none text-ink-3 opacity-70"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.75"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
              >
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="7 10 12 15 17 10" />
                <line x1="12" x2="12" y1="15" y2="3" />
              </svg>
            </label>

            <label
              class="flex cursor-pointer items-center gap-gap-md border-t border-line-soft/80 px-3.5 py-2.5
                {updateApp ? 'bg-accent/5' : 'bg-transparent hover:bg-field/50'}"
            >
              <input
                type="checkbox"
                bind:checked={installService}
                disabled={busy}
                class="h-3.5 w-3.5 rounded-control border-line-soft bg-field text-accent focus:ring-accent/25"
              />
              <span class="flex-1 text-xs text-ink-2">
                Also {status?.has_service ? 'refresh' : 'install'} user systemd service
              </span>
              {#if status?.has_service}
                <span class="rounded-chip border border-line-soft px-1.5 py-0.5 text-[10px] text-ink-3"
                  >present</span
                >
              {/if}
            </label>
          </div>
        </div>

        <!-- Shell / Zed integration tiles -->
        <div>
          <p class="micro mb-2 flex flex-wrap items-center gap-x-1.5 gap-y-0.5">
            <span>Shell helper</span>
            <span class="text-ink-3/50" aria-hidden="true">/</span>
            <span class="inline-flex items-center gap-1">
              <ZedIcon class="h-3 w-3" />
              Zed integration
            </span>
          </p>
          <p class="mb-2.5 text-xs leading-relaxed text-ink-3">
            Optional
            <span
              class="mx-0.5 inline-flex align-middle rounded-chip border border-line-soft bg-white/5 px-1.5 py-[2px] font-mono text-[10px] leading-none text-ink-2"
              >claude.todo</span
            >
            helper for
            <span class="font-mono font-semibold text-ink-2">$MHTODO_SESSION</span>.
          </p>
          <div class="grid grid-cols-2 gap-2">
            <button
              type="button"
              disabled={busy}
              aria-pressed={integrationZsh}
              onclick={() => (integrationZsh = !integrationZsh)}
              class="relative flex items-center gap-2 rounded-card border px-2.5 py-2 text-left transition-colors disabled:opacity-40
                {integrationZsh
                ? 'border-accent/45 bg-accent/10 shadow-[inset_3px_0_0_0_var(--color-accent)]'
                : 'border-line-soft bg-field/40 hover:border-line hover:bg-field/70'}"
            >
              <span class="font-mono text-[11px] text-ink-3">%</span>
              <span class="min-w-0 flex-1">
                <span class="block text-sm font-semibold leading-tight text-ink">zsh</span>
                <span class="block font-mono text-[10px] leading-tight text-ink-3">~/.zshrc</span>
              </span>
              {#if integrationZsh}
                <span
                  class="grid h-4 w-4 flex-none place-items-center rounded-full bg-accent text-accent-ink"
                  aria-hidden="true"
                >
                  <svg class="h-2.5 w-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                </span>
              {/if}
            </button>
            <button
              type="button"
              disabled={busy}
              aria-pressed={integrationBash}
              onclick={() => (integrationBash = !integrationBash)}
              class="relative flex items-center gap-2 rounded-card border px-2.5 py-2 text-left transition-colors disabled:opacity-40
                {integrationBash
                ? 'border-accent/45 bg-accent/10 shadow-[inset_3px_0_0_0_var(--color-accent)]'
                : 'border-line-soft bg-field/40 hover:border-line hover:bg-field/70'}"
            >
              <span class="font-mono text-[11px] text-ink-3">$</span>
              <span class="min-w-0 flex-1">
                <span class="block text-sm font-semibold leading-tight text-ink">bash</span>
                <span class="block font-mono text-[10px] leading-tight text-ink-3">~/.bashrc</span>
              </span>
              {#if integrationBash}
                <span
                  class="grid h-4 w-4 flex-none place-items-center rounded-full bg-accent text-accent-ink"
                  aria-hidden="true"
                >
                  <svg class="h-2.5 w-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                </span>
              {/if}
            </button>
          </div>
        </div>
      </div>

      <div class="flex flex-col gap-2 border-t border-line-soft px-5 py-3">
        {#if planBits.length}
          <p class="truncate text-[11px] text-ink-3" title={planBits.join(' · ')}>
            Will: <span class="text-ink-2">{planBits.join(' · ')}</span>
          </p>
        {:else}
          <p class="text-[11px] text-ink-3">Pick at least one action</p>
        {/if}
        <div class="flex items-center justify-end gap-2">
          <button
            type="button"
            onclick={onCancel}
            disabled={busy}
            class="rounded-control px-3 py-1.5 text-sm text-ink-2 transition-colors hover:bg-white/5 hover:text-ink disabled:opacity-40"
          >
            Cancel
          </button>
          <button
            type="button"
            onclick={submit}
            disabled={!canConfirm}
            class="rounded-control bg-accent px-4 py-1.5 text-sm font-medium text-accent-ink shadow-sm transition-colors hover:bg-accent-hi disabled:cursor-not-allowed disabled:opacity-40"
          >
            {busy ? 'Working…' : 'Continue'}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
