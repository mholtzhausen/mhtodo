/** Split a pr_url blob into ordered unique URLs (one per line). */
export function parsePRURLs(raw: string | null | undefined): string[] {
  if (!raw) return []
  const out: string[] = []
  const seen = new Set<string>()
  for (const line of raw.split('\n')) {
    const u = line.trim()
    if (!u || seen.has(u)) continue
    seen.add(u)
    out.push(u)
  }
  return out
}

/** Join URL lines into a normalized pr_url blob. */
export function joinPRURLs(urls: string[]): string {
  return parsePRURLs(urls.join('\n')).join('\n')
}

/** Short label for a PR URL in menus (path tail or host). */
export function prURLLabel(url: string): string {
  try {
    const u = new URL(url)
    const path = u.pathname.replace(/\/+$/, '')
    const tail = path.split('/').filter(Boolean).slice(-2).join('/')
    return tail || u.host || url
  } catch {
    return url
  }
}
