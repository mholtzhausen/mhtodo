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
mhtodo status ID icebox|pending|wip|waiting|review|pr|done
mhtodo status migrate FROM TO
mhtodo edit ID --session UUID|--progress N|--feedback S|--desc S|--title S|--pr-url URL [--pr-url URL…]
  # --pr-url is repeatable; each edit replaces the full PR list (not append)
mhtodo add TITLE [--parent ID] [--template REF] [--cwd PATH] [--json]
mhtodo activity add ID --activity "Label" --comment "…"
mhtodo activity list --task ID [--json]
mhtodo template search --cwd "$PWD"
mhtodo done ID
```

IDs: unique prefix of 4+ chars; show **13** chars in listings. Default `list` hides `done`, `icebox`, archived, and `human_only`. Icebox parks work outside the pipeline (`list --status icebox` or `--all` to see it).

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
 - when you open a pull request: put every PR URL on the root ticket with `mhtodo edit {rootShort8} --pr-url <url> [--pr-url <url>…]` (repeatable; each edit replaces the full list — `mhtodo show {rootShort8} --json` first if adding another). Never leave PR links only in feedback or comments.
 - (sub-task handoffs only) register pull requests against the main ticket `{rootShort8}`, not this sub-task id
---
```

Treat that as pointing at the ticket. Resolve with `mhtodo show` / `mhtodo open`. Follow the Instructions when you start work. `{rootShort8}` is the pasted ticket id for a root, or the parent id when the paste is a sub-task.

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
- Feedback (`--feedback`): short post-work summary at hand-back only — not a running log. **Never** put pull-request URLs only in feedback (or activity comments) as a substitute for `pr_url`.
- **Ownership:** user-originated → never change title; never overwrite description; hand back to `review` (user marks done), or set `--pr-url` on the **root** when you open a PR (auto-advances to `pr`). Agent-originated → may refine title/desc; may `done`.
- Do not narrate bookkeeping (“created a task…”) unless they asked about the board.

## 6. Step plan (sub-tasks)

One level only. Drive steps `pending → wip → done`. Blocking or hand-back uses the **parent** (`waiting` / `review` / `pr`). Multiple sub-tasks may be `wip` in parallel (subagents). Replan by adding/editing sub-tasks — not by narrowing the user’s root title/description. Pull-request URLs always go on the **root** (`edit <parent> --pr-url …`), never on a sub-task id.

## 7. Blocked / hand back / reopen / pull request

- **Blocked on user:** parent `waiting` + activity with the question.
- **Hand back (no PR yet):** every sub-task `done` → `--progress 100` + `--feedback` → parent `review` (or `done` if agent-owned). Never leave parent on `wip` at end of turn.
- **Opened a pull request:** after creating one or more PRs for this ticket, put **every** PR URL on the **root** ticket’s `pr_url` (if you adopted a sub-task, use the parent id — never register PRs on the sub-task). Do **not** leave PR links only in `--feedback` or activity comments.

  `edit --pr-url` **replaces** the whole list (it does not append). Pass every current PR in one command by repeating the flag. If a ticket already has PRs and you open another, `show --json` first, then re-set **all** URLs including the new one.

```bash
# first PR (or set the full list at once) — <root-id> is the root ticket (parent if you are on a sub-task)
mhtodo edit <root-id> --pr-url <https://…/pull/N> [--pr-url <https://…/pull/M> …]   # advances status to pr

# later: another PR opened — must include prior URLs too
mhtodo show <root-id> --json                        # read existing pr_url (newline-separated)
mhtodo edit <root-id> --pr-url <existing-1> --pr-url <existing-2> --pr-url <new>
mhtodo show <root-id> --json                        # confirm status is "pr" and pr_url lists every PR
```

  Do not skip the `show` check. Clearing with `--pr-url ""` does not move the ticket out of `pr`.
- **Reopen:** more work while in `review` or `pr` → parent `wip` + new sub-tasks; do not pile work under a review/PR card.

## 8. Task picker (“what’s next?”)

When they ask what’s next / todos / pick a task:

1. `mhtodo list --roots --json` (board order; do not re-sort or omit rows).
2. Present every row with the host picker (`AskUserQuestion` / `AskQuestion`): status, updated date, title, helpful description — **no ids in labels**. Include “none / something else”.
3. Adopt only after an explicit pick. Empty list → ask if they want a new root (still ask before creating).

## 9. Subagents

Share the parent ticket. Do not register a separate root per subagent. Each subagent drives its sub-task; the orchestrator owns parent `waiting` / `review` / `pr`, and registers any `--pr-url` on that parent.

## 10. Housekeeping

Never unprompted `archive` or `rm`. You may `rm` a task you created in error.
