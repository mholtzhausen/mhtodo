# Changelog

## 2.7.0 (5960ff8)

### Features and Improvements
- GUI themes (v0.6): SQLite `themes` table (migration v13) with categorized design tokens (colors, radii, spacing)
- Built-ins **Slate** (default), **Paper** (light), **Ember** (warm) — editable with Reset; Duplicate; built-ins cannot be deleted
- Settings → Themes with color/length pickers and live CSS-variable apply
- CLI: `mhtodo theme list|search|show|create|update|rm|activate|duplicate|reset` (`--json`)

### Bugfixes
- (none)

### Deprecations
- (none)

## 2.6.0 (db2bfd5)

### Features and Improvements
- Full task-template CLI: `mhtodo template list|search|show|create|update|rm` and `add --template REF` (explicit flags override presets)
- Template search supports `--mode fuzzy|regex` and `--cwd` (exact path); agent contract **v12** adds a `$PWD` cwd-probe workflow before creating or picking templates
- Board/list/detail sub-task UX: nested display improvements and related task-management polish

### Bugfixes
- Task card display updates; remove unused components

### Deprecations
- (none)

## 2.5.0 (26716d5)

### Features and Improvements
- Agent integration contract **v11**: live board signal is ticket → status → progress → sub-tasks (activities are audit); ask before creating root tasks; reopen `review` → `wip` with new sub-tasks when more work continues; mid-session ticket nudge (Behaviour D / `Stop`) plus stronger turn-start reminders
- Desktop GUI responsiveness: board columns use fluid `minmax` sizing with horizontal scroll; footer shortcuts hide on narrow widths; pinned detail can float when the main pane is tight
- Task load / detail updates coalesce and debounce; search debounce; Claude/Zed readiness cached once per settings change (not per card)

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 2.4.0 (dbffb30)

### Features and Improvements
- Claude spawn mode **`terminal`**: open Claude in a system terminal emulator, persist `terminal_pid`, and raise/focus the existing window on reopen (session PID + `mhtodo:<session>` title) instead of spawning duplicates
- Claude spawn settings: choose `herdr` | `terminal` | `disabled` (Herdr vs terminal fields shown conditionally)
- `todo_session` is a Claude session UUID (UUIDv7); launch uses `claude --session-id` / `--resume` with `--name`; legacy non-UUID values are minted on first Claude/Zed open
- Always-on-top: opening Claude or Zed hides mhtodo to the tray so the activated terminal/IDE is not covered

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 2.3.0 (e9f007c)

### Features and Improvements
- CLI: `mhtodo install [--prefix DIR] [--service|--no-service] [--integration bash|zsh|none]` copies this binary into `~/.local` (desktop + icon), then can install the user systemd unit and/or `claude.todo` shell helper (TTY prompts; flags for non-interactive)

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 2.2.0 (29f1e16)

### Features and Improvements
- CLI: `mhtodo service install|stop|start|restart|uninstall` manages the user systemd unit for this install (`~/.config/systemd/user/mhtodo.service`); `mhtodo update` still detects an attached unit and restarts it after a binary swap
- Zed: `ZedTicketCommand` / GUI tooltip shows the shell-equivalent open command (cwd + `MHTODO_SESSION`)

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 2.1.0 (54cf2a2)

### Features and Improvements
- Per-task **`todo_session`** (migrations v10–v11): auto-seeded to a space-free `{short8}-{slug}` on create; empty or legacy spaced values are backfilled on open
- CLI: `--session` on `add`/`edit`, and `mhtodo integration bash|zsh` to install a managed `claude.todo` helper that resumes `$MHTODO_SESSION`
- Zed integration: open tasks with `MHTODO_SESSION` set; Settings cover Claude, Herdr, and Zed
- Claude/Herdr resume and naming use the task session; GUI actions and docs (`mhtodo ai`) document the contract

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 2.0.0 (b9fe34c)

### Features and Improvements
- **Task templates** — named sets of task presets (title prefix, description, status, cwd, Slack thread, human-only, include-in-report) stored in the database (migration v9). Every preset is nullable: unset fields fall back to the normal defaults, while an empty string or an explicit `false` is a real override
- Templates are authored in **Settings → Task Templates**, with each template as its own sub-nav entry, a trash-can to clear any field back to unset, and tri-state controls for the boolean presets
- Apply a template from the new-task dialog's template picker (searchable with `/`, showing a chip per preset field), from the header's **New task** split button, or from the tray's **New Task from Template**
- Capture a template with **Save as template** in the new-task and task-detail headers: tick the fields to include and edit their values inline before saving
- Single-task archiving: `mhtodo archive ID` and a card archive action for done tasks, with clear errors when a task is already archived
- Task detail modal navigates to adjacent tasks consistently across the board, list, and activity views
- Slack thread URL can be set directly when creating a task in the new-task dialog
- Board and list views handle the archived filter without falling into a blank state, and the owner filter now explains itself when it matches nothing

### Bugfixes
- Template picker stays open when launched from the **New task** split button or the tray; the launching click was being read as a click-away
- Save-as-template modal no longer closes the task detail behind it when a checkbox is clicked, and Escape closes the modal rather than the task

### Deprecations
- (none)

### Notes
- Task templates ship **GUI-only** in this release. The rules live in `core.Service`, so `mhtodo template …` and `add --template` are a thin later addition — a deliberate, temporary exception to the CLI/GUI parity rule

