---
layout: post
title: "2.9.0 — Tray attention and notifications"
date: 2026-09-30 00:00:00 +0000
categories: [release]
tags: [gui, tray, notifications]
---

The system tray becomes a live attention surface: status summaries on the label, and submenus that jump straight to a task.

**Highlights**

- Tray attention label (e.g. `mhtodo · 2 waiting, 1 review`) plus configurable status submenus; click opens and focuses the task
- Settings → Notifications: which statuses feed the tray label/menus, max items per submenu, and `notify-send` toggles (→review on by default; →wip / →waiting / →done off)
- Header chrome: single Close (×) hides to tray; hold Ctrl while hovering to reveal Exit; Ctrl+click (or Ctrl+Q) quits

Upgrade with `mhtodo update`, or grab the release from GitHub. Full notes: [CHANGELOG — 2.9.0](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#290-0d24c03) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v2.9.0).
