---
layout: post
title: "3.1.1 — Multiple pull requests per ticket"
date: 2026-10-06 21:45:00 +0000
categories: [release]
tags: [gui, cli, agents, pull-request]
---

Some tickets ship more than one PR. **3.1.1** lets `pr_url` hold them all.

## One field, many URLs

`pr_url` stays a single string: **one URL per line**. The CLI flag is repeatable and **replaces** the full list each time:

```sh
mhtodo edit <id> --pr-url https://…/pull/10 --pr-url https://…/pull/11
mhtodo show <id> --json   # pr_url is newline-separated; status should be "pr"
```

When you open another PR later, read the existing list with `show`, then re-set every URL including the new one. Do not park PR links only in `--feedback`.

## GUI

The detail pane uses a multi-line Pull requests field. The PR icon next to Slack opens the link when there is one URL, or a floating picker when there are several (styled as a menu, above the board lanes).

## Upgrade

```sh
mhtodo update
```

Or grab the release from GitHub. Full notes: [CHANGELOG — 3.1.1](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#311-89887b4) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v3.1.1).
