package core_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"mhtodo/internal/core"
)

func laneIDs(t *testing.T, svc *core.Service) []string {
	t.Helper()
	lanes, err := svc.ListFocus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(lanes))
	for i, l := range lanes {
		out[i] = l.ID
	}
	return out
}

func eqIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestFocusOrderAndMove(t *testing.T) {
	svc, repo := newTestService(t)
	ctx := context.Background()
	for _, id := range []string{"a", "b", "c"} {
		seed(t, repo, id, id, core.StatusWIP, 0)
		if _, err := svc.Focus(ctx, id, nil); err != nil {
			t.Fatal(err)
		}
	}
	if got := laneIDs(t, svc); !eqIDs(got, []string{"a", "b", "c"}) {
		t.Fatalf("append order: %v", got)
	}
	before := "a"
	if _, err := svc.Focus(ctx, "c", &before); err != nil {
		t.Fatal(err)
	}
	before = "b"
	if _, err := svc.Focus(ctx, "a", &before); err != nil {
		t.Fatal(err)
	}
	if got := laneIDs(t, svc); !eqIDs(got, []string{"c", "a", "b"}) {
		t.Fatalf("after moves: %v", got)
	}
	// Plain add on a pinned task moves it to the end; on the last one it's a no-op.
	if _, err := svc.Focus(ctx, "c", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Focus(ctx, "c", nil); err != nil {
		t.Fatal(err)
	}
	if got := laneIDs(t, svc); !eqIDs(got, []string{"a", "b", "c"}) {
		t.Fatalf("move to end: %v", got)
	}
	seed(t, repo, "x", "x", core.StatusWIP, 0)
	before = "x"
	if _, err := svc.Focus(ctx, "a", &before); !errors.Is(err, core.ErrNotFocused) {
		t.Fatalf("before unpinned: %v", err)
	}
}

func TestFocusCapAndClosed(t *testing.T) {
	svc, repo := newTestService(t)
	ctx := context.Background()
	for i := 0; i < core.FocusMax; i++ {
		id := string(rune('a' + i))
		seed(t, repo, id, id, core.StatusWIP, 0)
		if _, err := svc.Focus(ctx, id, nil); err != nil {
			t.Fatal(err)
		}
	}
	seed(t, repo, "z", "z", core.StatusWIP, 0)
	if _, err := svc.Focus(ctx, "z", nil); !errors.Is(err, core.ErrFocusFull) {
		t.Fatalf("7th pin: %v", err)
	}
	seed(t, repo, "d0", "done", core.StatusDone, 100)
	if _, err := svc.Focus(ctx, "d0", nil); !errors.Is(err, core.ErrFocusClosed) {
		t.Fatalf("done pin: %v", err)
	}

	// Marking done auto-unpins and frees a lane.
	if _, err := svc.SetStatus(ctx, "a", core.StatusDone); err != nil {
		t.Fatal(err)
	}
	a, _ := svc.Get(ctx, "a")
	if a.FocusRank != nil {
		t.Fatalf("done task still focused: %v", *a.FocusRank)
	}
	if _, err := svc.Focus(ctx, "z", nil); err != nil {
		t.Fatalf("pin after done freed lane: %v", err)
	}

	n, err := svc.ClearFocus(ctx)
	if err != nil || n != core.FocusMax {
		t.Fatalf("clear: n=%d err=%v", n, err)
	}
	if got := laneIDs(t, svc); len(got) != 0 {
		t.Fatalf("after clear: %v", got)
	}
}

func TestFocusParentHiddenBySubtask(t *testing.T) {
	svc, repo := newTestService(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 19, 11, 0, 0, 0, time.UTC)
	pid := "p"
	for _, task := range []core.Task{
		{ID: "p", Title: "parent", Status: core.StatusWIP, CreatedAt: now, UpdatedAt: now},
		{ID: "s", Title: "sub", Status: core.StatusWIP, ParentID: &pid, CreatedAt: now, UpdatedAt: now},
		{ID: "o", Title: "other", Status: core.StatusWIP, CreatedAt: now, UpdatedAt: now},
	} {
		if err := repo.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"p", "o", "s"} {
		if _, err := svc.Focus(ctx, id, nil); err != nil {
			t.Fatal(err)
		}
	}
	if got := laneIDs(t, svc); !eqIDs(got, []string{"o", "s"}) {
		t.Fatalf("parent should be hidden: %v", got)
	}
	if _, err := svc.Unfocus(ctx, "s"); err != nil {
		t.Fatal(err)
	}
	if got := laneIDs(t, svc); !eqIDs(got, []string{"p", "o"}) {
		t.Fatalf("parent lane should return: %v", got)
	}
	// Unfocus is idempotent.
	if _, err := svc.Unfocus(ctx, "s"); err != nil {
		t.Fatal(err)
	}
}

func TestFocusClearedOnArchive(t *testing.T) {
	svc, repo := newTestService(t)
	ctx := context.Background()
	seed(t, repo, "a", "a", core.StatusWIP, 0)
	if _, err := svc.Focus(ctx, "a", nil); err != nil {
		t.Fatal(err)
	}
	// Only done tasks archive, and done already unpins; archive clears defensively.
	if _, err := svc.SetStatus(ctx, "a", core.StatusDone); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Archive(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if got := laneIDs(t, svc); len(got) != 0 {
		t.Fatalf("archived still focused: %v", got)
	}
}

func TestCreateWithFocus(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// Creating with Focus appends to Focus lanes immediately.
	t1, err := svc.Create(ctx, core.CreateInput{Title: "First", Status: core.StatusWIP, Focus: true})
	if err != nil {
		t.Fatal(err)
	}
	if t1.FocusRank == nil {
		t.Fatalf("expected focus_rank set on created task")
	}
	t2, err := svc.Create(ctx, core.CreateInput{Title: "Second", Focus: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := laneIDs(t, svc); !eqIDs(got, []string{t1.ID, t2.ID}) {
		t.Fatalf("lanes after create: %v", got)
	}

	// Done task cannot be created focused and must not be persisted.
	if _, err := svc.Create(ctx, core.CreateInput{Title: "Done", Status: core.StatusDone, Focus: true}); !errors.Is(err, core.ErrFocusClosed) {
		t.Fatalf("done+focus: want ErrFocusClosed, got %v", err)
	}

	// Fill remaining lanes up to FocusMax.
	for i := 2; i < core.FocusMax; i++ {
		if _, err := svc.Create(ctx, core.CreateInput{Title: "Fill", Focus: true}); err != nil {
			t.Fatalf("fill %d: %v", i, err)
		}
	}

	// 7th root task with Focus fails with ErrFocusFull and must not be persisted.
	if _, err := svc.Create(ctx, core.CreateInput{Title: "Overflow", Focus: true}); !errors.Is(err, core.ErrFocusFull) {
		t.Fatalf("7th focus create: want ErrFocusFull, got %v", err)
	}
	all, err := svc.List(ctx, core.ListFilter{IncludeDone: true, IncludeHumanOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != core.FocusMax {
		t.Fatalf("failed focus creates should not persist tasks: got %d tasks, want %d", len(all), core.FocusMax)
	}

	// A sub-task of an already-focused parent replaces the parent's lane, so it fits even at FocusMax.
	sub1, err := svc.Create(ctx, core.CreateInput{Title: "Sub 1", ParentID: t1.ID, Focus: true})
	if err != nil {
		t.Fatalf("sub-task of focused parent at cap: %v", err)
	}
	if sub1.FocusRank == nil {
		t.Fatalf("expected focus_rank set on sub-task")
	}
	// A second focused sub-task of the same parent would add a 7th visible lane → ErrFocusFull.
	if _, err := svc.Create(ctx, core.CreateInput{Title: "Sub 2", ParentID: t1.ID, Focus: true}); !errors.Is(err, core.ErrFocusFull) {
		t.Fatalf("second focused sub-task at cap: want ErrFocusFull, got %v", err)
	}
}
