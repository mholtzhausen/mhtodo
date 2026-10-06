---
layout: post
title: "2.12.0 — The Pull Request lane"
date: 2026-10-06 16:00:00 +0000
categories: [release]
tags: [gui, cli, agents, pull-request, board]
---

Review used to be the last stop before done. That worked when “ready for you” and “there’s an open PR” were the same signal. They aren’t. One is waiting on a human. The other is waiting on CI, reviewers, and merge.

**2.12.0** adds a sixth board lane — **Pull Request** (`pr`) — between review and done, plus an optional `pr_url` on every task so the card can take you straight to GitHub (or wherever the PR lives).

## Why a separate lane

The board is a glanceable status channel. When agents hand work back into `review`, you still need to know whether the next step is reading feedback or opening a pull request. Mixing those in one column turns the Done-adjacent strip into a catch-all.

`pr` is the waiting-on-merge state:

- Work is implemented and pushed as a PR
- The ticket is not done until the PR lands (or you deliberately mark it done)
- The link on the card is the source of truth for *which* PR

The column starts **collapsed**. Expand it when you’re in review/merge mode; leave it slim the rest of the time so the board stays focused on active work.

## Linking a PR

Store the URL on the ticket:

```sh
mhtodo edit <id> --pr-url https://github.com/org/repo/pull/42
mhtodo show <id> --json   # status should be "pr"
```

Setting `pr_url` from empty to a value **automatically advances** the ticket to the Pull Request lane (same for `add --pr-url`). Clearing the URL does not bounce the status back — you move lanes with `status` / drag as usual.

In the GUI, paste the URL in the detail pane’s **Pull request** field. On the board, when a URL is set, a small link control sits just left of the mark-done checkbox and opens the PR in your system browser.

## What agents should do

The installed Claude skill (`mhtodo ai`) now says: after you open a PR for a ticket, set `--pr-url` and confirm with `show` that status is `pr`. Hand-back without a PR still uses `review`. Reopen from `review` or `pr` to `wip` when more code work continues.

That keeps the live signal honest: **status → progress → sub-tasks**, with the PR link as the merge pointer.

## Also in this release

- Detail pane shows **Todo session** again (`todo_session` / `edit --session`)
- Markdown links in description, feedback, and activity open in the system browser (not the Wails webview)

Upgrade with `mhtodo update`, or grab the release from GitHub. Full notes: [CHANGELOG — 2.12.0](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#2120-PENDING) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v2.12.0).

Board habits for agents: [Claude and mhtodo]({{ site.baseurl }}/guide/2026/10/06/claude-and-mhtodo-primer.html) · [How the mhtodo Claude skill works]({{ site.baseurl }}/guide/2026/10/06/mhtodo-ai-skill.html).
