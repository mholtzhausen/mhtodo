package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"mhtodo/internal/core"
)

func openTestRepo(t *testing.T) *TaskRepo {
	t.Helper()
	repo, err := Open(filepath.Join(t.TempDir(), "sub", "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func mustCreate(t *testing.T, r *TaskRepo, id, title string, st core.Status, prog int) core.Task {
	t.Helper()
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	var completed *time.Time
	if st == core.StatusDone {
		completed = &now
	}
	task := core.Task{ID: id, Title: title, Status: st, Progress: prog, CreatedAt: now, UpdatedAt: now, CompletedAt: completed}
	if err := r.Create(context.Background(), task); err != nil {
		t.Fatalf("Create(%s): %v", id, err)
	}
	return task
}

func TestOpenMigratesAndIsIdempotent(t *testing.T) {
	// Nested dir so Open's MkdirAll(0700) actually creates it (t.TempDir's own
	// mode is not ours to assert).
	path := filepath.Join(t.TempDir(), "mhtodo", "mhtodo.db")
	repo, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	var version int
	if err := repo.db.QueryRow(`SELECT value FROM meta WHERE key='schema_version'`).Scan(&version); err != nil {
		t.Fatalf("read schema_version: %v", err)
	}
	if version != 1 {
		t.Fatalf("schema_version = %d, want 1", version)
	}
	repo.Close()

	// Reopen on the migrated DB must be a no-op and succeed.
	repo2, err := Open(path)
	if err != nil {
		t.Fatalf("second open (idempotency): %v", err)
	}
	defer repo2.Close()
	version = 0
	if err := repo2.db.QueryRow(`SELECT value FROM meta WHERE key='schema_version'`).Scan(&version); err != nil || version != 1 {
		t.Fatalf("schema_version after reopen = %d (err=%v), want 1", version, err)
	}

	// WAL must be active.
	var mode string
	if err := repo2.db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("journal_mode = %q (err=%v), want wal", mode, err)
	}

	// First-run permissions: dir 0700, file 0600.
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("db file mode = %o, want 600", perm)
	}
	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Errorf("data dir mode = %o, want 700", perm)
	}
}

func TestCRUDRoundtrip(t *testing.T) {
	ctx := context.Background()
	r := openTestRepo(t)
	now := time.Date(2026, 8, 19, 12, 30, 5, 0, time.UTC)
	completed := now.Add(time.Hour)

	task := core.Task{ID: "id-1", Title: "T", Description: "D", Status: core.StatusDone,
		Progress: 100, CreatedAt: now, UpdatedAt: now, CompletedAt: &completed}
	if err := r.Create(ctx, task); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := r.GetByID(ctx, "id-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "T" || got.Description != "D" || got.Status != core.StatusDone ||
		got.Progress != 100 || !got.CreatedAt.Equal(now) || got.CompletedAt == nil || !got.CompletedAt.Equal(completed) {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}

	if _, err := r.GetByID(ctx, "missing"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("GetByID(missing) err = %v, want ErrNotFound", err)
	}

	// Update (mutable fields), then verify.
	got.Status = core.StatusWIP
	got.CompletedAt = nil
	got.UpdatedAt = now.Add(time.Minute)
	if err := r.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	again, _ := r.GetByID(ctx, "id-1")
	if again.Status != core.StatusWIP || again.CompletedAt != nil {
		t.Fatalf("after update: %+v", again)
	}
	if err := r.Update(ctx, core.Task{ID: "ghost"}); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("Update(ghost) err = %v, want ErrNotFound", err)
	}

	// Delete returns the deleted row.
	del, err := r.Delete(ctx, "id-1")
	if err != nil || del.ID != "id-1" {
		t.Fatalf("Delete: (%+v, %v)", del, err)
	}
	if _, err := r.GetByID(ctx, "id-1"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("after delete, GetByID err = %v, want ErrNotFound", err)
	}
}

func TestFindByPrefix(t *testing.T) {
	ctx := context.Background()
	r := openTestRepo(t)
	mustCreate(t, r, "aaaa1111-0000-7000-8000-000000000001", "one", core.StatusPending, 0)
	mustCreate(t, r, "bbbb2222-0000-7000-8000-000000000001", "two", core.StatusWIP, 50)
	mustCreate(t, r, "bbbb3333-0000-7000-8000-000000000002", "three", core.StatusPending, 10)

	got, err := r.FindByPrefix(ctx, "aaaa")
	if err != nil || len(got) != 1 || got[0].ID != "aaaa1111-0000-7000-8000-000000000001" {
		t.Fatalf("FindByPrefix(aaaa) = (%v, %v)", got, err)
	}
	got, err = r.FindByPrefix(ctx, "bbbb")
	if err != nil || len(got) != 2 {
		t.Fatalf("FindByPrefix(bbbb) = (%v, %v), want 2 matches", got, err)
	}

	// LIKE wildcards in the prefix must not match.
	got, err = r.FindByPrefix(ctx, "a%")
	if err != nil || len(got) != 0 {
		t.Fatalf("FindByPrefix(a%%) = (%v, %v), want no matches", got, err)
	}
}

