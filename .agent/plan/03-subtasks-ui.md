# 03 — Sub-tasks UI

- Persist `localStorage['mhtodo.showSubtasks']` (default true); shortcut `s`
- List: group/indent children under parent when shown; omit children when hidden
- Board: nested child rows under parent card; only roots in columns / DnD
- Nested sub-tasks display in creation order (oldest first) on board, list, and detail
- Detail: “Add sub-task”; NewTask optional parent; subtask shows parent link (no full UUID in chrome)
- Delete confirm names child count when cascading
