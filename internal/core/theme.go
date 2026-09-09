package core

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Meta key for the active theme id (stored via TaskRepository GetMeta/SetMeta).
const MetaActiveThemeID = "active_theme_id"

// Theme is a named set of design tokens (v0.6). Tokens is a complete map of
// registry keys after read (missing keys merged from Slate). Active is computed
// from meta and not persisted on the themes row.
//
// JSON field names are a stable contract — do not rename.
type Theme struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	BuiltinKey *string           `json:"builtin_key"`
	Tokens     map[string]string `json:"tokens"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Active     bool              `json:"active"`
}

// ThemeInput carries fields for CreateTheme / UpdateTheme. Update is a full
// replace of name + tokens (not a patch).
type ThemeInput struct {
	Name   string
	Tokens map[string]string
}

// ErrEmptyThemeName is returned when a theme name is blank after trim.
var ErrEmptyThemeName = errors.New("theme name must not be empty")

// ErrThemeNotFound is returned when an ID or name matches no theme.
var ErrThemeNotFound = errors.New("theme not found")

// ErrBuiltinTheme is returned when deleting a built-in theme.
var ErrBuiltinTheme = errors.New("built-in themes cannot be deleted")

// ErrNotBuiltinTheme is returned when Reset is called on a non-built-in.
var ErrNotBuiltinTheme = errors.New("only built-in themes can be reset")

// ErrThemeSearchEmpty is returned when search has no query.
var ErrThemeSearchEmpty = errors.New("theme search requires a query")

// DuplicateThemeNameError is returned when a name collides (case-insensitive).
type DuplicateThemeNameError struct{ Name string }

func (e *DuplicateThemeNameError) Error() string {
	return fmt.Sprintf("a theme named %q already exists", e.Name)
}

// MaxThemeNameLen bounds the settings sidebar label.
const MaxThemeNameLen = 80

// ThemeNameTooLongError is returned for names beyond MaxThemeNameLen.
type ThemeNameTooLongError struct{ Len int }

func (e *ThemeNameTooLongError) Error() string {
	return fmt.Sprintf("theme name is %d characters, limit is %d", e.Len, MaxThemeNameLen)
}

// InvalidThemeSearchModeError is returned for unknown --mode values.
type InvalidThemeSearchModeError struct{ Mode string }

func (e *InvalidThemeSearchModeError) Error() string {
	return fmt.Sprintf("invalid theme search mode %q (want fuzzy|regex)", e.Mode)
}

// InvalidThemeSearchPatternError wraps a bad regex pattern.
type InvalidThemeSearchPatternError struct{ Err error }

func (e *InvalidThemeSearchPatternError) Error() string {
	return fmt.Sprintf("invalid theme search pattern: %v", e.Err)
}

func (e *InvalidThemeSearchPatternError) Unwrap() error { return e.Err }

func (in *ThemeInput) normalize() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return ErrEmptyThemeName
	}
	if n := utf8.RuneCountInString(in.Name); n > MaxThemeNameLen {
		return &ThemeNameTooLongError{Len: n}
	}
	normalized, err := NormalizeThemeTokens(in.Tokens)
	if err != nil {
		return err
	}
	in.Tokens = normalized
	return nil
}

// --- service methods ---------------------------------------------------------

// ListThemes returns every theme ordered by name, with Active set from meta.
func (s *Service) ListThemes(ctx context.Context) ([]Theme, error) {
	list, err := s.repo.ListThemes(ctx)
	if err != nil {
		return nil, err
	}
	activeID, err := s.activeThemeID(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Tokens = MergeThemeTokens(list[i].Tokens)
		list[i].Active = list[i].ID == activeID
	}
	return list, nil
}

// GetTheme resolves by exact ID, then by name (case-insensitive).
func (s *Service) GetTheme(ctx context.Context, ref string) (Theme, error) {
	t, err := s.resolveTheme(ctx, ref)
	if err != nil {
		return Theme{}, err
	}
	t.Tokens = MergeThemeTokens(t.Tokens)
	activeID, err := s.activeThemeID(ctx)
	if err != nil {
		return Theme{}, err
	}
	t.Active = t.ID == activeID
	return t, nil
}

// GetActiveTheme returns the currently active theme (falls back to Slate).
func (s *Service) GetActiveTheme(ctx context.Context) (Theme, error) {
	id, err := s.activeThemeID(ctx)
	if err != nil {
		return Theme{}, err
	}
	t, err := s.repo.GetThemeByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrThemeNotFound) {
			// Meta points at a deleted row — heal to Slate.
			if err := s.repo.SetMeta(ctx, MetaActiveThemeID, BuiltinSlateID); err != nil {
				return Theme{}, err
			}
			t, err = s.repo.GetThemeByID(ctx, BuiltinSlateID)
			if err != nil {
				return Theme{}, err
			}
		} else {
			return Theme{}, err
		}
	}
	t.Tokens = MergeThemeTokens(t.Tokens)
	t.Active = true
	return t, nil
}

// CreateTheme validates and stores a new user theme (no builtin_key).
func (s *Service) CreateTheme(ctx context.Context, in ThemeInput) (Theme, error) {
	if in.Tokens == nil {
		in.Tokens = SlateFactoryTokens()
	}
	if err := in.normalize(); err != nil {
		return Theme{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Theme{}, fmt.Errorf("generate id: %w", err)
	}
	now := s.now()
	t := Theme{
		ID:        id.String(),
		Name:      in.Name,
		Tokens:    cloneStringMap(in.Tokens),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateTheme(ctx, t); err != nil {
		return Theme{}, err
	}
	return s.GetTheme(ctx, t.ID)
}

// UpdateTheme replaces name + tokens on the theme identified by ref.
func (s *Service) UpdateTheme(ctx context.Context, ref string, in ThemeInput) (Theme, error) {
	if err := in.normalize(); err != nil {
		return Theme{}, err
	}
	existing, err := s.resolveTheme(ctx, ref)
	if err != nil {
		return Theme{}, err
	}
	t := Theme{
		ID:         existing.ID,
		Name:       in.Name,
		BuiltinKey: existing.BuiltinKey,
		Tokens:     cloneStringMap(in.Tokens),
		CreatedAt:  existing.CreatedAt,
		UpdatedAt:  s.now(),
	}
	if err := s.repo.UpdateTheme(ctx, t); err != nil {
		return Theme{}, err
	}
	return s.GetTheme(ctx, t.ID)
}

// DeleteTheme removes a user theme. Built-ins are refused. If the deleted theme
// was active, Slate becomes active.
func (s *Service) DeleteTheme(ctx context.Context, ref string) (Theme, error) {
	existing, err := s.GetTheme(ctx, ref)
	if err != nil {
		return Theme{}, err
	}
	if existing.BuiltinKey != nil {
		return Theme{}, ErrBuiltinTheme
	}
	deleted, err := s.repo.DeleteTheme(ctx, existing.ID)
	if err != nil {
		return Theme{}, err
	}
	deleted.Tokens = MergeThemeTokens(deleted.Tokens)
	if existing.Active {
		if err := s.repo.SetMeta(ctx, MetaActiveThemeID, BuiltinSlateID); err != nil {
			return Theme{}, err
		}
	}
	deleted.Active = false
	return deleted, nil
}

// DuplicateTheme copies tokens into a new user theme. optionalName empty → "X copy".
func (s *Service) DuplicateTheme(ctx context.Context, ref, optionalName string) (Theme, error) {
	src, err := s.GetTheme(ctx, ref)
	if err != nil {
		return Theme{}, err
	}
	name := strings.TrimSpace(optionalName)
	if name == "" {
		name = uniqueCopyName(src.Name, func(candidate string) bool {
			_, err := s.repo.GetThemeByName(ctx, candidate)
			return errors.Is(err, ErrThemeNotFound)
		})
	}
	return s.CreateTheme(ctx, ThemeInput{Name: name, Tokens: src.Tokens})
}

// ResetTheme restores factory tokens for a built-in theme.
func (s *Service) ResetTheme(ctx context.Context, ref string) (Theme, error) {
	existing, err := s.resolveTheme(ctx, ref)
	if err != nil {
		return Theme{}, err
	}
	if existing.BuiltinKey == nil {
		return Theme{}, ErrNotBuiltinTheme
	}
	factory, ok := FactoryTokens(*existing.BuiltinKey)
	if !ok {
		return Theme{}, ErrNotBuiltinTheme
	}
	return s.UpdateTheme(ctx, existing.ID, ThemeInput{
		Name:   existing.Name,
		Tokens: factory,
	})
}

// ActivateTheme sets meta.active_theme_id to the resolved theme.
func (s *Service) ActivateTheme(ctx context.Context, ref string) (Theme, error) {
	t, err := s.resolveTheme(ctx, ref)
	if err != nil {
		return Theme{}, err
	}
	if err := s.repo.SetMeta(ctx, MetaActiveThemeID, t.ID); err != nil {
		return Theme{}, err
	}
	return s.GetTheme(ctx, t.ID)
}

// ThemeSearchMode selects how ThemeSearchFilter.Query is matched.
type ThemeSearchMode string

const (
	ThemeSearchFuzzy ThemeSearchMode = "fuzzy"
	ThemeSearchRegex ThemeSearchMode = "regex"
)

// ThemeSearchFilter filters ListThemes by name.
type ThemeSearchFilter struct {
	Query string
	Mode  ThemeSearchMode // default fuzzy
}

// SearchThemes returns themes matching f, ordered by relevance or name.
func (s *Service) SearchThemes(ctx context.Context, f ThemeSearchFilter) ([]Theme, error) {
	query := strings.TrimSpace(f.Query)
	if query == "" {
		return nil, ErrThemeSearchEmpty
	}
	mode := f.Mode
	if mode == "" {
		mode = ThemeSearchFuzzy
	}
	switch mode {
	case ThemeSearchFuzzy, ThemeSearchRegex:
	default:
		return nil, &InvalidThemeSearchModeError{Mode: string(mode)}
	}

	var re *regexp.Regexp
	if mode == ThemeSearchRegex {
		compiled, err := regexp.Compile("(?i)" + query)
		if err != nil {
			return nil, &InvalidThemeSearchPatternError{Err: err}
		}
		re = compiled
	}

	all, err := s.ListThemes(ctx)
	if err != nil {
		return nil, err
	}

	type scored struct {
		t     Theme
		score int
	}
	matched := make([]scored, 0, len(all))
	for _, t := range all {
		switch mode {
		case ThemeSearchFuzzy:
			if sc := fuzzyScore(query, t.Name); sc >= 0 {
				matched = append(matched, scored{t: t, score: sc})
			}
		case ThemeSearchRegex:
			if re.MatchString(t.Name) {
				matched = append(matched, scored{t: t})
			}
		}
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].score != matched[j].score {
			return matched[i].score > matched[j].score
		}
		return strings.ToLower(matched[i].t.Name) < strings.ToLower(matched[j].t.Name)
	})

	out := make([]Theme, len(matched))
	for i, m := range matched {
		out[i] = m.t
	}
	return out, nil
}

func (s *Service) resolveTheme(ctx context.Context, ref string) (Theme, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Theme{}, ErrThemeNotFound
	}
	t, err := s.repo.GetThemeByID(ctx, ref)
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, ErrThemeNotFound) {
		return Theme{}, err
	}
	return s.repo.GetThemeByName(ctx, ref)
}

func (s *Service) activeThemeID(ctx context.Context) (string, error) {
	v, ok, err := s.repo.GetMeta(ctx, MetaActiveThemeID)
	if err != nil {
		return "", err
	}
	if !ok || strings.TrimSpace(v) == "" {
		return BuiltinSlateID, nil
	}
	return strings.TrimSpace(v), nil
}

func cloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// uniqueCopyName returns base+" copy", base+" copy 2", … until available(true).
func uniqueCopyName(base string, available func(string) bool) string {
	candidate := base + " copy"
	if available(candidate) {
		return candidate
	}
	for n := 2; ; n++ {
		candidate = fmt.Sprintf("%s copy %d", base, n)
		if available(candidate) {
			return candidate
		}
	}
}
