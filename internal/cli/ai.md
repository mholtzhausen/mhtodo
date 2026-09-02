<!-- Emitted by: mhtodo ai
     Integration contract version: {{INTEGRATION_VERSION}}
     mhtodo binary version:        {{MHTODO_VERSION}}
     Database:                     {{MHTODO_DB_PATH}}
     Generated:                    {{TIMESTAMP}}
     Everything in double-brace placeholders is interpolated by the CLI at emit time. -->

# mhtodo — agent integration instructions

You are reading this because someone ran `mhtodo ai`. This document is the
**complete, authoritative specification** for wiring mhtodo into an AI coding
agent's workflow. Read all of it before doing anything.

Your job now is to **install or upgrade that integration on this machine** — but
not silently. Follow the flow in [§7](#7-what-to-do-right-now).

---

## 1. What mhtodo is, and why the integration exists

`mhtodo` is a personal task manager with a CLI and a GUI dashboard. The dashboard
is open on the user's screen while agents work.

**It is a two-way channel between the user and their agents, not a log file.**

- **User → agent.** They write tasks and comments in the GUI, then point an agent
  at one: *"do that one"*, *"what's next?"*.
- **Agent → user.** The agent reports through status, progress, sub-tasks,
  **activities** (live running commentary), and **feedback** (a short post-work
  summary with notes and takeaways on the task itself).

Every design decision below follows from that. An integration that treats mhtodo
as a place to dump status lines has failed, even if every command succeeds.

### The one hard rule

> **An agent may read the board freely. An agent may NEVER adopt or start a task
> because it found one there — including tasks marked `human_only`.**

Work begins in exactly two ways: the user asks for something in the session (and
the agent then searches mhtodo for a matching existing task), or the user points
at a specific task. There is no third way. Listing is not starting. Skip any
`human_only` row even when the user did not use `--human-only` on list.

Any integration that lets an agent pick up work autonomously is wrong and must be
corrected during upgrade.

---

## 2. CLI surface

Authoritative for binary version `{{MHTODO_VERSION}}`. Re-read this section on
every upgrade; commands and flags change between versions.

```
mhtodo add TITLE [--desc S] [--feedback S] [--status S] [--progress N] [--parent ID] [--cwd S] [--slack-thread URL] [--human-only]
mhtodo edit ID [--title S] [--desc S] [--feedback S] [--progress N] [--cwd S] [--slack-thread URL] [--human-only | --no-human-only]  # at least one flag
mhtodo status ID {{STATUS_ENUM}}
mhtodo done ID [--notify]
mhtodo show ID
mhtodo list [--all] [--archived] [--roots] [--human-only] [--status S] [--search S] [--sort F] [--limit N]
mhtodo reorder ID [--before ID]
mhtodo activity add ID [--activity S] [--comment S]       # at least one
mhtodo activity list [--task ID ...] [--limit N]
mhtodo activity rm ID --yes
mhtodo rm ID --yes                                        # CASCADES to sub-tasks
mhtodo archive | mhtodo unarchive ID
mhtodo path
mhtodo slack report                                     # paste-ready board summary for Slack
mhtodo ai                                                 # this document
mhtodo update [--check] [--force]                         # self-update from GitHub Releases
```

- Global flags: `--json`, `-q/--quiet`.
- IDs accept any unique prefix of **4+ characters**. Prefixes are time-ordered, so
  tasks created seconds apart share long prefixes — **display 13 characters** in
  any generated listing, not 8.
- `--sort` fields: `{{SORT_FIELDS}}`; default `board`; suffix `-` ascending, `+`/none descending.
- Sub-tasks are **one level deep**. Passing a sub-task to `--parent` is rejected
  with `parent_is_child`.
- `mhtodo list` excludes `done`, archived, and **human-only** tasks by default.
  Pass `--human-only` to include human-only rows (for the human's own review, not
  for agent task pickers).

**Statuses:** `{{STATUS_ENUM}}`

| Status | Meaning |
|---|---|
| `pending` | Not started. The user's queue of work waiting for an agent. |
| `wip` | **A live session is working on this right now.** |
| `waiting` | Blocked on the user. A question was asked and cannot proceed. |
| `review` | Delivered, waiting on the user's judgement. |
| `done` | Complete and verified. |

**Task fields:** `id`, `title`, `description`, `feedback`, `status`, `progress` (0–100),
`parent_id`, `board_rank`, `cwd`, `human_only`, `slack_thread`, `created_at`, `updated_at`, `completed_at`, `archived_at`.
**Activity fields:** `id`, `task_id`, `activity`, `comment`, `created_at`.

**`cwd`** is an optional absolute path to the project or working directory the task
belongs to. Set it with `--cwd` on `add`/`edit` when the job is tied to a specific
checkout. The GUI folder picker sets the same field.

**`slack_thread`** is an optional Slack thread URL for this ticket. When set, `show`,
`show --markdown`, and `slack report` include the reminder:
`the primary thread on slack for communication regarding this ticket is : <url>`.
Set with `--slack-thread` on `add`/`edit`; empty string clears it.

**`human_only`** marks a task the user will handle themselves. Agents must **never**
adopt, start, or update human-only tasks. Default `mhtodo list` hides them; only
include them when the user explicitly asks to see their personal queue
(`mhtodo list --human-only`).

### Markdown fields

The GUI **markdown-renders** these when not in an input/textarea:

| Field | Flag / channel | Notes |
|---|---|---|
| Description | `--desc` | User brief (don't overwrite on adopted tasks) or agent's current-state snapshot |
| Feedback | `--feedback` | Post-work summary — see §3.2 / §3.7 |
| Activity comment | `--comment` on `activity add` | Detail under a chip label |

Use light markdown where it helps (short lists, `` `paths` ``, **emphasis**). Do **not**
put markdown in `--activity` labels — those are plain Title Case chips. Titles stay
plain text.

---

## 3. The behavioural contract

This is what must end up in the agent's always-on instructions (a skill, a rules
file, a system prompt — see [§4](#4-host-mapping)). Encode all of it.

### 3.1 Session state

One pointer file per agent session:

```
${XDG_STATE_HOME:-$HOME/.local/state}/mhtodo-agent/<session_id>
```

Two lines:

```
01a0426d-8d19-7066-b1ca-8de5128f60f8
origin=user
```

`origin=user` means the human wrote the task and the agent adopted it.
`origin=agent` means the agent registered it. A file with only line 1 is treated
as `origin=agent`.

A sibling `<session_id>.seen` file holds the RFC3339 timestamp of the newest
activity already shown to the agent, for the comment relay in §3.6.

### 3.2 Ownership — who may touch what

| | **User's task** | **Agent's task** |
|---|---|---|
| Title | Never change. It is how they find it. | Agent may refine. |
| Description | **Never overwrite.** It is their brief, possibly the whole spec. | Agent's; keep it a current-state snapshot (markdown OK). |
| Feedback | Agent sets/updates at hand-back (`--feedback`). | Same. |
| Status / progress | Agent moves it. | Agent moves it. |
| Narration | **Activities only** while working. | Activities (and the description). |
| Closing | Take to `review`. **The user** marks it done. | Agent may mark `done`. |
| Decomposition | Step plan as sub-tasks on start (§3.5). Never narrow their row. | Same — immediate step plan on start. |

**Feedback** is a short **post-work summary**: outcome in a sentence or two, plus any
notes and takeaways the user should keep (gotchas, follow-ups, decisions). It lives
on the task card (GUI shows it when non-empty). It is **not** a running log —
that is activities. Write or refresh feedback when handing back (§3.7); do not drip
mid-job updates into it.

Unifying habit: **progress narration always goes to activities.** Closing summary
goes to **feedback**.

### 3.3 Starting work — search, then adopt or register

Before investigating anything substantive:

```bash
mhtodo list --roots --json --search "<2-3 distinctive keywords from the request>"
```

- **A result is the same job** → adopt it. A duplicate row beside the user's is
  worse than no task at all.
- **Nothing matches** → register a new one.
- **Genuinely unsure** → ask, in one line.

Adopt:

```bash
mhtodo status <id> wip
mhtodo edit <id> --progress 5          # progress only — never --desc, never --title
mhtodo activity add <id> --activity "Task Picked Up" \
  --comment "<how the brief was read and what happens first>"
printf '%s\norigin=user\n' <id> > "$pointer"
```

That opening activity is the user's chance to correct a misread brief before
anything is built.

Register:

```bash
mhtodo add "[<context>] <short imperative title>" \
  --desc "<one-line goal>" --status wip --progress 5 \
  [--cwd "<absolute path>"] --json
printf '%s\norigin=agent\n' <id> > "$pointer"
```

**Titles** are `[<context>] <imperative phrase>`, where context is the working
directory basename or a short label like `[cloudflare]`. The bracket prefix is how
the user tells concurrent agents apart on the board. Keep titles under ~60 chars.

Skip registration entirely for conversational turns, one-line lookups, and work
already covered by the session's open task.

Immediately after adopting or registering, draft the step plan (§3.5).

### 3.4 Activities — the primary reporting channel

```bash
mhtodo activity add <id> --activity "<Short Label>" [--comment "<detail>"]
```

**Which `<id>`?** When the job has sub-tasks, post activities on the sub-task
being worked — they are actions taken to complete that step. When there are no
sub-tasks, post on the parent. Use judgment for job-wide moments (hand-back,
plan changes) — parent is fine.

**`--activity` is a label, not a sentence.** Two to four words, Title Case, a noun
phrase — the kind of thing that fits on a chip in a list. The GUI renders it as a
label beside the task, so a sentence overflows and stops being scannable.

**`--comment` carries everything else** — numbers, paths, queries, findings, a
decision and its reasoning. Detail belongs here (markdown OK), never in the label.

```bash
# Bad — the whole finding jammed into the label
--activity "Infra clean: 0 restarts in 2d, 0 not-ready, 0 OOMKills. Memory: app 151MiB, exporter 132MiB"

# Still bad — a sentence, just a shorter one
--activity "Infra checks came back clean"

# Good — the label is a chip, the comment carries the finding
--activity "Infra Checks Clean" \
  --comment "0 container restarts over 2d, 0 pods not ready, 0 OOMKills. Working set: app 151 MiB, exporter 132 MiB."
```

Post one when:

| Moment | Label |
|---|---|
| Adopting or registering the task | `Task Picked Up` |
| Finishing investigation, shape now known | `Root Cause Found` |
| Touching a significant file | `GCS Loader Rewritten` |
| Making a decision the user might challenge | `Patching Upstream` |
| Hitting a surprise | `Pillow-SIMD Gap` |
| Committing, pushing or opening a PR | `PR #142 Opened` |
| Handing back | `Handed Back` |

**Granularity: one activity per step forward.** The unit is a step that moved the
job along, not a tool call. Sometimes that is a single tool call; more often it is a
small run of them that together settled one thing. If the agent can say what
changed because of it, it is an activity. If it cannot, fold it into the next one.

**Lean fine-grained rather than coarse.** A log that shows *how* the agent got there
is worth more on the board than four summary paragraphs posted at the end. The
failed attempt, the stale cache, the label that turned out not to exist — those are
steps forward and they belong in the log. Two dozen short labelled entries across a
long job is a healthy trace, not noise.

Roughly one every few minutes of real work. **Twenty minutes of silence reads as a
stalled agent.**

### 3.5 Sub-tasks — step plan, one level

When a root task is adopted or registered and work begins, **immediately** draft
a step plan for the job.

- **Two steps or fewer** — work on the parent only. No sub-tasks.
- **Three or more steps** — create one sub-task per step under the parent as soon
  as the plan is clear:

```bash
mhtodo add "<step title>" --parent <parent-id> --status pending --json
# when work on a step starts (possibly in parallel — §3.9):
mhtodo status <step-id> wip
# when a step is finished:
mhtodo status <step-id> done
```

Sub-tasks are the visible breakdown — especially valuable when the user wrote a
coarse task: the plan appears without touching their row, and they can object to
it. Three or four steps is a solid plan; twelve is a checklist nobody reads.

> **Sub-task or activity?** A sub-task is a *planned step* in the breakdown.
> An activity is an *action taken while completing* a step. "Add CSV serializer
> endpoint" is a sub-task; `Serializer Implemented` is an activity on that
> sub-task.

Sub-tasks use **`pending` → `wip` → `done` only** — never `review` or `waiting`.

**Parent row is the job.** Parent status and progress reflect the overall job —
informed by sub-task completion, but not mechanically derived from it. Do not
treat "all sub-tasks done" as synonymous with "job finished" unless the work is
actually complete.

**Replan freely.** Add, remove, or reorder sub-tasks whenever the shape of the
job changes — new findings, user input, blockers, scope shifts. Drop obsolete
steps (`rm` only for rows created in error; otherwise mark `done` with a brief
activity explaining why).

**Blocking moves the parent.** Sub-tasks render inside the parent card on the
board, not as separate kanban columns. When user input is required, set the
**parent** to `waiting` — the whole job is blocked until they respond (§3.7).
Context injection flips the parent back to `wip` on the next user turn.

### 3.6 Reading the user's comments

The user writes comments on tasks in the GUI mid-job. Treat a comment from them as
an instruction, and acknowledge it with an activity on that same task so they can
see it landed.

```bash
mhtodo activity list --task <id> --json   # the task they commented on; repeat --task for each sub-task if needed
```

### 3.7 Handing back

Never end a turn leaving the **parent** on `wip` — that claims a live agent is on it.

If the job has sub-tasks, **every sub-task must be `done`** before the parent
can move to `review`. Incomplete steps mean the job is not ready to hand back.

```bash
mhtodo activity add <parent-id> --activity "Handed Back" --comment "<files, PR, outcome>"
mhtodo edit <parent-id> --progress 100 --feedback "<short summary + notes/takeaways>"
mhtodo status <parent-id> review     # user's task — they close it
# or, for the agent's own task:
mhtodo edit <parent-id> --desc "<what was delivered, past tense>" \
  --feedback "<short summary + notes/takeaways>" && mhtodo done <parent-id>
```

**`--feedback` at hand-back** is required whenever substantive work happened: a
brief outcome plus notes and takeaways (markdown OK — e.g. a short bullet list).
Keep it scannable; put the blow-by-blow in activities, not here.

Use `waiting` on the **parent** instead when blocked on an answer — the whole
card moves to the Waiting column (sub-tasks stay nested inside it). The closing
activity is the step-trace the user can skim; **feedback** is what they read
later for the digest — files, PRs, outcomes, gotchas — not "finished the task".

### 3.8 Picking a task — "what's next?", todos, the list

When the user wants to see their board, pick work, or continue something — e.g.
*what's next?*, *next task*, *todos*, *todo list*, *what should I work on?*,
*pick up a task*, *show me my tasks*, *what's on my list?* — treat it as a
**task-picker turn**, not a guess.

**Read-only until they choose.** Never adopt or start a task from this flow
without an explicit pick (§1).

1. **Fetch the board** — root tasks only (what appears as cards on the kanban):

```bash
mhtodo list --roots --json
```

Use default list behaviour: open agent-eligible tasks (excludes `done`, archived,
and **human-only**) in **board order** — the same order the GUI uses for agent
work (status workflow → rank → `updated_at`). **Do not re-sort, split into groups,
or omit rows** (including `wip`).

2. **Present every row with AskUserQuestion** — the host's structured
   multiple-choice UI. Each option must clearly show:

   - **status** (`pending`, `wip`, `waiting`, `review`)
   - **date** — `updated_at` from the JSON, human-readable
   - **title**
   - **description** — include when non-empty and it helps distinguish similar titles

   Do **not** show task ids in the picker labels — the user picks by meaning; map
   the selection back to the task id from the JSON you already fetched.

   List options in **exactly** the order `mhtodo list` returned them. Include at
   least one escape hatch (e.g. *None of these / something else*) so the user is
   not forced to pick from the list.

   | Host | Mechanism |
   |---|---|
   | Claude Code | `AskUserQuestion` |
   | Cursor | `AskQuestion` |
   | Other | Host equivalent — never prose-only when a picker exists |

   Do **not** answer by narrating a single recommendation or auto-picking the
   first `pending` row.

3. **After they pick** — adopt per §3.3 (or continue if it is already the
   session's open task). Only then set `wip` and begin work.

If the list is empty, say so and offer to register new work — still do not
register until they describe what they want.

### 3.9 Subagents

Subagents share the parent session's task and pointer, and must **not** register
their own root task — that fragments one job into several dashboard rows.

Each subagent works a step: set that sub-task to `wip` and post activities on
**that sub-task**. Multiple steps may be `wip` in parallel across subagents.
The orchestrator owns the parent row — creating and replanning sub-tasks, setting
the parent to `waiting` or `review`, and closing the job.

### 3.10 Never narrate the bookkeeping

No "let me create a task for this", no "I've updated the task", no reciting ids
back. It happens silently alongside the real work. **The dashboard is the
notification** — that is the entire point. The only exception is when the user
asks about the board directly.

### 3.11 Housekeeping is the user's call

`mhtodo archive` and `mhtodo rm` are never run unprompted. A `done` task staying
visible is how the user sees what just finished, and `rm` cascades to sub-tasks.
An agent may only `rm` a task it created in error.

---

## 4. Host mapping

The contract above is host-agnostic. Map it onto whatever the agent runtime
offers. Three behaviours must be automated, because they cannot be left to the
agent remembering:

| Behaviour | Trigger | Effect |
|---|---|---|
| **A. Context injection + candidate search** | Before each user turn is processed | Inject the active task's state, origin, sub-tasks, feedback, and any new activity; when no task is registered, keyword-search the user's prompt against open root tasks and offer candidates — **unless** the turn is a task-picker request (§3.8), in which case follow §3.8 instead. Flip a `waiting` task back to `wip`. |
| **B. Ghost cleanup** | Session ends | If the task is still `wip`, set `waiting` and post an activity saying why. **Leave the pointer file in place** so a resumed session picks it back up. |
| **C. Idle detection** | Runtime goes idle awaiting user input | If the task is `wip`, set `waiting`. **Skip permission/approval prompts** — nothing reliably fires when one is granted mid-turn, so the task would strand on `waiting` through an hour of real work. |

### Claude Code (reference implementation)

| Behaviour | Hook event | Script |
|---|---|---|
| Contract (§3) | — | Skill at `~/.claude/skills/mhtodo/SKILL.md` (or a plugin skill) |
| A | `UserPromptSubmit` | `~/.claude/hooks/mhtodo-reminder.sh` |
| B | `SessionEnd` | `~/.claude/hooks/mhtodo-session-end.sh` |
| C | `Notification` | `~/.claude/hooks/mhtodo-notification.sh` |

Hooks receive a JSON payload on stdin (`session_id`, `cwd`, `prompt`, `message`,
`reason` depending on event). A `UserPromptSubmit` hook's stdout is injected into
the turn's context. Register them in `~/.claude/settings.json` under `hooks`,
**appending to** any existing array for that event — never replacing it.

### Other hosts

- **Cursor / Windsurf / Copilot-style rules files** — write §3 into the rules
  file. Behaviours A–C are usually unavailable; instead add an explicit
  instruction to run `mhtodo show <id>` at the start of each turn and to set a
  terminal status before replying. **§3.8 task-picker turns must use `AskQuestion`
  (or equivalent)** — not a prose list. Tell the user which parts could not be
  automated.
- **Agent SDKs / custom harnesses** — map A/B/C onto the pre-turn, session-teardown
  and idle callbacks.
- **No hook mechanism at all** — install the contract only, and say so plainly.

### Hook implementation notes

Every hook must be **defensive**: `command -v mhtodo || exit 0`, wrap `mhtodo`
calls with a timeout, swallow parse failures, and **always exit 0**. A task
tracker that breaks the user's agent is worse than no task tracker.

The context-injection hook should also:
- track a `.seen` marker so activities are relayed once, not replayed every turn —
  and treat a *missing* marker as "baseline, show nothing" while an *empty* marker
  means "show everything", or the very first activity is silently swallowed;
- display 13-character id prefixes (see §2);
- surface non-empty `feedback` when injecting task state (it is the last hand-back
  digest, not live narration).

---

## 5. Installation state and versioning

Record what was installed, so a later `mhtodo ai` can tell an upgrade from a fresh
install:

```
${XDG_STATE_HOME:-$HOME/.local/state}/mhtodo-agent/integration.json
```

```json
{
  "integration_version": {{INTEGRATION_VERSION}},
  "mhtodo_version": "{{MHTODO_VERSION}}",
  "installed_at": "<RFC3339>",
  "host": "claude-code",
  "artifacts": [
    {"path": "~/.claude/skills/mhtodo/SKILL.md", "role": "contract"},
    {"path": "~/.claude/hooks/mhtodo-reminder.sh", "role": "context-injection"},
    {"path": "~/.claude/hooks/mhtodo-session-end.sh", "role": "ghost-cleanup"},
    {"path": "~/.claude/hooks/mhtodo-notification.sh", "role": "idle-detection"}
  ],
  "settings_touched": ["~/.claude/settings.json"],
  "settings_backup": "~/.claude/settings.json.bak-mhtodo-<stamp>"
}
```

Write it after a successful install or upgrade.

---

## 6. Upgrading an existing integration

If `integration.json` exists, or the artifacts above are present, this is an
**upgrade**, not an install. Do not start from scratch and do not blindly
overwrite — the user may have hand-tuned things.

1. **Compare** `integration_version` in the manifest against
   `{{INTEGRATION_VERSION}}` at the top of this document. Equal *and* artifacts
   present and healthy → report "already current at v{{INTEGRATION_VERSION}}" and
   stop. No confirmation prompt needed for a no-op.
2. **Read every installed artifact** and diff its behaviour against §3 and §4.
   Note specifically:
   - contract rules that are missing, and rules present that this document has
     since **changed or reversed** — removals matter as much as additions;
   - commands or flags used that no longer exist in §2;
   - hook behaviours A/B/C that are absent, or wired to the wrong event;
   - **any mechanism that lets the agent start work autonomously** — that
     violates §1 and must be removed;
   - missing **`--feedback` at hand-back** and missing **markdown-field** guidance.
3. **Preserve local customisation.** If an artifact contains user-authored
   material outside this contract, keep it and merge rather than replace.
4. **Present the diff** as a short plan: for each file, what changes and why.
   Then confirm (§7).
5. **Back up** anything you modify (`<file>.bak-mhtodo-<stamp>`) before writing.
6. **Update the manifest** to the new version.

Removed features are the dangerous case: if a previous contract version told the
agent to do something this one no longer mentions, delete that instruction rather
than leaving it to rot into a stale habit.

#### Known reversals to check for

Earlier contract versions carried the opposite of some rules now in §3. An upgrade
that only *adds* the new wording, leaving the old wording in place further up the
file, produces a self-contradicting contract — and the agent will follow whichever
it reads first. Search the installed artifact for these and **delete them**:

| Old wording (delete it) | Now says |
|---|---|
| "Do not post per tool call", "`Ran a grep` is noise" | §3.4: one activity per **step forward**; lean fine-grained |
| "`--activity` is the headline … short and past tense" | §3.4: `--activity` is a **Title Case label, not a sentence** — 2–4 words, noun phrase |
| Sentence-style example labels (`Picked up in a session`, `Traced it to the async GCS loader`, `Handed back for review — 3 files changed`) | Chip-style labels (`Task Picked Up`, `Root Cause Found`, `Handed Back`) |
| Hand-back that only sets progress/status with no `--feedback` | §3.7: set `--feedback` with summary + notes/takeaways |
| Sub-tasks only "as useful" / when pieces have "their own lifecycle" / activity when it "doesn't deserve its own progress bar" | §3.5: immediate step plan on start; sub-tasks are planned steps, activities are actions within a step |
| `waiting` or `review` on sub-tasks | §3.5: sub-tasks use `pending` → `wip` → `done` only; blocking moves the **parent** to `waiting` |
| Task list as prose groups, omitting `wip`, or auto-picking the first `pending` row | §3.8: `mhtodo list --roots --json` in board order → **AskUserQuestion** / **AskQuestion** picker (status, date, title; no ids in labels) |
| Adopting or updating tasks with `human_only: true` | §1 / §2: human-only tasks are user-owned; default list hides them — never adopt |

Example labels matter more than they look: they sit above the rule in the file and
are what an agent actually copies. A correct rule underneath a table of sentences
will not change behaviour.

---

## 7. What to do right now

1. **Detect the host.** Which agent runtime is this? Look for `~/.claude`,
   `.cursor/rules`, `AGENTS.md`, an SDK harness, and so on. If ambiguous, ask.
2. **Detect existing state.** Read `integration.json` and probe for the artifacts
   in §4. Decide: fresh install, upgrade, or already current.
3. **Verify the CLI.** Run `mhtodo --version` and `mhtodo --help`, plus
   `--help` on any subcommand you will generate calls for. **If the real CLI
   disagrees with §2, the CLI wins** — this document may lag the binary.
4. **Present a plan and STOP.** List every file to be created or modified, one
   line each, and what changes. Then ask for explicit confirmation.
   **Write nothing before the user says yes.** They may not have expected `mhtodo
   ai` to modify their machine.
5. **On confirmation, apply it.** Back up first. Merge into existing config arrays
   rather than replacing them.
6. **Verify.** For a hook-capable host, feed each hook a synthetic JSON payload on
   stdin and check the output and any resulting state change. At minimum:
   - context-injection hook with a registered task → prints task state;
   - same hook with no task and a prompt matching an existing task → offers it as
     a candidate;
   - idle hook with a permission-style message → **no** status change;
   - idle hook with an idle-style message → `wip` becomes `waiting`;
   - context-injection hook afterwards → `waiting` returns to `wip`;
   - session-end hook on a `wip` task → becomes `waiting` with an activity.

   Create a throwaway task and pointer for this, and delete both afterwards.
7. **Write the manifest** (§5).
8. **Report** what was installed or changed, what was verified, and — honestly —
   anything the host could not support.

Do not narrate steps 1–3 at length. A short plan, a confirmation, then the work.

---

## 8. Uninstalling

Remove the artifacts listed in the manifest, remove the corresponding entries from
the host's config (leaving other entries untouched), and delete the manifest.
Leave `${XDG_STATE_HOME}/mhtodo-agent/` pointer files and the task database alone
unless explicitly asked — the tasks are the user's data.

---

## 9. Changelog

The CLI should render its own history here so an upgrading agent can see what
changed rather than diffing blind.

```
{{INTEGRATION_CHANGELOG}}
```
