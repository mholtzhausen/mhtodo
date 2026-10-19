package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// FocusMax caps how many lanes the Focus view shows.
const FocusMax = 6

// ErrFocusFull is returned when pinning would exceed FocusMax lanes.
var ErrFocusFull = fmt.Errorf("focus is full (%d lanes) — unfocus a task first", FocusMax)

// ErrFocusClosed is returned when pinning a done or archived task.
var ErrFocusClosed = errors.New("done or archived tasks cannot be focused")

// ErrNotFocused is returned when --before points at a task that is not focused.
var ErrNotFocused = errors.New("before task is not focused")

// FocusLanes applies the lane rule to focused tasks (already in focus order):
// a focused parent with a focused sub-task gets no lane of its own — only the
// sub-task shows. The parent stays pinned and reappears once no child is.
func FocusLanes(pins []Task) []Task {
	hidden := map[string]bool{}
	for _, t := range pins {
		if t.ParentID != nil {
			hidden[*t.ParentID] = true
		}
	}
	out := make([]Task, 0, len(pins))
	for _, t := range pins {
		if !hidden[t.ID] {
			out = append(out, t)
		}
	}
	return out
}

// ListFocus returns the Focus view lanes in order (see FocusLanes).
func (s *Service) ListFocus(ctx context.Context) ([]Task, error) {
	pins, err := s.repo.ListFocused(ctx)
	if err != nil {
		return nil, err
	}
	return FocusLanes(pins), nil
}

// Focus pins a task to the Focus view, or moves an already-pinned one.
// beforeRef nil appends (moves a pinned task to the end, like reorder);
// otherwise the task goes immediately before that focused task. Done/archived tasks are refused, and
// a new pin may not push the visible lane count past FocusMax.
func (s *Service) Focus(ctx context.Context, ref string, beforeRef *string) (Task, error) {
	t, err := s.Get(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	if t.Status == StatusDone || t.ArchivedAt != nil {
		return Task{}, ErrFocusClosed
	}
	pins, err := s.repo.ListFocused(ctx)
	if err != nil {
		return Task{}, err
	}

	var others []Task
	for _, p := range pins {
		if p.ID != t.ID {
			others = append(others, p)
		}
	}
	if t.FocusRank == nil && len(FocusLanes(append(append([]Task{}, others...), t))) > FocusMax {
		return Task{}, ErrFocusFull
	}

	var beforeID *string
	if beforeRef != nil && strings.TrimSpace(*beforeRef) != "" {
		bid, err := s.ResolveID(ctx, strings.TrimSpace(*beforeRef))
		if err != nil {
			return Task{}, err
		}
		if bid == t.ID {
			if t.FocusRank == nil {
				return Task{}, ErrNotFocused
			}
			return t, nil
		}
		beforeID = &bid
	} else if t.FocusRank != nil && (len(others) == 0 || *others[len(others)-1].FocusRank < *t.FocusRank) {
		return t, nil // already last
	}

	rank, err := s.computeFocusRank(ctx, others, beforeID)
	if err != nil {
		return Task{}, err
	}
	if err := s.repo.UpdateFocusRank(ctx, t.ID, &rank); err != nil {
		return Task{}, err
	}
	t.FocusRank = &rank
	return t, nil
}

// Unfocus removes a task from the Focus view. Unpinned tasks are a no-op.
func (s *Service) Unfocus(ctx context.Context, ref string) (Task, error) {
	t, err := s.Get(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	if t.FocusRank == nil {
		return t, nil
	}
	if err := s.repo.UpdateFocusRank(ctx, t.ID, nil); err != nil {
		return Task{}, err
	}
	t.FocusRank = nil
	return t, nil
}

// ClearFocus unpins every task and returns how many were pinned.
func (s *Service) ClearFocus(ctx context.Context) (int, error) {
	return s.repo.ClearFocus(ctx)
}

// computeFocusRank places a task among others (focus order, excluding the
// task itself): appended when beforeID is nil, else just before beforeID.
func (s *Service) computeFocusRank(ctx context.Context, others []Task, beforeID *string) (float64, error) {
	if beforeID == nil {
		if len(others) == 0 {
			return 1.0, nil
		}
		return *others[len(others)-1].FocusRank + 1.0, nil
	}
	idx := -1
	for i, p := range others {
		if p.ID == *beforeID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return 0, ErrNotFocused
	}
	if idx == 0 {
		return *others[0].FocusRank - 1.0, nil
	}
	prev, next := *others[idx-1].FocusRank, *others[idx].FocusRank
	if next-prev >= minBoardRankGap {
		return (prev + next) / 2, nil
	}
	// Ranks too close — renumber 1..n (one statement per row) and retry.
	for i := range others {
		r := float64(i + 1)
		if err := s.repo.UpdateFocusRank(ctx, others[i].ID, &r); err != nil {
			return 0, err
		}
		others[i].FocusRank = &r
	}
	return s.computeFocusRank(ctx, others, beforeID)
}
