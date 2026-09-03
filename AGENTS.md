# AGENTS.md — mhtodo

## What this project is

**mhtodo** is a personal todo manager written in Go with two frontends over one shared core:

- **CLI** (`mhtodo add|list|show|edit|status|done|reorder|rm|path|slack|ai|update`) — the interface for **agentic tool
  access**. Scriptable, `--json` everywhere, stable exit codes and JSON field names (a documented contract).
  `mhtodo ai` emits the install/upgrade contract for wiring this app into an agent host.
  `mhtodo update` checks GitHub Releases and installs in place (restarts the user systemd unit when present).
- **GUI** (Wails v2 webview + system tray) — the human view. Board/list views, task detail editing,
  desktop notifications, live sync so CLI changes appear without restart.

One binary: bare `mhtodo` launches the GUI; any subcommand runs the CLI and exits. All data lives in a
single SQLite database at `$XDG_DATA_HOME/mhtodo/mhtodo.db` (override: `MHTODO_DB_PATH`). Linux-first.

## Scope

**In scope (v0.1):**
- Task fields: title, description, feedback (agent-authored; GUI shows when non-empty),
  status (`pending | wip | waiting | review | done`), progress 0–100,
  created_at / updated_at / completed_at. UUIDv7 IDs with short-prefix lookup.
- Full CLI ↔ GUI feature parity (the bound-API table in the plan is the contract).
- System tray: show/hide window, new task, quit; close-to-tray behavior; single-instance lock.
  GUI also supports always-on-top (persisted in DB `meta`), Esc-to-hide, and a global X11 hotkey
  (`Ctrl+Shift+Alt+T`, hardcoded for now) to toggle show/hide and raise the window.
- Desktop notifications on →done and →waiting (`notify-send`).
- Comprehensive Makefile: `dev`, `build`, `test`, `lint`, `release` (linux amd64/arm64), `install`.

**Post-v0.1 (v0.2, shipped 2026-08-20):** archive/unarchive for done tasks — bulk archive from the
board's Done column, `mhtodo archive` / `unarchive`, archived filter in list view; see plan docs + `.agent/plan/PROGRESS.md`.
Single-task archive: `mhtodo archive ID` and the card archive action (done tasks only).

**v0.3 (shipped):** one-level sub-tasks (`parent_id`), agent-authored activity/comment entries +
Activity view, detail-pane pin, `review` status (after waiting), rebalanced list columns. See
[`.agent/plan/`](.agent/plan/README.md).

**Also:** `mhtodo ai` prints the agent-integration contract (embedded `internal/cli/ai.md`,
interpolated at emit time). `mhtodo update` self-updates from GitHub Releases (see README).

**Board order (v0.4):** root tasks have optional `board_rank` (migration v5). Board and list default
sort is `board` (status workflow → rank → `updated_at`). GUI: drag root cards within a column to
reorder; cross-column drag changes status (appends to target column). CLI: `mhtodo reorder`.

**Task templates (v0.5):** named sets of task presets (`title_prefix`, `description`, `status`, `cwd`,
`slack_thread`, `human_only`, `include_in_report`) in a `task_templates` table (migration v9). Every
preset column is NULLable: `NULL` means "not part of this template" and the normal default applies,
while an empty string or an explicit `false` is a real override. Authored in Settings → Task Templates
(per-template sub-nav); applied from the template picker in the new-task dialog, the header split
button, or the tray's *New Task from Template*; captured via *Save as template* from the new-task and
task-detail headers. **GUI-only for now** — a deliberate, temporary break from the parity rule below.
All rules live in `core.Service` (including `CreateFromTemplate`), so `mhtodo template …` and
`add --template` are a thin later add. See [`.agent/plan/08-task-templates.md`](.agent/plan/08-task-templates.md).

**Out of scope (stretch):** cross-column insert index, sub-task reorder, list-view drag reorder,
light theme, Windows/macOS support, tags/labels/projects, due dates/reminders,
CLI surface for task templates.

**GUI display:** description, feedback, and activity comments are markdown-rendered
(when not in an input/textarea). Detail-pane description & feedback grow with content
up to 500px, then scroll.

## Hard constraints

- **Build tags:** this distro ships webkit2gtk-4.1 only → all Go builds need `-tags webkit2_41`
  (Makefile `TAGS` variable). Do not remove it without checking `pkg-config --list-all | grep webkit`.
  GUI binaries also need a Wails *mode* tag: `wails build`/`wails dev` inject `production`/`dev`
  automatically; plain `go build` must add `-tags "webkit2_41 production"` or the binary fails at runtime.
- **Parity is structural:** CLI and GUI both call the same `internal/core.Service`; neither may contain
 business rules or SQL of its own. New capability = core method + CLI command + bound GUI method.
 (One tracked exception: task templates ship GUI-only in v0.5; the core methods exist, the CLI commands
 do not yet.)
- **DB concurrency:** WAL mode + busy_timeout; single-statement transactions only (CLI and GUI run concurrently).
- **Agent contract stability:** CLI JSON field names, flags, and exit codes are API — change deliberately and document in README.

## The plan

The current implementation plan lives in **[`.agent/plan/README.md`](.agent/plan/README.md)** (start here).
Task detail files are numbered and live in the same folder — see [`.agent/plan/AGENTS.md`](.agent/plan/AGENTS.md)
for how this folder is organized.

**Build in milestone order.** The first milestone is a system-tray integration spike and must pass before any
other work — tray + Wails dual GTK main loops on Linux are the top project risk.

## Progress tracking

Progress against this plan lives in **[`.agent/plan/PROGRESS.md`](.agent/plan/PROGRESS.md)**: a checkbox
summary of every task, kept current as work lands (tick boxes, update the date/status line). It is written to be
copy-pasted into Slack verbatim for team updates — keep it plain-text pasteable. See [`.agent/plan/AGENTS.md`](.agent/plan/AGENTS.md)
for how this file and the plan folder are structured.

## Working agreements for agents

- Run `make test` after touching `internal/core`, `internal/store`, or `internal/cli`; golden tests use
  temp-dir DBs via `MHTODO_DB_PATH`.
- The repo is empty until M1 — do not scaffold ahead of the milestone you are on.
- Keep this file updated when scope, constraints, or the plan location changes.