func TestListFilters(t *testing.T) {
	ctx := context.Background()
	r := openTestRepo(t)
	base := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	mk := func(id, title string, st core.Status, prog int, ageMin int) {
		ts := base.Add(time.Duration(ageMin) * time.Minute)
		if err := r.Create(ctx, core.Task{ID: id, Title: title, Status: st, Progress: prog, CreatedAt: ts, UpdatedAt: ts}); err != nil {
			t.Fatal(err)
		}
	}
	mk("t1", "Alpha refactor", core.StatusPending, 0, 5)
	mk("t2", "beta deploy", core.StatusWIP, 40, 3)
	mk("t3", "Gamma docs", core.StatusDone, 100, 8)
	mk("t4", "alpha search test", core.StatusWaiting, 10, 1)

	// Default: excludes done.
	got, err := r.List(ctx, core.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("default list = %d tasks, want 3 (done excluded)", len(got))
	}

	// IncludeDone.
	got, _ = r.List(ctx, core.ListFilter{IncludeDone: true})
	if len(got) != 4 {
		t.Fatalf("--all list = %d tasks, want 4", len(got))
	}

	// Status filter (done is reachable explicitly).
	got, _ = r.List(ctx, core.ListFilter{Status: core.StatusDone})
	if len(got) != 1 || got[0].ID != "t3" {
		t.Fatalf("status=done list = %+v", got)
	}

	// Case-insensitive search over title + description; LIKE wildcards escaped.
	got, _ = r.List(ctx, core.ListFilter{Search: "ALPHA"})
	if len(got) != 2 {
		t.Fatalf("search ALPHA = %d tasks, want 2", len(got))
	}
	got, _ = r.List(ctx, core.ListFilter{Search: "%"})
	if len(got) != 0 {
		t.Fatalf("search %% should match nothing (escaped), got %d", len(got))
	}

	// Sort by progress desc (default direction) and asc. Non-done set: t2=40, t4=10, t1=0.
	got, _ = r.List(ctx, core.ListFilter{Sort: "progress"})
	if got[0].ID != "t2" || got[len(got)-1].ID != "t1" {
		t.Fatalf("sort progress desc = %v", ids(got))
	}
	got, _ = r.List(ctx, core.ListFilter{Sort: "progress", Ascending: true})
	if got[0].ID != "t1" || got[len(got)-1].ID != "t2" {
		t.Fatalf("sort progress asc = %v", ids(got))
	}

	// Default sort is updated_at desc.
	got, _ = r.List(ctx, core.ListFilter{})
	if got[0].ID != "t1" || got[len(got)-1].ID != "t4" {
		t.Fatalf("default sort = %v, want t1..t4 by updated desc", ids(got))
	}

	// Limit.
	got, _ = r.List(ctx, core.ListFilter{Limit: 2})
	if len(got) != 2 {
		t.Fatalf("limit 2 = %d tasks", len(got))
	}

	// Unknown sort field falls back to updated_at (no SQL injection).
	got, err = r.List(ctx, core.ListFilter{Sort: "title; DROP TABLE tasks"})
	if err != nil || len(got) != 3 {
		t.Fatalf("injection-ish sort: (%d, %v)", len(got), err)
	}
}

// TestConcurrentWriters simulates the CLI+GUI concurrency contract: two
// independent connections on one WAL file writing at the same time must not
// error (busy_timeout absorbs brief lock contention).
func TestConcurrentWriters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mhtodo.db")
	r1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r1.Close()
	r2, err := Open(path) // second process (e.g. CLI while GUI is open)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for i, repo := range []*TaskRepo{r1, r2} {
		wg.Add(1)
		go func(i int, repo *TaskRepo) {
			defer wg.Done()
			for n := 0; n < 32; n++ {
				id := fmt.Sprintf("w%02d-%04d-7000-8000-00000000000%d", i, n, i)
				now := time.Now().UTC()
				task := core.Task{ID: id, Title: "t", Status: core.StatusPending, CreatedAt: now, UpdatedAt: now}
				if err := repo.Create(ctx, task); err != nil {
					errs <- fmt.Errorf("create %s: %w", id, err)
					return
				}
				task.Progress = n % 101
				task.UpdatedAt = now.Add(time.Millisecond)
				if err := repo.Update(ctx, task); err != nil {
					errs <- fmt.Errorf("update %s: %w", id, err)
					return
				}
			}
		}(i, repo)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	var count int
	if err := r1.db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&count); err != nil || count != 64 {
		t.Fatalf("row count = %d (err=%v), want 64", count, err)
	}
}

func ids(ts []core.Task) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.ID
	}
	return out
}
