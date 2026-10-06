---
layout: post
title: "3.1.0 — Pull request beside Slack"
date: 2026-10-06 20:00:00 +0000
categories: [release]
tags: [gui, pull-request, board]
---

Tickets already carried an optional `pr_url`. **3.1.0** puts that link where you already look for Slack — in the card footer actions and the detail modal header.

## One place to open the PR

When `pr_url` is set, a pull-request icon appears next to the Slack thread icon. Click it and the URL opens in your system browser (same path as Slack and markdown links).

The old board-only control beside the mark-done checkbox is gone, so cards and detail stay consistent: copy report, ticket ref, Slack, **PR**, Zed, then the human-only / report toggles.

## Upgrade

```sh
mhtodo update
```

Or grab the release from GitHub. Full notes: [CHANGELOG — 3.1.0](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#310-a6aa230) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v3.1.0).
