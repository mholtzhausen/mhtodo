# 01 — Schema v3 and core Service

## Migration v3

SQLite cannot ALTER a CHECK — rebuild `tasks` in one transaction:

- Copy all columns including `archived_at`
- Status CHECK: `('pending','wip','done','waiting','review')`
- Add `parent_id TEXT NULL REFERENCES tasks(id) ON DELETE CASCADE`
- Index `idx_tasks_parent`
- New `activity` table (id, task_id, activity, comment, created_at) + indexes

## Core

- `StatusReview = "review"`; update `allStatuses` and error text
- `Task.ParentID *string` JSON `parent_id`
- `CreateInput.ParentID`; one-level enforcement (parent must be a root)
- `ListFilter.RootsOnly` for optional root-only lists
- Activity types + `AddActivity` / `ListActivity` / `DeleteActivity`
- Cascade delete via FK; CountChildren for confirm messaging
- Repo: scan/insert/update parent_id; ActivityRepo methods on same DB
