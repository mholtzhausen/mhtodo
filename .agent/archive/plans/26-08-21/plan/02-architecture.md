# Architecture

## Process model

One binary, two entry paths:

```
mhtodo            → GUI (Wails app + tray)        [also: mhtodo gui]
mhtodo <command>  → CLI, exits when done          [agentic path]
```

Dispatch in `main.go`: if `len(os.Args) > 1` and the first arg is a known cobra command (or `-h/--help`,
`--version`) → run CLI. Otherwise → start GUI. Cobra's own help/version flags are handled by building
the root command and checking `root.Find(args)` before deciding, so `mhtodo --help` prints CLI help
and bare `mhtodo` opens the app.

Both paths construct the same `core.Service` over the same DB file. There is **no** shared in-memory
state between processes — SQLite (WAL) is the coordination layer.

## Project layout

```
mhtodo/
├── main.go                  # dispatch: CLI vs GUI; version vars (ldflags)
├── app.go                   # Wails App struct + bound methods + lifecycle (GUI path only)
├── go.mod / go.sum
├── Makefile                 # see 06-makefile.md
├── internal/
│   ├── core/                # DOMAIN — shared by CLI and GUI, no UI/CLI imports
│   │   ├── task.go          #   Task struct, Status type + transitions, validation
│   │   ├── service.go       #   Create/List/Get/Edit/SetStatus/Delete; invariants
│   │   └── service_test.go
│   ├── store/               # PERSISTENCE — sqlite only
│   │   ├── db.go            #   Open(path): WAL, busy_timeout, driver registration
│   │   ├── migrate.go       #   schema_version meta table, forward migrations
│   │   ├── repo.go          #   TaskRepo: SQL for all queries (filter/sort/search)
│   │   └── repo_test.go
│   ├── cli/                 # CLI — cobra commands → core.Service
│   │   ├── root.go          #   root cmd, global flags (--json), output helpers
│   │   ├── add.go list.go show.go edit.go status.go archive.go rm.go path.go gui.go
│   │   └── cli_test.go      #   golden tests: run command against temp DB, assert stdout/exit
│   ├── tray/                # GUI-only — systray wiring (icon, menu, callbacks)
│   │   └── tray.go
│   ├── notify/              # desktop notifications (notify-send wrapper + dedupe)
│   │   └── notify.go
│   └── sync/                # live sync: fsnotify on DB file → callback
│       └── watcher.go
├── frontend/                # Vite + Svelte 5 + Tailwind v4 (Wails-managed bindings in wailsjs/)
│   ├── src/
│   │   ├── App.svelte
│   │   ├── lib/api.ts       # thin wrapper over generated wailsjs bindings
│   │   ├── lib/store.svelte.js   # svelte 5 runes store: tasks, filter, selection
│   │   └── components/{Board,List,TaskCard,TaskDetail,NewTaskDialog,FilterBar}.svelte
│   └── package.json / vite.config.js / tailwind (v4 css-first config)
├── assets/
│   ├── icon.png             # 512 app icon; tray icon derived 22×22 template
│   └── tray.png
└── packaging/
    ├── mhtodo.desktop       # Terminal=false, Exec=mhtodo
    └── README.release.md
```

## Bound API (Go ↔ JS) — parity contract

`app.go` exposes exactly the service surface; frontend never touches SQL or business rules:

| Method | Maps to CLI | Notes |
|---|---|---|
| `ListTasks(filter ListFilter) []Task` | `list` | filter: status, search, limit, sort, includeDone, archived (v0.2; default hides archived) |
| `GetTask(id string) Task` | `show` | prefix match allowed (same helper as CLI) |
| `CreateTask(in CreateInput) Task` | `add` | |
| `UpdateTask(id string, patch UpdateInput) Task` | `edit` | title/description/progress |
| `SetStatus(id string, status Status) Task` | `status` / `done` | fires notify + event on done/waiting |
| `ArchiveDone() []Task` | `archive` | v0.2: bulk-archives all done tasks (board Done-column button); no-op when empty |
| `Unarchive(id string) Task` | `unarchive` | v0.2: archived → pending, progress reset to 0 |
| `DeleteTask(id string)` | `rm` | |
| `DBPath() string` | `path` | shown in settings/about |

After every mutation the app emits Wails event **`tasks:changed`** (payload: `{id, op}`). The external
watcher (`internal/sync`) also emits it when the DB file changes from another process. Frontend
handler = refetch current view; cheap at this scale and always consistent.

## Concurrency rules

- SQLite opened with `PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000`.
- GUI holds one long-lived connection pool (max 1 writer); CLI opens/closes per invocation.
- All writes are single-statement transactions in the repo layer. No cross-process locking beyond SQLite's.
