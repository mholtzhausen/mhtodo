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

const taskColumns = `id, title, description, feedback, status, progress, created_at, updated_at, completed_at, archived_at, parent_id, board_rank`

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
		archived  sql.NullString
		parent    sql.NullString
		boardRank sql.NullFloat64
	)
	if err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Feedback, &status, &t.Progress,
		&createdAt, &updatedAt, &completed, &archived, &parent, &boardRank); err != nil {
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
	if archived.Valid {
		at, err := parseTS(archived.String)
		if err != nil {
			return core.Task{}, err
		}
		t.ArchivedAt = &at
	}
	if parent.Valid {
		pid := parent.String
		t.ParentID = &pid
	}
	if boardRank.Valid {
		r := boardRank.Float64
		t.BoardRank = &r
	}
	return t, nil
}

// --- core.TaskRepository ----------------------------------------------------

func (r *TaskRepo) Create(ctx context.Context, t core.Task) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (`+taskColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Description, t.Feedback, string(t.Status), t.Progress,
		formatTS(t.CreatedAt), formatTS(t.UpdatedAt), nullTS(t.CompletedAt), nullTS(t.ArchivedAt),
		nullStr(t.ParentID), nullFloat(t.BoardRank))
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
	} else if !f.IncludeDone && !f.Archived {
		// The archived view shows everything that is archived (almost always
		// done), so the default done-exclusion does not apply there.
		conds = append(conds, `status <> 'done'`)
	}
	if f.Archived {
		conds = append(conds, "archived_at IS NOT NULL")
	} else {
		conds = append(conds, "archived_at IS NULL") // archived tasks are hidden unless explicitly requested
	}
	if f.RootsOnly {
		conds = append(conds, "parent_id IS NULL")
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
		`UPDATE tasks SET title = ?, description = ?, feedback = ?, status = ?, progress = ?,
		 completed_at = ?, archived_at = ?, parent_id = ?, board_rank = ?, updated_at = ? WHERE id = ?`,
		t.Title, t.Description, t.Feedback, string(t.Status), t.Progress,
		nullTS(t.CompletedAt), nullTS(t.ArchivedAt), nullStr(t.ParentID), nullFloat(t.BoardRank),
		formatTS(t.UpdatedAt), t.ID)
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
// Children are removed by ON DELETE CASCADE when foreign_keys are on.
func (r *TaskRepo) Delete(ctx context.Context, id string) (core.Task, error) {
	row := r.db.QueryRowContext(ctx,
		`DELETE FROM tasks WHERE id = ? RETURNING `+taskColumns, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Task{}, core.ErrNotFound
	}
	return t, err
}

// ArchiveDone archives every non-archived done task and returns them.
func (r *TaskRepo) ArchiveDone(ctx context.Context, at time.Time) ([]core.Task, error) {
	ts := formatTS(at)
	rows, err := r.db.QueryContext(ctx,
		`UPDATE tasks SET archived_at = ?, updated_at = ?
		 WHERE status = 'done' AND archived_at IS NULL
		 RETURNING `+taskColumns, ts, ts)
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

// CountOpen counts tasks not in done status (tray tooltip; no CLI equivalent).
func (r *TaskRepo) CountOpen(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE status <> 'done'`).Scan(&n)
	return n, err
}

func (r *TaskRepo) CountChildren(ctx context.Context, parentID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE parent_id = ?`, parentID).Scan(&n)
	return n, err
}

// MaxBoardRank returns the highest board_rank among root tasks in a status column.
// ok is false when no ranked roots exist in that column.
func (r *TaskRepo) MaxBoardRank(ctx context.Context, status core.Status) (max float64, ok bool, err error) {
	var maxNull sql.NullFloat64
	err = r.db.QueryRowContext(ctx,
		`SELECT MAX(board_rank) FROM tasks WHERE status = ? AND parent_id IS NULL AND board_rank IS NOT NULL`,
		string(status)).Scan(&maxNull)
	if err != nil {
		return 0, false, err
	}
	if !maxNull.Valid {
		return 0, false, nil
	}
	return maxNull.Float64, true, nil
}

// ListRootsInStatus returns root tasks in a status column ordered by board sort.
func (r *TaskRepo) ListRootsInStatus(ctx context.Context, status core.Status) ([]core.Task, error) {
	return r.List(ctx, core.ListFilter{
		Status:    status,
		Sort:      "board",
		Ascending: false,
		RootsOnly: true,
		IncludeDone: true,
	})
}

// UpdateBoardRank sets board_rank without touching updated_at (pure reorder).
func (r *TaskRepo) UpdateBoardRank(ctx context.Context, id string, rank float64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE tasks SET board_rank = ? WHERE id = ?`, rank, id)
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

