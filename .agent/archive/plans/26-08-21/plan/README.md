# mhtodo — Implementation Plan

A todo app in Go with two frontends over one shared core:

- **CLI** (`mhtodo add|list|show|edit|status|...`) — for agentic tool access, JSON-friendly.
- **GUI** (Wails v2 webview + system tray) — the human view: board/list views, task detail,
  desktop notifications, live sync with CLI changes.

One binary, one SQLite database in the XDG data dir. Linux-first.

## Files in this plan

| File | Contents |
|---|---|
| [01-stack-and-environment.md](01-stack-and-environment.md) | Chosen stack, versions, machine-specific facts |
| [02-architecture.md](02-architecture.md) | Project layout, process model, shared core design |
| [03-data-model.md](03-data-model.md) | SQLite schema, IDs, storage path, concurrency rules |
| [04-cli-spec.md](04-cli-spec.md) | Full CLI command/flag spec, JSON shapes, exit codes |
| [05-gui-spec.md](05-gui-spec.md) | Views, tray behavior, notifications, live sync |
| [06-makefile.md](06-makefile.md) | Complete draft Makefile (build/dev/release/install) |
| [07-milestones-and-risks.md](07-milestones-and-risks.md) | Phased build order + risk register |

## Key decisions (summary)

1. **Single binary.** No args → GUI; any subcommand → CLI. One install, one `PATH` entry for agents.
2. **Wails v2** (not v3): stable, matches installed CLI 2.11, known tray-integration patterns.
   Build tag `webkit2_41` is required on this distro (only webkit2gtk-4.1 is present).
3. **Shared core package** (`internal/core`) used by both CLI and GUI — parity is structural, not aspirational.
4. **SQLite via `modernc.org/sqlite`** (pure Go, no cgo for the DB layer), WAL mode + busy_timeout so
   CLI and GUI can run concurrently against the same file.
5. **UUIDv7 IDs**, time-ordered; short-prefix lookup like git (`mhtodo show 019ab3`).
6. **Data location:** `$XDG_DATA_HOME/mhtodo/mhtodo.db` (default `~/.local/share/mhtodo/`),
   overridable via `MHTODO_DB_PATH` for tests and agents.
7. **Tray** via `getlantern/systray` in a goroutine — the riskiest integration point, so it gets a
   dedicated spike (M0) before any real work. Fallback: `leaanthony/systray` fork.
8. **Frontend:** Vite + Svelte 5 + Tailwind CSS v4, dark theme by default. Views: Board (kanban),
   List, Detail drawer. Drag-and-drop is a stretch goal; status changes via buttons/menus first.
9. **Live sync:** GUI watches the DB file (fsnotify) with a 2s poll fallback → Wails event
   `tasks:changed` → frontend refetches. CLI edits appear in the UI without restart.

## Status model

`pending → wip → done`, plus `waiting` as an orthogonal "blocked on external" state.
`progress` is 0–100, independent of status except: setting `done` forces progress to 100 and stamps
`completed_at`; leaving `done` clears `completed_at`.
