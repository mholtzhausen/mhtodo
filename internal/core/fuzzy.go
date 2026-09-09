package core

import "unicode"

// fuzzyScore returns how well query matches candidate (case-insensitive
// subsequence). Higher is better; -1 means no match. Contiguous runs and
// matches at the start of the string or after a non-letter score higher so
// short unique fragments rank above scattered letter hits.
func fuzzyScore(query, candidate string) int {
	if query == "" {
		return 0
	}
	q := []rune(toLower(query))
	c := []rune(toLower(candidate))
	if len(q) > len(c) {
		return -1
	}

	score := 0
	ci := 0
	prevMatch := -2 // force first match to count as a "gap" break
	for _, qr := range q {
		found := -1
		for j := ci; j < len(c); j++ {
			if c[j] == qr {
				found = j
				break
			}
		}
		if found < 0 {
			return -1
		}
		score += 1
		if found == prevMatch+1 {
			score += 5 // contiguous run
		} else if found == 0 || isWordBoundary(c, found) {
			score += 3 // start / word boundary
		}
		prevMatch = found
		ci = found + 1
	}
	// Prefer denser matches (query covers more of the candidate).
	score += 2 * (len(q) * 100 / (len(c) + 1))
	return score
}

func toLower(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		runes[i] = unicode.ToLower(r)
	}
	return string(runes)
}

func isWordBoundary(runes []rune, i int) bool {
	if i <= 0 {
		return true
	}
	prev := runes[i-1]
	return !unicode.IsLetter(prev) && !unicode.IsDigit(prev)
}
