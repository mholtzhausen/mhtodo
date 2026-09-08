package core

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// NewTodoSessionID returns a fresh UUIDv7 for Claude --session-id / --resume.
func NewTodoSessionID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate todo session id: %w", err)
	}
	return id.String(), nil
}

// DefaultTodoSession builds a space-free display slug for Claude --name,
// Herdr-independent labels, and MHTODO_SESSION_NAME: "{shortID}-{slugified-title}",
// or just shortID when title yields an empty slug.
func DefaultTodoSession(shortID, title string) string {
	shortID = strings.TrimSpace(shortID)
	slug := SessionSlug(title)
	if slug == "" {
		return shortID
	}
	return shortID + "-" + slug
}

// ClaudeDisplayName picks the --name value for a Claude launch. Non-UUID
// todo_session values (legacy slugs / user-set names) are used as the display
// name; UUID sessions use DefaultTodoSession(shortID, title).
func ClaudeDisplayName(shortID, title, todoSession string) string {
	todoSession = strings.TrimSpace(todoSession)
	if todoSession != "" && !LooksLikeSessionUUID(todoSession) {
		return todoSession
	}
	return DefaultTodoSession(shortID, title)
}

// EnsureClaudeSessionID returns a Claude session UUID. When todoSession is
// already a UUID it is returned unchanged (generated=false). Otherwise a new
// UUIDv7 is minted (generated=true) so callers can persist it.
func EnsureClaudeSessionID(todoSession string) (session string, generated bool, err error) {
	todoSession = strings.TrimSpace(todoSession)
	if LooksLikeSessionUUID(todoSession) {
		return todoSession, false, nil
	}
	id, err := NewTodoSessionID()
	if err != nil {
		return "", false, err
	}
	return id, true, nil
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
// hyphen, trimmed, capped at 40 characters (safe for shell/env/--name).
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
// (used to choose --name display slug vs session id).
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
