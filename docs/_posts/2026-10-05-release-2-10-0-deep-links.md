---
layout: post
title: "2.10.0 — Deep links and ticket handoff"
date: 2026-10-05 00:00:00 +0000
categories: [release]
tags: [cli, gui, deep-links, agents]
---

Point agents (and yourself) at a ticket with a pasteable reference block and desktop deep links. Direct Claude/Herdr spawn from the app is gone — the board + CLI remain the channel.

**Highlights**

- `mhtodo open TARGET` and `mhtodo://task/{id}` raise the GUI on a task and open its detail modal (`x-scheme-handler/mhtodo` on the desktop entry)
- Board/list **Copy ticket reference** puts a `---` block on the clipboard (MHTodo Ticket URI, Status, Title, reminder to keep sub-tasks and activity current)
- Zed integration opens the task working directory only
- Agent contract **v15** documents the multi-line ticket reference and deep links
- Removed: Claude/Herdr/terminal spawn from Settings/GUI, `mhtodo integration` / `claude.todo` helper, `--session` on `add`/`edit` (legacy DB columns kept inert)

Upgrade with `mhtodo update`, or grab the release from GitHub. Full notes: [CHANGELOG — 2.10.0](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#2100-4f7c458) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v2.10.0).

For how Claude (or any agent) is expected to use the board, see [Claude and mhtodo]({{ site.baseurl }}/guide/2026/10/06/claude-and-mhtodo-primer.html).
