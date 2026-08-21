# mhtodo

A personal todo manager in Go. **One binary, two frontends over one shared core:**

- **CLI** — the scriptable, agentic interface: `--json` everywhere, stable exit codes and JSON field names.
- **GUI** — Wails v2 webview + system tray: board (kanban) and list views, task detail editing, desktop notifications, live sync so CLI changes appear without a restart.

Both frontends call the same `core.Service`; neither contains business rules or SQL of its own. Every CLI command has an exact GUI equivalent and vice versa (parity table below). All data lives in one SQLite database — safe to drive from both at once.

## Install

### From source

Requirements: Go ≥ 1.25, Node.js (frontend build), the Wails v2 CLI
(`go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0`), and Linux dev packages for
**webkit2gtk-4.1** + **libayatana-appindicator3** (this distro ships 4.1 only, so the Makefile bakes in
`-tags webkit2_41`; on a webkit2gtk-4.0 system override with `make build TAGS=`).

```sh
make install   # builds and installs into ~/.local: binary + launcher entry + icon
mhtodo         # opens the GUI; tray icon appears in the panel
```

Other useful targets:

| Target | What it does |
|---|---|
| `make build` | release-mode local build → `bin/mhtodo` (builds the frontend too) |
| `make dev` | Wails hot-reload development (GUI) |
| `make test` / `make lint` | Go tests (incl. CLI golden tests) / golangci-lint or go vet fallback |
| `make release` | cross-build linux tarballs → `dist/` (arm64 needs `aarch64-linux-gnu-gcc`; without it, amd64 only + warning) |
| `make install` / `uninstall` | user-local install into `$PREFIX` (default `~/.local`) |
| `make path` | print where the DB lives |

### From a release tarball

```sh
tar xzf mhtodo_0.1.0_linux_amd64.tar.gz && cd mhtodo_0.1.0
install -Dm755 mhtodo ~/.local/bin/mhtodo
install -Dm644 mhtodo.desktop ~/.local/share/applications/
install -Dm644 icon.png ~/.local/share/icons/hicolor/512x512/apps/mhtodo.png
update-desktop-database ~/.local/share/applications
```

## CLI reference (agent contract)

Bare `mhtodo` opens the GUI; any subcommand runs the CLI and exits. All commands are safe to run
concurrently with the GUI.

### Global flags

| Flag | Meaning |
|---|---|
| `--json` | emit JSON instead of human format (objects for single tasks, arrays for lists) |
| `-q`, `--quiet` | suppress non-essential output (e.g. only print the ID on `add`) |
| `MHTODO_DB_PATH` env | override the DB file location |

### Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | usage/validation error (bad flag, bad status, progress out of range) |
| 2 | not found / ambiguous ID |
| 3 | storage error (DB locked beyond busy_timeout, corrupt file, permissions) |

Errors go to **stderr** as `mhtodo: <message>`; with `--json`, stderr carries the envelope
`{"error":"<code>","message":"..."}`. Error codes: `not_found`, `ambiguous_id`, `empty_title`,
`invalid_status`, `progress_range`, `no_fields`, `not_archived`, `usage`, `storage`.

### Commands

| Command | Synopsis | Notes |
|---|---|---|
| `add` | `mhtodo add TITLE [--desc TEXT] [--status pending\|wip\|done\|waiting] [--progress 0-100]` | prints the created object (or just the ID with `-q`) |
| `list` (`ls`) | `mhtodo list [--status S] [--search TEXT] [--limit N] [--sort FIELD[+\|-]] [--all] [--archived]` | default: excludes done **and archived**, sorted `updated_at desc`; `--all` includes done; `--archived` shows archived tasks only (they are hidden everywhere else); `--search` = case-insensitive substring over title + description; sort fields: `created`, `updated`, `status`, `progress`, `title` — suffix `-` ascending, `+` or none descending |
| `show` (`get`) | `mhtodo show ID` | full detail; ID may be a unique prefix (≥ 4 chars) |
| `edit` | `mhtodo edit ID [--title TEXT] [--desc TEXT] [--progress 0-100]` | at least one flag required; never changes status |
| `status` (`set`) | `mhtodo status ID pending\|wip\|done\|waiting` | prints the updated object (transition + timestamps) |
| `done` | `mhtodo done ID [--notify]` | shortcut for `status ID done`; `--notify` sends a desktop notification (opt-in; the GUI always notifies on →done/→waiting) |
| `archive` | `mhtodo archive` | archives **all** currently-done tasks in one step (no per-task form); prints the archived objects, nothing when there was none; reversible via `unarchive` |
| `unarchive` | `mhtodo unarchive ID` | restores an archived task: it goes back to `pending`, progress resets to 0, `completed_at` cleared; non-archived ID → exit 1 (`not_archived`) |
| `rm` (`remove`) | `mhtodo rm ID [--yes]` | interactive confirmation on a TTY; **non-TTY requires `--yes`** (agents must pass it — exit 1 otherwise); prints only the deleted ID |
| `path` | `mhtodo path` | print the DB file path |
| `gui` | `mhtodo gui` | explicit GUI launch, identical to bare `mhtodo` |

