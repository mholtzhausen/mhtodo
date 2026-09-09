package store

import (
	"database/sql"
	"fmt"
	"strings"

	"mhtodo/internal/core"
)

// Forward-only, versioned migrations. v1 ships as the initial migration (not
// inline DDL) so the pattern is proven from day one.
type migration struct {
	version int
	up      string              // optional SQL (may be empty when fn alone does the work)
	fn      func(*sql.Tx) error // optional Go step after SQL, same transaction
}

var migrations = []migration{
	{version: 1, up: schemaV1},
	{version: 2, up: schemaV2},
	{version: 3, up: schemaV3},
	{version: 4, up: schemaV4},
	{version: 5, up: schemaV5},
	{version: 6, up: schemaV6},
	{version: 7, up: schemaV7},
	{version: 8, up: schemaV8},
	{version: 9, up: schemaV9},
	{version: 10, up: schemaV10},
	{version: 11, fn: backfillEmptyTodoSessions},
	{version: 12, fn: addTerminalPIDColumn},
	{version: 13, fn: seedThemesV13},
}

// v2 adds the archive (v0.2): archived_at is set when a done task is archived
// and cleared on unarchive. Archived tasks are hidden from default lists and
// the board; only an explicit archived filter shows them.
const schemaV2 = `
ALTER TABLE tasks ADD COLUMN archived_at TEXT; -- set on archive, cleared on unarchive

CREATE INDEX idx_tasks_archived ON tasks(archived_at);
`

// v3 (v0.3): rebuild tasks to expand status CHECK + add parent_id; add activity.
// SQLite cannot ALTER a CHECK constraint, so we copy into a new table.
const schemaV3 = `
CREATE TABLE tasks_v3 (
  id           TEXT PRIMARY KEY,
  title        TEXT NOT NULL,
  description  TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending','wip','waiting','review','done')),
  progress     INTEGER NOT NULL DEFAULT 0
               CHECK (progress BETWEEN 0 AND 100),
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL,
  completed_at TEXT,
  archived_at  TEXT,
  parent_id    TEXT REFERENCES tasks_v3(id) ON DELETE CASCADE
);

INSERT INTO tasks_v3 (id, title, description, status, progress, created_at, updated_at, completed_at, archived_at, parent_id)
SELECT id, title, description, status, progress, created_at, updated_at, completed_at, archived_at, NULL
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_v3 RENAME TO tasks;

CREATE INDEX idx_tasks_status   ON tasks(status);
CREATE INDEX idx_tasks_updated  ON tasks(updated_at DESC);
CREATE INDEX idx_tasks_archived ON tasks(archived_at);
CREATE INDEX idx_tasks_parent   ON tasks(parent_id);

CREATE TABLE activity (
  id          TEXT PRIMARY KEY,
  task_id     TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  activity    TEXT NOT NULL DEFAULT '',
  comment     TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL
);
CREATE INDEX idx_activity_created ON activity(created_at DESC);
CREATE INDEX idx_activity_task    ON activity(task_id);
`

// v4: agent-authored feedback shown in the GUI when non-empty.
const schemaV4 = `
ALTER TABLE tasks ADD COLUMN feedback TEXT NOT NULL DEFAULT '';
`

// v5: persisted board column order for root tasks (GUI reorder).
const schemaV5 = `
ALTER TABLE tasks ADD COLUMN board_rank REAL;

CREATE INDEX idx_tasks_status_rank ON tasks(status, board_rank);
`

// v6: optional task cwd and human-only flag (agents exclude from default lists).
const schemaV6 = `
ALTER TABLE tasks ADD COLUMN cwd TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN human_only INTEGER NOT NULL DEFAULT 0
  CHECK (human_only IN (0, 1));
`

// v7: per-task opt-out from Slack board report (default included).
const schemaV7 = `
ALTER TABLE tasks ADD COLUMN include_in_report INTEGER NOT NULL DEFAULT 1
  CHECK (include_in_report IN (0, 1));
`

// v8: optional Slack thread URL for per-task communication context.
const schemaV8 = `
ALTER TABLE tasks ADD COLUMN slack_thread TEXT NOT NULL DEFAULT '';
`

