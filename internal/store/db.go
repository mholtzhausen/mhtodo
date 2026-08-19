package store

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registered as "sqlite"
)

// DBPath resolves the database file location: $MHTODO_DB_PATH (full path)
// first, then $XDG_DATA_HOME/mhtodo/mhtodo.db (default ~/.local/share).
func DBPath() string {
	if p := os.Getenv("MHTODO_DB_PATH"); p != "" {
		return p
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			base = ".local/share" // last resort; Open will surface the error
		} else {
			base = filepath.Join(home, ".local", "share")
		}
	}
	return filepath.Join(base, "mhtodo", "mhtodo.db")
}

// Open opens (creating if needed) the SQLite database at path with WAL mode
// and a 5s busy timeout — the concurrency contract for CLI+GUI coexistence.
// Migrations run on open and are idempotent. The directory is created 0700
// and the file 0600 on first run.
func Open(path string) (*TaskRepo, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}
	}
	// Pre-create with restrictive perms before the driver sees the path.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open db file: %w", err)
	}
	f.Close()
	if err := os.Chmod(path, 0o600); err != nil { // umask may have narrowed the create mode
		return nil, fmt.Errorf("chmod db file: %w", err)
	}

	dsn := (&url.URL{Scheme: "file", Path: path}).RequestURI() +
		"?_journal_mode=wal&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxIdleConns(2)

	if err := applyMigrations(db); err != nil {
		db.Close()
		return nil, err
	}
	repo := NewTaskRepo(db)
	if err := repo.checkWAL(); err != nil {
		db.Close()
		return nil, err
	}
	return repo, nil
}

// Close releases the underlying pool.
func (r *TaskRepo) Close() error { return r.db.Close() }

// checkWAL fails fast if journal_mode did not take (e.g. read-only fs).
func (r *TaskRepo) checkWAL() error {
	var mode string
	if err := r.db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		return fmt.Errorf("query journal_mode: %w", err)
	}
	if mode != "wal" {
		return fmt.Errorf("journal_mode is %q, want wal", mode)
	}
	return nil
}
