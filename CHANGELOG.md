# Changelog

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
