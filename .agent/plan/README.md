# mhtodo — Implementation Plan (v0.3)

## Goal

Ship **v0.3**: one-level nested sub-tasks with show/hide, agent-authored activity/comment
entries plus a global Activity view, detail-pane pin, new `review` status, and a rebalanced
list layout — full CLI ↔ GUI parity over `internal/core.Service`.

## Decisions (locked)

- **Sub-tasks:** One level only. Children are full tasks (`parent_id`) with their own
  status/progress. Board: children nest under the parent card (never own column cards).
  List: indent under parent when shown. No progress rollup. Show/hide toggle (persisted).
- **Activity:** Agent/user-authored only — **no auto events**. Each entry has `activity`
  and `comment` text fields (at least one required), plus `id`, `task_id`, `created_at`.
- **Status order:** `pending → wip → waiting → review → done`. Notify on →done / →waiting only.
- **Delete parent:** Cascade-delete children; confirm dialog names the child count.

## Files in this plan

| File | Contents |
|---|---|
| `README.md` | Goal + decisions (this file) |
| `PROGRESS.md` | Checkbox progress |
| `01-schema-and-core.md` | Migration v3, parent_id, review, activity table, Service API |
| `02-cli-contract.md` | Flags, JSON, exit codes, examples |
| `03-subtasks-ui.md` | Nesting, show/hide, cascade delete UX |
| `04-activity-ui.md` | Detail composer + Activity view + ticket filter |
| `05-detail-pin-and-list.md` | Pin layout + list columns + format helpers |
| `06-review-status.md` | Enum, board/filter/keys, CHECK rebuild |
| `07-docs-version.md` | README agent contract, AGENTS.md scope, VERSION bump |

## Milestone order

1. Schema v3 + core + unit tests
2. CLI + golden tests
3. GUI: review + list columns
4. GUI: sub-tasks nesting + show/hide
5. GUI: detail pin
6. GUI: activity composer + Activity view
7. Docs + VERSION + `make test` / `make build`
