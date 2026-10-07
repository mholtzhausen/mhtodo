---
layout: post
title: "3.2.0 — Panel Notifications for collapsed lanes"
date: 2026-10-07 07:45:00 +0000
categories: [release]
tags: [gui, settings, board]
---

Collapsed columns can hide tickets you still care about. **3.2.0** makes occupancy visible at a glance.

## Soft pulse while collapsed

When a board lane is collapsed and has root tasks, it can wash with a slow status-colored background. Empty lanes stay still. Defaults match the first cut of this behavior: all statuses except Done, every five seconds, moderate intensity.

## Settings → Notifications → Panel Notifications

Tune the effect without leaving the app:

- **Which statuses** pulse when collapsed and occupied (Done included as an opt-in)
- **How often** (1–30 seconds; default 5)
- **Intensity** (1–100; default 40) for how strong the wash is

Prefs live with the rest of Notifications in `config.yml`.

## Upgrade

```sh
mhtodo update
```

Or grab the release from GitHub. Full notes: [CHANGELOG — 3.2.0](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#320-pending) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v3.2.0).
