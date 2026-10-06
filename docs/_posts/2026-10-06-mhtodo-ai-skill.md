---
layout: post
title: "How the mhtodo Claude skill works"
date: 2026-10-06 12:00:00 +0000
categories: [guide]
tags: [agents, claude, skill, cli]
---

Agent integration is now a **Claude skill**, not a hook stack or a versioned contract dump. One command installs it; the board stays the live channel.

## Install and refresh

```sh
mhtodo ai          # write ~/.claude/skills/mhtodo/SKILL.md + strip legacy leftovers
mhtodo ai --check  # report freshness / leftovers; no writes
```

`mhtodo ai` embeds the skill from the binary (`internal/aiskill/SKILL.md`), installs it under `~/.claude/skills/mhtodo/`, and removes old Claude Code hooks, settings entries, `integration.json`, session pointer files, and `claude.todo` helpers.

After a successful `mhtodo update`, skill install runs automatically. When `claude` is on PATH, a non-interactive refresh may also run so the host picks up the new skill. Skip that pass with `--no-skill-refresh` or `MHTODO_SKIP_SKILL_REFRESH=1` (refresh failures warn only; the binary update still counts).

## What the skill teaches

The skill is a set of **named workflows** Claude loads when you mention todos, paste an MHTodo ticket block, or start work that belongs on the board. The important bits:

1. **Hard rule** — read the board freely; never adopt a task just because it appeared in a list (including `human_only`). Work starts only when you ask for something or point at a ticket.
2. **Ticket handoff** — a pasted `MHTodo Ticket` / `mhtodo://task/…` block is an explicit pointer. On adopt, record the session with `mhtodo edit ID --session <uuid>` (that also posts a Claude Session activity). Before code work: clean `master`/`main`, clear dirt with the user if needed, then a new branch.
3. **Live signal** — keep **status → progress → sub-tasks** current. Activities are audit; feedback is a short hand-back summary, not a running log.
4. **Ownership** — user-originated tickets hand back to `review` (you mark done). Agent-originated ones may go to `done`. Reopen `review` → `wip` with new sub-tasks when work continues.
5. **Task picker** — when you ask “what’s next?”, list roots and present every row in the host picker (no autonomous pickup).

Full cheat-sheet and steps live in the installed `SKILL.md`; `mhtodo --help` remains the flag reference.

## What went away

There is no IntegrationVersion, printed agent-contract document, or Claude Code hook install path anymore. If you still have leftovers from older `mhtodo ai` / `mhtodo integration` installs, a normal `mhtodo ai` pass strips them; `--check` lists anything left without writing.

## Day-to-day with the GUI

Keep the board visible. Point Claude with **Copy ticket reference**, or use **Open in Zed** (when enabled) — that opens the task `cwd` and copies the same ticket reference block so you can paste it into the session immediately.

For the short human-facing board primer, see [Claude and mhtodo]({{ site.baseurl }}/guide/2026/10/06/claude-and-mhtodo-primer.html). This skill model ships in [2.11.0]({{ site.baseurl }}/release/2026/10/06/release-2-11-0-claude-skill.html).
