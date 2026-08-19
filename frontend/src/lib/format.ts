export function shortId(id: string): string {
  return id.slice(0, 8)
}

export function relTime(iso: string): string {
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return iso
  const s = Math.max(0, (Date.now() - then) / 1000)
  if (s < 45) return 'just now'
  const m = s / 60
  if (m < 60) return `${Math.round(m)}m ago`
  const h = m / 60
  if (h < 24) return `${Math.round(h)}h ago`
  const d = h / 24
  if (d < 7) return `${Math.round(d)}d ago`
  return new Date(iso).toLocaleDateString()
}

export function absShort(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

export const STATUS_LABELS: Record<string, string> = {
  pending: 'Pending',
  wip: 'In progress',
  waiting: 'Waiting',
  done: 'Done'
}
