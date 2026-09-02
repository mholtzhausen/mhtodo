# mhtodo

A personal todo manager in Go. **One binary, two frontends over one shared core:**

- **CLI** — the scriptable, agentic interface: `--json` everywhere, stable exit codes and JSON field names.
- **GUI** — Wails v2 webview + system tray: board (kanban) and list views, task detail editing
  (description/feedback/activity comments are markdown-rendered outside inputs), desktop
  notifications, live sync so CLI changes appear without a restart.

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
| `make dev` | Wails hot-reload (window starts **hidden**; show from tray) |
| `make test` / `make lint` | Go tests (incl. CLI golden tests) / golangci-lint or go vet fallback |
| `make release` | cross-build linux tarballs → `dist/` (arm64 needs `aarch64-linux-gnu-gcc`; without it, amd64 only + warning) |
| `make release-tag [BUMP=major\|minor\|patch]` | **release process** — asks for major/minor/patch (or takes `BUMP=`), bumps `VERSION`, commits + tags, builds tarballs, publishes a GitHub Release and pushes main + tag |
| `make publish` | cross-build with the current version, then `gh release create v$(VERSION)` + push main and the tag |
| `make install` / `uninstall` | user-local install into `$PREFIX` (default `~/.local`) |
| `make service-install` / `service-remove` | build + install, then run as a user systemd service at login (re-running replaces the installed version) |
| `make path` | print where the DB lives |

### From a release tarball

```sh
tar xzf mhtodo_0.1.0_linux_amd64.tar.gz && cd mhtodo_0.1.0
install -Dm755 mhtodo ~/.local/bin/mhtodo
install -Dm644 mhtodo.desktop ~/.local/share/applications/
install -Dm644 icon.png ~/.local/share/icons/hicolor/512x512/apps/mhtodo.png
update-desktop-database ~/.local/share/applications
```

### From the installer script (`install.sh`)

A one-shot installer that downloads the latest prebuilt release binary (no Go/wails/node needed), or falls back to cloning + `make` if no release is reachable — with a prompt for **folder app** vs **systemd service**, and it detects an existing install to update in place. Pipe it directly into bash:

```sh
curl -fsSL https://raw.githubusercontent.com/mholtzhausen/mhtodo/main/install.sh | bash
```

Pass `--service` (user systemd unit at login) or `--app` (binary on PATH) to skip the prompt when piping, e.g. `… | bash -s -- --service`. See `install.sh --help` for flags (`--force`, `--no-build`, `--prefix`, `--repo-url`).

Once installed, upgrade in place with:

