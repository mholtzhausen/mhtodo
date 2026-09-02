<script lang="ts">
  let {
    value,
    placeholder = '',
    multiline = false,
    rows = 4,
    mono = false,
    onChange
  }: {
    value: string
    placeholder?: string
    multiline?: boolean
    rows?: number
    mono?: boolean
    onChange: (value: string) => void
  } = $props()

  const fieldClass =
    'w-full rounded border border-line-soft bg-field py-1.5 pl-3 text-ink shadow-[inset_0_1px_2px_rgba(6,8,12,0.35)] placeholder:text-ink-3 focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/25' +
    (value.trim() ? ' pr-8' : ' pr-3') +
    (mono ? ' font-mono text-xs leading-relaxed resize-y' : ' text-sm')
</script>

<div class="relative">
  {#if value.trim()}
    <button
      type="button"
      title="Clear"
      aria-label="Clear field"
      onclick={() => onChange('')}
      class="absolute right-1.5 top-1.5 z-10 rounded p-0.5 text-ink-3 transition-colors hover:bg-white/10 hover:text-ink"
    >
      <svg
        class="h-3.5 w-3.5"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M3 6h18" />
        <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" />
        <path d="M10 11v6" />
        <path d="M14 11v6" />
      </svg>
    </button>
  {/if}
  {#if multiline}
    <textarea
      {rows}
      {value}
      {placeholder}
      oninput={(e) => onChange((e.currentTarget as HTMLTextAreaElement).value)}
      class={fieldClass}
    ></textarea>
  {:else}
    <input
      {value}
      {placeholder}
      oninput={(e) => onChange((e.currentTarget as HTMLInputElement).value)}
      class={fieldClass}
    />
  {/if}
</div>
