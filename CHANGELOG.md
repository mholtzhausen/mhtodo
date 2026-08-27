# Changelog

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
