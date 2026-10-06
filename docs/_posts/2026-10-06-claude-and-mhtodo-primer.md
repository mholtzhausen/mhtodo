---
layout: post
title: "Claude and mhtodo — a short primer"
date: 2026-10-06 00:00:00 +0000
categories: [guide]
tags: [agents, claude, cli]
---

mhtodo is a **two-way channel** between you and your coding agents — not a log file. The GUI stays open while Claude (or any agent) works; you steer by pointing at tickets, and the agent reports through the board.

The authoritative install/upgrade contract is whatever `mhtodo ai` prints on your machine (from the embedded skill). This post is the human-facing summary of how that relationship is meant to work.

## What you watch on the board

Scan in this order:

1. **Ticket** (title / card)
2. **Status** (column)
3. **Progress** (0–100)
4. **Sub-tasks** (the live step plan)

**Activities** are mostly an audit trail (why a choice was made). **Feedback** is a short post-work summary on the card — not a running log. If sub-tasks are stale but the activity feed is busy, the integration has failed the design.

| Status | Meaning |
|---|---|
| `pending` | Not started — your queue waiting for an agent |
| `wip` | A live session is working on this now |
| `waiting` | Blocked on you (a question was asked) |
| `review` | Delivered — waiting on your judgement |
| `done` | Complete and verified |

## The one hard rule

Agents may **read** the board freely. They must **never** adopt or start a task just because they found it there — including anything marked `human_only`.

Work starts in exactly two ways:

1. You ask for something in the session (the agent then searches for a matching task), or
2. You point at a specific task (paste an id, or the ticket reference block).

Listing is not starting. Default `mhtodo list` hides human-only rows; agents must skip those even if they appear.

## Pointing Claude at a ticket

On a board or list card, use **Copy ticket reference**. You get a handoff block like:

```
---
MHTodo Ticket: mhtodo://task/{short8}
Status: {status}
Title: {title}
 *remember to update subtasks and activity comments if you do work on this*
---
```

Paste that into Claude. On an installed mhtodo, the `mhtodo://task/…` URI (or `mhtodo open …`) raises the GUI focused on that task. The agent should resolve it with `mhtodo show` and keep **status, progress, and sub-tasks** current while working.

You can also say “work on the foo ticket” and let the agent search — still no autonomous pickup from a silent board scan.

## Starting and continuing work

- **Before creating a root task**, the agent should search (`mhtodo list --roots …`) and **ask you** if nothing matches. Unprompted root tickets clutter the board.
- Sub-tasks under an adopted parent are the step plan; those do not need a separate ask.
- If a card is in **`review`** and more work continues, the agent should reopen it to **`wip`**, add new sub-tasks, and continue — not leave new work stranded under a review card.
- For work you will handle yourself, mark the task **human-only** so agents never adopt it.

## Wiring Claude (install / upgrade)

On a machine with mhtodo installed, ask Claude to run:

```sh
mhtodo ai
```

That emits the full agent-integration contract for the installed binary version. Claude’s job is to install or upgrade the host wiring (skill, hooks, always-on instructions) from that document — not to invent a parallel protocol.

Re-run `mhtodo ai` after upgrades when the contract version bumps (release notes call this out).

## Practical habits that help

- Keep the GUI visible; treat columns and progress as the live signal.
- Hand off with the ticket reference block when switching chats or sessions.
- Prefer clear titles and a brief description on root tasks you create for agents.
- When you are done reviewing, mark **done** yourself on user-originated tickets (agents take those to `review`).

For releases that shaped this handoff model, see [2.10.0 — Deep links and ticket handoff]({{ site.baseurl }}/2026/10/05/release-2-10-0-deep-links.html).
