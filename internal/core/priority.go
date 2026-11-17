package core

import (
	"fmt"
	"strings"
)

// Priority is a ticket's urgency signal (v20). It is purely visual/filtering:
// it never changes board_rank, focus_rank or the default board sort.
type Priority string

const (
	PriorityNone   Priority = "none"
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// AllPriorities is ascending order; the index is the stored level (0..4).
var AllPriorities = []Priority{PriorityNone, PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent}

// Level returns the stored integer (0 = none … 4 = urgent); unknown → 0.
func (p Priority) Level() int {
	for i, q := range AllPriorities {
		if q == p {
			return i
		}
	}
	return 0
}

// PriorityFromLevel maps a stored integer back to a Priority (out of range → none).
func PriorityFromLevel(n int) Priority {
	if n < 0 || n >= len(AllPriorities) {
		return PriorityNone
	}
	return AllPriorities[n]
}

// ParsePriority validates a level name (case-insensitive). "" → none.
func ParsePriority(s string) (Priority, error) {
	v := strings.ToLower(strings.TrimSpace(s))
	if v == "" {
		return PriorityNone, nil
	}
	for _, p := range AllPriorities {
		if v == string(p) {
			return p, nil
		}
	}
	return "", &InvalidPriorityError{Priority: s}
}

// ParsePriorityFilter parses a list filter: comma-separated levels and/or
// thresholds (">=high", ">low", "<=medium", "<urgent"). The result is the
// union of all terms; nil means "no filter".
func ParsePriorityFilter(s string) ([]Priority, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	set := map[Priority]bool{}
	for raw := range strings.SplitSeq(s, ",") {
		term := strings.TrimSpace(raw)
		if term == "" {
			continue
		}
		op := ""
		for _, cand := range []string{">=", "<=", ">", "<"} {
			if strings.HasPrefix(term, cand) {
				op = cand
				term = strings.TrimSpace(strings.TrimPrefix(term, cand))
				break
			}
		}
		p, err := ParsePriority(term)
		if err != nil || term == "" {
			return nil, &InvalidPriorityError{Priority: raw}
		}
		lv := p.Level()
		for _, q := range AllPriorities {
			ql := q.Level()
			match := false
			switch op {
			case "":
				match = ql == lv
			case ">=":
				match = ql >= lv
			case "<=":
				match = ql <= lv
			case ">":
				match = ql > lv
			case "<":
				match = ql < lv
			}
			if match {
				set[q] = true
			}
		}
	}
	out := make([]Priority, 0, len(set))
	for _, p := range AllPriorities {
		if set[p] {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		// e.g. ">urgent": a valid expression that matches nothing. Keep it a
		// filter (not "no filter") by returning an empty non-nil slice.
		return []Priority{}, nil
	}
	return out, nil
}

// InvalidPriorityError is returned for an unknown priority level or filter term.
type InvalidPriorityError struct{ Priority string }

func (e *InvalidPriorityError) Error() string {
	return fmt.Sprintf("invalid priority %q (want none, low, medium, high or urgent; filters also accept >=, <=, >, < and commas)", e.Priority)
}
