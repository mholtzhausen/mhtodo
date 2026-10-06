---
name: mhtodo
description: >-
  Drive mhtodo (personal todo CLI+GUI) as a two-way channel with the user.
  Use when the user pastes an MHTodo Ticket / mhtodo://task/ block, mentions
  todos / what's next / pick up a task / hand back / review / pull request / PR,
  asks about the board, or you start or continue work that should be tracked on
  a ticket. Covers adopt, --session, sub-tasks, status/progress, waiting/review/pr,
  and never adopting human_only or autonomous board pickup.
---

# mhtodo — agent workflows

`mhtodo` is a live board the user watches. You report through **status → progress → sub-tasks**. Activities are audit. Feedback is a short hand-back summary. Full flags: `mhtodo --help`.

## CLI cheat-sheet

```
mhtodo list --roots [--search S] [--json]
mhtodo show ID
mhtodo open TARGET
mhtodo status ID pending|wip|waiting|review|pr|done
mhtodo edit ID --session UUID|--progress N|--feedback S|--desc S|--title S|--pr-url URL
mhtodo add TITLE [--parent ID] [--template REF] [--cwd PATH] [--json]
mhtodo activity add ID --activity "Label" --comment "…"
mhtodo activity list --task ID [--json]
mhtodo template search --cwd "$PWD"
mhtodo done ID
```

IDs: unique prefix of 4+ chars; show **13** chars in listings. Default `list` hides `done`, archived, and `human_only`.

---

## 1. Hard rule — no autonomous pickup

Read the board freely. **Never** adopt or start a task because it appeared in a list — including `human_only`. Work starts only when the user asks for something or points at a ticket (paste / id). Listing is not starting.

## 2. Ticket reference handoff

If the user pastes:

```
---
MHTodo Ticket: mhtodo://task/{short8}
Status: …
Title: …
Instructions: (only if you start working on this task) 
 - record your current claude session against the ticket: `mhtodo edit {short8} --session <claude-session-uuid>`
 - remember to update subtasks and activity comments if you do work on this
 - before starting: be on a clean `master` (or `main` if that is the default trunk); if the working tree is dirty, stop and clear it up with the user first; then create a new branch for this ticket and only then start work
---
```

Treat that as pointing at the ticket. Resolve with `mhtodo show` / `mhtodo open`. Follow the Instructions when you start work.

## 3. Adopt (search → session → plan)

Before substantive work:

1. `mhtodo list --roots --json --search "<keywords>"` (or use a pasted id).
2. Same job → adopt. In `review` or `pr` with more work → reopen to `wip`. Nothing matches → **ask** before creating a root. Unsure → ask.
3. **Git hygiene (required before any code work):** checkout `master` (or `main` if that is the repo default). Working tree must be clean. If anything is dirty (uncommitted changes, untracked work you did not expect, mid-rebase, etc.), **stop and clear it up with the user** — do not stash, discard, or commit on their behalf unless they explicitly approve. Only then create a new branch for this ticket and start work.
4. Adopt:

```bash
mhtodo status <id> wip
mhtodo edit <id> --session <claude-session-uuid>   # required; posts Claude Session activity
mhtodo edit <id> --progress 5                      # never rewrite user title/description
mhtodo activity add <id> --activity "Task Picked Up" --comment "<brief>"
```

5. Re-set `--session` after `/new`, `/clear`, or a new chat.
6. Non-trivial job → create one-level sub-tasks as the step plan (2+ steps) immediately.

**Templates:** before a project-shaped root, `mhtodo template search --cwd "$PWD"`; ask which match; `add --template REF` when appropriate.

## 4. Sync checklist

**At the start of a turn and before you finish a reply:**

- Confirm the linked ticket and `--session` still match this chat.
- Refresh parent **status**, **progress**, and **sub-tasks** for work just done.
- If continuing a `waiting`, `review`, or `pr` card → set parent `wip` and add sub-tasks as needed.
- Never auto-create a root from this checklist.

## 5. Live board rules

- User scans: ticket → status → progress → **sub-tasks**. Keep those current.
- Activities: Title Case 2–4 word label + detail in `--comment` (audit only).
- Feedback (`--feedback`): short post-work summary at hand-back only — not a running log.
- **Ownership:** user-originated → never change title; never overwrite description; hand back to `review` (user marks done), or set `--pr-url` when you open a PR (auto-advances to `pr`). Agent-originated → may refine title/desc; may `done`.
- Do not narrate bookkeeping (“created a task…”) unless they asked about the board.

## 6. Step plan (sub-tasks)

One level only. Drive steps `pending → wip → done`. Blocking or hand-back uses the **parent** (`waiting` / `review` / `pr`). Multiple sub-tasks may be `wip` in parallel (subagents). Replan by adding/editing sub-tasks — not by narrowing the user’s root title/description.

## 7. Blocked / hand back / reopen / pull request

- **Blocked on user:** parent `waiting` + activity with the question.
- **Hand back (no PR yet):** every sub-task `done` → `--progress 100` + `--feedback` → parent `review` (or `done` if agent-owned). Never leave parent on `wip` at end of turn.
- **Opened a pull request:** after creating the PR for this ticket:

```bash
mhtodo edit <id> --pr-url <https://…/pull/N>   # also advances status to pr
mhtodo show <id> --json                        # confirm status is "pr" and pr_url is set
```

  Do not skip the `show` check. Clearing `--pr-url` does not move the ticket out of `pr`.
- **Reopen:** more work while in `review` or `pr` → parent `wip` + new sub-tasks; do not pile work under a review/PR card.

## 8. Task picker (“what’s next?”)

When they ask what’s next / todos / pick a task:

1. `mhtodo list --roots --json` (board order; do not re-sort or omit rows).
2. Present every row with the host picker (`AskUserQuestion` / `AskQuestion`): status, updated date, title, helpful description — **no ids in labels**. Include “none / something else”.
3. Adopt only after an explicit pick. Empty list → ask if they want a new root (still ask before creating).

## 9. Subagents

Share the parent ticket. Do not register a separate root per subagent. Each subagent drives its sub-task; the orchestrator owns parent `waiting` / `review` / `pr`.

## 10. Housekeeping

Never unprompted `archive` or `rm`. You may `rm` a task you created in error.
