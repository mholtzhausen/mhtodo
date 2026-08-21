package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// Forward-only, versioned migrations. v1 ships as the initial migration (not
// inline DDL) so the pattern is proven from day one.
type migration struct {
	version int
	up      string
}

var migrations = []migration{
	{version: 1, up: schemaV1},
	{version: 2, up: schemaV2},
}

// v2 adds the archive (v0.2): archived_at is set when a done task is archived
// and cleared on unarchive. Archived tasks are hidden from default lists and
// the board; only an explicit archived filter shows them.
const schemaV2 = `
ALTER TABLE tasks ADD COLUMN archived_at TEXT; -- set on archive, cleared on unarchive

CREATE INDEX idx_tasks_archived ON tasks(archived_at);
`

const schemaV1 = `
CREATE TABLE meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
INSERT INTO meta VALUES ('schema_version', '1');

CREATE TABLE tasks (
  id           TEXT PRIMARY KEY,            -- UUIDv7 string
  title        TEXT NOT NULL,
  description  TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending','wip','done','waiting')),
  progress     INTEGER NOT NULL DEFAULT 0
               CHECK (progress BETWEEN 0 AND 100),
  created_at   TEXT NOT NULL,               -- RFC3339 UTC
  updated_at   TEXT NOT NULL,
  completed_at TEXT                         -- set on →done, cleared when leaving done
);

CREATE INDEX idx_tasks_status  ON tasks(status);
CREATE INDEX idx_tasks_updated ON tasks(updated_at DESC);
`

// applyMigrations brings the schema up to the latest version. Idempotent: a
// fresh DB runs v1; an already-migrated DB is a no-op. All pending steps run
// in one transaction.
func applyMigrations(db *sql.DB) error {
	current, err := readSchemaVersion(db)
	if err != nil {
		return err
	}
	pending := make([]migration, 0, len(migrations))
	for _, m := range migrations {
		if m.version > current {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}
	defer tx.Rollback() // no-op after Commit

	for _, m := range pending {
		if _, err := tx.Exec(m.up); err != nil {
			return fmt.Errorf("apply migration v%d: %w", m.version, err)
		}
		if _, err := tx.Exec(`UPDATE meta SET value = ? WHERE key = 'schema_version'`,
			fmt.Sprintf("%d", m.version)); err != nil {
			return fmt.Errorf("record schema version %d: %w", m.version, err)
		}
	}
	return tx.Commit()
}

// readSchemaVersion returns 0 when the meta table does not exist yet.
func readSchemaVersion(db *sql.DB) (int, error) {
	var v int
	err := db.QueryRow(`SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&v)
	if err == sql.ErrNoRows || isNoSuchTable(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return v, nil
}

// isNoSuchTable matches modernc's "no such table" error text.
func isNoSuchTable(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "no such table")
}
