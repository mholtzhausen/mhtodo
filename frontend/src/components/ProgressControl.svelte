<script lang="ts">
  // Progress editor for the detail drawer and the new-task dialog — slider +
  // −/+ stepper, replacing the old native number input (Slate design).
  // Commits whole steps; the slider commits on release (onchange) so a drag
  // issues one API call, same semantics as the pre-redesign control.
  let {
    value,
    step = 5,
    disabled = false,
    onCommit
  }: { value: number; step?: number; disabled?: boolean; onCommit: (v: number) => void } = $props()

  const clamp = (v: number) => Math.max(0, Math.min(100, v))
  function bump(d: number) {
    if (disabled) return
    onCommit(clamp(value + d))
  }
  function onRelease(e: Event) {
    if (disabled) return
    onCommit(clamp(Number((e.target as HTMLInputElement).value)))
  }
</script>

<div class="flex items-center gap-gap-md {disabled ? 'opacity-55' : ''}">
  <input
    type="range"
    min="0"
    max="100"
    step={step}
    value={value}
    disabled={disabled}
    onchange={onRelease}
    aria-label="Progress"
    class="prog-range min-w-0 flex-1"
    style="background: linear-gradient(to right, var(--color-accent) {value}%, var(--color-track) {value}%)"
  />
  <div class="flex flex-none items-center gap-1.5">
    <button
      type="button"
      class="stepbtn"
      title="Decrease {step}%"
      aria-label="Decrease progress"
      disabled={disabled}
      onclick={() => bump(-step)}
    >
      −
    </button>
    <span class="w-11 text-center font-mono text-xs text-ink"
      >{value}<span class="text-[9px] text-ink-3">%</span></span
    >
    <button
      type="button"
      class="stepbtn"
      title="Increase {step}%"
      aria-label="Increase progress"
      disabled={disabled}
      onclick={() => bump(step)}
    >
      +
    </button>
  </div>
</div>
