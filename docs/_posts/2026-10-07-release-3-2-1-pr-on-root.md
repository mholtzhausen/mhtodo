---
layout: post
title: "3.2.1 — Register PRs on the root ticket"
date: 2026-10-07 13:00:00 +0000
categories: [release]
tags: [gui, agents, pull-request, skill]
---

Sub-task handoffs were easy to mis-file against the wrong id. **3.2.1** makes the rule explicit in the paste block and the Claude skill.

## Ticket reference Instructions

**Copy ticket reference** (and Open in Zed) now tell agents to put every PR URL on the **root** ticket with repeatable `mhtodo edit … --pr-url`. If the copied card is a sub-task, the Instructions name the parent id and say not to register PRs on the sub-task.

## Claude skill (`mhtodo ai`)

The embedded skill matches: handoff sample, ownership, step plan, PR hand-back, and subagent sections all say `--pr-url` belongs on the parent/root. Run `mhtodo ai` after upgrade if your installed skill is stale.

## Upgrade

```sh
mhtodo update
```

Or grab the release from GitHub. Full notes: [CHANGELOG — 3.2.1](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#321-pending) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v3.2.1).