// --- activity ----------------------------------------------------------------

const activityColumns = `id, task_id, activity, comment, created_at`

func scanActivity(row interface{ Scan(...any) error }) (core.Activity, error) {
	var (
		a         core.Activity
		createdAt string
	)
	if err := row.Scan(&a.ID, &a.TaskID, &a.Activity, &a.Comment, &createdAt); err != nil {
		return core.Activity{}, err
	}
	ts, err := parseTS(createdAt)
	if err != nil {
		return core.Activity{}, err
	}
	a.CreatedAt = ts
	return a, nil
}

func (r *TaskRepo) CreateActivity(ctx context.Context, a core.Activity) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO activity (`+activityColumns+`) VALUES (?, ?, ?, ?, ?)`,
		a.ID, a.TaskID, a.Activity, a.Comment, formatTS(a.CreatedAt))
	return err
}

func (r *TaskRepo) GetActivityByID(ctx context.Context, id string) (core.Activity, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+activityColumns+` FROM activity WHERE id = ?`, id)
	a, err := scanActivity(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Activity{}, core.ErrNotFound
	}
	return a, err
}

func (r *TaskRepo) FindActivityByPrefix(ctx context.Context, prefix string) ([]core.Activity, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+activityColumns+` FROM activity WHERE id LIKE ? ESCAPE '\' ORDER BY id`,
		escapeLike(prefix)+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Activity
	for rows.Next() {
		a, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *TaskRepo) ListActivity(ctx context.Context, f core.ActivityFilter) ([]core.Activity, error) {
	var conds []string
	var args []any
	if !f.IncludeArchived {
		conds = append(conds, `EXISTS (SELECT 1 FROM tasks t WHERE t.id = activity.task_id AND t.archived_at IS NULL)`)
	}
	if len(f.TaskIDs) > 0 {
		ph := make([]string, len(f.TaskIDs))
		for i, id := range f.TaskIDs {
			ph[i] = "?"
			args = append(args, id)
		}
		conds = append(conds, "task_id IN ("+strings.Join(ph, ",")+")")
	}
	q := `SELECT ` + activityColumns + ` FROM activity`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY created_at DESC, id DESC"
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", f.Limit)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Activity
	for rows.Next() {
		a, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *TaskRepo) DeleteActivity(ctx context.Context, id string) (core.Activity, error) {
	row := r.db.QueryRowContext(ctx,
		`DELETE FROM activity WHERE id = ? RETURNING `+activityColumns, id)
	a, err := scanActivity(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Activity{}, core.ErrNotFound
	}
	return a, err
}

// --- helpers -----------------------------------------------------------------

func nullTS(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTS(*t)
}

func nullStr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func nullFloat(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

// escapeLike escapes LIKE wildcards so user input matches literally.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

var sortColumns = map[string]string{
	"created":  "created_at",
	"updated":  "updated_at",
	"status":   "status",
	"progress": "progress",
	"title":    "title",
}

const boardStatusOrder = `CASE status
  WHEN 'pending' THEN 0 WHEN 'wip' THEN 1 WHEN 'waiting' THEN 2
  WHEN 'review' THEN 3 WHEN 'done' THEN 4 ELSE 5
END`

// sortClause whitelists the sort field (no user SQL) and appends id as a
// deterministic tiebreaker. Unknown fields fall back to board order.
func sortClause(field string, ascending bool) string {
	if field == "" || field == "board" {
		rankDir := "ASC"
		updatedDir := "DESC"
		idDir := "DESC"
		if ascending {
			rankDir = "DESC"
			updatedDir = "ASC"
			idDir = "ASC"
		}
		return boardStatusOrder + " ASC, board_rank " + rankDir + " NULLS LAST, updated_at " + updatedDir + ", id " + idDir
	}
	col, ok := sortColumns[field]
	if !ok {
		return boardStatusOrder + " ASC, board_rank ASC NULLS LAST, updated_at DESC, id DESC"
	}
	dir := "DESC"
	if ascending {
		dir = "ASC"
	}
	return col + " " + dir + ", id " + dir
}
