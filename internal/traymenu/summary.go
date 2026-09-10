// Package traymenu holds pure helpers for tray label + status submenu payloads.
// Kept free of systray/GTK so unit tests need no CGO display stack.
package traymenu

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// MaxItemsHardCap is the fixed slot count preallocated in the tray menu.
const MaxItemsHardCap = 20

// DefaultMaxItemsPerStatus is the factory default for Notifications.MaxItemsPerStatus.
const DefaultMaxItemsPerStatus = 10

// DefaultTitleTruncate is the max rune length for submenu item labels.
const DefaultTitleTruncate = 48

// TaskRef is one row in a status submenu.
type TaskRef struct {
	ID    string
	Title string
}

// StatusSection is the payload for one status submenu root.
type StatusSection struct {
	Status  string
	Label   string // e.g. "Waiting (2)"
	Visible bool
	Count   int
	Items   []TaskRef
}

// FormatAttentionLabel builds the activity suffix from counts in status order.
// Only statuses with count > 0 are included. Empty when nothing needs attention.
// Example: "2 waiting, 1 review".
func FormatAttentionLabel(order []string, counts map[string]int) string {
	var parts []string
	for _, st := range order {
		st = strings.TrimSpace(strings.ToLower(st))
		if st == "" {
			continue
		}
		n := counts[st]
		if n <= 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%d %s", n, statusNoun(st, n)))
	}
	return strings.Join(parts, ", ")
}

func statusNoun(status string, n int) string {
	if status == "review" && n != 1 {
		return "reviews"
	}
	return status
}

// FormatTrayLabel returns the icon label / tooltip text.
// When attention is non-empty: "mhtodo · 2 waiting, 1 review".
// Else falls back to open-count: "mhtodo (N)" or "mhtodo".
func FormatTrayLabel(attention string, openCount int) string {
	attention = strings.TrimSpace(attention)
	if attention != "" {
		return "mhtodo · " + attention
	}
	if openCount > 0 {
		return fmt.Sprintf("mhtodo (%d)", openCount)
	}
	return "mhtodo"
}

// StatusMenuTitle is the submenu root label, e.g. "Waiting (2)".
func StatusMenuTitle(status string, count int) string {
	return fmt.Sprintf("%s (%d)", StatusDisplayName(status), count)
}

// StatusDisplayName returns a short title-case name for menu roots.
func StatusDisplayName(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "Pending"
	case "wip":
		return "WIP"
	case "waiting":
		return "Waiting"
	case "review":
		return "Review"
	case "done":
		return "Done"
	default:
		s := strings.TrimSpace(status)
		if s == "" {
			return "Status"
		}
		return strings.ToUpper(s[:1]) + s[1:]
	}
}

// TruncateTitle shortens s to at most max runes, appending "…" when trimmed.
func TruncateTitle(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 {
		max = DefaultTitleTruncate
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	if max == 1 {
		return "…"
	}
	return string(runes[:max-1]) + "…"
}

// ClampMaxItems bounds n to [1, MaxItemsHardCap]; 0 or negative → default.
func ClampMaxItems(n int) int {
	if n <= 0 {
		return DefaultMaxItemsPerStatus
	}
	if n > MaxItemsHardCap {
		return MaxItemsHardCap
	}
	return n
}

// BuildSections builds visible status sections from menu order + per-status tasks.
// counts may be larger than len(items) when capped; Count uses counts when set,
// otherwise len(items).
func BuildSections(menuOrder []string, counts map[string]int, tasks map[string][]TaskRef, maxItems int) []StatusSection {
	maxItems = ClampMaxItems(maxItems)
	out := make([]StatusSection, 0, len(menuOrder))
	seen := map[string]bool{}
	for _, st := range menuOrder {
		st = strings.TrimSpace(strings.ToLower(st))
		if st == "" || seen[st] {
			continue
		}
		seen[st] = true
		items := tasks[st]
		if len(items) > maxItems {
			items = items[:maxItems]
		}
		copied := append([]TaskRef(nil), items...)
		n := counts[st]
		if n < len(copied) {
			n = len(copied)
		}
		out = append(out, StatusSection{
			Status:  st,
			Label:   StatusMenuTitle(st, n),
			Visible: true,
			Count:   n,
			Items:   copied,
		})
	}
	return out
}
