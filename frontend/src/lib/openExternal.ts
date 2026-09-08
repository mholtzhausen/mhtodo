/** Open an http(s) URL in the system browser (Wails) or a new tab (browser). */
export async function openExternalUrl(raw: string): Promise<boolean> {
  const url = raw.trim()
  if (!url) return false
  let parsed: URL
  try {
    parsed = new URL(url)
  } catch {
    return false
  }
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    return false
  }
  const href = parsed.href
  const inWails = typeof window !== 'undefined' && !!(window as any).runtime
  if (inWails) {
    const { BrowserOpenURL } = await import('../../wailsjs/runtime/runtime.js')
    BrowserOpenURL(href)
    return true
  }
  window.open(href, '_blank', 'noopener,noreferrer')
  return true
}
