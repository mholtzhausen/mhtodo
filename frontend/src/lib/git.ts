// Git repo/branch helpers for the GUI. core.NormalizeGitRepo stores repos as
// "host/path" (e.g. github.com/org/x) or a local path; only the former link.

/** Splits a git_branches blob into trimmed, unique, non-empty branch names. */
export function parseGitBranches(s: string | null | undefined): string[] {
  return [...new Set((s ?? '').split('\n').map((l) => l.trim()).filter(Boolean))]
}

/**
 * Web URL for a normalized repo, or for a branch in it. Returns '' for local
 * paths and anything without a dotted host. GitLab hosts use /-/tree/; other
 * hosts (GitHub, Gitea, Bitbucket-style) use /tree/.
 */
export function gitWebUrl(repo: string | null | undefined, branch?: string): string {
  const r = (repo ?? '').trim().replace(/\/+$/, '')
  if (!r || /^[/~.]/.test(r) || r.includes('://')) return ''
  const slash = r.indexOf('/')
  if (slash <= 0) return ''
  const host = r.slice(0, slash).toLowerCase()
  if (!host.includes('.') || host.includes(':')) return ''
  const base = `https://${r}`
  if (!branch) return base
  const ref = branch.split('/').map(encodeURIComponent).join('/')
  return host.includes('gitlab') ? `${base}/-/tree/${ref}` : `${base}/tree/${ref}`
}