```sh
mhtodo update          # download + install if a newer release exists
mhtodo update --check  # report only
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
`invalid_status`, `progress_range`, `no_fields`, `not_archived`, `parent_is_child`, `not_root`,
`reorder_status_mismatch`, `empty_activity`,
`usage`, `storage`, `update`.

### Commands

| Command | Synopsis | Notes |
|---|---|---|
| `add` | `mhtodo add TITLE [--desc TEXT] [--feedback TEXT] [--status pending\|wip\|waiting\|review\|done] [--progress 0-100] [--parent ID] [--cwd PATH] [--human-only]` | prints the created object (or just the ID with `-q`); `--parent` creates a one-level sub-task; `--feedback` is agent-authored (GUI shows it when set); `--cwd` optional working directory; `--human-only` marks a user-owned task agents must skip |
| `list` (`ls`) | `mhtodo list [--status S] [--search TEXT] [--limit N] [--sort FIELD[+\|-]] [--all] [--archived] [--roots] [--human-only]` | default: excludes done, archived, **and human-only**, sorted **board order** (status workflow → `board_rank` → `updated_at`); `--all` includes done; `--archived` shows archived only; `--roots` top-level only; `--human-only` includes human-only rows (default hides them); list stays flat for agents (`parent_id` field); sort fields: `board`, `created`, `updated`, `status`, `progress`, `title` |
| `show` (`get`) | `mhtodo show ID` | full detail; ID may be a unique prefix (≥ 4 chars) |
| `edit` | `mhtodo edit ID [--title TEXT] [--desc TEXT] [--feedback TEXT] [--progress 0-100] [--cwd PATH] [--slack-thread URL] [--human-only \| --no-human-only]` | at least one flag required; never changes status; `--cwd ""` clears the path; `--slack-thread ""` clears the thread |
| `status` (`set`) | `mhtodo status ID pending\|wip\|waiting\|review\|done` | prints the updated object (transition + timestamps); root tasks append to the target column’s board order |
| `reorder` | `mhtodo reorder ID [--before ID]` | move a root task within its status column; `--before` omitted appends to column end |
| `done` | `mhtodo done ID [--notify]` | shortcut for `status ID done`; `--notify` sends a desktop notification (opt-in; the GUI always notifies on →done/→waiting) |
| `archive` | `mhtodo archive` | archives **all** currently-done tasks in one step; reversible via `unarchive` |
| `unarchive` | `mhtodo unarchive ID` | restores an archived task to `pending`, progress 0; non-archived → exit 1 (`not_archived`) |
| `activity add` | `mhtodo activity add ID --activity TEXT [--comment TEXT]` | agent/user-authored entry (at least one of activity/comment); not auto-logged |
| `activity list` | `mhtodo activity list [--task ID]… [--limit N]` | newest first; non-archived tasks by default |
| `activity rm` | `mhtodo activity rm ID [--yes]` | non-TTY requires `--yes` |
| `rm` (`remove`) | `mhtodo rm ID [--yes]` | interactive confirmation on a TTY; **non-TTY requires `--yes`**; cascades to sub-tasks |
| `path` | `mhtodo path` | print the DB file path |
| `slack report` | `mhtodo slack report` | paste-ready board summary for Slack (Completed / Todo / WIP); `--json` emits the text as a JSON string |
| `ai` | `mhtodo ai` | print agent integration instructions (install/upgrade contract; interpolates version, DB path, status/sort enums; documents human-only and cwd rules) |
| `update` | `mhtodo update [--check] [--force]` | check GitHub Releases for a newer linux binary; download, verify sha256, install over the running binary (and desktop/icon when under `$PREFIX/bin/mhtodo`); if `~/.config/systemd/user/mhtodo.service` is present, stop → rewrite unit → `enable --now`. Auth: `GH_TOKEN` / `GITHUB_TOKEN`. `--check` reports only; `--force` reinstalls even when current |
| `gui` | `mhtodo gui` | explicit GUI launch, identical to bare `mhtodo` |

### Canonical JSON object

```json
{
  "id": "01958b2e-4c1a-7f3d-9a6b-2c8e4f5a6b7c",
  "title": "Ship mhtodo v0.1",
  "description": "Ship mhtodo v0.1 (see .agent/plan/)",
  "feedback": "",
  "status": "wip",
  "progress": 40,
  "created_at": "2025-08-19T07:59:00Z",
  "updated_at": "2025-08-19T08:30:12Z",
  "completed_at": null,
  "archived_at": null,
  "parent_id": null,
  "board_rank": 1.0,
  "cwd": "/home/me/projects/mhtodo",
  "human_only": false,
  "slack_thread": ""
}
```

Activity entry:

```json
{
  "id": "01958b2e-aaaa-7f3d-9a6b-2c8e4f5a6b7c",
  "task_id": "01958b2e-4c1a-7f3d-9a6b-2c8e4f5a6b7c",
  "activity": "Ran migration dry-run",
  "comment": "No schema diffs",
  "created_at": "2025-08-19T08:45:00Z"
}
```

`--json list` returns an array of task objects. Timestamps are RFC3339 UTC; `completed_at` is set on
→done and cleared when leaving done; `archived_at` is set by `archive` and cleared by `unarchive`;
`parent_id` is set for one-level sub-tasks; `board_rank` is set on root tasks for board/list ordering
(lower = higher on the board). `cwd` is an optional absolute path to the task's project or working
directory. `human_only` marks a task the user handles themselves — agents must not adopt or update
such tasks; default `list` hides them unless `--human-only` is passed. IDs are UUIDv7 (time-ordered).

### Agent usage examples

```bash
mhtodo add "Refactor auth" --desc "Split token + session" --cwd "$PWD" --json | jq -r .id
mhtodo add "Write tests" --parent 01958b2e --json
mhtodo add "Renew passport" --human-only --json
mhtodo list --status wip --json
mhtodo list --roots --json
mhtodo list --human-only --json   # include user-owned tasks
mhtodo show 01958b2e --json
mhtodo edit 01958b2e --progress 60
mhtodo status 01958b2e review
mhtodo activity add 01958b2e --activity "Opened PR #42" --comment "awaiting review" --json
mhtodo activity list --task 01958b2e --json
mhtodo done 01958b2e
mhtodo archive --json | jq -r '.[].id'
mhtodo list --archived --json
mhtodo unarchive 01958b2e
```

**Contract stability:** JSON field names, flags, and exit codes are API. They change deliberately,
and any change is documented here first. An agent can drive the full task lifecycle (create → edit →
status transitions → activity → delete) using only this CLI.

## GUI

- **Board view (default):** five kanban columns — pending / wip / waiting / review / done — with live
  counts; root cards show title, progress, relative time; human-only cards show a person icon top-right.
  Sub-tasks nest under the parent card when shown (never own column cards). Drag a **root** card to
  change status. Per-column **+** opens new-task preset to that status. Filter chips: **All** /
  **Agents** (hide human-only) / **Human** (human-only only).
- **List view:** status + progress stacked in one column; human-only rows show a person icon before
  the status chip; title takes remaining width; updated shows elapsed + absolute time. Sub-tasks indent
  under parents when shown. Same human filter as the board. Toggle Board / List / Activity with `b` /
  `l` / `a`; choice persists.
- **Activity view:** feed of agent/user activity across non-archived tickets (newest first), filterable
  by ticket checkbox dropdown.
- **Detail pane:** edit fields (including working directory with folder picker, human-only checkbox),
  activity composer, Add sub-task (roots only). **Pin** switches from overlay drawer to an in-flow
  right pane (persisted). Esc closes modals/unpinned detail, otherwise hides to tray.
- **New task dialog:** optional working directory (text + system folder picker) and human-only checkbox.
- **Sub-tasks toggle:** header control or `s` (persisted).
- **Always on top:** pin icon in the header; preference stored in the SQLite `meta` table.
- **Window position:** last position is saved on hide/quit and periodically while visible (`meta.window_pos`), restored on show. On Ubuntu 24+ Wayland sessions the app defaults to the XWayland backend so GTK can read/write coordinates reliably; set `MHTODO_WAYLAND=1` to keep native Wayland (position may not persist).
- **Keyboard:** `/` search · `n` new · `s` sub-tasks · `esc` dismiss/hide · `1–5` status filter · `6` archived
  (list) · `b`/`l`/`a` views · `Ctrl+Shift+Alt+T` global show/hide · `Ctrl+Q` quit.
- **System tray:** Show/Hide, New Task, Quit; close hides to tray; label shows open-task count.
  Global hotkey (X11) toggles the window and raises it on show. The grab is renewed periodically and after resume from suspend (screen lock can drop passive X11 grabs).
- **Notifications:** on real →done and →waiting only (not →review).
- **Live sync:** CLI writes appear via fsnotify + 2s poll; same SQLite WAL DB.
- **Single instance:** second launch focuses the existing window.

## Data & concurrency

- Database: `$XDG_DATA_HOME/mhtodo/mhtodo.db` (override with `MHTODO_DB_PATH`; `mhtodo path` prints it).
- SQLite in WAL mode with `busy_timeout=5000` and `foreign_keys=ON`; single-statement transactions —
  concurrent CLI + GUI use is safe by design.

## Parity contract

| Bound method (GUI) | CLI command | Notes |
|---|---|---|
| `ListTasks(filter)` | `list` | filter: status, search, limit, sort, includeDone, archived, rootsOnly, includeHumanOnly (GUI defaults true; CLI default excludes human-only) |
| `GetTask(id)` | `show` | prefix match allowed |
| `CreateTask(in)` | `add` | optional ParentID, Cwd, HumanOnly |
| `UpdateTask(id, patch)` | `edit` | title/description/feedback/progress/cwd/human_only |
| `PickDirectory()` | — | system folder picker (GUI cwd field) |
| `SetStatus(id, status)` | `status` / `done` | notifies on →done/→waiting; assigns end rank on column change |
| `ReorderBoardTask(id, beforeID)` | `reorder` | same-lane board order; empty `beforeID` appends |
| `ArchiveDone()` | `archive` | bulk done → archive |
| `Unarchive(id)` | `unarchive` | |
| `DeleteTask(id)` | `rm` | cascades to children |
| `CountChildren(id)` | (confirm helper) | GUI delete confirm |
| `AddActivity` / `ListActivity` / `DeleteActivity` | `activity add\|list\|rm` | agent-authored |
| `GetAlwaysOnTop` / `SetAlwaysOnTop` | — | GUI preference (`meta.always_on_top`) |
| `DBPath()` | `path` | GUI footer |
| `SlackReport()` | `slack report` | GUI header copies report to clipboard |

After every mutation the app emits `tasks:changed` (activity ops use `op: activity`); the external
watcher emits the same event for CLI-side writes. New capability = core method + CLI command + bound
method — never business logic in either frontend.

## Development notes

- **Build tags:** this distro ships webkit2gtk-4.1 only → all Go builds need `-tags webkit2_41`
  (Makefile `TAGS`). GUI binaries also need a Wails *mode* tag: `wails build`/`wails dev` inject
  `production`/`dev` automatically; plain `go build` must add it (`-tags "webkit2_41 production"`).
- **Tests:** `make test` runs core/store unit tests, CLI golden tests (temp-dir DBs via
  `MHTODO_DB_PATH`, asserting stdout + exit code), and the instance-lock tests.
- **Frontend:** Vite + Svelte 5 + Tailwind v4 in `frontend/`; Wails bindings are generated into
  `frontend/wailsjs` (`make fe-bindings`).
- The full implementation plan lives in [`.agent/plan/`](.agent/plan/README.md); progress is tracked
  in [`.agent/plan/PROGRESS.md`](.agent/plan/PROGRESS.md).
