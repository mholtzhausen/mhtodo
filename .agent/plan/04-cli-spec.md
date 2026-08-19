# CLI Specification (agentic interface)

Design goals: predictable, scriptable, low-noise. Every command supports `--json` (machine output);
default human output is compact plain text (no ANSI unless TTY). All commands are safe to run
concurrently with the GUI.

## Global flags

| Flag | Meaning |
|---|---|
| `--json` | emit JSON instead of human format (objects for single tasks, arrays for lists) |
| `-q, --quiet` | suppress non-essential output (e.g. only print ID on `add`) |
| `MHTODO_DB_PATH` env | override DB file location |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | usage/validation error (bad flag, bad status, progress out of range) |
| 2 | not found / ambiguous ID |
| 3 | storage error (DB locked beyond busy_timeout, corrupt file, permissions) |

Errors go to **stderr** as `mhtodo: <message>`; with `--json`, stderr carries `{"error":"<code>","message":"..."}`.

## Commands

### add
```
mhtodo add TITLE [--desc TEXT] [--status pending|wip|done|waiting] [--progress 0-100]
```
Creates a task; prints the full created object (JSON with `--json`, one line `id  status  title` otherwise).

### list / ls
```
mhtodo list [--status S] [--search TEXT] [--limit N] [--sort FIELD[+|-]] [--all]
```
- Default: **excludes done**, sorted `updated_at desc`. `--all` includes done.
- `--search`: case-insensitive LIKE over title + description.
- `--sort`: `created`, `updated`, `status`, `progress`, `title`; default `updated`; `-` prefix = ascending.
- Human format: aligned columns `ID(8)  STATUS  PROG  UPDATED(rel+abs)  TITLE`.

### show / get
```
mhtodo show ID            # full detail, all fields incl. created_at/updated_at/completed_at
```
Human format is a labeled block; JSON is the canonical object shape (below).

### edit
```
mhtodo edit ID [--title TEXT] [--desc TEXT] [--progress 0-100]
```
At least one flag required. Prints updated object.

### status / set
```
mhtodo status ID pending|wip|done|waiting
mhtodo done ID            # shortcut for `status ID done`
```
Prints updated object (so agents can confirm the transition + timestamps).

### rm / remove
```
mhtodo rm ID [--yes]
```
Interactive confirmation on a TTY; **non-TTY requires `--yes`** (agents must pass it) — fails with exit 1 otherwise. Prints deleted task's id only.

### path
```
mhtodo path               # prints the DB file path (useful for agents/debugging)
```

### gui
```
mhtodo gui                # explicit GUI launch; identical to bare `mhtodo`
```

## Canonical JSON object

```json
{
  "id": "01958b2e-4c1a-7f3d-9a6b-2c8e4f5a6b7c",
  "title": "Ship mhtodo v0.1",
  "description": "Ship mhtodo v0.1 (see .agent/plan/)",
  "status": "wip",
  "progress": 40,
  "created_at": "2025-08-19T07:59:00Z",
  "updated_at": "2025-08-19T08:30:12Z",
  "completed_at": null
}
```

`--json list` → array of these. Field names are stable API — document in README as the agent contract.

## Agent usage examples (for README)

```bash
mhtodo add "Refactor auth" --desc "Split token + session" --json | jq -r .id
mhtodo list --status wip --json
mhtodo show 01958b2e --json
mhtodo edit 01958b2e --progress 60
mhtodo status 01958b2e waiting
mhtodo done 01958b2e
```
