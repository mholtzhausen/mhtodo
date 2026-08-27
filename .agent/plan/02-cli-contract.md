# 02 — CLI contract (v0.3)

## Status

All status flags/help: `pending|wip|waiting|review|done`.

## Sub-tasks

- `mhtodo add TITLE --parent ID` — create child under root parent
- `mhtodo list --roots` — top-level only (default list stays flat with `parent_id` field)
- `mhtodo rm ID` — if parent has children, confirm mentions cascade count; non-TTY still needs `--yes`

## Activity

```
mhtodo activity add ID --activity TEXT [--comment TEXT]
mhtodo activity list [--task ID]... [--limit N] [--json]
mhtodo activity rm ID [--yes]
```

At least one of `--activity` / `--comment` required. Errors: empty both → usage (1); unknown task → 2.

## JSON

Task includes `"parent_id": null | "<uuid>"`. Activity object as in plan README.
