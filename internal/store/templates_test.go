package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"mhtodo/internal/core"
)

func templateFixture(id, name string) core.Template {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	return core.Template{ID: id, Name: name, CreatedAt: now, UpdatedAt: now}
}

func TestTemplateCRUD(t *testing.T) {
	repo := openTestRepo(t)
	ctx := context.Background()

	prefix := "BUG: "
	cwd := "/tmp/proj"
	humanOnly := true
	includeInReport := false
	status := core.StatusWIP

	tpl := templateFixture("t1", "Event Logger")
	tpl.TitlePrefix = &prefix
	tpl.Cwd = &cwd
	tpl.HumanOnly = &humanOnly
	tpl.IncludeInReport = &includeInReport
	tpl.Status = &status

	if err := repo.CreateTemplate(ctx, tpl); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	got, err := repo.GetTemplateByID(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTemplateByID: %v", err)
	}
	if got.Name != "Event Logger" {
		t.Fatalf("name = %q, want Event Logger", got.Name)
	}
	if got.TitlePrefix == nil || *got.TitlePrefix != prefix {
		t.Fatalf("title_prefix = %v, want %q", got.TitlePrefix, prefix)
	}
	if got.Status == nil || *got.Status != core.StatusWIP {
		t.Fatalf("status = %v, want wip", got.Status)
	}
	if got.HumanOnly == nil || !*got.HumanOnly {
		t.Fatalf("human_only = %v, want true", got.HumanOnly)
	}
	// An explicit false must round-trip as set-and-false, not as unset.
	if got.IncludeInReport == nil || *got.IncludeInReport {
		t.Fatalf("include_in_report = %v, want pointer to false", got.IncludeInReport)
	}
	// Fields never set must stay nil so task creation falls back to defaults.
	if got.Description != nil || got.SlackThread != nil {
		t.Fatalf("unset fields materialized: desc=%v slack=%v", got.Description, got.SlackThread)
	}

	// Name lookup is case-insensitive (NOCASE column).
	if _, err := repo.GetTemplateByName(ctx, "event logger"); err != nil {
		t.Fatalf("GetTemplateByName(case-insensitive): %v", err)
	}

	// Update is a full replace: clearing a pointer clears the column.
	upd := got
	upd.TitlePrefix = nil
	upd.HumanOnly = nil
	upd.Name = "Event Logger v2"
	upd.UpdatedAt = got.UpdatedAt.Add(time.Minute)
	if err := repo.UpdateTemplate(ctx, upd); err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	got, err = repo.GetTemplateByID(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTemplateByID after update: %v", err)
	}
	if got.TitlePrefix != nil || got.HumanOnly != nil {
		t.Fatalf("cleared fields survived: prefix=%v human=%v", got.TitlePrefix, got.HumanOnly)
	}
	if got.Cwd == nil || *got.Cwd != cwd {
		t.Fatalf("cwd = %v, want %q (untouched by update)", got.Cwd, cwd)
	}
	if !got.CreatedAt.Equal(upd.CreatedAt) {
		t.Fatalf("created_at drifted: %v", got.CreatedAt)
	}

	deleted, err := repo.DeleteTemplate(ctx, "t1")
	if err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	if deleted.Name != "Event Logger v2" {
		t.Fatalf("deleted.Name = %q", deleted.Name)
	}
	if _, err := repo.GetTemplateByID(ctx, "t1"); !errors.Is(err, core.ErrTemplateNotFound) {
		t.Fatalf("get after delete = %v, want ErrTemplateNotFound", err)
	}
}

func TestTemplateDuplicateName(t *testing.T) {
	repo := openTestRepo(t)
	ctx := context.Background()

	if err := repo.CreateTemplate(ctx, templateFixture("t1", "Deploy")); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	// Case-insensitive collision must surface as the typed domain error, not a
	// raw driver error.
	err := repo.CreateTemplate(ctx, templateFixture("t2", "deploy"))
	var dup *core.DuplicateTemplateNameError
	if !errors.As(err, &dup) {
		t.Fatalf("CreateTemplate(duplicate) = %v, want DuplicateTemplateNameError", err)
	}

	if err := repo.CreateTemplate(ctx, templateFixture("t2", "Other")); err != nil {
		t.Fatalf("CreateTemplate(distinct): %v", err)
	}
	renamed := templateFixture("t2", "DEPLOY")
	if err := repo.UpdateTemplate(ctx, renamed); !errors.As(err, &dup) {
		t.Fatalf("UpdateTemplate(duplicate) = %v, want DuplicateTemplateNameError", err)
	}
}

func TestTemplateStatusCheckConstraint(t *testing.T) {
	repo := openTestRepo(t)
	ctx := context.Background()

	bogus := core.Status("archived")
	tpl := templateFixture("t1", "Bad")
	tpl.Status = &bogus
	if err := repo.CreateTemplate(ctx, tpl); err == nil {
		t.Fatal("CreateTemplate with invalid status succeeded, want CHECK violation")
	}
}

func TestListTemplatesOrdersByName(t *testing.T) {
	repo := openTestRepo(t)
	ctx := context.Background()

	for id, name := range map[string]string{"t1": "zeta", "t2": "Alpha", "t3": "middle"} {
		if err := repo.CreateTemplate(ctx, templateFixture(id, name)); err != nil {
			t.Fatalf("CreateTemplate(%s): %v", name, err)
		}
	}
	got, err := repo.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	want := []string{"Alpha", "middle", "zeta"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Fatalf("templates[%d] = %q, want %q", i, got[i].Name, name)
		}
	}
}
