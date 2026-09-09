import { marked } from 'marked'
import DOMPurify from 'dompurify'

marked.setOptions({
  gfm: true,
  breaks: true // single newlines → <br> (matches how people write task notes)
})

const CACHE_MAX = 128
const cache = new Map<string, string>()

function cacheGet(key: string): string | undefined {
  const hit = cache.get(key)
  if (hit === undefined) return undefined
  // refresh LRU order
  cache.delete(key)
  cache.set(key, hit)
  return hit
}

function cacheSet(key: string, value: string) {
  if (cache.has(key)) cache.delete(key)
  cache.set(key, value)
  while (cache.size > CACHE_MAX) {
    const oldest = cache.keys().next().value
    if (oldest === undefined) break
    cache.delete(oldest)
  }
}

/** Render markdown to sanitized HTML safe for `{@html …}`. */
export function renderMarkdown(source: string | null | undefined): string {
  const raw = (source ?? '').trim()
  if (!raw) return ''
  const hit = cacheGet(raw)
  if (hit !== undefined) return hit
  const html = marked.parse(raw, { async: false }) as string
  const sanitized = DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true }
  })
  cacheSet(raw, sanitized)
  return sanitized
}
