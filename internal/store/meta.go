package store

import (
	"context"
	"database/sql"
	"fmt"

	"mhtodo/internal/core"
)

// Meta keys for app preferences (alongside schema_version). Values are opaque
// strings; callers define the encoding (e.g. "true"/"false").
const (
	MetaAlwaysOnTop = "always_on_top"
	MetaWindowPos   = "window_pos" // "x,y" — last shown position
	// MetaShowOnNextStart ("true") is a one-shot: the next GUI start shows the
	// window even when Start hidden is set (in-app service update restart).
	MetaShowOnNextStart = "show_on_next_start"
	MetaActiveThemeID   = core.MetaActiveThemeID
)

// GetMeta returns the value for key. ok is false when the key is absent.
func (r *TaskRepo) GetMeta(ctx context.Context, key string) (value string, ok bool, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get meta %q: %w", key, err)
	}
	return value, true, nil
}

// SetMeta upserts a preference key. Single-statement write (WAL contract).
func (r *TaskRepo) SetMeta(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	if err != nil {
		return fmt.Errorf("set meta %q: %w", key, err)
	}
	return nil
}

// DeleteMeta removes a preference key (no-op when absent).
func (r *TaskRepo) DeleteMeta(ctx context.Context, key string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM meta WHERE key = ?`, key); err != nil {
		return fmt.Errorf("delete meta %q: %w", key, err)
	}
	return nil
}