### Canonical JSON object

```json
{
  "id": "01958b2e-4c1a-7f3d-9a6b-2c8e4f5a6b7c",
  "title": "Ship mhtodo v0.1",
  "description": "Ship mhtodo v0.1 (see .agent/plan/)",
  "status": "wip",
  "progress": 40,
  "created_at": "2025-08-19T07:59:00Z",
  "updated_at": "2025-08-19T08:30:12Z",
  "completed_at": null,
  "archived_at": null
}
```

`--json list` returns an array of these. Timestamps are RFC3339 UTC; `completed_at` is set on the
→done transition and cleared when a task leaves done; `archived_at` is set by `archive` and cleared
by `unarchive`. IDs are UUIDv7 (time-ordered).

### Agent usage examples

```bash
mhtodo add "Refactor auth" --desc "Split token + session" --json | jq -r .id
mhtodo list --status wip --json
mhtodo show 01958b2e --json
mhtodo edit 01958b2e --progress 60
mhtodo status 01958b2e waiting
mhtodo done 01958b2e
mhtodo archive --json | jq -r '.[].id'      # sweep the Done column into the archive
mhtodo list --archived --json               # inspect archived tasks
mhtodo unarchive 01958b2e                   # back to pending, progress reset
```

**Contract stability:** JSON field names, flags, and exit codes are API. They change deliberately,
and any change is documented here first. An agent can drive the full task lifecycle (create → edit →
status transitions → delete) using only this CLI.

## GUI

- **Board view (default):** four kanban columns — pending / wip / waiting / done — with live counts;
  cards show title, progress bar, and relative update time. Drag a card to another column to change
  its status (same code path as `mhtodo status`). The per-column **+** button opens the new-task
  dialog preset to that column's status.
- **List view:** mirrors the CLI `list` flags — status filter, search, sort + direction. Toggle with
  the header switch or the `b` / `l` keys; your choice persists across launches.
- **Detail drawer** (click a card): edit title/description/status/progress, see timestamps, delete
  with confirmation. New-task dialog from the header button, tray menu, or `n`.
- **Keyboard:** `/` search · `n` new task · `esc` close · `1–4` toggle status filter (list view) ·
  `b`/`l` board/list · `Ctrl+Q` quit.
- **System tray:** Show/Hide window, New Task, Quit. Closing the window hides to tray; the real exit
  paths are tray → Quit, Ctrl+Q, or SIGINT/SIGTERM. The tray label shows open-task count
  ("mhtodo (N)").
- **Notifications:** desktop notification (`notify-send`) on real →done and →waiting transitions.
- **Live sync:** CLI-side changes appear in the GUI automatically (fsnotify on the DB + a 2s poll
  safety net); GUI-side changes are immediately visible to the CLI — same database, WAL mode.
- **Single instance:** launching `mhtodo` while it runs focuses the existing window and exits.

## Data & concurrency

- Database: `$XDG_DATA_HOME/mhtodo/mhtodo.db` (override with `MHTODO_DB_PATH`; `mhtodo path` prints it).
- SQLite in WAL mode with `busy_timeout=5000`; single-statement transactions only — concurrent CLI +
  GUI use is safe by design.

## Parity contract

| Bound method (GUI) | CLI command | Notes |
|---|---|---|
| `ListTasks(filter)` | `list` | filter: status, search, limit, sort, includeDone |
| `GetTask(id)` | `show` | prefix match allowed (same helper as the CLI) |
| `CreateTask(in)` | `add` | |
| `UpdateTask(id, patch)` | `edit` | title/description/progress only |
| `SetStatus(id, status)` | `status` / `done` | fires notification + event on →done/→waiting |
| `DeleteTask(id)` | `rm` | |
| `DBPath()` | `path` | shown in the GUI footer |

After every mutation the app emits a Wails event (`tasks:changed`) that the frontend handles with one
refetch path; an external file watcher emits the same event for CLI-side writes. New capability =
core method + CLI command + bound method — never business logic in either frontend.

## Development notes

- **Build tags:** this distro ships webkit2gtk-4.1 only → all Go builds need `-tags webkit2_41`
  (Makefile `TAGS`). GUI binaries also need a Wails *mode* tag: `wails build`/`wails dev` inject
  `production`/`dev` automatically; plain `go build` must add it (`-tags "webkit2_41 production"`).
- **Tests:** `make test` runs core/store unit tests, CLI golden tests (temp-dir DBs via
  `MHTODO_DB_PATH`, asserting stdout + exit code), and the instance-lock tests.
- **Frontend:** Vite + Svelte 5 + Tailwind v4 in `frontend/`; Wails bindings are generated into
  `frontend/wailsjs` (`make fe-bindings`).
- The full implementation plan lives in [`.agent/plan/`](.agent/plan/README.md); progress is tracked
  in [`.agent/PROGRESS.md`](.agent/PROGRESS.md).
