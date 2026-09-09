package core

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
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

// TemplateSearchMode selects how TemplateSearchFilter.Query is matched.
type TemplateSearchMode string

const (
	TemplateSearchFuzzy TemplateSearchMode = "fuzzy"
	TemplateSearchRegex TemplateSearchMode = "regex"
)

// TemplateSearchFilter filters ListTemplates results in memory. At least one of
// Query or Cwd must be set. Text search covers name, title_prefix, description,
// and cwd. Cwd is an exact path match after filepath.Clean (templates with a
// nil/empty cwd never match a --cwd filter).
type TemplateSearchFilter struct {
	Query string
	Mode  TemplateSearchMode // default fuzzy when Query is set
	Cwd   string
}

// InvalidTemplateSearchModeError is returned for unknown --mode values.
type InvalidTemplateSearchModeError struct{ Mode string }

func (e *InvalidTemplateSearchModeError) Error() string {
	return fmt.Sprintf("invalid template search mode %q (want fuzzy|regex)", e.Mode)
}

// InvalidTemplateSearchPatternError wraps a bad regex pattern.
type InvalidTemplateSearchPatternError struct{ Err error }

func (e *InvalidTemplateSearchPatternError) Error() string {
	return fmt.Sprintf("invalid template search pattern: %v", e.Err)
}

func (e *InvalidTemplateSearchPatternError) Unwrap() error { return e.Err }

// ErrTemplateSearchEmpty is returned when search has neither a query nor a cwd.
var ErrTemplateSearchEmpty = errors.New("template search requires a query and/or --cwd")

// SearchTemplates returns templates matching f, ordered by relevance (fuzzy
// score desc, then name) or by name for regex / cwd-only searches.
func (s *Service) SearchTemplates(ctx context.Context, f TemplateSearchFilter) ([]Template, error) {
	query := strings.TrimSpace(f.Query)
	cwd := strings.TrimSpace(f.Cwd)
	if query == "" && cwd == "" {
		return nil, ErrTemplateSearchEmpty
	}

	mode := f.Mode
	if mode == "" {
		mode = TemplateSearchFuzzy
	}
	switch mode {
	case TemplateSearchFuzzy, TemplateSearchRegex:
	default:
		return nil, &InvalidTemplateSearchModeError{Mode: string(mode)}
	}

	var re *regexp.Regexp
	if query != "" && mode == TemplateSearchRegex {
		compiled, err := regexp.Compile("(?i)" + query)
		if err != nil {
			return nil, &InvalidTemplateSearchPatternError{Err: err}
		}
		re = compiled
	}

	all, err := s.repo.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}

	type scored struct {
		t     Template
		score int
	}
	matched := make([]scored, 0, len(all))
	wantCwd := ""
	if cwd != "" {
		wantCwd = filepath.Clean(cwd)
	}

	for _, t := range all {
		if wantCwd != "" {
			if t.Cwd == nil || strings.TrimSpace(*t.Cwd) == "" {
				continue
			}
			if filepath.Clean(*t.Cwd) != wantCwd {
				continue
			}
		}
		if query == "" {
			matched = append(matched, scored{t: t})
			continue
		}
		switch mode {
		case TemplateSearchFuzzy:
			best := -1
			for _, hay := range templateSearchHaystacks(t) {
				if sc := fuzzyScore(query, hay); sc > best {
					best = sc
				}
			}
			if best >= 0 {
				matched = append(matched, scored{t: t, score: best})
			}
		case TemplateSearchRegex:
			for _, hay := range templateSearchHaystacks(t) {
				if re.MatchString(hay) {
					matched = append(matched, scored{t: t})
					break
				}
			}
		}
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].score != matched[j].score {
			return matched[i].score > matched[j].score
		}
		return strings.ToLower(matched[i].t.Name) < strings.ToLower(matched[j].t.Name)
	})

	out := make([]Template, len(matched))
	for i, m := range matched {
		out[i] = m.t
	}
	return out, nil
}

func templateSearchHaystacks(t Template) []string {
	h := []string{t.Name}
	if t.TitlePrefix != nil {
		h = append(h, *t.TitlePrefix)
	}
	if t.Description != nil {
		h = append(h, *t.Description)
	}
	if t.Cwd != nil {
		h = append(h, *t.Cwd)
	}
	return h
}

// AsInput copies the template's presets into a TemplateInput suitable for a
// full-replace UpdateTemplate after CLI/GUI patching.
func (t Template) AsInput() TemplateInput {
	return TemplateInput{
		Name:            t.Name,
		TitlePrefix:     clonePtr(t.TitlePrefix),
		Description:     clonePtr(t.Description),
		Status:          clonePtr(t.Status),
		Cwd:             clonePtr(t.Cwd),
		SlackThread:     clonePtr(t.SlackThread),
		HumanOnly:       clonePtr(t.HumanOnly),
		IncludeInReport: clonePtr(t.IncludeInReport),
	}
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
