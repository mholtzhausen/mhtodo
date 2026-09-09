package store

import (
	"context"
	"database/sql"
	"errors"

	"mhtodo/internal/core"
)

// --- themes (v0.6) -----------------------------------------------------------

const themeColumns = `id, name, builtin_key, tokens, created_at, updated_at`

func scanTheme(row interface{ Scan(...any) error }) (core.Theme, error) {
	var (
		t          core.Theme
		builtinKey sql.NullString
		tokensRaw  string
		createdAt  string
		updatedAt  string
	)
	if err := row.Scan(&t.ID, &t.Name, &builtinKey, &tokensRaw, &createdAt, &updatedAt); err != nil {
		return core.Theme{}, err
	}
	if builtinKey.Valid {
		v := builtinKey.String
		t.BuiltinKey = &v
	}
	tokens, err := core.UnmarshalThemeTokens(tokensRaw)
	if err != nil {
		return core.Theme{}, err
	}
	t.Tokens = tokens
	if t.CreatedAt, err = parseTS(createdAt); err != nil {
		return core.Theme{}, err
	}
	if t.UpdatedAt, err = parseTS(updatedAt); err != nil {
		return core.Theme{}, err
	}
	return t, nil
}

func (r *TaskRepo) CreateTheme(ctx context.Context, t core.Theme) error {
	raw, err := core.MarshalThemeTokens(t.Tokens)
	if err != nil {
		return err
	}
	var builtin any
	if t.BuiltinKey != nil {
		builtin = *t.BuiltinKey
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO themes (`+themeColumns+`) VALUES (?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, builtin, raw, formatTS(t.CreatedAt), formatTS(t.UpdatedAt))
	if isUniqueViolation(err) {
		return &core.DuplicateThemeNameError{Name: t.Name}
	}
	return err
}

func (r *TaskRepo) GetThemeByID(ctx context.Context, id string) (core.Theme, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+themeColumns+` FROM themes WHERE id = ?`, id)
	t, err := scanTheme(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Theme{}, core.ErrThemeNotFound
	}
	return t, err
}

func (r *TaskRepo) GetThemeByName(ctx context.Context, name string) (core.Theme, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+themeColumns+` FROM themes WHERE name = ?`, name)
	t, err := scanTheme(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Theme{}, core.ErrThemeNotFound
	}
	return t, err
}

func (r *TaskRepo) ListThemes(ctx context.Context) ([]core.Theme, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+themeColumns+` FROM themes ORDER BY name COLLATE NOCASE ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Theme
	for rows.Next() {
		t, err := scanTheme(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TaskRepo) UpdateTheme(ctx context.Context, t core.Theme) error {
	raw, err := core.MarshalThemeTokens(t.Tokens)
	if err != nil {
		return err
	}
	var builtin any
	if t.BuiltinKey != nil {
		builtin = *t.BuiltinKey
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE themes SET name = ?, builtin_key = ?, tokens = ?, updated_at = ? WHERE id = ?`,
		t.Name, builtin, raw, formatTS(t.UpdatedAt), t.ID)
	if isUniqueViolation(err) {
		return &core.DuplicateThemeNameError{Name: t.Name}
	}
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return core.ErrThemeNotFound
	}
	return nil
}

func (r *TaskRepo) DeleteTheme(ctx context.Context, id string) (core.Theme, error) {
	row := r.db.QueryRowContext(ctx,
		`DELETE FROM themes WHERE id = ? RETURNING `+themeColumns, id)
	t, err := scanTheme(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Theme{}, core.ErrThemeNotFound
	}
	return t, err
}
