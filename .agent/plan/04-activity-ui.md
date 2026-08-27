# 04 — Activity UI

- Detail pane composer: activity + comment fields → AddActivity
- View tab **Activity** (`a`); persist `mhtodo.view`
- Feed: non-archived tasks' activity, newest first
- Filter: checkbox multi-select of tickets (empty = all)
- Card: dual date (rel + absList) + `task` / `task > subtask` header; content `[activity pill] — comment`
- Listen on `tasks:changed` with `op: activity` (same event bus)
