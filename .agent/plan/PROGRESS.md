# mhtodo — progress (updated 26-08-27)

## v0.3 — Sub-tasks, activity, review, UI polish (complete)

- [x] Schema v3 + core (parent_id, review, activity table, Service API)
- [x] CLI contract (--parent, review, activity add/list/rm, goldens)
- [x] GUI: review status + rebalanced list columns
- [x] GUI: sub-tasks nesting + show/hide toggle
- [x] GUI: detail pin (overlay vs right pane)
- [x] GUI: activity composer + Activity view
- [x] Docs / VERSION / make test+build

## Notes / blockers

- SQLite CHECK rebuild for `review` + `parent_id` requires table copy (migration v3).
- `foreign_keys=ON` required for cascade delete of sub-tasks and activity rows.
- Wails `generate module` may focus a running GUI instance; bindings updated by hand when needed.
- `mhtodo ai` (integration contract v3) embeds `internal/cli/ai.md` and interpolates version/DB/enums at emit time.
