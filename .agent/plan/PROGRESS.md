# mhtodo — progress (updated 26-09-03)

## v0.5 — Task templates (complete)

- [x] Migration v9: `task_templates` table (every preset column NULLable = "not set")
  - [x] `COLLATE NOCASE UNIQUE` name; status CHECK constraint
  - [x] Store CRUD + duplicate-name mapped from SQLITE_CONSTRAINT_UNIQUE
- [x] Core: `Template` / `TemplateInput` pointer types, `Template.Apply`, full-replace update
  - [x] `ListTemplates` / `GetTemplate` (id or name) / Create / Update / Delete
  - [x] `CreateFromTemplate` so a later CLI has nothing to reimplement
  - [x] Unit tests: validation, duplicate names, unset-vs-explicit-false, apply overlay
- [x] Bound GUI methods + `templates:changed` event (bindings hand-written, see notes)
- [x] Settings: Task Templates section with per-template sub-nav + `+ New template`
  - [x] Per-field trash-can clears back to unset; own debounced autosave + flush on unmount
  - [x] `TriStateCheck` for human-only / include-in-report (unset / off / on)
- [x] New-task dialog: template icon + picker popover (`/` filter, arrows, chips with values)
  - [x] Title focused with caret after the prefix; switching templates swaps the prefix
- [x] Save as template from the new-task dialog and the task-detail header
- [x] Entry points: header split button, tray `New Task from Template`
- [x] Docs (README, AGENTS.md, `08-task-templates.md`)
- [ ] CLI `mhtodo template …` + `add --template` — deliberately deferred

## v0.4 — Board reorder (complete)

- [x] Migration v5: `board_rank` column + index
- [x] Core: `ReorderBoardTask`, rank on create/status/unarchive, `sort=board` default
- [x] CLI: `reorder` command, `list --sort board` default
- [x] GUI: same-lane drag reorder + board/list default sort
- [x] Tests + docs (README, AGENTS.md, ai.md)

## v0.3 — Sub-tasks, activity, review, UI polish (complete)

- [x] Schema v3 + core (parent_id, review, activity table, Service API)
- [x] CLI contract (--parent, review, activity add/list/rm, goldens)
- [x] GUI: review status + rebalanced list columns
- [x] GUI: sub-tasks nesting + show/hide toggle
- [x] GUI: detail pin (overlay vs right pane)
- [x] GUI: activity composer + Activity view
- [x] Docs / VERSION / make test+build

## Self-update — `mhtodo update` (complete)

- [x] Detect install path + optional user systemd unit
- [x] GitHub Releases latest + sha256 verify + in-place install
- [x] Service stop / rewrite unit / enable --now when attached
- [x] CLI flags `--check` / `--force` / `--json` + unit tests
- [x] Docs (README, AGENTS, CHANGELOG)

## Agent integration contract v4 (complete)

- [x] Adopt activity chip labels + step-forward granularity from updated instructions
- [x] Document `--feedback` as post-work summary + notes/takeaways
- [x] Document markdown for description / feedback / activity comments
- [x] §6 known reversals + IntegrationVersion bump to 4

## Notes / blockers

- SQLite CHECK rebuild for `review` + `parent_id` requires table copy (migration v3).
- `foreign_keys=ON` required for cascade delete of sub-tasks and activity rows.
- Wails `generate module` may focus a running GUI instance; bindings updated by hand when needed.
- `mhtodo ai` (integration contract v4) embeds `internal/cli/ai.md` and interpolates version/DB/enums at emit time.
- `mhtodo update` is CLI-only (no GUI parity); replaces via write-temp+rename; uses `GH_TOKEN`/`GITHUB_TOKEN` when set.
- Default list sort changed from `updated` to `board` (agent contract); scripts relying on implicit default should pass `--sort` explicitly.
- Task templates are GUI-only for now — a deliberate, temporary break from the CLI/GUI parity rule. All logic sits in `core.Service` (incl. `CreateFromTemplate`), so the CLI is a thin add when wanted.
- Template preset columns are NULLable so "not part of this template" stays distinct from an explicit empty string / false. Tasks keep the older NOT NULL + empty-default convention.
- `wails generate module` exits early when a GUI instance holds the single-instance lock; the v0.5 template bindings in `frontend/wailsjs/` were added by hand.
- `go fmt ./...` rewrites files that were never gofmt-clean (cli/edit.go, core/task.go, store/repo.go, …). Format only the files you touch.
