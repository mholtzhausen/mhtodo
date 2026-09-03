package store

import (
	"context"
	"database/sql"
	"errors"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"mhtodo/internal/core"
)

// --- task templates (v0.5) ---------------------------------------------------

const templateColumns = `id, name, title_prefix, description, status, cwd, slack_thread, human_only, include_in_report, created_at, updated_at`

func scanTemplate(row interface{ Scan(...any) error }) (core.Template, error) {
	var (
		t               core.Template
		titlePrefix     sql.NullString
		description     sql.NullString
		status          sql.NullString
		cwd             sql.NullString
		slackThread     sql.NullString
		humanOnly       sql.NullInt64
		includeInReport sql.NullInt64
		createdAt       string
		updatedAt       string
	)
	if err := row.Scan(&t.ID, &t.Name, &titlePrefix, &description, &status, &cwd, &slackThread,
		&humanOnly, &includeInReport, &createdAt, &updatedAt); err != nil {
		return core.Template{}, err
	}
	t.TitlePrefix = strPtr(titlePrefix)
	t.Description = strPtr(description)
	if status.Valid {
		st := core.Status(status.String)
		t.Status = &st
	}
	t.Cwd = strPtr(cwd)
	t.SlackThread = strPtr(slackThread)
	t.HumanOnly = intBoolPtr(humanOnly)
	t.IncludeInReport = intBoolPtr(includeInReport)
	var err error
	if t.CreatedAt, err = parseTS(createdAt); err != nil {
		return core.Template{}, err
	}
	if t.UpdatedAt, err = parseTS(updatedAt); err != nil {
		return core.Template{}, err
	}
	return t, nil
}

// templateArgs flattens a template into the templateColumns order minus the id
// and timestamps, which the callers supply positionally.
func templateArgs(t core.Template) []any {
	return []any{
		t.Name,
		nullStr(t.TitlePrefix),
		nullStr(t.Description),
		nullStatus(t.Status),
		nullStr(t.Cwd),
		nullStr(t.SlackThread),
		nullBool(t.HumanOnly),
		nullBool(t.IncludeInReport),
	}
}

func (r *TaskRepo) CreateTemplate(ctx context.Context, t core.Template) error {
	args := append([]any{t.ID}, templateArgs(t)...)
	args = append(args, formatTS(t.CreatedAt), formatTS(t.UpdatedAt))
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO task_templates (`+templateColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		args...)
	if isUniqueViolation(err) {
		return &core.DuplicateTemplateNameError{Name: t.Name}
	}
	return err
}

func (r *TaskRepo) GetTemplateByID(ctx context.Context, id string) (core.Template, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+templateColumns+` FROM task_templates WHERE id = ?`, id)
	t, err := scanTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Template{}, core.ErrTemplateNotFound
	}
	return t, err
}

// GetTemplateByName matches case-insensitively (the name column is NOCASE).
func (r *TaskRepo) GetTemplateByName(ctx context.Context, name string) (core.Template, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+templateColumns+` FROM task_templates WHERE name = ?`, name)
	t, err := scanTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Template{}, core.ErrTemplateNotFound
	}
	return t, err
}

// ListTemplates returns every template ordered by name, which is also the order
// the settings sidebar renders them in.
func (r *TaskRepo) ListTemplates(ctx context.Context) ([]core.Template, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+templateColumns+` FROM task_templates ORDER BY name COLLATE NOCASE ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateTemplate replaces every preset field (nil clears it) by t.ID.
func (r *TaskRepo) UpdateTemplate(ctx context.Context, t core.Template) error {
	args := append(templateArgs(t), formatTS(t.UpdatedAt), t.ID)
	res, err := r.db.ExecContext(ctx,
		`UPDATE task_templates SET name = ?, title_prefix = ?, description = ?, status = ?,
		 cwd = ?, slack_thread = ?, human_only = ?, include_in_report = ?, updated_at = ?
		 WHERE id = ?`, args...)
	if isUniqueViolation(err) {
		return &core.DuplicateTemplateNameError{Name: t.Name}
	}
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return core.ErrTemplateNotFound
	}
	return nil
}

// DeleteTemplate removes the template and returns it in one statement.
func (r *TaskRepo) DeleteTemplate(ctx context.Context, id string) (core.Template, error) {
	row := r.db.QueryRowContext(ctx,
		`DELETE FROM task_templates WHERE id = ? RETURNING `+templateColumns, id)
	t, err := scanTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Template{}, core.ErrTemplateNotFound
	}
	return t, err
}

// --- helpers -----------------------------------------------------------------

func strPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}

func intBoolPtr(n sql.NullInt64) *bool {
	if !n.Valid {
		return nil
	}
	v := n.Int64 != 0
	return &v
}

func nullBool(b *bool) any {
	if b == nil {
		return nil
	}
	return boolInt(*b)
}

func nullStatus(s *core.Status) any {
	if s == nil {
		return nil
	}
	return string(*s)
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure,
// which is how a duplicate template name surfaces from the NOCASE UNIQUE index.
func isUniqueViolation(err error) bool {
	var serr *sqlite.Error
	if errors.As(err, &serr) {
		return serr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
	}
	return false
}
