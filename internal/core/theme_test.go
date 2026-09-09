package core_test

import (
	"context"
	"errors"
	"testing"

	"mhtodo/internal/core"
)

func TestThemesSeededOnOpen(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	list, err := svc.ListThemes(ctx)
	if err != nil {
		t.Fatalf("ListThemes: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("len = %d, want 3 built-ins", len(list))
	}
	active, err := svc.GetActiveTheme(ctx)
	if err != nil {
		t.Fatalf("GetActiveTheme: %v", err)
	}
	if active.ID != core.BuiltinSlateID || !active.Active {
		t.Fatalf("active = %+v, want Slate", active)
	}
	if active.Tokens["color.canvas"] != "#252b37" {
		t.Fatalf("slate canvas = %q", active.Tokens["color.canvas"])
	}
}

func TestCreateUpdateDeleteTheme(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	tokens := core.SlateFactoryTokens()
	tokens["color.accent"] = "#ff0000"
	created, err := svc.CreateTheme(ctx, core.ThemeInput{Name: "  Custom  ", Tokens: tokens})
	if err != nil {
		t.Fatalf("CreateTheme: %v", err)
	}
	if created.Name != "Custom" || created.BuiltinKey != nil {
		t.Fatalf("created = %+v", created)
	}
	if created.Tokens["color.accent"] != "#ff0000" {
		t.Fatalf("accent = %q", created.Tokens["color.accent"])
	}

	var dup *core.DuplicateThemeNameError
	if _, err := svc.CreateTheme(ctx, core.ThemeInput{Name: "custom", Tokens: tokens}); !errors.As(err, &dup) {
		t.Fatalf("dup = %v", err)
	}

	tokens["color.accent"] = "#00ff00"
	updated, err := svc.UpdateTheme(ctx, created.ID, core.ThemeInput{Name: "Custom", Tokens: tokens})
	if err != nil {
		t.Fatalf("UpdateTheme: %v", err)
	}
	if updated.Tokens["color.accent"] != "#00ff00" {
		t.Fatalf("accent after update = %q", updated.Tokens["color.accent"])
	}

	if _, err := svc.DeleteTheme(ctx, "Slate"); !errors.Is(err, core.ErrBuiltinTheme) {
		t.Fatalf("delete builtin = %v, want ErrBuiltinTheme", err)
	}

	deleted, err := svc.DeleteTheme(ctx, "Custom")
	if err != nil || deleted.Name != "Custom" {
		t.Fatalf("DeleteTheme = %+v, %v", deleted, err)
	}
}

func TestActivateDuplicateReset(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	paper, err := svc.ActivateTheme(ctx, "Paper")
	if err != nil || !paper.Active || paper.ID != core.BuiltinPaperID {
		t.Fatalf("ActivateTheme Paper = %+v, %v", paper, err)
	}

	copyTheme, err := svc.DuplicateTheme(ctx, "Paper", "")
	if err != nil {
		t.Fatalf("DuplicateTheme: %v", err)
	}
	if copyTheme.Name != "Paper copy" || copyTheme.BuiltinKey != nil {
		t.Fatalf("copy = %+v", copyTheme)
	}
	if copyTheme.Tokens["color.canvas"] != paper.Tokens["color.canvas"] {
		t.Fatalf("tokens not copied")
	}

	// Mutate Paper then reset.
	mut := core.SlateFactoryTokens() // wrong palette on purpose
	mut["color.canvas"] = "#111111"
	if _, err := svc.UpdateTheme(ctx, "Paper", core.ThemeInput{Name: "Paper", Tokens: mut}); err != nil {
		t.Fatalf("mutate Paper: %v", err)
	}
	reset, err := svc.ResetTheme(ctx, "Paper")
	if err != nil {
		t.Fatalf("ResetTheme: %v", err)
	}
	factory, _ := core.FactoryTokens(core.BuiltinPaper)
	if reset.Tokens["color.canvas"] != factory["color.canvas"] {
		t.Fatalf("reset canvas = %q, want %q", reset.Tokens["color.canvas"], factory["color.canvas"])
	}

	if _, err := svc.ResetTheme(ctx, copyTheme.ID); !errors.Is(err, core.ErrNotBuiltinTheme) {
		t.Fatalf("reset custom = %v", err)
	}
}

func TestDeleteActiveSwitchesToSlate(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	custom, err := svc.CreateTheme(ctx, core.ThemeInput{Name: "Temp", Tokens: core.SlateFactoryTokens()})
	if err != nil {
		t.Fatalf("CreateTheme: %v", err)
	}
	if _, err := svc.ActivateTheme(ctx, custom.ID); err != nil {
		t.Fatalf("ActivateTheme: %v", err)
	}
	if _, err := svc.DeleteTheme(ctx, custom.ID); err != nil {
		t.Fatalf("DeleteTheme: %v", err)
	}
	active, err := svc.GetActiveTheme(ctx)
	if err != nil || active.ID != core.BuiltinSlateID {
		t.Fatalf("active after delete = %+v, %v", active, err)
	}
}

func TestThemeTokenValidation(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	bad := core.SlateFactoryTokens()
	bad["color.nope"] = "#fff"
	var unk *core.UnknownThemeTokenError
	if _, err := svc.CreateTheme(ctx, core.ThemeInput{Name: "Bad", Tokens: bad}); !errors.As(err, &unk) {
		t.Fatalf("unknown key = %v", err)
	}

	bad = core.SlateFactoryTokens()
	bad["color.accent"] = "not-a-color"
	var inv *core.InvalidThemeTokenValueError
	if _, err := svc.CreateTheme(ctx, core.ThemeInput{Name: "Bad2", Tokens: bad}); !errors.As(err, &inv) {
		t.Fatalf("bad color = %v", err)
	}
}

func TestSearchThemes(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	list, err := svc.SearchThemes(ctx, core.ThemeSearchFilter{Query: "slt"})
	if err != nil || len(list) != 1 || list[0].Name != "Slate" {
		t.Fatalf("fuzzy = %+v, %v", list, err)
	}
	list, err = svc.SearchThemes(ctx, core.ThemeSearchFilter{Query: "^Ember$", Mode: core.ThemeSearchRegex})
	if err != nil || len(list) != 1 || list[0].Name != "Ember" {
		t.Fatalf("regex = %+v, %v", list, err)
	}
	if _, err := svc.SearchThemes(ctx, core.ThemeSearchFilter{}); !errors.Is(err, core.ErrThemeSearchEmpty) {
		t.Fatalf("empty = %v", err)
	}
}

func TestTokenToCSSVar(t *testing.T) {
	cases := map[string]string{
		"color.canvas":    "--color-canvas",
		"color.cardHi":    "--color-card-hi",
		"color.ink2":      "--color-ink-2",
		"color.stPending": "--color-st-pending",
		"radius.card":     "--radius-card",
		"space.gapMd":     "--spacing-gap-md",
		"space.padLg":     "--spacing-pad-lg",
	}
	for k, want := range cases {
		if got := core.TokenToCSSVar(k); got != want {
			t.Errorf("TokenToCSSVar(%q) = %q, want %q", k, got, want)
		}
	}
}
