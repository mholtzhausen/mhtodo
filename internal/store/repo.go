package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"mhtodo/internal/core"
)

// TaskRepo is the SQLite implementation of core.TaskRepository. All writes are
// single statements; concurrency safety comes from WAL + busy_timeout, not
// application locks.
type TaskRepo struct {
	db *sql.DB
}

var _ core.TaskRepository = (*TaskRepo)(nil)

func NewTaskRepo(db *sql.DB) *TaskRepo { return &TaskRepo{db: db} }

const taskColumns = `id, title, description, status, progress, created_at, updated_at, completed_at`

// --- time helpers (RFC3339 UTC strings in the DB) ---------------------------

func formatTS(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func parseTS(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", s, err)
	}
	return t.UTC(), nil
}

func scanTask(row interface{ Scan(...any) error }) (core.Task, error) {
	var (
		t         core.Task
		status    string
		createdAt string
		updatedAt string
		completed sql.NullString
	)
	if err := row.Scan(&t.ID, &t.Title, &t.Description, &status, &t.Progress,
		&createdAt, &updatedAt, &completed); err != nil {
		return core.Task{}, err
	}
	t.Status = core.Status(status)
	var err error
	if t.CreatedAt, err = parseTS(createdAt); err != nil {
		return core.Task{}, err
	}
	if t.UpdatedAt, err = parseTS(updatedAt); err != nil {
		return core.Task{}, err
	}
	if completed.Valid {
		ct, err := parseTS(completed.String)
		if err != nil {
			return core.Task{}, err
		}
		t.CompletedAt = &ct
	}
	return t, nil
}

// --- core.TaskRepository ----------------------------------------------------

func (r *TaskRepo) Create(ctx context.Context, t core.Task) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (`+taskColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Description, string(t.Status), t.Progress,
		formatTS(t.CreatedAt), formatTS(t.UpdatedAt), nullTS(t.CompletedAt))
	return err
}

func (r *TaskRepo) GetByID(ctx context.Context, id string) (core.Task, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Task{}, core.ErrNotFound
	}
	return t, err
}

func (r *TaskRepo) FindByPrefix(ctx context.Context, prefix string) ([]core.Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE id LIKE ? ESCAPE '\' ORDER BY id`,
		escapeLike(prefix)+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TaskRepo) List(ctx context.Context, f core.ListFilter) ([]core.Task, error) {
	var conds []string
	var args []any
	if f.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, string(f.Status))
	} else if !f.IncludeDone {
		conds = append(conds, `status <> 'done'`)
	}
	if strings.TrimSpace(f.Search) != "" {
		pat := "%" + escapeLike(strings.TrimSpace(f.Search)) + "%"
		conds = append(conds, `(title LIKE ? ESCAPE '\' OR description LIKE ? ESCAPE '\')`)
		args = append(args, pat, pat)
	}

	q := `SELECT ` + taskColumns + ` FROM tasks`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY " + sortClause(f.Sort, f.Ascending)
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", f.Limit) // int → safe to inline
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TaskRepo) Update(ctx context.Context, t core.Task) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET title = ?, description = ?, status = ?, progress = ?,
		 completed_at = ?, updated_at = ? WHERE id = ?`,
		t.Title, t.Description, string(t.Status), t.Progress,
		nullTS(t.CompletedAt), formatTS(t.UpdatedAt), t.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return core.ErrNotFound
	}
	return nil
}

// Delete removes the task and returns it in one statement (RETURNING).
func (r *TaskRepo) Delete(ctx context.Context, id string) (core.Task, error) {
	row := r.db.QueryRowContext(ctx,
		`DELETE FROM tasks WHERE id = ? RETURNING `+taskColumns, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Task{}, core.ErrNotFound
	}
	return t, err
}

// --- helpers -----------------------------------------------------------------

func nullTS(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTS(*t)
}

// escapeLike escapes LIKE wildcards so user input matches literally.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

var sortColumns = map[string]string{
	"created":  "created_at",
	"updated":  "updated_at",
	"status":   "status",
	"progress": "progress",
	"title":    "title",
}

// sortClause whitelists the sort field (no user SQL) and appends id as a
// deterministic tiebreaker. Unknown fields fall back to updated_at.
func sortClause(field string, ascending bool) string {
	col, ok := sortColumns[field]
	if !ok {
		col = "updated_at"
	}
	dir := "DESC"
	if ascending {
		dir = "ASC"
	}
	return col + " " + dir + ", id " + dir
}
