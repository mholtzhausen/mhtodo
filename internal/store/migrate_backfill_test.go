package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"mhtodo/internal/core"
)

func TestBackfillEmptyTodoSessions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backfill.db")
	// Build a v10 DB with empty todo_session rows, then open via Open so v11 runs.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	// Apply through v10 manually via Open then roll schema_version back… simpler:
	// Open fully, insert with empty session via raw SQL, reset version to 10, reopen.
	db.Close()

	repo, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	id := "019be00a-5f3a-7abc-8000-abc123456789"
	if err := repo.Create(ctx, core.Task{
		ID: id, Title: "Backfill me", Status: core.StatusPending,
		CreatedAt: now, UpdatedAt: now, IncludeInReport: true,
		TodoSession: "keep-me",
	}); err != nil {
		t.Fatal(err)
	}
	idEmpty := "019be00a-5f3a-7abc-8000-abc123456790"
	if _, err := repo.db.ExecContext(ctx,
		`INSERT INTO tasks (id, title, description, feedback, status, progress, created_at, updated_at, cwd, human_only, include_in_report, slack_thread, todo_session)
		 VALUES (?, ?, '', '', 'pending', 0, ?, ?, '', 0, 1, '', '')`,
		idEmpty, "Needs seed", formatTS(now), formatTS(now)); err != nil {
		t.Fatal(err)
	}
	idLegacy := "019be00a-5f3a-7abc-8000-abc123456791"
	legacy := core.LegacyTodoSession(core.ShortID(idLegacy), "Old spaced")
	if _, err := repo.db.ExecContext(ctx,
		`INSERT INTO tasks (id, title, description, feedback, status, progress, created_at, updated_at, cwd, human_only, include_in_report, slack_thread, todo_session)
		 VALUES (?, ?, '', '', 'pending', 0, ?, ?, '', 0, 1, '', ?)`,
		idLegacy, "Old spaced", formatTS(now), formatTS(now), legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.Exec(`UPDATE meta SET value = '10' WHERE key = 'schema_version'`); err != nil {
		t.Fatal(err)
	}
	repo.Close()

	repo2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer repo2.Close()

	var version int
	if err := repo2.db.QueryRow(`SELECT value FROM meta WHERE key='schema_version'`).Scan(&version); err != nil || version != 13 {
		t.Fatalf("version=%d err=%v want 13", version, err)
	}

	kept, err := repo2.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if kept.TodoSession != "keep-me" {
		t.Fatalf("existing session overwritten: %q", kept.TodoSession)
	}

	seeded, err := repo2.GetByID(ctx, idEmpty)
	if err != nil {
		t.Fatal(err)
	}
	want := core.DefaultTodoSession(core.ShortID(idEmpty), "Needs seed")
	if seeded.TodoSession != want {
		t.Fatalf("seeded todo_session = %q, want %q", seeded.TodoSession, want)
	}

	rewritten, err := repo2.GetByID(ctx, idLegacy)
	if err != nil {
		t.Fatal(err)
	}
	wantLegacy := core.DefaultTodoSession(core.ShortID(idLegacy), "Old spaced")
	if rewritten.TodoSession != wantLegacy {
		t.Fatalf("legacy rewrite = %q, want %q", rewritten.TodoSession, wantLegacy)
	}
}
