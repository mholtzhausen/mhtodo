# AGENTS.md — mhtodo

## What this project is

**mhtodo** is a personal todo manager written in Go with two frontends over one shared core:

- **CLI** (`mhtodo add|list|show|edit|status|done|reorder|rm|path|slack|integration|ai|install|update|service|template|theme`) — the interface for **agentic tool
 access**. Scriptable, `--json` everywhere, stable exit codes and JSON field names (a documented contract).
 `mhtodo ai` emits the install/upgrade contract for wiring this app into an agent host.
 `mhtodo install` copies this binary into `~/.local` (desktop + icon), then can install the user
 systemd unit and/or `claude.todo` shell helper (prompts on a TTY; flags for non-interactive).
 `mhtodo update` checks GitHub Releases and installs in place (restarts the user systemd unit when present).
 `mhtodo service install|stop|start|restart|uninstall` manages the user systemd unit for this install.
 `mhtodo integration bash|zsh` installs a managed `claude.todo` shell helper for `$MHTODO_SESSION`.
 `mhtodo template list|search|show|create|update|rm` manages task templates
 (search: fuzzy/regex + `--cwd`); `add --template` applies one when creating a task.
 `mhtodo theme list|search|show|create|update|rm|activate|duplicate|reset` manages GUI themes.
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
  GUI is frameless: drag the app header to move; double-click the header (outside tabs/actions)
  toggles maximize; header Close hides to tray and Quit exits (same as Esc / Ctrl+Q). Also
  supports always-on-top (persisted in DB `meta`), Esc-to-hide, and a global X11 hotkey
  (`Ctrl+Shift+Alt+T`, hardcoded for now) to toggle show/hide and raise the window.
  With always-on-top on, a successful Claude or Zed open hides the window to tray.
  Terminal spawn raises an existing Claude window by session PID / `mhtodo:<session>` title
  instead of opening a duplicate.
- Desktop notifications on →done and →waiting (`notify-send`).
- Comprehensive Makefile: `dev`, `build`, `test`, `lint`, `release` (linux amd64/arm64), `install`.

**Post-v0.1 (v0.2, shipped 2026-08-20):** archive/unarchive for done tasks — bulk archive from the
board's Done column, `mhtodo archive` / `unarchive`, archived filter in list view; see plan docs + `.agent/plan/PROGRESS.md`.
Single-task archive: `mhtodo archive ID` and the card archive action (done tasks only).

**v0.3 (shipped):** one-level sub-tasks (`parent_id`), agent-authored activity/comment entries +
Activity view, detail-pane pin, `review` status (after waiting), rebalanced list columns. See
[`.agent/plan/`](.agent/plan/README.md). New sub-tasks in the GUI seed cwd, todo session, Slack
thread, and human-only from the parent; `include_in_report` defaults to false (CLI `add --parent`
likewise).

**Also:** `mhtodo ai` prints the agent-integration contract (embedded `internal/cli/ai.md`,
interpolated at emit time). The live board signal for agents is status → progress →
sub-tasks (activities are audit); agents must ask before creating root tasks and
reopen `review` → `wip` with new sub-tasks when more work continues.
`mhtodo update` self-updates from GitHub Releases (see README).
`mhtodo service …` installs/controls/removes `~/.config/systemd/user/mhtodo.service` for the
running binary (from-source bootstrap remains `make service-install`).
`mhtodo install` is the user-facing folder install (`~/.local`) with optional service + shell
integration prompts.
Per-task `todo_session` (migration v10–v11) is a Claude session UUID (UUIDv7 on
create). Launch uses `claude --session-id <uuid> --name <slug> || claude --resume
<uuid> --name <slug>` (Herdr, system terminal, `claude.todo`). Display slug remains
`{short8}-{slug}` via `--name` / `MHTODO_SESSION_NAME`. Zed sets `MHTODO_SESSION`.
Empty or legacy spaced auto-seeds are backfilled on open; non-UUID values are
minted to a UUID on first Claude/Zed open. Claude spawn mode in Settings is `herdr` | `terminal` | `disabled`
(Herdr fields or terminal binary/env_start show conditionally). Terminal spawn opens Claude
in a system terminal emulator window (raise existing by `terminal_pid` when alive). Migration v12 adds `terminal_pid`
for the managed process. Zed remains a separate integration.

