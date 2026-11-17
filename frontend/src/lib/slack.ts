import type { SlackLink } from './api'

/** Ordered Slack links for a task; falls back to the legacy single thread. */
export function slackLinksOf(t: { slack_links?: SlackLink[] | null; slack_thread?: string }): SlackLink[] {
  const links = (t.slack_links ?? []).filter((l) => (l.url ?? '').trim())
  if (links.length) return links
  const u = (t.slack_thread ?? '').trim()
  return u ? [{ url: u, description: '' }] : []
}

const KIND_LABEL: Record<string, string> = {
  thread: 'Thread',
  channel: 'Channel',
  dm: 'Direct message',
  group: 'Group DM',
  workspace: 'Workspace',
  link: 'Slack link'
}

/**
 * Synchronous display label (no validation — core.DescribeSlackLink is the
 * authority and is what the add popover uses). Mirrors its kind rules closely
 * enough for list/menu rows.
 */
export function slackLinkLabel(raw: string): string {
  try {
    const u = new URL(raw.trim())
    let kind = 'link'
    let ws = ''
    if (u.protocol === 'slack:') {
      ws = u.searchParams.get('team') ?? ''
      const host = u.host || u.pathname.replace(/^\/\//, '').split(/[/?]/)[0]
      if (host === 'channel') kind = chanKind(u.searchParams.get('id') ?? '')
      else if (host === 'user') kind = 'dm'
      else if (host === 'open') kind = 'workspace'
    } else {
      const host = u.hostname.toLowerCase()
      const sub = host.endsWith('.slack.com') ? host.slice(0, -'.slack.com'.length) : ''
      if (sub && sub !== 'app' && sub !== 'www') ws = sub.replace(/\.enterprise$/, '')
      const parts = u.pathname.split('/').filter(Boolean)
      let ch = ''
      parts.forEach((p, i) => {
        if ((p === 'archives' || p === 'client') && parts[i + 1]) {
          ch = parts[i + 1]
          if (p === 'client' && parts[i + 2]) {
            if (!ws) ws = parts[i + 1]
            ch = parts[i + 2]
          }
        }
      })
      if (ch) kind = chanKind(ch)
      if (parts.some((p) => /^p\d+$/.test(p) || p === 'thread')) kind = 'thread'
    }
    return ws ? `${KIND_LABEL[kind]} · ${ws}` : KIND_LABEL[kind]
  } catch {
    return raw
  }
}

function chanKind(id: string): string {
  if (id.startsWith('D')) return 'dm'
  if (id.startsWith('G')) return 'group'
  return id ? 'channel' : 'link'
}

/** Row text: description first (what the human wrote), recognised label second. */
export function slackLinkText(l: SlackLink): { primary: string; secondary: string } {
  const label = slackLinkLabel(l.url)
  const d = (l.description ?? '').trim()
  return d ? { primary: d, secondary: label } : { primary: label, secondary: '' }
}
