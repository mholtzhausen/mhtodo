# AGENTS.md — mhtodo

## What this project is

**mhtodo** is a personal todo manager written in Go with two frontends over one shared core:

- **CLI** (`mhtodo add|list|show|edit|status|done|rm|path`) — the interface for **agentic tool
  access**. Scriptable, `--json` everywhere, stable exit codes and JSON field names (a documented contract).
- **GUI** (Wails v2 webview + system tray) — the human view. Board/list views, task detail editing,
  desktop notifications, live sync so CLI changes appear without restart.

One binary: bare `mhtodo` launches the GUI; any subcommand runs the CLI and exits. All data lives in a
single SQLite database at `$XDG_DATA_HOME/mhtodo/mhtodo.db` (override: `MHTODO_DB_PATH`). Linux-first.

## Scope

**In scope (v0.1):**
- Task fields: title, description, status (`pending | wip | done | waiting`), progress 0–100,
  created_at / updated_at / completed_at. UUIDv7 IDs with short-prefix lookup.
- Full CLI ↔ GUI feature parity (the bound-API table in the plan is the contract).
- System tray: show/hide window, new task, quit; close-to-tray behavior; single-instance lock.
- Desktop notifications on →done and →waiting (`notify-send`).
- Comprehensive Makefile: `dev`, `build`, `test`, `lint`, `release` (linux amd64/arm64), `install`.

**Out of scope for v0.1 (stretch only):** drag-and-drop kanban, markdown rendering in descriptions,
light theme, Windows/macOS support, tags/labels/projects, due dates/reminders.

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

Full implementation plan lives in **[`.agent/plan/`](.agent/plan/README.md)**:

| File | Contents |
|---|---|
| `.agent/plan/README.md` | Key decisions summary (start here) |
| `01-stack-and-environment.md` | Stack choices + machine-specific facts |
| `02-architecture.md` | Project layout, process model, parity contract |
| `03-data-model.md` | SQLite schema, IDs, storage path, concurrency rules |
| `04-cli-spec.md` | CLI command/flag spec, JSON shapes, exit codes |
| `05-gui-spec.md` | Views, tray behavior, notifications, live sync |
| `06-makefile.md` | Complete draft Makefile |
| `07-milestones-and-risks.md` | M0–M6 build order + risk register |

**Build in milestone order (M0 → M6).** M0 is a system-tray integration spike and must pass before any
other work — tray + Wails dual GTK main loops on Linux are the top project risk.

## Progress tracking

Progress against this plan lives in **[`.agent/PROGRESS.md`](.agent/PROGRESS.md)**: a checkbox summary of
every milestone task, kept current as work lands (tick boxes, update the date/status line). It is written to be
copy-pasted into Slack verbatim for team updates — keep it plain-text pasteable.

## Working agreements for agents

- Run `make test` after touching `internal/core`, `internal/store`, or `internal/cli`; golden tests use
  temp-dir DBs via `MHTODO_DB_PATH`.
- The repo is empty until M1 — do not scaffold ahead of the milestone you are on.
- Keep this file updated when scope, constraints, or the plan location changes.
