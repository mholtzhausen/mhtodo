package core

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// DefaultTodoSession builds a space-free session slug for Claude --name/--resume,
// MHTODO_SESSION, and claude.todo: "{shortID}-{slugified-title}", or just shortID
// when title yields an empty slug.
//
// Herdr tab labels stay human-readable separately (see integrations.ticketTabLabel).
func DefaultTodoSession(shortID, title string) string {
	shortID = strings.TrimSpace(shortID)
	slug := SessionSlug(title)
	if slug == "" {
		return shortID
	}
	return shortID + "-" + slug
}

// LegacyTodoSession is the pre-space-free slug ("{shortID} - {title≤40}") used to
// detect auto-seeded values that should be rewritten on migration.
func LegacyTodoSession(shortID, title string) string {
	shortID = strings.TrimSpace(shortID)
	title = strings.TrimSpace(title)
	if title == "" {
		return shortID
	}
	if utf8.RuneCountInString(title) > 40 {
		runes := []rune(title)
		title = string(runes[:40])
	}
	return shortID + " - " + title
}

// SessionSlug lowercases title and replaces non-alphanumeric runs with a single
// hyphen, trimmed, capped at 40 characters (safe for shell/env/--name/--resume).
func SessionSlug(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(title))
	prevHyphen := false
	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevHyphen = false
		default:
			if b.Len() == 0 || prevHyphen {
				continue
			}
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if utf8.RuneCountInString(s) > 40 {
		runes := []rune(s)
		s = strings.Trim(string(runes[:40]), "-")
	}
	return s
}

// LooksLikeSessionUUID reports whether s looks like a Claude session UUID
// (used to choose -n display name vs resume id).
func LooksLikeSessionUUID(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 36 {
		return false
	}
	// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}
