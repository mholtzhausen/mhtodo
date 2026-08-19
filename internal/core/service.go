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
	Delete(ctx context.Context, id string) (Task, error)
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
// done effects (progress 100, completed_at set).
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

	now := s.now()
	id, err := uuid.NewV7() // time-ordered, lowercase — ORDER BY id ≈ creation order
	if err != nil {
		return Task{}, fmt.Errorf("generate id: %w", err)
	}
	t := Task{
		ID:          id.String(),
		Title:       title,
		Description: in.Description,
		Status:      st,
		Progress:    in.Progress,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if st == StatusDone {
		t.Progress = 100
		t.CompletedAt = &now
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// List returns tasks per filter (defaults: exclude done, updated_at desc).
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

// Edit applies the set fields (title/description/progress). Status is not
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
	if in.Progress != nil {
		t.Progress = *in.Progress
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
	if err := s.repo.Update(ctx, t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// Delete removes a task and returns it (callers print the ID).
func (s *Service) Delete(ctx context.Context, ref string) (Task, error) {
	id, err := s.ResolveID(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	return s.repo.Delete(ctx, id)
}
