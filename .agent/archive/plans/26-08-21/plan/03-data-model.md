# Data Model & Storage

## Location

```
$XDG_DATA_HOME/mhtodo/mhtodo.db        # default: ~/.local/share/mhtodo/mhtodo.db
```

- Helper `store.DBPath()`: honors `$MHTODO_DB_PATH` (full file path) first, then XDG data dir.
  Env override exists for tests and for agents that want an isolated DB.
- Directory created 0700, db file 0600 on first run; migrations run at open (idempotent).

## Schema (v1 → v2)

```sql
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
  created_at   TEXT NOT NULL,               -- RFC3339 UTC, e.g. 2025-08-19T07:59:00Z
  updated_at   TEXT NOT NULL,
  completed_at TEXT                         -- set on →done, cleared when leaving done
);

CREATE INDEX idx_tasks_status  ON tasks(status);
CREATE INDEX idx_tasks_updated ON tasks(updated_at DESC);
```

**v2 (archive, v0.2):**

```sql
ALTER TABLE tasks ADD COLUMN archived_at TEXT; -- set on archive, cleared on unarchive
CREATE INDEX idx_tasks_archived ON tasks(archived_at);
```

Archived tasks are hidden from every default view (CLI `list` incl. `--all`, board columns) and only
appear via an explicit archived filter (`list --archived`, GUI List → Archived chip). Archive is
bulk-only: the Done column's archive button / `mhtodo archive` moves all done tasks at once; there is
no per-task archive in v0.2.

Timestamps stored as RFC3339 UTC strings (not unix ints) so CLI/JSON output is human-readable without conversion.

## IDs

- **UUIDv7** (`uuid.NewV7()`): time-ordered → `ORDER BY id` ≈ creation order; 26-char lowercase string.
- **Prefix lookup:** any command taking an ID accepts a unique prefix ≥ 4 chars (CLI and GUI).
  Ambiguous prefix → error listing candidates. Implemented once in `core`, used by both frontends.

## Status & progress semantics (enforced in `core.Service`)

| Transition | Effect |
|---|---|
| any → `done` | `progress = 100`, `completed_at = now` |
| `done` → anything else | `completed_at = NULL`; progress left as-is unless explicitly set |
| archive (bulk, done only) | `archived_at = now`; status/progress untouched; hidden from default views |
| unarchive | `status = pending`, `progress = 0`, `completed_at = NULL`, `archived_at = NULL` — a fresh start |
| any other change | `updated_at = now` always; `created_at` immutable |

- `waiting` is a first-class status (blocked on external dependency), not a flag.
- No transition restrictions beyond the above — agents may move tasks freely between states.
- Validation errors return typed errors (`ErrNotFound`, `ErrAmbiguousID`, `ErrInvalidStatus`) so CLI
  exit codes and GUI toasts can be specific.

## Migration strategy

Forward-only, versioned: `migrate.go` holds `[]migration{version int, up string}`; applied in a
transaction when `meta.schema_version < latest`. v1 ships as the initial migration (not inline DDL)
so the pattern is proven from day one.
