# 10 — Tray menu notifications

Menu-only attention surface (no floating toast window).

## Behavior

- Tray **label** shows configured attention statuses with counts (`mhtodo · 2 waiting, 1 review`), else open-task count.
- Tray **status submenus** (order from Settings) list root tasks (newest `updated_at` first, capped); click → show window + `focus-task`.
- Tray **Settings** item shows the window and opens the Settings dialog.
- `notify-send` on →waiting / →done remains, gated by Settings toggles.
- Refresh on every `tasks:changed` and on settings save (CLI/external included).

## Config (`config.yml` → `notifications`)

- `tray_label_statuses` (default `waiting`, `review`)
- `tray_menu_statuses` (default `waiting`, `review`)
- `max_items_per_status` (default 10, max 20)
- `notify_send_wip` / `notify_send_waiting` / `notify_send_done` (default **false**)
- `notify_send_review` (default **true**)

## Code

- `internal/traymenu` — pure label/section helpers
- `internal/tray` — fixed slot submenus + `UpdateStatusMenus`
- `app.refreshTray` / `openFocusTaskFromTray`
- Settings page + `GUISettings.Notifications`
