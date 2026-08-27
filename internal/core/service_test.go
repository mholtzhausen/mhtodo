package core_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"mhtodo/internal/core"
	"mhtodo/internal/store"
)

// newTestService builds a Service over a temp-dir DB with a deterministic,
// advancing clock.
func newTestService(t *testing.T) (*core.Service, *store.TaskRepo) {
	t.Helper()
	repo, err := store.Open(filepath.Join(t.TempDir(), "mhtodo.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	svc := core.NewService(repo)
	base := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	n := 0
	svc.SetNowFunc(func() time.Time { n++; return base.Add(time.Duration(n) * time.Second) })
	return svc, repo
}

func seed(t *testing.T, repo *store.TaskRepo, id, title string, st core.Status, prog int) {
	t.Helper()
	now := time.Date(2026, 8, 19, 11, 0, 0, 0, time.UTC)
	task := core.Task{ID: id, Title: title, Status: st, Progress: prog, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(context.Background(), task); err != nil {
		t.Fatalf("seed %s: %v", id, err)
	}
}

func TestCreateValidation(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, core.CreateInput{Title: "   "}); !errors.Is(err, core.ErrEmptyTitle) {
		t.Errorf("empty title: %v, want ErrEmptyTitle", err)
	}
	_, err := svc.Create(ctx, core.CreateInput{Title: "x", Status: core.Status("nope")})
	var ise *core.InvalidStatusError
	if !errors.As(err, &ise) {
		t.Errorf("bad status: %v, want InvalidStatusError", err)
	}
	for _, p := range []int{-1, 101} {
		_, err := svc.Create(ctx, core.CreateInput{Title: "x", Progress: p})
		var pre *core.ProgressRangeError
		if !errors.As(err, &pre) || pre.Value != p {
			t.Errorf("progress %d: %v, want ProgressRangeError", p, err)
		}
	}

	got, err := svc.Create(ctx, core.CreateInput{Title: "  Trim me  ", Description: "d"})
	if err != nil {
		t.Fatalf("valid create: %v", err)
	}
	if got.Title != "Trim me" || got.Status != core.StatusPending || got.Progress != 0 ||
		got.CompletedAt != nil || len(got.ID) != 36 {
		t.Errorf("defaults wrong: %+v", got)
	}
}

func TestCreateDirectlyDone(t *testing.T) {
	svc, _ := newTestService(t)
	got, err := svc.Create(context.Background(), core.CreateInput{Title: "x", Status: core.StatusDone})
	if err != nil {
		t.Fatal(err)
	}
	if got.Progress != 100 || got.CompletedAt == nil {
		t.Errorf("create-as-done effects missing: %+v", got)
	}
}

func TestStatusTransitions(t *testing.T) {
	svc, repo := newTestService(t)
	ctx := context.Background()
	seed(t, repo, "aaaa1111-0000-7000-8000-000000000001", "task", core.StatusWIP, 40)

	// wip → done: progress forced to 100, completed_at set.
	got, err := svc.SetStatus(ctx, "aaaa", core.StatusDone)
	if err != nil {
		t.Fatal(err)
	}
	if got.Progress != 100 || got.CompletedAt == nil {
		t.Fatalf("→done effects: %+v", got)
	}

	// done → waiting: completed_at cleared, progress left as-is (spec).
	got, err = svc.SetStatus(ctx, "aaaa", core.StatusWaiting)
	if err != nil {
		t.Fatal(err)
	}
	if got.CompletedAt != nil || got.Progress != 100 {
		t.Fatalf("done→waiting effects: %+v", got)
	}

	// waiting → pending.
	got, err = svc.SetStatus(ctx, "aaaa", core.StatusPending)
	if err != nil || got.Status != core.StatusPending || got.CompletedAt != nil {
		t.Fatalf("→pending: (%+v, %v)", got, err)
	}

	// updated_at must advance on every change.
	first, _ := svc.Get(ctx, "aaaa")
	if _, err := svc.SetStatus(ctx, "aaaa", core.StatusWIP); err != nil {
		t.Fatal(err)
	}
	second, _ := svc.Get(ctx, "aaaa")
	if !second.UpdatedAt.After(first.UpdatedAt) {
		t.Errorf("updated_at did not advance: %v → %v", first.UpdatedAt, second.UpdatedAt)
	}

	// Invalid status rejected before lookup.
	_, err = svc.SetStatus(ctx, "aaaa", core.Status("bogus"))
	var ise *core.InvalidStatusError
	if !errors.As(err, &ise) {
		t.Errorf("bad status: %v", err)
	}
}

func TestPrefixLookup(t *testing.T) {
	svc, repo := newTestService(t)
	ctx := context.Background()
	seed(t, repo, "aaaa1111-0000-7000-8000-000000000001", "one", core.StatusPending, 0)
	seed(t, repo, "bbbb2222-0000-7000-8000-000000000001", "two", core.StatusWIP, 50)
	seed(t, repo, "bbbb3333-0000-7000-8000-000000000002", "three", core.StatusPending, 10)

	// Unique prefix resolves.
	got, err := svc.Get(ctx, "aaaa")
	if err != nil || got.ID != "aaaa1111-0000-7000-8000-000000000001" {
		t.Fatalf("unique prefix: (%+v, %v)", got, err)
	}

	// Full ID works.
	if _, err := svc.Get(ctx, "bbbb2222-0000-7000-8000-000000000001"); err != nil {
		t.Fatalf("full id: %v", err)
	}

	// Ambiguous prefix → typed error with sorted candidates.
	_, err = svc.Get(ctx, "bbbb")
	var amb *core.AmbiguousIDError
	if !errors.As(err, &amb) || len(amb.Candidates) != 2 {
		t.Fatalf("ambiguous: %v", err)
	}
	if amb.Candidates[0] != "bbbb2222-0000-7000-8000-000000000001" ||
		amb.Candidates[1] != "bbbb3333-0000-7000-8000-000000000002" {
		t.Errorf("candidates not sorted/complete: %v", amb.Candidates)
	}

	// Too-short non-exact ref → ErrNotFound (not ambiguous).
	if _, err := svc.Get(ctx, "aaa"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("short ref: %v, want ErrNotFound", err)
	}
	if _, err := svc.Get(ctx, "zzzz9999-0000-7000-8000-000000000001"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("missing id: %v, want ErrNotFound", err)
	}
}

func TestEdit(t *testing.T) {
	svc, repo := newTestService(t)
	ctx := context.Background()
	seed(t, repo, "cccc1111-0000-7000-8000-000000000001", "orig", core.StatusPending, 5)

	// No fields → error.
	if _, err := svc.Edit(ctx, "cccc", core.UpdateInput{}); !errors.Is(err, core.ErrNoFieldsToUpdate) {
		t.Fatalf("empty edit: %v", err)
	}

	title, desc, feedback, prog := "new title", "", "shipped cleanly", 77
	got, err := svc.Edit(ctx, "cccc", core.UpdateInput{Title: &title, Desc: &desc, Feedback: &feedback, Progress: &prog})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "new title" || got.Description != "" || got.Feedback != "shipped cleanly" || got.Progress != 77 || got.Status != core.StatusPending {
		t.Errorf("edit result: %+v", got)
	}

	// created_at immutable, updated_at bumped.
	again, _ := svc.Get(ctx, "cccc")
	if !again.CreatedAt.Equal(time.Date(2026, 8, 19, 11, 0, 0, 0, time.UTC)) {
		t.Errorf("created_at changed: %v", again.CreatedAt)
	}

	// Validation on edit.
	empty := ""
	if _, err := svc.Edit(ctx, "cccc", core.UpdateInput{Title: &empty}); !errors.Is(err, core.ErrEmptyTitle) {
		t.Errorf("empty title edit: %v", err)
	}
	bad := 150
	_, err = svc.Edit(ctx, "cccc", core.UpdateInput{Progress: &bad})
	var pre *core.ProgressRangeError
	if !errors.As(err, &pre) {
		t.Errorf("progress edit: %v", err)
	}

	// Edit does not touch status effects (done task keeps completed_at).
	doneAt := time.Date(2026, 8, 19, 11, 30, 0, 0, time.UTC)
	if err := repo.Create(context.Background(), core.Task{
		ID: "dddd1111-0000-7000-8000-000000000002", Title: "done one", Status: core.StatusDone,
		Progress: 100, CreatedAt: doneAt, UpdatedAt: doneAt, CompletedAt: &doneAt,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Edit(ctx, "dddd", core.UpdateInput{Progress: &prog}); err != nil {
		t.Fatal(err)
	}
	got, _ = svc.Get(ctx, "dddd")
	if got.Status != core.StatusDone || got.CompletedAt == nil {
		t.Errorf("edit on done task should not clear completed_at: %+v", got)
	}
}

func TestDelete(t *testing.T) {
	svc, repo := newTestService(t)
	ctx := context.Background()
	seed(t, repo, "eeee1111-0000-7000-8000-000000000001", "doomed", core.StatusPending, 0)

	got, err := svc.Delete(ctx, "eeee")
	if err != nil || got.ID != "eeee1111-0000-7000-8000-000000000001" {
		t.Fatalf("delete: (%+v, %v)", got, err)
	}
	if _, err := svc.Get(ctx, "eeee"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
}

func TestListViaService(t *testing.T) {
	svc, repo := newTestService(t)
	ctx := context.Background()
	seed(t, repo, "ffff1111-0000-7000-8000-000000000001", "a", core.StatusPending, 0)
	seed(t, repo, "ffff2222-0000-7000-8000-000000000002", "b", core.StatusDone, 100)

	got, err := svc.List(ctx, core.ListFilter{})
	if err != nil || len(got) != 1 {
		t.Fatalf("default list: (%d, %v)", len(got), err)
	}
	_, err = svc.List(ctx, core.ListFilter{Status: core.Status("bogus")})
	var ise *core.InvalidStatusError
	if !errors.As(err, &ise) {
		t.Errorf("bad filter status: %v", err)
	}
}

// TestArchiveDoneAndUnarchive covers the v0.2 archive lifecycle: bulk archive
// of done tasks, exclusion from default lists, explicit archived view, and
// unarchive → pending with progress reset.
func TestArchiveDoneAndUnarchive(t *testing.T) {
	svc, repo := newTestService(t)
	ctx := context.Background()

	seed(t, repo, "aaaa1111-0000-7000-8000-000000000001", "done one", core.StatusDone, 100)
	seed(t, repo, "bbbb2222-0000-7000-8000-000000000001", "done two", core.StatusDone, 100)
	seed(t, repo, "cccc3333-0000-7000-8000-000000000001", "wip task", core.StatusWIP, 50)
	seed(t, repo, "dddd4444-0000-7000-8000-000000000001", "pending task", core.StatusPending, 0)

	// ArchiveDone returns exactly the done tasks, stamped with archived_at.
	archived, err := svc.ArchiveDone(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(archived) != 2 {
		t.Fatalf("archived %d tasks, want 2: %+v", len(archived), archived)
	}
	for _, tsk := range archived {
		if tsk.ArchivedAt == nil || tsk.Status != core.StatusDone {
			t.Errorf("%s wrong after archive: %+v", tsk.ID, tsk)
		}
	}

	// Second call is a no-op.
	again, err := svc.ArchiveDone(ctx)
	if err != nil || len(again) != 0 {
		t.Fatalf("second ArchiveDone = (%d, %v), want (0, nil)", len(again), err)
	}

	// Default list hides archived (and done); the wip+pending tasks remain.
	defList, err := svc.List(ctx, core.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(defList) != 2 {
		t.Errorf("default list = %d tasks, want 2 (wip + pending): %+v", len(defList), defList)
	}

	// Explicit archived view shows exactly the two done tasks.
	archList, err := svc.List(ctx, core.ListFilter{Archived: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(archList) != 2 {
		t.Fatalf("archived list = %d tasks, want 2: %+v", len(archList), archList)
	}

	// Unarchive by prefix → pending, progress reset to 0, completed_at cleared.
	got, err := svc.Unarchive(ctx, "aaaa")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != core.StatusPending || got.Progress != 0 ||
		got.CompletedAt != nil || got.ArchivedAt != nil {
		t.Errorf("unarchived task wrong: %+v", got)
	}

	// It is back in the default list.
	defList, _ = svc.List(ctx, core.ListFilter{})
	if len(defList) != 3 {
		t.Errorf("default list after unarchive = %d tasks, want 3: %+v", len(defList), defList)
	}

	// Unarchiving a non-archived task → ErrNotArchived.
	_, err = svc.Unarchive(ctx, "cccc")
	if !errors.Is(err, core.ErrNotArchived) {
		t.Errorf("unarchive of wip task: %v, want ErrNotArchived", err)
	}

	// Unknown id still maps to ErrNotFound.
	_, err = svc.Unarchive(ctx, "zzzz9999")
	if !errors.Is(err, core.ErrNotFound) {
		t.Errorf("unarchive unknown: %v, want ErrNotFound", err)
	}
}

func TestCreateWithParent(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	parent, err := svc.Create(ctx, core.CreateInput{Title: "Parent"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := svc.Create(ctx, core.CreateInput{Title: "Child", ParentID: parent.ID})
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Fatalf("child parent_id: %+v", child)
	}
	// One level only: cannot nest under a child.
	_, err = svc.Create(ctx, core.CreateInput{Title: "Grand", ParentID: child.ID})
	if !errors.Is(err, core.ErrParentIsChild) {
		t.Fatalf("nest under child: %v, want ErrParentIsChild", err)
	}
	n, err := svc.CountChildren(ctx, parent.ID)
	if err != nil || n != 1 {
		t.Fatalf("CountChildren = (%d, %v), want 1", n, err)
	}
	// Cascade delete.
	if _, err := svc.Delete(ctx, parent.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, child.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("child should be cascaded: %v", err)
	}
}

func TestReviewStatus(t *testing.T) {
	svc, repo := newTestService(t)
	ctx := context.Background()
	seed(t, repo, "rrrr1111-0000-7000-8000-000000000001", "r", core.StatusPending, 0)
	got, err := svc.SetStatus(ctx, "rrrr", core.StatusReview)
	if err != nil || got.Status != core.StatusReview {
		t.Fatalf("→review: (%+v, %v)", got, err)
	}
}

func TestActivityCRUD(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	task, err := svc.Create(ctx, core.CreateInput{Title: "T"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.AddActivity(ctx, task.ID, core.ActivityInput{})
	if !errors.Is(err, core.ErrEmptyActivity) {
		t.Fatalf("empty: %v", err)
	}
	a, err := svc.AddActivity(ctx, task.ID, core.ActivityInput{Activity: " Ran tests ", Comment: " ok "})
	if err != nil {
		t.Fatal(err)
	}
	if a.Activity != "Ran tests" || a.Comment != "ok" || a.TaskID != task.ID {
		t.Fatalf("activity: %+v", a)
	}
	list, err := svc.ListActivity(ctx, core.ActivityFilter{})
	if err != nil || len(list) != 1 {
		t.Fatalf("list: (%d, %v)", len(list), err)
	}
	del, err := svc.DeleteActivity(ctx, a.ID[:8])
	if err != nil || del.ID != a.ID {
		t.Fatalf("delete: (%+v, %v)", del, err)
	}
}
