---
layout: post
title: "3.0.0 — Modal detail and Archived"
date: 2026-10-06 18:00:00 +0000
categories: [release]
tags: [gui, archive, detail, major]
---

The GUI had three ways to show a ticket and a list view that mostly existed to browse archived work. **3.0.0** collapses that into one detail shape and one place for the archive.

## Modal-only detail

Opening a task always raises the centered modal. Pin and float panes are gone — including the cramped-window fallback that turned a pinned pane into a floating strip. Esc closes the modal; `←` / `→` still move between adjacent tickets.

The modal header now carries the same action row as board cards: copy markdown report, copy ticket reference, Slack thread, Zed, human-only, include-in-report, and archive when applicable. The form still has the human-only and Slack-report checkboxes, so either control works.

## Board · Activity · Archived

Tabs are **Board**, **Activity**, then **Archived** floated to the right (lighter when idle). The old List view is removed entirely — not hidden behind a preference.

Archived is a table of archived tickets with separators and a clear row hover. Open one to unarchive from the detail footer. While archived, fields are read-only: status, progress, description, cwd, checkboxes, and activity posting stay locked until you restore the ticket. An ox-red **Archived** chip sits beside the ID hash so the modal cannot be mistaken for an active edit.

Shortcuts: `b` board · `a` activity · `r` or `7` archived.

## Upgrade

```sh
mhtodo update
```

Or grab the release from GitHub. Full notes: [CHANGELOG — 3.0.0](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#300-a0b290b) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v3.0.0).
