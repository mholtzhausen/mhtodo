# mhtodo — progress (updated 26-08-27)

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
