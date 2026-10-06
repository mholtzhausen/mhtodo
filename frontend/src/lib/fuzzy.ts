// Port of internal/core/fuzzy.go — case-insensitive subsequence score.
// Higher is better; -1 means no match. Contiguous runs and matches at the
// start of the string or after a non-letter/digit score higher so short unique
// fragments rank above scattered letter hits.

function isWordBoundary(chars: string[], i: number): boolean {
  if (i <= 0) return true
  const prev = chars[i - 1]
  // Mirror unicode.IsLetter / unicode.IsDigit roughly via JS regex.
  return !/\p{L}|\p{N}/u.test(prev)
}

/** Case-insensitive subsequence fuzzy score. Returns -1 when there is no match. */
export function fuzzyScore(query: string, candidate: string): number {
  if (query === '') return 0
  const q = Array.from(query.toLowerCase())
  const c = Array.from(candidate.toLowerCase())
  if (q.length > c.length) return -1

  let score = 0
  let ci = 0
  let prevMatch = -2 // force first match to count as a "gap" break
  for (const qr of q) {
    let found = -1
    for (let j = ci; j < c.length; j++) {
      if (c[j] === qr) {
        found = j
        break
      }
    }
    if (found < 0) return -1
    score += 1
    if (found === prevMatch + 1) {
      score += 5 // contiguous run
    } else if (found === 0 || isWordBoundary(c, found)) {
      score += 3 // start / word boundary
    }
    prevMatch = found
    ci = found + 1
  }
  // Prefer denser matches (query covers more of the candidate).
  score += 2 * Math.floor((q.length * 100) / (c.length + 1))
  return score
}
