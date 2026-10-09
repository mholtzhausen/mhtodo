---
layout: post
title: "3.3.0 — Icebox and configurable lanes"
date: 2026-10-09 10:00:00 +0000
categories: [release]
tags: [gui, board, settings, themes, agents]
---

Not every ticket belongs in the active pipeline. **3.3.0** adds an Icebox lane and lets you choose which statuses show on the board.

## Icebox

A new leftmost status, `icebox`, parks work that is not in flight. Default CLI `list` hides icebox the same way it hides `done`; use `--status icebox` or `--all` when you need those tickets.

## Statuses / Lanes settings

**Settings → Statuses / Lanes** has a submenu per status:

- Toggle **visible on board** (at least one lane must stay on)
- Edit the lane **color** on the **active** theme (status colors moved out of Themes)
- See how many tickets sit in the lane; when hiding a lane that still has tickets, **leave** them or **migrate** them to another visible status

Hidden lanes are omitted from the board and the detail StatusPicker (a ticket already in a hidden lane still shows its current status).

## Bulk migrate

```sh
mhtodo status migrate FROM TO
```

Moves every non-archived task from one status to another. The Settings hide flow uses the same path.

## Upgrade

```sh
mhtodo update
```

Or grab the release from GitHub. Full notes: [CHANGELOG — 3.3.0](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#330-000f228) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v3.3.0).
