package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TaskRepository is the persistence surface core needs. Implemented by
// internal/store; core never touches SQL itself.
type TaskRepository interface {
	Create(ctx context.Context, t Task) error
	GetByID(ctx context.Context, id string) (Task, error) // ErrNotFound if missing
	FindByPrefix(ctx context.Context, prefix string) ([]Task, error)
	List(ctx context.Context, f ListFilter) ([]Task, error)
	Update(ctx context.Context, t Task) error // mutable fields by t.ID; ErrNotFound if missing
	UpdateBoardRank(ctx context.Context, id string, rank float64) error
	MaxBoardRank(ctx context.Context, status Status) (max float64, ok bool, err error)
	ListRootsInStatus(ctx context.Context, status Status) ([]Task, error)
	Delete(ctx context.Context, id string) (Task, error)
	// ArchiveDone archives every non-archived done task in one statement and
	// returns the archived tasks (v0.2). at is the service clock (tests inject it).
	ArchiveDone(ctx context.Context, at time.Time) ([]Task, error)
	CountOpen(ctx context.Context) (int, error) // non-done tasks; tray tooltip only
	CountChildren(ctx context.Context, parentID string) (int, error)

	CreateActivity(ctx context.Context, a Activity) error
	GetActivityByID(ctx context.Context, id string) (Activity, error)
	FindActivityByPrefix(ctx context.Context, prefix string) ([]Activity, error)
	ListActivity(ctx context.Context, f ActivityFilter) ([]Activity, error)
	DeleteActivity(ctx context.Context, id string) (Activity, error)
}

// Service holds all business rules shared by CLI and GUI. Neither frontend
// may contain its own rules or SQL — new capability = method here + both frontends.
type Service struct {
	repo TaskRepository
	now  func() time.Time // injectable for tests; defaults to UTC now
}

// The default clock truncates to whole seconds so returned objects match what
// the store persists (RFC3339 second precision) — keeps --json output stable.
func NewService(repo TaskRepository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now().UTC().Truncate(time.Second) }}
}

// SetNowFunc is test-only.
func (s *Service) SetNowFunc(f func() time.Time) { s.now = f }

// Create validates input and stores a new task. →done at creation applies the
// done effects (progress 100, completed_at set). ParentID (if set) must resolve
// to a top-level task (one-level nesting only).
func (s *Service) Create(ctx context.Context, in CreateInput) (Task, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return Task{}, ErrEmptyTitle
	}
	st := in.Status
	if st == "" {
		st = StatusPending
	} else if _, err := ParseStatus(string(st)); err != nil {
		return Task{}, err
	}
	if in.Progress < 0 || in.Progress > 100 {
		return Task{}, &ProgressRangeError{Value: in.Progress}
	}

	var parentID *string
	if ref := strings.TrimSpace(in.ParentID); ref != "" {
		parent, err := s.Get(ctx, ref)
		if err != nil {
			return Task{}, err
		}
		if parent.ParentID != nil {
			return Task{}, ErrParentIsChild
		}
		pid := parent.ID
		parentID = &pid
	}

	now := s.now()
	id, err := uuid.NewV7() // time-ordered, lowercase — ORDER BY id ≈ creation order
	if err != nil {
		return Task{}, fmt.Errorf("generate id: %w", err)
	}
	t := Task{
		ID:          id.String(),
		Title:       title,
		Description: in.Description,
		Feedback:    in.Feedback,
		Status:      st,
		Progress:    in.Progress,
		CreatedAt:   now,
		UpdatedAt:   now,
		ParentID:    parentID,
		Cwd:         strings.TrimSpace(in.Cwd),
		HumanOnly:   in.HumanOnly,
	}
	if st == StatusDone {
		t.Progress = 100
		t.CompletedAt = &now
	}
	if parentID == nil {
		if err := s.assignEndBoardRank(ctx, &t); err != nil {
			return Task{}, err
		}
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// List returns tasks per filter (defaults: exclude done, board order).
func (s *Service) List(ctx context.Context, f ListFilter) ([]Task, error) {
	if f.Status != "" {
		if _, err := ParseStatus(string(f.Status)); err != nil {
			return nil, err
		}
	}
	return s.repo.List(ctx, f)
}

// Get fetches a task by full ID or unique prefix (≥ MinPrefixLen chars).
func (s *Service) Get(ctx context.Context, ref string) (Task, error) {
	id, err := s.ResolveID(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	return s.repo.GetByID(ctx, id)
}

// ResolveID maps a full ID or unique prefix to the canonical ID.
func (s *Service) ResolveID(ctx context.Context, ref string) (string, error) {
	if _, err := s.repo.GetByID(ctx, ref); err == nil {
		return ref, nil // exact match wins
	} else if !errors.Is(err, ErrNotFound) {
		return "", err
	}
	if len(ref) < MinPrefixLen {
		return "", fmt.Errorf("%w: %q", ErrNotFound, ref)
	}
	matches, err := s.repo.FindByPrefix(ctx, ref)
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: %q", ErrNotFound, ref)
	case 1:
		return matches[0].ID, nil
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.ID
		}
		sort.Strings(ids)
		return "", &AmbiguousIDError{Ref: ref, Candidates: ids}
	}
}

