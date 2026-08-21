# GUI Specification (Wails v2 + tray)

## Window & lifecycle

- Default size 1100×720, min 800×560. Title `mhtodo`. Dark theme default (follows a light/dark toggle in the UI; CSS custom properties).
- **Close button → hide to tray** (intercepted via Wails `OnBeforeClose` returning true), *except* when
  quitting from the tray menu or `Ctrl+Q`. First launch shows the window; subsequent launches while
  already running: focus existing instance — enforce single-instance with a lock file in
  `$XDG_RUNTIME_DIR/mhtodo.lock` (stale-lock detection via pid).
- Tray icon: monochrome template PNG (22×22) for AppIndicator. Tooltip: `mhtodo — N open tasks`
  (open = not done), refreshed on every change.

## Tray menu

```
Show / Hide mhtodo        # toggles window visibility + focus
New Task                  # shows window and opens the create dialog
─────────────
Quit                      # real exit: systray.Quit() + wails quit
```

Implementation: `systray.Run(onReady, onExit)` in a dedicated goroutine started **before** `wails.Run`
(M0 spike validates this ordering on Linux; fallback library `leaanthony/systray`). Menu callbacks talk
to the Wails app through channels — never touch GTK from non-main threads beyond what systray itself does.

## Views (frontend)

Single-page app, three views + detail drawer:

1. **Board** (default): kanban columns `pending | wip | waiting | done`, cards show title, progress
   bar, relative updated time. Column headers show counts and allow quick-add per column. The Done
   column header additionally carries an **archive button** (v0.2): one click archives everything in
   the column (reversible — no confirm dialog), shows a toast with the count, and empties the column
   via the normal `tasks:changed` refetch. Archived tasks never appear on the board.
2. **List**: sortable table mirroring CLI `list` flags (status filter chips, search box, sort by any
   field). Row click → detail drawer. Filter chips include a neutral **Archived** chip (v0.2):
   All/status views hide archived tasks; only the Archived chip shows them.
3. **Detail drawer** (right side): full task — title (inline edit), description (textarea, markdown-ish
   plain text for v1), status segmented control, progress slider + numeric input, created/updated/completed
   timestamps, delete button (confirm dialog). Every field editable → full CLI parity. For archived
   tasks the drawer shows an **Archived** timestamp and an **Unarchive → pending** button (v0.2):
   restoring moves the task to pending with progress reset (rule owned by `core.Unarchive`).

Toasts (errors/info, e.g. the archive count) auto-dismiss after **3s** by default; individual call
sites may pass a different lifetime when a message needs it.

Plus: **New Task dialog** (title required; desc/status/progress optional) reachable from toolbar, tray,
and `Ctrl+N`. Keyboard: `/` focus search, `n` new task, `Esc` close drawer/dialog/confirm, `1..4` switch status filter, `5` toggle archived view (list), `Delete` deletes the selected task (confirmation dialog).

## Notifications (`internal/notify`)

- Backend: `exec.Command("notify-send", "-a", "mhtodo", "-i", "dialog-task", summary, body)`;
  app name constant for grouping; failures are logged, never fatal.
- Triggers (GUI process): task → **done** ("Task completed: <title>"), task → **waiting**
  ("Waiting on external: <title>"). Not triggered by plain edits/progress ticks (noise).
- Dedupe: suppress repeat notification for same id+status within 60s.
- CLI does **not** notify by default; `mhtodo done ID --notify` flag as opt-in for agents that want it.

## Live sync (CLI → GUI)

```
internal/sync.Watcher(dbPath, onChange)
  ├─ fsnotify on the DB file (WAL writes touch it)
  └─ fallback: poll `SELECT max(updated_at)` every 2s; emit only when changed
→ app emits Wails event "tasks:changed" → frontend refetches current view
```

Debounce 300ms. The same event is emitted after local (GUI-initiated) mutations, so the frontend has
exactly one refresh path regardless of origin.

## Styling direction ("modern beautiful")

- Tailwind v4 (CSS-first config), dark slate/zinc palette, single accent color (e.g. indigo).
- Inter (system fallback stack; no webfont download — offline-friendly).
- Subtle motion only: drawer slide-in, card hover lift, progress bar transition (svelte/transition, ≤150ms).
- No component library in v1 — hand-rolled components keep the bundle tiny and the look distinctive.
