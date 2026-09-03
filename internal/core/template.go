package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Template is a named set of task presets (v0.5). Every preset field is a
// pointer: nil means "not part of this template", so task creation falls back
// to the normal default. A non-nil pointer to the zero value is an explicit
// override (e.g. IncludeInReport=false, or Cwd="" to force no working dir).
//
// JSON field names are a stable contract — do not rename.
type Template struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	TitlePrefix     *string   `json:"title_prefix"`
	Description     *string   `json:"description"`
	Status          *Status   `json:"status"`
	Cwd             *string   `json:"cwd"`
	SlackThread     *string   `json:"slack_thread"`
	HumanOnly       *bool     `json:"human_only"`
	IncludeInReport *bool     `json:"include_in_report"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TemplateInput carries the fields accepted by CreateTemplate / UpdateTemplate.
// Update semantics are full replace, not patch: a nil field clears the preset.
// The settings editor always submits the complete template, so this keeps the
// "trash-can clears a field" behaviour a plain nil rather than a double pointer.
type TemplateInput struct {
	Name            string
	TitlePrefix     *string
	Description     *string
	Status          *Status
	Cwd             *string
	SlackThread     *string
	HumanOnly       *bool
	IncludeInReport *bool
}

// Apply overlays the template's set fields onto a CreateInput, leaving fields
// the template does not define untouched so caller defaults survive. Title is
// prefixed rather than replaced.
func (t Template) Apply(in CreateInput) CreateInput {
	if t.TitlePrefix != nil {
		in.Title = *t.TitlePrefix + in.Title
	}
	if t.Description != nil {
		in.Description = *t.Description
	}
	if t.Status != nil {
		in.Status = *t.Status
	}
	if t.Cwd != nil {
		in.Cwd = *t.Cwd
	}
	if t.SlackThread != nil {
		in.SlackThread = *t.SlackThread
	}
	if t.HumanOnly != nil {
		in.HumanOnly = *t.HumanOnly
	}
	if t.IncludeInReport != nil {
		v := *t.IncludeInReport
		in.IncludeInReport = &v
	}
	return in
}

// ErrEmptyTemplateName is returned when a template name is blank after trim.
var ErrEmptyTemplateName = errors.New("template name must not be empty")

// ErrTemplateNotFound is returned when an ID or name matches no template.
var ErrTemplateNotFound = errors.New("template not found")

// DuplicateTemplateNameError is returned when a name collides with an existing
// template. Names are compared case-insensitively.
type DuplicateTemplateNameError struct{ Name string }

func (e *DuplicateTemplateNameError) Error() string {
	return fmt.Sprintf("a template named %q already exists", e.Name)
}

// MaxTemplateNameLen bounds the nav label so the settings sidebar stays usable.
const MaxTemplateNameLen = 80

// TemplateNameTooLongError is returned for names beyond MaxTemplateNameLen.
type TemplateNameTooLongError struct{ Len int }

func (e *TemplateNameTooLongError) Error() string {
	return fmt.Sprintf("template name is %d characters, limit is %d", e.Len, MaxTemplateNameLen)
}

// normalize trims the name and every string preset, and validates the status.
// Pointers stay nil (unset) — only their contents are cleaned.
func (in *TemplateInput) normalize() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return ErrEmptyTemplateName
	}
	if n := len([]rune(in.Name)); n > MaxTemplateNameLen {
		return &TemplateNameTooLongError{Len: n}
	}
	if in.Status != nil {
		st, err := ParseStatus(string(*in.Status))
		if err != nil {
			return err
		}
		in.Status = &st
	}
	// Paths and URLs are trimmed; title prefix and description are not, because
	// a prefix's trailing space ("BUG: ") is meaningful. Trimming writes to a
	// fresh pointer so the caller's string is never mutated in place.
	in.Cwd = trimmed(in.Cwd)
	in.SlackThread = trimmed(in.SlackThread)
	return nil
}

func trimmed(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}

// --- service methods ---------------------------------------------------------

// ListTemplates returns every template ordered by name.
func (s *Service) ListTemplates(ctx context.Context) ([]Template, error) {
	return s.repo.ListTemplates(ctx)
}

// GetTemplate resolves a template by exact ID, then by name
// (case-insensitive). Templates are few and user-named, so there is no prefix
// matching here — unlike tasks, the name is the natural handle.
func (s *Service) GetTemplate(ctx context.Context, ref string) (Template, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Template{}, ErrTemplateNotFound
	}
	t, err := s.repo.GetTemplateByID(ctx, ref)
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, ErrTemplateNotFound) {
		return Template{}, err
	}
	return s.repo.GetTemplateByName(ctx, ref)
}

// CreateTemplate validates and stores a new template.
func (s *Service) CreateTemplate(ctx context.Context, in TemplateInput) (Template, error) {
	if err := in.normalize(); err != nil {
		return Template{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Template{}, fmt.Errorf("generate id: %w", err)
	}
	now := s.now()
	t := templateFrom(id.String(), in)
	t.CreatedAt = now
	t.UpdatedAt = now
	if err := s.repo.CreateTemplate(ctx, t); err != nil {
		return Template{}, err
	}
	return t, nil
}

// UpdateTemplate replaces every preset field on the template identified by ref.
// This is a full replace, not a patch: fields left nil in the input are cleared.
func (s *Service) UpdateTemplate(ctx context.Context, ref string, in TemplateInput) (Template, error) {
	if err := in.normalize(); err != nil {
		return Template{}, err
	}
	existing, err := s.GetTemplate(ctx, ref)
	if err != nil {
		return Template{}, err
	}
	t := templateFrom(existing.ID, in)
	t.CreatedAt = existing.CreatedAt
	t.UpdatedAt = s.now()
	if err := s.repo.UpdateTemplate(ctx, t); err != nil {
		return Template{}, err
	}
	return t, nil
}

// DeleteTemplate removes a template and returns it.
func (s *Service) DeleteTemplate(ctx context.Context, ref string) (Template, error) {
	existing, err := s.GetTemplate(ctx, ref)
	if err != nil {
		return Template{}, err
	}
	return s.repo.DeleteTemplate(ctx, existing.ID)
}

// CreateFromTemplate applies a template to in and creates the resulting task.
// Fields already set on in win over the template only for Title, which is
// prefixed rather than replaced.
func (s *Service) CreateFromTemplate(ctx context.Context, ref string, in CreateInput) (Task, error) {
	tpl, err := s.GetTemplate(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	return s.Create(ctx, tpl.Apply(in))
}

// templateFrom builds a Template from validated input. Pointers are copied so
// the stored value cannot be mutated through the caller's input.
func templateFrom(id string, in TemplateInput) Template {
	return Template{
		ID:              id,
		Name:            in.Name,
		TitlePrefix:     clonePtr(in.TitlePrefix),
		Description:     clonePtr(in.Description),
		Status:          clonePtr(in.Status),
		Cwd:             clonePtr(in.Cwd),
		SlackThread:     clonePtr(in.SlackThread),
		HumanOnly:       clonePtr(in.HumanOnly),
		IncludeInReport: clonePtr(in.IncludeInReport),
	}
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
