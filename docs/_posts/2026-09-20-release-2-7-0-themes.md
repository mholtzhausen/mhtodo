---
layout: post
title: "2.7.0 — GUI themes"
date: 2026-09-20 12:00:00 +0000
categories: [release]
tags: [gui, themes, cli]
---

Themes land in the GUI and CLI: named sets of design tokens (colors, radii, spacing) stored in SQLite.

**Highlights**

- Built-ins **Slate** (default), **Paper** (light), and **Ember** (warm) — editable with Reset-to-factory; Duplicate always available; built-ins cannot be deleted
- Settings → Themes with color and length pickers; the active theme applies as CSS variables at runtime
- CLI: `mhtodo theme list|search|show|create|update|rm|activate|duplicate|reset` (use `--json` for agents)

Upgrade with `mhtodo update`, or grab the release from GitHub. Full notes: [CHANGELOG — 2.7.0](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#270-5960ff8) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v2.7.0).