// Edit applies the set fields (title/description/feedback/progress). Status is not
// changed here — use SetStatus. created_at is immutable; updated_at always bumps.
func (s *Service) Edit(ctx context.Context, ref string, in UpdateInput) (Task, error) {
	if !in.hasFields() {
		return Task{}, ErrNoFieldsToUpdate
	}
	if in.Title != nil && strings.TrimSpace(*in.Title) == "" {
		return Task{}, ErrEmptyTitle
	}
	if in.Progress != nil && (*in.Progress < 0 || *in.Progress > 100) {
		return Task{}, &ProgressRangeError{Value: *in.Progress}
	}

	t, err := s.Get(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	if in.Title != nil {
		t.Title = strings.TrimSpace(*in.Title)
	}
	if in.Desc != nil {
		t.Description = *in.Desc
	}
	if in.Feedback != nil {
		t.Feedback = *in.Feedback
	}
	if in.Progress != nil {
		t.Progress = *in.Progress
	}
	if in.Cwd != nil {
		t.Cwd = strings.TrimSpace(*in.Cwd)
	}
	if in.HumanOnly != nil {
		t.HumanOnly = *in.HumanOnly
	}
	t.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// SetStatus moves a task to a new status and applies the transition effects:
//
//	any → done      progress = 100, completed_at = now
//	done → other    completed_at cleared; progress left as-is unless it was 100 by the rule above
//	any change     updated_at = now always
func (s *Service) SetStatus(ctx context.Context, ref string, st Status) (Task, error) {
	if _, err := ParseStatus(string(st)); err != nil {
		return Task{}, err
	}
	t, err := s.Get(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	oldStatus := t.Status
	now := s.now()
	switch {
	case st == StatusDone && t.Status != StatusDone:
		t.Progress = 100
		t.CompletedAt = &now
	case t.Status == StatusDone && st != StatusDone:
		t.CompletedAt = nil // progress left as-is per spec
	}
	t.Status = st
	t.UpdatedAt = now
	if t.ParentID == nil && oldStatus != st {
		if err := s.assignEndBoardRank(ctx, &t); err != nil {
			return Task{}, err
		}
	}
	if err := s.repo.Update(ctx, t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// ReorderBoardTask moves a root task within its current status column.
// beforeRef nil appends to the end; otherwise inserts immediately before that root.
func (s *Service) ReorderBoardTask(ctx context.Context, ref string, beforeRef *string) (Task, error) {
	id, err := s.ResolveID(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if t.ParentID != nil {
		return Task{}, ErrNotRoot
	}
	if t.ArchivedAt != nil {
		return Task{}, fmt.Errorf("%w: %q", ErrNotFound, ref)
	}

	var beforeID *string
	if beforeRef != nil && strings.TrimSpace(*beforeRef) != "" {
		bid, err := s.ResolveID(ctx, strings.TrimSpace(*beforeRef))
		if err != nil {
			return Task{}, err
		}
		before, err := s.repo.GetByID(ctx, bid)
		if err != nil {
			return Task{}, err
		}
		if before.ParentID != nil {
			return Task{}, ErrNotRoot
		}
		if before.Status != t.Status {
			return Task{}, ErrReorderStatusMismatch
		}
		if before.ID == t.ID {
			return t, nil
		}
		beforeID = &before.ID
	}

	column, err := s.repo.ListRootsInStatus(ctx, t.Status)
	if err != nil {
		return Task{}, err
	}
	if !reorderWouldChange(t, beforeID, column) {
		return t, nil
	}

	rank, err := s.computeBoardRank(ctx, t, beforeID)
	if err != nil {
		return Task{}, err
	}
	if err := s.repo.UpdateBoardRank(ctx, t.ID, rank); err != nil {
		return Task{}, err
	}
	t.BoardRank = &rank
	return t, nil
}

// ArchiveDone archives all done tasks at once (board Done-column button / CLI
// `archive`). It is reversible via Unarchive; no notification fires (the tasks
// were already done). Returns the archived tasks, empty when there was nothing.
func (s *Service) ArchiveDone(ctx context.Context) ([]Task, error) {
	return s.repo.ArchiveDone(ctx, s.now())
}

// Unarchive restores an archived task: it goes back to pending with progress
// reset to 0 and completed_at cleared (the done→other transition rule), so the
// user starts fresh. Non-archived tasks → ErrNotArchived.
func (s *Service) Unarchive(ctx context.Context, ref string) (Task, error) {
	id, err := s.ResolveID(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if t.ArchivedAt == nil {
		return Task{}, ErrNotArchived
	}
	now := s.now()
	t.Status = StatusPending
	t.Progress = 0
	t.CompletedAt = nil // done → other rule
	t.ArchivedAt = nil
	t.UpdatedAt = now
	if t.ParentID == nil {
		if err := s.assignEndBoardRank(ctx, &t); err != nil {
			return Task{}, err
		}
	}
	if err := s.repo.Update(ctx, t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// CountChildren returns how many direct sub-tasks a task has (for delete confirm).
func (s *Service) CountChildren(ctx context.Context, ref string) (int, error) {
	id, err := s.ResolveID(ctx, ref)
	if err != nil {
		return 0, err
	}
	return s.repo.CountChildren(ctx, id)
}

// Delete removes a task (and cascades to children via FK) and returns the
// deleted parent task.
func (s *Service) Delete(ctx context.Context, ref string) (Task, error) {
	id, err := s.ResolveID(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	return s.repo.Delete(ctx, id)
}

// CountOpen returns the number of non-done tasks. Tray tooltip only — not part
// of the CLI parity surface (no business rules involved).
func (s *Service) CountOpen(ctx context.Context) (int, error) { return s.repo.CountOpen(ctx) }

// AddActivity records an agent/user-authored activity entry on a task.
func (s *Service) AddActivity(ctx context.Context, taskRef string, in ActivityInput) (Activity, error) {
	act := strings.TrimSpace(in.Activity)
	cmt := strings.TrimSpace(in.Comment)
	if act == "" && cmt == "" {
		return Activity{}, ErrEmptyActivity
	}
	taskID, err := s.ResolveID(ctx, taskRef)
	if err != nil {
		return Activity{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Activity{}, fmt.Errorf("generate id: %w", err)
	}
	a := Activity{
		ID:        id.String(),
		TaskID:    taskID,
		Activity:  act,
		Comment:   cmt,
		CreatedAt: s.now(),
	}
	if err := s.repo.CreateActivity(ctx, a); err != nil {
		return Activity{}, err
	}
	return a, nil
}

// ListActivity returns activity entries newest-first.
func (s *Service) ListActivity(ctx context.Context, f ActivityFilter) ([]Activity, error) {
	resolved := make([]string, 0, len(f.TaskIDs))
	for _, ref := range f.TaskIDs {
		id, err := s.ResolveID(ctx, ref)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, id)
	}
	f.TaskIDs = resolved
	return s.repo.ListActivity(ctx, f)
}

// ResolveActivityID maps a full ID or unique prefix to an activity ID.
func (s *Service) ResolveActivityID(ctx context.Context, ref string) (string, error) {
	if _, err := s.repo.GetActivityByID(ctx, ref); err == nil {
		return ref, nil
	} else if !errors.Is(err, ErrNotFound) {
		return "", err
	}
	if len(ref) < MinPrefixLen {
		return "", fmt.Errorf("%w: %q", ErrNotFound, ref)
	}
	matches, err := s.repo.FindActivityByPrefix(ctx, ref)
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: %q", ErrNotFound, ref)
	case 1:
		return matches[0].ID, nil
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.ID
		}
		sort.Strings(ids)
		return "", &AmbiguousIDError{Ref: ref, Candidates: ids}
	}
}

// DeleteActivity removes an activity entry by ID or unique prefix.
func (s *Service) DeleteActivity(ctx context.Context, ref string) (Activity, error) {
	id, err := s.ResolveActivityID(ctx, ref)
	if err != nil {
		return Activity{}, err
	}
	return s.repo.DeleteActivity(ctx, id)
}
