# AGENTS.md — mhtodo plan folder conventions

This folder holds the current implementation plan. It has two parts:

## Overall progress — `PROGRESS.md`

`[PROGRESS.md](PROGRESS.md)` is the single source of truth for where the project stands. It is a
checkbox summary of every task, kept current as work lands: tick boxes, and update the date + status
line at the top. Write it to be copy-pasted into Slack verbatim — keep it plain-text pasteable.

Inspired by the format used while building v0.1/v0.2: group progress by release/version heading, list
each task with sub-checkpoints under it, and end with a "Notes / blockers" section for findings that
mattered (machine-specific gotchas, design decisions, things deferred). Example shape:

```
# mhtodo — progress (updated yy-mm-dd)

## v0.3 — <feature name> (<status>)

- [ ] Task title
  - [ ] sub-checkpoint
  - [ ] sub-checkpoint
- [x] Completed task
  - [x] detail

## Notes / blockers
- <finding or blocker>
```

## Task detail files — numbered, in this folder

Each task gets its own **numbered detail file** in this same folder (e.g. `01-*.md`, `02-*.md`, …).
`PROGRESS.md` tracks the checkbox-level status; these files hold the full design, decisions, and
notes for each piece. Keep the numbers stable so links don't rot as the plan grows.

| File | Contents |
|---|---|
| `README.md` | Goal of this plan + initial-planning discussion (start here) |
| `PROGRESS.md` | Overall checkbox progress (this file's conventions live in AGENTS.md) |
| `01-*.md`, … | Per-task detail files (one numbered file per task) |

## Working with this folder

- Read `README.md` first to understand the goal, then `PROGRESS.md` for current status.
- Update `PROGRESS.md` whenever a task's state changes — it is what gets pasted into Slack.
- Add or renumber detail files as needed; keep their numbers stable once linked.
