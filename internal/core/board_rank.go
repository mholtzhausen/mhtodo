package core

import (
	"context"
	"fmt"
)

const minBoardRankGap = 1e-6

// appendBoardRank returns a rank that places a task at the end of a status column.
func (s *Service) appendBoardRank(ctx context.Context, st Status) (float64, error) {
	max, ok, err := s.repo.MaxBoardRank(ctx, st)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 1.0, nil
	}
	return max + 1.0, nil
}

func (s *Service) assignEndBoardRank(ctx context.Context, t *Task) error {
	if t.ParentID != nil {
		return nil
	}
	rank, err := s.appendBoardRank(ctx, t.Status)
	if err != nil {
		return err
	}
	t.BoardRank = &rank
	return nil
}

func (s *Service) renumberBoardColumn(ctx context.Context, st Status) error {
	column, err := s.repo.ListRootsInStatus(ctx, st)
	if err != nil {
		return err
	}
	for i, t := range column {
		rank := float64(i + 1)
		if err := s.repo.UpdateBoardRank(ctx, t.ID, rank); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) computeBoardRank(ctx context.Context, t Task, beforeID *string) (float64, error) {
	column, err := s.repo.ListRootsInStatus(ctx, t.Status)
	if err != nil {
		return 0, err
	}
	var others []Task
	for _, row := range column {
		if row.ID != t.ID {
			others = append(others, row)
		}
	}

	if beforeID == nil {
		return s.appendBoardRank(ctx, t.Status)
	}

	beforeIdx := -1
	for i, row := range others {
		if row.ID == *beforeID {
			beforeIdx = i
			break
		}
	}
	if beforeIdx < 0 {
		return 0, fmt.Errorf("%w: %q", ErrNotFound, *beforeID)
	}

	if beforeIdx == 0 {
		before := others[0]
		if before.BoardRank == nil {
			if err := s.renumberBoardColumn(ctx, t.Status); err != nil {
				return 0, err
			}
			return s.computeBoardRank(ctx, t, beforeID)
		}
		return *before.BoardRank - 1.0, nil
	}

	prev := others[beforeIdx-1]
	next := others[beforeIdx]
	if prev.BoardRank == nil || next.BoardRank == nil {
		if err := s.renumberBoardColumn(ctx, t.Status); err != nil {
			return 0, err
		}
		return s.computeBoardRank(ctx, t, beforeID)
	}
	gap := *next.BoardRank - *prev.BoardRank
	if gap < minBoardRankGap {
		if err := s.renumberBoardColumn(ctx, t.Status); err != nil {
			return 0, err
		}
		return s.computeBoardRank(ctx, t, beforeID)
	}
	return (*prev.BoardRank + *next.BoardRank) / 2, nil
}

// reorderWouldChange reports whether moving t before beforeID changes visual order.
func reorderWouldChange(t Task, beforeID *string, column []Task) bool {
	if beforeID == nil {
		if len(column) == 0 || column[len(column)-1].ID == t.ID {
			return false
		}
		return true
	}
	for i, row := range column {
		if row.ID != *beforeID {
			continue
		}
		if i == 0 {
			return row.ID != t.ID
		}
		return column[i-1].ID != t.ID
	}
	return true
}