## 1.10.0 (ee9a638)

### Features and Improvements
- Task `slack_thread`: optional Slack thread URL on tasks (`--slack-thread` on add/edit); shown in `show`, markdown export, and Slack board report with a communication reminder
- Settings → General: **Start hidden in system tray** — choose whether the app launches with the window visible or only in the tray (takes effect on next launch)
- Task detail modal: left-nav sections (Task, Sub-tasks, Activity); header delete; sub-task list with navigation
- Task detail: **Include in Slack report** toggle per task
- New-task dialog: Claude/Herdr **Start task** respects the **Require cwd** integration setting; backdrop click no longer closes the dialog (avoids interrupting text selection)
- Herdr integration: `TaskEligible` honors `require_cwd` when deciding whether a task can open in Herdr

### Bugfixes
- Board: selected card highlight no longer shows a heavy yellow accent border

### Deprecations
- (none)

## 1.9.0 (775f6d9)

### Features and Improvements
- Archive done: bulk archive from the Done column and `mhtodo archive` skips subtasks by default; enable **Archive done subtasks** in Settings → General to include them
- Task markdown report: `mhtodo show ID --markdown` prints a paste-ready summary (subtasks, activity, feedback)
- Herdr: optional close ticket tab when a task moves to done (`close_tab_on_done` in Claude integration settings)
- GUI: task activity actions panel, clearable settings fields, list/board UI polish

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 1.8.0 (1e1e3ab)

### Features and Improvements
- Slack board report: `mhtodo slack report` prints a paste-ready summary (Completed / Todo / WIP with status icons); GUI header button copies the same report to the clipboard

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 1.7.0 (99fd39f)

### Features and Improvements
- Herdr workspace readiness check: `EnsureHerdrReady` verifies workspace status when integration is enabled; new-task dialog reflects Herdr/Claude readiness with async status refresh
- Directory picking: invalid starting paths no longer break the folder dialog (reopens gracefully)
- Toast notifications: multi-toast support with unique IDs and enter/exit transitions

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 1.6.0 (d66e0b3)

### Features and Improvements
- GUI Settings dialog: persisted preferences (Herdr path, default terminal, etc.) with directory picking
- Herdr integration: open workspace from task `cwd`, Claude ticket prompt in terminal tabs, status sync, and env-start helpers
- Short task ID copy button in TaskDetail
- Shared short-prefix ID resolution in core (CLI, GUI, and Herdr)

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 1.5.0 (c4c1545)

### Features and Improvements
- Task `cwd` field: optional absolute working-directory path on tasks (`--cwd` on add/edit; GUI in new-task dialog and detail pane)
- `human_only` flag: mark user-owned tasks agents must skip; default `list` hides them (`--human-only` to include); GUI filter chips (Agents / Human / All) and person icon on board cards
- Agent integration contract **v8** (`mhtodo ai`): documents `cwd`, `human_only`, and stricter never-adopt-human-only guidance

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 1.4.0 (6e3b43c)

### Features and Improvements
- Board task reordering: drag root cards within a column, cross-column drag changes status; CLI `mhtodo reorder`; default list/board sort uses `board_rank`
- Status updates can reorder within the column when appropriate
- Modal detail mode, resizable pin/float panels, and board arrow navigation
- Agent integration contract **v7** (`mhtodo ai`)

### Bugfixes
- Window position persistence and global hotkey (`Ctrl+Shift+Alt+T`) on Ubuntu 24+ Wayland: default to XWayland, periodic hotkey re-grab, resume-from-suspend re-grab, safer position capture

### Deprecations
- (none)

## 1.3.0 (1ae273f)

### Features and Improvements
- `mhtodo update` checks GitHub Releases for a newer linux binary, verifies the asset sha256, installs over the current binary (and desktop/icon when under `$PREFIX/bin/mhtodo`), and restarts the user systemd unit when `mhtodo.service` is present (`--check`, `--force`, `--json`)
- Agent integration contract **v4** (`mhtodo ai`): Title Case activity chip labels + one activity per step forward (reverses v3 coarse/tool-call guidance); `--feedback` as post-work summary + notes/takeaways at hand-back; markdown guidance for description/feedback/activity comments

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 1.2.0 (e6dfa47)

### Features and Improvements
- Agent-authored `feedback` field on tasks (CLI `--feedback` on add/edit; GUI shows when non-empty)
- Markdown rendering for description, feedback, and activity comments in the GUI (detail panes grow up to 500px, then scroll)

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 1.1.0 (ed977a6)

### Features and Improvements
- `mhtodo ai` prints the agent integration contract (embedded instructions with interpolated binary version, DB path, status/sort enums, and changelog)

### Bugfixes
- (none in this release range)

### Deprecations
- (none)

## 1.0.0 (9c56116)

### Features and Improvements
- One-level sub-tasks (`parent_id`), agent-authored activity/comment entries, Activity view, detail-pane pin, and `review` status (after waiting), with rebalanced list columns
- Always-on-top (persisted in DB meta), window position persistence, Esc-to-hide, and a global X11 hotkey (`Ctrl+Shift+Alt+T`) to toggle show/hide and raise the window

### Bugfixes
- (none in this release range)

### Deprecations
- (none)
