package core

import "strings"

// ParsePRURLs splits a pr_url blob into ordered unique URLs (one per line).
// Blank lines are dropped; first occurrence wins on duplicates.
func ParsePRURLs(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		u := strings.TrimSpace(line)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

// JoinPRURLs joins URL strings with newlines after ParsePRURLs normalization.
func JoinPRURLs(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	return strings.Join(ParsePRURLs(strings.Join(urls, "\n")), "\n")
}

// NormalizePRURL canonicalizes a pr_url blob: trim lines, drop blanks, dedupe.
func NormalizePRURL(s string) string {
	return strings.Join(ParsePRURLs(s), "\n")
}
