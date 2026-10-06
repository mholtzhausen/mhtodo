---
layout: post
title: "2.11.0 — Claude skill installer"
date: 2026-10-06 14:00:00 +0000
categories: [release]
tags: [cli, agents, claude, skill, zed]
---

Agent setup is skill-first: `mhtodo ai` installs a Claude workflow skill, strips legacy hooks, and stays in sync after `mhtodo update`. Session linking and ticket handoff are back on the board path.

**Highlights**

- **`mhtodo ai`** installs/updates `~/.claude/skills/mhtodo/SKILL.md` and removes legacy hooks/settings/integration leftovers; `--check` reports only
- **`mhtodo update`** re-installs the skill after a binary swap and may refresh Claude non-interactively (`--no-skill-refresh` / `MHTODO_SKIP_SKILL_REFRESH=1` to skip)
- Restore **`mhtodo edit ID --session`**: links a Claude session UUID; non-empty values auto-post a Claude Session activity
- Ticket reference Instructions again tell agents to record `--session` on adopt
- **Open in Zed** also copies the ticket reference block to the clipboard
- Removed: versioned agent-integration contract document and Claude Code hook install path from `mhtodo ai`

Upgrade with `mhtodo update`, or grab the release from GitHub. Full notes: [CHANGELOG — 2.11.0](https://github.com/mholtzhausen/mhtodo/blob/main/CHANGELOG.md#2110-pending) · [Release](https://github.com/mholtzhausen/mhtodo/releases/tag/v2.11.0).

How the skill is meant to be used: [How the mhtodo Claude skill works]({{ site.baseurl }}/guide/2026/10/06/mhtodo-ai-skill.html). Board habits: [Claude and mhtodo]({{ site.baseurl }}/guide/2026/10/06/claude-and-mhtodo-primer.html).
