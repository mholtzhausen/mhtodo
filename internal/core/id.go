package core

import "strings"

// ShortID returns a compact human-facing identifier for a task or activity UUID:
// the last 8 hex digits with hyphens removed. Unlike the first 8 characters of a
// UUIDv7 string, this suffix is unique per generated id.
func ShortID(id string) string {
	compact := strings.ReplaceAll(strings.TrimSpace(id), "-", "")
	if compact == "" {
		return ""
	}
	if len(compact) <= 8 {
		return compact
	}
	return compact[len(compact)-8:]
}
