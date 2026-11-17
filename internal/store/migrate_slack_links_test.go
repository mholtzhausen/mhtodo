package store

import (
	"context"
	"path/filepath"
	"testing"
)

// A v20 database with a slack_thread migrates to v21 with that thread as
// link #1 (no description) and slack_thread unchanged; empty threads get [].
func TestV21BackfillsSlackLinks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mhtodo.db")
	repo, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`ALTER TABLE tasks DROP COLUMN slack_links`,
		`INSERT INTO tasks (id, title, status, progress, created_at, updated_at, slack_thread) VALUES ('t1', 'linked', 'pending', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', ' https://acme.slack.com/archives/C1/p1 ')`,
		`INSERT INTO tasks (id, title, status, progress, created_at, updated_at) VALUES ('t2', 'bare', 'pending', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`UPDATE meta SET value = '20' WHERE key = 'schema_version'`,
	} {
		if _, err := repo.db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	repo.Close()

	repo2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer repo2.Close()
	ctx := context.Background()
	t1, err := NewTaskRepo(repo2.db).GetByID(ctx, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(t1.SlackLinks) != 1 || t1.SlackLinks[0].URL != "https://acme.slack.com/archives/C1/p1" || t1.SlackLinks[0].Description != "" {
		t.Fatalf("t1 links = %+v", t1.SlackLinks)
	}
	if t1.SlackThread != " https://acme.slack.com/archives/C1/p1 " {
		t.Fatalf("slack_thread changed by migration: %q", t1.SlackThread)
	}
	t2, _ := NewTaskRepo(repo2.db).GetByID(ctx, "t2")
	if t2.SlackLinks == nil || len(t2.SlackLinks) != 0 {
		t.Fatalf("t2 links = %#v", t2.SlackLinks)
	}
}
