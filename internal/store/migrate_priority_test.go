package store

import (
	"path/filepath"
	"testing"
)

// A v19 database (no priority columns, existing rows) migrates to v20 with
// every task at priority none (0) and templates without a preset.
func TestV20AddsPriority(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mhtodo.db")
	repo, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`ALTER TABLE tasks DROP COLUMN priority`,
		`ALTER TABLE task_templates DROP COLUMN priority`,
		`INSERT INTO tasks (id, title, status, progress, created_at, updated_at) VALUES ('t1', 'old', 'pending', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`UPDATE meta SET value = '19' WHERE key = 'schema_version'`,
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
	var prio int
	if err := repo2.db.QueryRow(`SELECT priority FROM tasks WHERE id = 't1'`).Scan(&prio); err != nil || prio != 0 {
		t.Fatalf("migrated priority = %d (err=%v), want 0", prio, err)
	}
	var n int
	if err := repo2.db.QueryRow(`SELECT count(*) FROM pragma_table_info('task_templates') WHERE name = 'priority'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("task_templates.priority columns = %d (err=%v)", n, err)
	}
	if _, err := repo2.db.Exec(`UPDATE tasks SET priority = 5 WHERE id = 't1'`); err == nil {
		t.Fatal("CHECK allowed priority 5")
	}
}