**Board order (v0.4):** root tasks have optional `board_rank` (migration v5). Board and list default
sort is `board` (status workflow → rank → `updated_at`). GUI: drag root cards within a column to
reorder; cross-column drag changes status (appends to target column). CLI: `mhtodo reorder`.
Nested sub-tasks on the board/list/detail are shown in creation order (oldest first), not board
`updated_at` order. Board cards have a top-right mark-done control that sets status to `done`
immediately. Board columns collapse to a slim vertical strip (rotated title + count); state is
persisted in `localStorage` (`mhtodo.collapsedColumns`).

**Task templates (v0.5):** named sets of task presets (`title_prefix`, `description`, `status`, `cwd`,
`slack_thread`, `human_only`, `include_in_report`) in a `task_templates` table (migration v9). Every
preset column is NULLable: `NULL` means "not part of this template" and the normal default applies,
while an empty string or an explicit `false` is a real override. Authored in Settings → Task Templates
(per-template sub-nav); applied from the template picker in the new-task dialog, the header split
button, or the tray's *New Task from Template*; captured via *Save as template* from the new-task and
task-detail headers. CLI: `mhtodo template list|search|show|create|update|rm`
(`search` supports `--mode fuzzy|regex` and `--cwd`) and `add --template REF`.
See [`.agent/plan/08-task-templates.md`](.agent/plan/08-task-templates.md).

**GUI themes (v0.6):** named sets of design tokens (colors, radii, spacing) in a `themes` table
(migration v13). Built-ins **Slate** (default active), **Paper** (light), and **Ember** (warm) are
editable with Reset-to-factory; Duplicate always available; built-ins cannot be deleted. Active
theme id lives in DB `meta`. Settings → Themes mirrors the Task Templates sub-nav with color and
length pickers. Runtime apply sets CSS variables on `documentElement`. CLI: `mhtodo theme
list|search|show|create|update|rm|activate|duplicate|reset`. See
[`.agent/plan/09-themes.md`](.agent/plan/09-themes.md).

**Out of scope (stretch):** cross-column insert index, sub-task reorder, list-view drag reorder,
Windows/macOS support, tags/labels/projects, due dates/reminders.

**GUI display:** description, feedback, and activity comments are markdown-rendered
(when not in an input/textarea). Detail-pane description & feedback grow with content
up to 500px, then scroll.

**GUI window / responsiveness (desktop):** Frameless Wails window (800×560 floor, default
1100×720); drag the header to move, double-click header (outside tabs/actions) to toggle
maximize. The board uses `minmax(200px, 1fr)` columns with horizontal scroll instead of
crushing five columns. Pinned detail auto-renders as floating when the main pane would be
under ~640px (preference unchanged). Footer shortcuts hide below ~900px. Reloads coalesce/
`tasks:changed` debounce; search is debounced; Claude/Zed readiness is cached once per
settings change (not per card).
Header Install icon (left of Settings) is enabled when a release update is available; hold
Ctrl while hovering to force-enable. Latest version is cached 60 minutes (hover refreshes when
stale; dialog has a manual refresh). Confirmation offers install/upgrade (+ as service) and
optional `mhtodo integration zsh|bash`.

## Hard constraints

- **Build tags:** this distro ships webkit2gtk-4.1 only → all Go builds need `-tags webkit2_41`
  (Makefile `TAGS` variable). Do not remove it without checking `pkg-config --list-all | grep webkit`.
  GUI binaries also need a Wails *mode* tag: `wails build`/`wails dev` inject `production`/`dev`
  automatically; plain `go build` must add `-tags "webkit2_41 production"` or the binary fails at runtime.
- **Parity is structural:** CLI and GUI both call the same `internal/core.Service`; neither may contain
 business rules or SQL of its own. New capability = core method + CLI command + bound GUI method.
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