// v9: named task templates (v0.5). Every preset field is NULLable on purpose —
// unlike tasks, which are NOT NULL with empty-string defaults, a template must distinguish
// "not part of this template" (NULL → task creation falls back to the normal
// default) from an explicit empty/false value that overrides that default.
const schemaV9 = `
CREATE TABLE task_templates (
  id                TEXT PRIMARY KEY,            -- UUIDv7 string
  name              TEXT NOT NULL COLLATE NOCASE UNIQUE,
  title_prefix      TEXT,
  description       TEXT,
  status            TEXT CHECK (status IS NULL OR status IN
                      ('pending','wip','waiting','review','done')),
  cwd               TEXT,
  slack_thread      TEXT,
  human_only        INTEGER CHECK (human_only IS NULL OR human_only IN (0, 1)),
  include_in_report INTEGER CHECK (include_in_report IS NULL OR include_in_report IN (0, 1)),
  created_at        TEXT NOT NULL,               -- RFC3339 UTC
  updated_at        TEXT NOT NULL
);
`

// v10: per-task session identity for Claude --session-id/--resume/--name, Zed MHTODO_SESSION, shell claude.todo.
const schemaV10 = `
ALTER TABLE tasks ADD COLUMN todo_session TEXT NOT NULL DEFAULT '';
`

// v12: OS PID of mhtodo-managed Claude terminal session (spawn=terminal).
// Idempotent: skip when the column already exists (test re-opens after version rollbacks).
func addTerminalPIDColumn(tx *sql.Tx) error {
	rows, err := tx.Query(`PRAGMA table_info(tasks)`)
	if err != nil {
		return fmt.Errorf("pragma table_info: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "terminal_pid" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE tasks ADD COLUMN terminal_pid INTEGER NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("add terminal_pid: %w", err)
	}
	return nil
}

// seedThemesV13 creates the themes table, inserts Slate/Paper/Ember, and sets
// meta.active_theme_id to Slate. Idempotent if the table already exists.
func seedThemesV13(tx *sql.Tx) error {
	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS themes (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL COLLATE NOCASE UNIQUE,
  builtin_key TEXT,
  tokens      TEXT NOT NULL,
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);
`); err != nil {
		return fmt.Errorf("create themes: %w", err)
	}

	now := "2026-09-09T00:00:00Z"
	for _, seed := range core.BuiltinThemeSeeds {
		tokens, ok := core.FactoryTokens(seed.BuiltinKey)
		if !ok {
			return fmt.Errorf("missing factory tokens for %q", seed.BuiltinKey)
		}
		raw, err := core.MarshalThemeTokens(tokens)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO themes (id, name, builtin_key, tokens, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			seed.ID, seed.Name, seed.BuiltinKey, raw, now, now,
		); err != nil {
			return fmt.Errorf("seed theme %s: %w", seed.BuiltinKey, err)
		}
	}
	if _, err := tx.Exec(
		`INSERT INTO meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO NOTHING`,
		core.MetaActiveThemeID, core.BuiltinSlateID,
	); err != nil {
		return fmt.Errorf("set active_theme_id: %w", err)
	}
	return nil
}

// backfillEmptyTodoSessions seeds todo_session for rows still empty, and rewrites
// auto-seeded legacy values that contained spaces ("{shortID} - {title}") to the
// current space-free slug. Custom non-empty sessions are left alone.
func backfillEmptyTodoSessions(tx *sql.Tx) error {
	rows, err := tx.Query(`SELECT id, title, todo_session FROM tasks`)
	if err != nil {
		return fmt.Errorf("select todo_session rows: %w", err)
	}
	defer rows.Close()

	type row struct{ id, title, session string }
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.title, &r.session); err != nil {
			return err
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`UPDATE tasks SET todo_session = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range pending {
		short := core.ShortID(r.id)
		next := core.DefaultTodoSession(short, r.title)
		if next == "" {
			continue
		}
		cur := strings.TrimSpace(r.session)
		if cur != "" && cur != core.LegacyTodoSession(short, r.title) {
			continue
		}
		if cur == next {
			continue
		}
		if _, err := stmt.Exec(next, r.id); err != nil {
			return fmt.Errorf("backfill todo_session for %s: %w", r.id, err)
		}
	}
	return nil
}

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
		if strings.TrimSpace(m.up) != "" {
			if _, err := tx.Exec(m.up); err != nil {
				return fmt.Errorf("apply migration v%d: %w", m.version, err)
			}
		}
		if m.fn != nil {
			if err := m.fn(tx); err != nil {
				return fmt.Errorf("apply migration v%d fn: %w", m.version, err)
			}
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
