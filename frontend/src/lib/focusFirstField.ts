/**
 * Modal open focus: prefer an explicit primary field, then the first text-like
 * input, then a marked/autofocus button (confirm dialogs with no fields).
 *
 * Skips header chrome (close buttons) and checkboxes/radios so Settings lands
 * on the first real text field, not a nav button or toggle.
 */

const TEXT_FIELD =
  'input:not([type="hidden"]):not([type="checkbox"]):not([type="radio"]):not([type="button"]):not([type="submit"]):not([type="reset"]):not([disabled]), textarea:not([disabled]), select:not([disabled])'

function canFocus(el: HTMLElement | null | undefined): el is HTMLElement {
  if (!el) return false
  if (el.getAttribute('tabindex') === '-1') return false
  if ((el as HTMLInputElement).disabled) return false
  const style = getComputedStyle(el)
  if (style.visibility === 'hidden' || style.display === 'none') return false
  return true
}

/** Focus the best first field inside `root`. Returns the focused element, or null. */
export function focusFirstField(root: ParentNode | null | undefined): HTMLElement | null {
  if (!root) return null

  const primary = root.querySelector<HTMLElement>('[data-focus-primary]')
  if (canFocus(primary)) {
    primary.focus()
    return primary
  }

  const required = root.querySelector<HTMLElement>(`${TEXT_FIELD}[required]`)
  if (canFocus(required)) {
    required.focus()
    return required
  }

  const text = root.querySelector<HTMLElement>(TEXT_FIELD)
  if (canFocus(text)) {
    text.focus()
    return text
  }

  const autoBtn = root.querySelector<HTMLElement>('button[autofocus], [data-focus-primary]')
  if (canFocus(autoBtn)) {
    autoBtn.focus()
    return autoBtn
  }

  return null
}

/**
 * Svelte action: when the node mounts (typical `{#if open}` modal body), focus
 * the first field after paint so fly transitions / layout have settled.
 * Pass `false` to disable (e.g. TaskDetail pin/float modes).
 */
export function focusOnOpen(node: HTMLElement, enabled: boolean = true) {
  let cancelled = false
  let raf = 0

  function run() {
    cancelAnimationFrame(raf)
    if (!enabled || cancelled) return
    raf = requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        if (!cancelled && enabled) focusFirstField(node)
      })
    })
  }

  run()

  return {
    update(next: boolean = true) {
      enabled = next
      if (enabled) run()
    },
    destroy() {
      cancelled = true
      cancelAnimationFrame(raf)
    }
  }
}

/** Schedule focusFirstField after the next paint (for async-ready modals). */
export function scheduleFocusFirstField(root: ParentNode | null | undefined) {
  requestAnimationFrame(() => {
    requestAnimationFrame(() => focusFirstField(root))
  })
}
