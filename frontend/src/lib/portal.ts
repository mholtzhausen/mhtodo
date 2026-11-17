/** Svelte action: move the node to <body> so overflow/scroll containers never clip it. */
export function portal(node: HTMLElement) {
  document.body.appendChild(node)
  return {
    destroy() {
      node.remove()
    }
  }
}
