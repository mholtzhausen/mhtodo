import { marked } from 'marked'
import DOMPurify from 'dompurify'

marked.setOptions({
  gfm: true,
  breaks: true // single newlines → <br> (matches how people write task notes)
})

/** Render markdown to sanitized HTML safe for `{@html …}`. */
export function renderMarkdown(source: string | null | undefined): string {
  const raw = (source ?? '').trim()
  if (!raw) return ''
  const html = marked.parse(raw, { async: false }) as string
  return DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true }
  })
}
