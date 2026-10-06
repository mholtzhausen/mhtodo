---
layout: post
title: "2.8.0 — Frameless window and in-app install"
date: 2026-09-25 12:00:00 +0000
categories: [release]
tags: [gui, install, subtasks]
---

The desktop window goes frameless, and the header gains a one-click path to install or upgrade from GitHub Releases.

**Highlights**

- Drag the app header to move; double-click (outside tabs/actions) toggles maximize; header Close hides to tray
- Header **Install** control: enabled when a newer release is available (hold Ctrl while hovering to force-enable); cached version check with manual refresh; confirm dialog for install/upgrade, optional user systemd service, and shell helper
- New sub-tasks (GUI and `add --parent`) seed cwd, Slack thread, and human-only from the parent; `include_in_report` defaults to false for sub-tasks

Upgrade with `mhtodo update`, or grab the release from GitHub. Full notes: [CHANGELOG — 2.8.0](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#280-a1e87a6) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v2.8.0).
