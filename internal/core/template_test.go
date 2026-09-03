package core_test

import (
	"context"
	"errors"
	"testing"

	"mhtodo/internal/core"
)

func ptr[T any](v T) *T { return &v }

func TestCreateTemplateValidation(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateTemplate(ctx, core.TemplateInput{Name: "   "}); !errors.Is(err, core.ErrEmptyTemplateName) {
		t.Fatalf("blank name = %v, want ErrEmptyTemplateName", err)
	}

	longName := make([]rune, core.MaxTemplateNameLen+1)
	for i := range longName {
		longName[i] = 'x'
	}
	var tooLong *core.TemplateNameTooLongError
	if _, err := svc.CreateTemplate(ctx, core.TemplateInput{Name: string(longName)}); !errors.As(err, &tooLong) {
		t.Fatalf("over-long name = %v, want TemplateNameTooLongError", err)
	}

	var badStatus *core.InvalidStatusError
	_, err := svc.CreateTemplate(ctx, core.TemplateInput{Name: "T", Status: ptr(core.Status("nope"))})
	if !errors.As(err, &badStatus) {
		t.Fatalf("bad status = %v, want InvalidStatusError", err)
	}
}

func TestCreateTemplateNormalizesAndTrims(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	tpl, err := svc.CreateTemplate(ctx, core.TemplateInput{
		Name:        "  Event Logger  ",
		TitlePrefix: ptr("BUG: "), // trailing space is meaningful, must survive
		Cwd:         ptr("  /tmp/proj  "),
		SlackThread: ptr("  https://slack.example/x  "),
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if tpl.Name != "Event Logger" {
		t.Fatalf("name = %q, want trimmed", tpl.Name)
	}
	if *tpl.TitlePrefix != "BUG: " {
		t.Fatalf("title_prefix = %q, want trailing space preserved", *tpl.TitlePrefix)
	}
	if *tpl.Cwd != "/tmp/proj" {
		t.Fatalf("cwd = %q, want trimmed", *tpl.Cwd)
	}
	if *tpl.SlackThread != "https://slack.example/x" {
		t.Fatalf("slack_thread = %q, want trimmed", *tpl.SlackThread)
	}
	if tpl.ID == "" || tpl.CreatedAt.IsZero() || tpl.UpdatedAt.IsZero() {
		t.Fatalf("template missing id/timestamps: %+v", tpl)
	}
}

func TestTemplateDuplicateNameFromService(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateTemplate(ctx, core.TemplateInput{Name: "Deploy"}); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	var dup *core.DuplicateTemplateNameError
	if _, err := svc.CreateTemplate(ctx, core.TemplateInput{Name: " deploy "}); !errors.As(err, &dup) {
		t.Fatalf("duplicate = %v, want DuplicateTemplateNameError", err)
	}
}

func TestGetTemplateByIDAndName(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	created, err := svc.CreateTemplate(ctx, core.TemplateInput{Name: "Event Logger"})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	byID, err := svc.GetTemplate(ctx, created.ID)
	if err != nil || byID.ID != created.ID {
		t.Fatalf("GetTemplate(id) = %+v, %v", byID, err)
	}
	byName, err := svc.GetTemplate(ctx, "event LOGGER")
	if err != nil || byName.ID != created.ID {
		t.Fatalf("GetTemplate(name) = %+v, %v", byName, err)
	}
	if _, err := svc.GetTemplate(ctx, "nothing here"); !errors.Is(err, core.ErrTemplateNotFound) {
		t.Fatalf("GetTemplate(missing) = %v, want ErrTemplateNotFound", err)
	}
	if _, err := svc.GetTemplate(ctx, "  "); !errors.Is(err, core.ErrTemplateNotFound) {
		t.Fatalf("GetTemplate(blank) = %v, want ErrTemplateNotFound", err)
	}
}

func TestUpdateTemplateIsFullReplace(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	created, err := svc.CreateTemplate(ctx, core.TemplateInput{
		Name:        "Event Logger",
		TitlePrefix: ptr("BUG: "),
		Cwd:         ptr("/tmp/proj"),
		HumanOnly:   ptr(true),
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	// Only Cwd is submitted, so the prefix and human_only presets are cleared.
	updated, err := svc.UpdateTemplate(ctx, created.ID, core.TemplateInput{
		Name: "Event Logger",
		Cwd:  ptr("/tmp/other"),
	})
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if updated.TitlePrefix != nil || updated.HumanOnly != nil {
		t.Fatalf("omitted fields not cleared: %+v", updated)
	}
	if *updated.Cwd != "/tmp/other" {
		t.Fatalf("cwd = %q", *updated.Cwd)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("created_at drifted: %v -> %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("updated_at not advanced: %v", updated.UpdatedAt)
	}

	if _, err := svc.UpdateTemplate(ctx, "missing", core.TemplateInput{Name: "x"}); !errors.Is(err, core.ErrTemplateNotFound) {
		t.Fatalf("update missing = %v, want ErrTemplateNotFound", err)
	}
}

func TestDeleteTemplateByName(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateTemplate(ctx, core.TemplateInput{Name: "Scratch"}); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	deleted, err := svc.DeleteTemplate(ctx, "scratch")
	if err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	if deleted.Name != "Scratch" {
		t.Fatalf("deleted.Name = %q", deleted.Name)
	}
	list, err := svc.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("templates after delete = %d, want 0", len(list))
	}
}

func TestTemplateApplySkipsUnsetFields(t *testing.T) {
	tpl := core.Template{
		Name:        "Partial",
		TitlePrefix: ptr("BUG: "),
		Cwd:         ptr("/tmp/proj"),
	}
	in := core.CreateInput{
		Title:           "broken login",
		Cwd:             "/default/cwd",
		HumanOnly:       true,
		IncludeInReport: ptr(false),
		SlackThread:     "https://slack.example/default",
		Status:          core.StatusWaiting,
	}
	got := tpl.Apply(in)

	if got.Title != "BUG: broken login" {
		t.Fatalf("title = %q, want prefixed", got.Title)
	}
	if got.Cwd != "/tmp/proj" {
		t.Fatalf("cwd = %q, want template override", got.Cwd)
	}
	// Everything the template does not define must survive untouched.
	if !got.HumanOnly {
		t.Fatal("human_only default lost")
	}
	if got.IncludeInReport == nil || *got.IncludeInReport {
		t.Fatalf("include_in_report default lost: %v", got.IncludeInReport)
	}
	if got.SlackThread != "https://slack.example/default" {
		t.Fatalf("slack_thread default lost: %q", got.SlackThread)
	}
	if got.Status != core.StatusWaiting {
		t.Fatalf("status default lost: %q", got.Status)
	}
}

func TestTemplateApplyExplicitFalseOverridesDefault(t *testing.T) {
	tpl := core.Template{
		Name:            "Quiet",
		Cwd:             ptr(""), // explicit empty clears the caller's default
		HumanOnly:       ptr(false),
		IncludeInReport: ptr(false),
	}
	in := core.CreateInput{
		Title:           "task",
		Cwd:             "/default/cwd",
		HumanOnly:       true,
		IncludeInReport: ptr(true),
	}
	got := tpl.Apply(in)

	if got.Cwd != "" {
		t.Fatalf("cwd = %q, want cleared by explicit empty preset", got.Cwd)
	}
	if got.HumanOnly {
		t.Fatal("human_only = true, want overridden to false")
	}
	if got.IncludeInReport == nil || *got.IncludeInReport {
		t.Fatalf("include_in_report = %v, want overridden to false", got.IncludeInReport)
	}
}

func TestCreateFromTemplate(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.CreateTemplate(ctx, core.TemplateInput{
		Name:        "Event Logger",
		TitlePrefix: ptr("[EL] "),
		Status:      ptr(core.StatusWIP),
		HumanOnly:   ptr(true),
	}); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	task, err := svc.CreateFromTemplate(ctx, "event logger", core.CreateInput{Title: "parse feed"})
	if err != nil {
		t.Fatalf("CreateFromTemplate: %v", err)
	}
	if task.Title != "[EL] parse feed" {
		t.Fatalf("title = %q", task.Title)
	}
	if task.Status != core.StatusWIP {
		t.Fatalf("status = %q, want wip", task.Status)
	}
	if !task.HumanOnly {
		t.Fatal("human_only not applied")
	}
	// Not defined by the template, so the normal Create default applies.
	if !task.IncludeInReport {
		t.Fatal("include_in_report = false, want the Create default of true")
	}

	if _, err := svc.CreateFromTemplate(ctx, "missing", core.CreateInput{Title: "x"}); !errors.Is(err, core.ErrTemplateNotFound) {
		t.Fatalf("CreateFromTemplate(missing) = %v, want ErrTemplateNotFound", err)
	}
}

func TestTemplateInputPointersAreCloned(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	cwd := "  /tmp/proj  "
	in := core.TemplateInput{Name: "Cloned", Cwd: &cwd}
	tpl, err := svc.CreateTemplate(ctx, in)
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	// Normalizing must not trim through the caller's pointer.
	if cwd != "  /tmp/proj  " {
		t.Fatalf("caller's string was mutated in place: %q", cwd)
	}
	// Mutating the caller's pointer must not reach the returned template.
	cwd = "/mutated"
	if *tpl.Cwd != "/tmp/proj" {
		t.Fatalf("cwd = %q, want the value captured at create time", *tpl.Cwd)
	}
}
