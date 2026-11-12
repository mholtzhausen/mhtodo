package store

import (
	"path/filepath"
	"testing"
)

// A DB that ran the early v17 (git_refs only) must gain git_repo/git_branches via v18.
func TestEarlyV17GitRefsDBIsRepaired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mhtodo.db")
	repo, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`ALTER TABLE tasks DROP COLUMN git_repo`,
		`ALTER TABLE tasks DROP COLUMN git_branches`,
		`ALTER TABLE tasks ADD COLUMN git_refs TEXT NOT NULL DEFAULT ''`,
		`UPDATE meta SET value = '17' WHERE key = 'schema_version'`,
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
	var n int
	if err := repo2.db.QueryRow(`SELECT count(*) FROM pragma_table_info('tasks') WHERE name IN ('git_repo','git_branches')`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("git columns = %d (err=%v), want 2", n, err)
	}
}
