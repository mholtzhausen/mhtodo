# mhtodo

A personal todo manager in Go. **One binary, two frontends over one shared core:**

- **CLI** — the scriptable, agentic interface: `--json` everywhere, stable exit codes and JSON field names.
- **GUI** — Wails v2 webview + system tray: board (kanban), archive, and activity views, task detail editing
  (description/feedback/activity comments are markdown-rendered outside inputs; links open in
  the system browser), desktop notifications, live sync so CLI changes appear without a restart.

Both frontends call the same `core.Service`; neither contains business rules or SQL of its own. Every CLI command has an exact GUI equivalent and vice versa (parity table below). All data lives in one SQLite database — safe to drive from both at once.

**Blog:** [mholtzhausen.github.io/mhtodo](https://mholtzhausen.github.io/mhtodo/) (release notes and guides) · [posts as Markdown](https://github.com/mholtzhausen/mhtodo/tree/main/docs/_posts) · Minima skin in `docs/assets/main.scss`

## Install

### From source

Requirements: Go ≥ 1.25, Node.js (frontend build), the Wails v2 CLI
(`go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0`), and Linux dev packages for
**webkit2gtk-4.1** + **libayatana-appindicator3** (this distro ships 4.1 only, so the Makefile bakes in
`-tags webkit2_41`; on a webkit2gtk-4.0 system override with `make build TAGS=`).

```sh
make install   # builds and installs into ~/.local: binary + launcher entry + icon
# or with an existing binary: mhtodo install  # same layout; prompts for service + shell helper
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
| `make service-install` / `service-remove` | build + install, then run as a user systemd service at login (re-running replaces the installed version); after install, prefer `mhtodo service install\|stop\|start\|restart\|uninstall` |
| `make path` | print where the DB lives |

### From a release tarball

```sh
tar xzf mhtodo_0.1.0_linux_amd64.tar.gz && cd mhtodo_0.1.0
install -Dm755 mhtodo ~/.local/bin/mhtodo
install -Dm644 mhtodo.desktop ~/.local/share/applications/
install -Dm644 icon.png ~/.local/share/icons/hicolor/512x512/apps/mhtodo.png
update-desktop-database ~/.local/share/applications
xdg-mime default mhtodo.desktop x-scheme-handler/mhtodo
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
`invalid_status`, `progress_range`, `no_fields`, `not_archived`, `not_done`, `already_archived`,
`parent_is_child`, `not_root`,
`reorder_status_mismatch`, `empty_activity`,
`usage`, `storage`, `update`, `service`, `install`.

### Commands

| Command | Synopsis | Notes |
|---|---|---|
| `add` | `mhtodo add TITLE [--template REF] [--desc TEXT] [--feedback TEXT] [--status pending\|wip\|waiting\|review\|pr\|done] [--progress 0-100] [--parent ID] [--cwd PATH] [--slack-thread URL] [--pr-url URL …] [--human-only] [--include-in-report \| --no-include-in-report]` | prints the created object (or just the ID with `-q`); `--template` applies a named template (CLI flags override presets); `--parent` creates a one-level sub-task; `--feedback` is agent-authored (GUI shows it when set); `--cwd` optional working directory; `--pr-url` optional PR URL(s), repeatable, stored one-per-line (non-empty advances status to `pr`); `--human-only` marks a user-owned task agents must skip; Slack report inclusion defaults to on for root tasks and off for sub-tasks |
| `list` (`ls`) | `mhtodo list [--status S] [--search TEXT] [--limit N] [--sort FIELD[+\|-]] [--all] [--archived] [--roots] [--human-only]` | default: excludes done, archived, **and human-only**, sorted **board order** (status workflow → `board_rank` → `updated_at`); `--all` includes done; `--archived` shows archived only; `--roots` top-level only; `--human-only` includes human-only rows (default hides them); list stays flat for agents (`parent_id` field); sort fields: `board`, `created`, `updated`, `status`, `progress`, `title` |
| `show` (`get`) | `mhtodo show ID` | full detail; ID may be a unique prefix (≥ 4 chars) |
| `open` | `mhtodo open TARGET` | raise the GUI focused on a task; TARGET is an id/prefix or `mhtodo://task/{id}` deep link (also registered as desktop `x-scheme-handler/mhtodo`) |
| `edit` | `mhtodo edit ID [--title TEXT] [--desc TEXT] [--feedback TEXT] [--progress 0-100] [--cwd PATH] [--slack-thread URL] [--pr-url URL …] [--session UUID] [--human-only \| --no-human-only] [--include-in-report \| --no-include-in-report]` | at least one flag required; does not change status except `--pr-url` empty→non-empty advances to `pr`; `--pr-url` is repeatable (full replacement list, one URL per line in JSON); `--cwd ""` / `--slack-thread ""` / `--pr-url ""` / `--session ""` clear those fields; `--session` links a Claude session UUID on the ticket |
| `status` (`set`) | `mhtodo status ID pending\|wip\|waiting\|review\|pr\|done` | prints the updated object (transition + timestamps); root tasks append to the target column’s board order |
| `reorder` | `mhtodo reorder ID [--before ID]` | move a root task within its status column; `--before` omitted appends to column end |
| `done` | `mhtodo done ID [--notify]` | shortcut for `status ID done`; `--notify` sends a desktop notification (opt-in; GUI notify-send is Settings → Notifications) |
| `archive` | `mhtodo archive [ID]` | with no ID, archives **all** currently-done tasks; with ID, archives that single done task only (must be done; already archived → `already_archived`); reversible via `unarchive` |
| `unarchive` | `mhtodo unarchive ID` | restores an archived task to `pending`, progress 0; non-archived → exit 1 (`not_archived`) |
| `activity add` | `mhtodo activity add ID --activity TEXT [--comment TEXT]` | agent/user-authored entry (at least one of activity/comment); not auto-logged |
| `activity list` | `mhtodo activity list [--task ID]… [--limit N]` | newest first; non-archived tasks by default |
| `activity rm` | `mhtodo activity rm ID [--yes]` | non-TTY requires `--yes` |
| `rm` (`remove`) | `mhtodo rm ID [--yes]` | interactive confirmation on a TTY; **non-TTY requires `--yes`**; cascades to sub-tasks |
| `path` | `mhtodo path` | print the DB file path |
| `slack report` | `mhtodo slack report` | paste-ready board summary for Slack (Completed / Todo / WIP); `--json` emits the text as a JSON string |
| `ai` | `mhtodo ai [--check]` | install/update `~/.claude/skills/mhtodo/SKILL.md` (embedded workflows) and strip legacy Claude hooks/settings/integration artifacts; `--check` reports skill freshness + leftovers without writing |
| `install` | `mhtodo install [--prefix DIR] [--service \| --no-service]` | copy this binary into `$PREFIX` (default `~/.local`) with desktop launcher + icon; on a TTY, prompt for user systemd service; flags skip prompts (non-TTY skips optionals unless flagged) |
| `update` | `mhtodo update [--check] [--force] [--no-skill-refresh]` | check GitHub Releases for a newer linux binary; download, verify sha256, install over the running binary (and desktop/icon when under `$PREFIX/bin/mhtodo`); if `~/.config/systemd/user/mhtodo.service` is attached to this binary, stop → rewrite unit → `enable --now`. After a successful binary update, runs `mhtodo ai` and (when `claude` is on PATH) a non-interactive skill refresh; human output announces each skill step before it starts (Claude refresh can take up to a few minutes); refresh failures warn only. Auth: `GH_TOKEN` / `GITHUB_TOKEN`. `--check` reports only; `--force` reinstalls even when current; `--no-skill-refresh` / `MHTODO_SKIP_SKILL_REFRESH=1` skip the Claude pass |
| `service` | `mhtodo service install\|stop\|start\|restart\|uninstall` | manage the user systemd unit for this install (`~/.config/systemd/user/mhtodo.service`); `install` writes `ExecStart=<this binary> gui` and enables it; `uninstall` removes the unit (binary stays). From-source bootstrap remains `make service-install` |
| `template list` | `mhtodo template list` | all templates, name order |
| `template search` | `mhtodo template search [QUERY] [--mode fuzzy\|regex] [--cwd PATH]` | fuzzy (default) or regex over name/title_prefix/description/cwd; `--cwd` exact path match; query and/or `--cwd` required |
| `template show` | `mhtodo template show REF` | one template by id or name |
| `template create` | `mhtodo template create NAME [--title-prefix S] [--desc S] [--status S] [--cwd S] [--slack-thread URL] [--human-only \| --no-human-only] [--include-in-report \| --no-include-in-report]` | only passed flags become presets (omitted = unset); `-q` prints id |
| `template update` | `mhtodo template update REF [--name S] … [--clear-title-prefix\|--clear-desc\|--clear-status\|--clear-cwd\|--clear-slack-thread\|--clear-human-only\|--clear-include-in-report]` | patch by id or name; `--clear-*` unsets a preset; at least one flag required |
| `template rm` | `mhtodo template rm REF [--yes]` | non-TTY requires `--yes`; prints deleted id |
| `theme list` | `mhtodo theme list` | all themes, name order (`*` = active) |
| `theme search` | `mhtodo theme search QUERY [--mode fuzzy\|regex]` | fuzzy (default) or regex over name |
| `theme show` | `mhtodo theme show REF` | one theme by id or name (full tokens) |
| `theme create` | `mhtodo theme create NAME [--from REF] [--activate]` | tokens from Slate factory or `--from`; optional activate |
| `theme update` | `mhtodo theme update REF [--name S] [--set KEY=VALUE]… [--tokens-json '{…}']` | merge token patches; at least one flag required |
| `theme rm` | `mhtodo theme rm REF [--yes]` | user themes only (built-ins refused); non-TTY requires `--yes` |
| `theme activate` | `mhtodo theme activate REF` | set the active GUI theme |
| `theme duplicate` | `mhtodo theme duplicate REF [NAME]` | copy into a new user theme |
| `theme reset` | `mhtodo theme reset REF` | restore factory tokens for a built-in |
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
  "include_in_report": true,
  "slack_thread": "",
  "pr_url": ""
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
`parent_id` is set for one-level sub-tasks; `board_rank` is set on root tasks for board ordering
(lower = higher on the board). `cwd` is an optional absolute path to the task's project or working
directory. `human_only` marks a task the user handles themselves — agents must not adopt or update
such tasks; default `list` hides them unless `--human-only` is passed. IDs are UUIDv7 (time-ordered).
`todo_session` holds the Claude session UUID linked when an agent adopts the ticket
(`mhtodo edit ID --session …`). `pr_url` holds optional pull-request URL(s), one per line;
setting the field empty→non-empty advances status to `pr`. Repeat `--pr-url` on `add`/`edit`
to set multiple. Legacy `terminal_pid` may still appear as `0` — ignore it.

### Zed (Settings → Integrations)

When Zed is enabled and found on PATH, board/archive cards show an Open-in-Zed action for tasks that
have a working directory. It runs the configured binary with the task `cwd` (optional `env_start`
prefix) and copies the same **ticket reference** block to the clipboard. There is no direct
Claude/Herdr/terminal spawn in the app; agent hosts still install via `mhtodo ai`
(installs/updates the Claude skill under `~/.claude/skills/mhtodo/`; no hooks).

Card actions also copy a paste-ready **markdown report** or a **ticket reference** block
(deep link + status/title + agent Instructions):

```
---
MHTodo Ticket: mhtodo://task/{short8}
Status: {status}
Title: {title}
Instructions: (only if you start working on this task) 
 - record your current claude session against the ticket: `mhtodo edit {short8} --session <claude-session-uuid>`
 - remember to update subtasks and activity comments if you do work on this
 - before starting: be on a clean `master` (or `main` if that is the default trunk); if the working tree is dirty, stop and clear it up with the user first; then create a new branch for this ticket and only then start work
 - when you open a pull request: put every PR URL on the root ticket with `mhtodo edit {rootShort8} --pr-url <url> [--pr-url <url>…]` (repeatable; each edit replaces the full list — `mhtodo show {rootShort8} --json` first if adding another). Never leave PR links only in feedback or comments.
 - (sub-task handoffs only) register pull requests against the main ticket `{rootShort8}`, not this sub-task id
---
```

`{rootShort8}` is the pasted ticket id for a root card, or the parent id when copying a sub-task.

Clicking `mhtodo://task/…` (or running `mhtodo open mhtodo://task/{short8}`) raises the
GUI on that task when mhtodo is installed with its desktop entry.

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

- **Board view (default):** six kanban columns — pending / wip / waiting / review / pr / done — with live
  counts; root cards show title, progress, relative time; human-only / Slack-report flags live in the
  card footer actions (not duplicated in the title row). When `pr_url` is set, a pull-request icon
  sits next to the Slack thread icon in those footer actions (and in the detail modal header);
  one URL opens directly, two or more open a picker dropdown. The Pull Request column is collapsed by default. Columns collapse
  via a header caret into a slim vertical strip (rotated title + count); collapsed state persists across
  restarts (`mhtodo.collapsedColumns.v2`). Collapsed lanes with tickets can pulse a slow status-tinted
  background (Settings → Notifications → Panel Notifications: which statuses, interval, intensity;
  defaults exclude Done, 5s, intensity 40). Sub-tasks nest under the parent card when shown (never own
  column cards), in creation order (oldest first). Drag a **root** card to change status (including onto
  a collapsed lane). Per-column **+** opens new-task preset to that status. Filter chips: **All** /
  **Agents** (hide human-only) / **Human** (human-only only).
- **Archived view:** table of archived tasks (status + progress, title, updated); sub-tasks indent
  under parents when shown (creation order). Same human filter as the board. Open a row to unarchive
  from the detail modal. Tabs: Board / Activity / Archived (`b` / `a` / `r`, or `7` for Archived);
  choice persists.
- **Activity view:** feed of agent/user activity across non-archived tickets (newest first), with
  shared search/human filters plus a ticket checkbox dropdown (closes on outside click / Esc).
- **Detail modal:** edit fields (including working directory with fuzzy typeahead over template
  cwds and previously used ticket paths — plus folder picker / free-typed path — Todo session /
  Claude session UUID, human-only / Slack report checkboxes, Slack thread URL, pull-request URLs
  one-per-line), activity composer,
  Add sub-task (roots only). Header repeats card actions (copy markdown / ticket ref, Slack, pull
  request when `pr_url` is set — picker when multiple — Zed, human-only / include-in-report toggles,
  archive). Feedback is agent/CLI-authored (read-only in the
  GUI). Markdown http(s) links open externally. Esc closes the modal, otherwise hides to tray.
  Click another task to switch; `←`/`→` move to adjacent tasks.
- **New task dialog:** optional working directory (same typeahead + folder picker as detail), Slack
  thread, human-only, include-in-Slack-report (defaults from Settings), and initial status. Header
  icons apply a **task template** or save the current fields as one.
- **Task templates:** named sets of presets — title prefix, description, status, working directory,
  Slack thread, human-only, include-in-report. Authored under **Settings → Task Templates**, where
  each template is its own sub-nav item. Only the fields you set are applied; anything left unset
  keeps its normal default, so a template that only sets a working directory still picks up your
  default human-only and report settings. Clear a field with its trash-can to unset it; the two
  boolean fields use a three-state control (`default` / `off` / `on`). Apply one from the template
  icon in the new-task dialog, the right half of the header's **New task** split button, or the
  tray's *New Task from Template* — the picker filters with `/`, navigates with arrows, and shows
  chips for the fields each template presets. Save the current new-task form or an existing task as
  a template from the save icon in either header. CLI: `mhtodo template
  list|search|show|create|update|rm` and `add --template REF` (`search --cwd "$PWD"`
  is the agent-friendly probe; `--mode fuzzy|regex` for text).
- **Themes:** Settings → Themes authors design tokens (colors, radii, spacing). Built-ins **Slate**
  (default active), **Paper** (light), and **Ember** (warm) are editable with Reset-to-factory;
  Duplicate always available; built-ins cannot be deleted. The active theme applies live via CSS
  variables. CLI: `mhtodo theme list|search|show|create|update|rm|activate|duplicate|reset`.
- **Sub-tasks toggle:** header control (persisted).
- **Always on top:** pin icon in the header; preference stored in the SQLite `meta` table.
  When on, opening Zed for a task hides mhtodo to the tray so the activated
  terminal/IDE is not covered.
- **Install / update:** download icon left of Settings. Enabled when a newer release is
  available; hold **Ctrl** while hovering to force-enable. Hover refreshes the GitHub
  version check when the 60-minute cache is stale; the dialog shows current/target versions
  with a manual refresh control (`install` vs `upgrade`, optional service).
- **Window:** frameless; drag the app header to move, double-click header (outside tabs/actions) to toggle maximize. Header Close / Esc hide to tray; hold **Ctrl** while hovering Close to reveal Exit, then Ctrl+click (or `Ctrl+Q`) to quit.
- **Window position:** last position is saved on hide/quit and periodically while visible (`meta.window_pos`), restored on show. On Ubuntu 24+ Wayland sessions the app defaults to the XWayland backend so GTK can read/write coordinates reliably; set `MHTODO_WAYLAND=1` to keep native Wayland (position may not persist).
- **Keyboard:** `/` search · `n` new · `esc` dismiss/hide · `1–6` status filter · `7`/`r` archived view ·
  `b`/`a` board/activity · `←`/`→` adjacent task in modal ·
  detail short-ID: copy button for short ID, `Ctrl+click` (⌘-click) for full UUID ·
  `Ctrl+Shift+Alt+T` global show/hide · `Ctrl+Q` quit.
- **System tray:** Show/Hide, New Task, New Task from Template, Settings, Quit; close hides to tray. Label shows
  attention counts when configured statuses have root tasks (e.g. `mhtodo · 2 waiting, 1 review`),
  otherwise the open-task count. Configurable status submenus list recent root tasks; click opens the
  window and selects the task (Settings → Notifications).
  Global hotkey (X11) toggles the window and raises it on show. The grab is renewed periodically and after resume from suspend (screen lock can drop passive X11 grabs).
- **Notifications:** Settings → Notifications configures tray label/menu statuses and `notify-send` on
  GUI status transitions (defaults: →review on; →wip / →waiting / →done off). Sub-page **Panel
  Notifications** controls which collapsed board lanes pulse when occupied, how often (default 5s),
  and intensity (default 40). Tray menus refresh on local and CLI-driven DB changes.
- **Live sync:** CLI writes appear via fsnotify + 2s poll; same SQLite WAL DB.
- **Single instance:** second launch focuses the existing window.
- **Window size:** default 1100×720, minimum 800×560 (desktop-only; no mobile layout). Near the floor, the board keeps ~200px columns and scrolls horizontally; footer shortcut legend hides below ~900px width.
- **GUI refresh:** `tasks:changed` is debounced/coalesced; single-task updates patch in place when possible. Search input is debounced (~200ms). Zed binary readiness is cached app-wide (not per board card).

## Data & concurrency

- Database: `$XDG_DATA_HOME/mhtodo/mhtodo.db` (override with `MHTODO_DB_PATH`; `mhtodo path` prints it).
- SQLite in WAL mode with `busy_timeout=5000` and `foreign_keys=ON`; single-statement transactions —
  concurrent CLI + GUI use is safe by design.

## Parity contract

| Bound method (GUI) | CLI command | Notes |
|---|---|---|
| `ListTasks(filter)` | `list` | filter: status, search, limit, sort, includeDone, archived, rootsOnly, parentId (direct children), includeHumanOnly (GUI defaults true; CLI default excludes human-only) |
| `GetTask(id)` | `show` | prefix match allowed |
| `CreateTask(in)` | `add` | optional ParentID, Cwd, HumanOnly, IncludeInReport (*bool, default true), SlackThread, TodoSession |
| `UpdateTask(id, patch)` | `edit` | title/description/feedback/progress/cwd/human_only/include_in_report/slack_thread/pr_url/todo_session; `pr_url` is newline-separated URL(s); empty→non-empty advances to `pr` |
| `PickDirectory()` | — | system folder picker (GUI cwd field) |
| `SetStatus(id, status)` | `status` / `done` | optional notify-send per Settings (wip/waiting/review/done); assigns end rank on column change |
| `ReorderBoardTask(id, beforeID)` | `reorder` | same-lane board order; empty `beforeID` appends |
| `Archive(id)` | `archive ID` | single done task → archive |
| `ArchiveDone()` | `archive` | bulk done → archive |
| `Unarchive(id)` | `unarchive` | |
| `DeleteTask(id)` | `rm` | cascades to children |
| `CountChildren(id)` | (confirm helper) | GUI delete confirm |
| `AddActivity` / `ListActivity` / `DeleteActivity` | `activity add\|list\|rm` | agent-authored |
| `GetAlwaysOnTop` / `SetAlwaysOnTop` | — | GUI preference (`meta.always_on_top`) |
| `DBPath()` | `path` | GUI footer |
| `SlackReport()` | `slack report` | GUI header copies report to clipboard |
| `GetInstallStatus(force)` | `update --check` (+ detect) | 60m cache; `force` bypasses; `show` when update available; GUI Ctrl+hover force |
| `RunInstallActions(in)` | `update` / `service install` | GUI confirmation; execs this binary’s CLI |
| `ListTemplates` / `GetTemplate` / `CreateTemplate` / `UpdateTemplate` / `DeleteTemplate` | `template list\|search\|show\|create\|update\|rm`; `add --template` | task templates (v0.5); CLI `search` uses core `SearchTemplates` (fuzzy/regex + cwd); update is full replace in core (CLI patches then replace); `add --template` applies then lets changed flags override |
| `ListThemes` / `GetTheme` / `GetActiveTheme` / `CreateTheme` / `UpdateTheme` / `DeleteTheme` / `ActivateTheme` / `DuplicateTheme` / `ResetTheme` | `theme list\|search\|show\|create\|update\|rm\|activate\|duplicate\|reset` | GUI themes (v0.6); tokens JSON map; active id in `meta.active_theme_id`; built-ins Slate/Paper/Ember |

After every mutation the app emits `tasks:changed` (activity ops use `op: activity`); the external
watcher emits the same event for CLI-side writes. Template mutations emit `templates:changed`
and theme mutations emit `themes:changed`, so they do not trigger a task reload. New capability =
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
  in [`.agent/plan/PROGRESS.md`](.agent/plan/PROGRESS.md).
