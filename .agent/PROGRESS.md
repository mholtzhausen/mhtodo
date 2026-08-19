# mhtodo — v0.1 progress (updated 2026-08-19)

Status: M1 done ✅ — core + store with full unit tests; ready for M2 (CLI). Everything below is copy-pasteable to Slack as-is.

## Milestones
- [x] **M0** Tray spike — Wails + systray coexist on this machine (gate for everything else; fallbacks not needed)
  - [x] Tray icon appears (AppIndicator), menu works
  - [x] Show/hide window from tray, clean quit
- [x] **M1** Core + store — internal/store (WAL, migrations) + internal/core (Task/Status/Service); unit tests for transitions, prefix matching, validation, migration idempotency (+ concurrent-writer test; `go test -race` green)
- [ ] **M2** CLI — all commands per 04-cli-spec.md, --json everywhere, exit codes, non-TTY rm guard; golden tests incl. error paths → fully usable agentic interface with zero GUI
- [ ] **M3** GUI MVP — Wails frontend (Vite+Svelte+Tailwind v4), app.go bindings per parity contract, single-instance lock; list view + new task dialog + detail drawer = full CLI parity in UI
- [ ] **M4** Tray + notifications + live sync — hide-to-tray close, tooltip counts, notify-send on →done/→waiting, fsnotify watcher (+2s poll fallback) → tasks:changed
- [ ] **M5** Polish — board/kanban view + column quick-add, keyboard shortcuts, styling pass, empty states, error toasts (stretch: drag-and-drop, light theme)
- [ ] **M6** Packaging & docs — Makefile finalize (arm64 guard), .desktop file, app+tray icons, README with agent contract section, make install smoke test

## Definition of done (v0.1)
- [ ] make install → launcher entry, tray icon works, close hides to tray
- [ ] Full CLI ↔ GUI parity (table in 02-architecture.md holds both ways)
- [ ] mhtodo list --json stable & documented; an agent can drive the full lifecycle via CLI alone
- [ ] make test && make lint green; DB survives concurrent CLI+GUI use

## Notes / blockers
- **M0 findings (2026-08-19):** validated pattern = `systray.Register()` called *before* `wails.Run()` (never `systray.Run()` — that starts a second gtk_main). Register does gtk_init + AppIndicator pre-loop; all later tray mutations queue via g_idle_add onto Wails' single GTK loop. Show/hide/quit verified headless (`./mhtodo --selftest` → exit 0) and manually on the desktop (icon, menu, JS show/hide).
- **Wails v2 mode-tag gotcha:** raw `go build -tags webkit2_41` compiles but fails at runtime ("will not build without the correct build tags"). GUI builds need a mode tag: `wails build`/`wails dev` inject `production`/`dev` automatically; plain `go build` must use `-tags "webkit2_41 production"`. CLI-only paths (M2) never call wails.Run and are unaffected.
- Spike code lives at repo root (`main.go`, `frontend/index.html`, `assets/tray.png`) — reworked into the M3 layout; tray wiring moves to `internal/tray` in M4 using the validated Register pattern.
- **M1 design notes (2026-08-19):** `core` defines the domain types + a `TaskRepository` interface and holds all business rules (transition effects, prefix resolution ≥4 chars with sorted-candidate ambiguity error, validation); `store` implements it over SQLite (WAL + busy_timeout=5000 via DSN `_journal_mode=wal&_busy_timeout=5000`, versioned forward migrations in one tx, single-statement writes incl. `DELETE … RETURNING`). Timestamps are RFC3339 UTC strings per spec; UUIDv7 IDs. Edit never changes status (done tasks keep `completed_at`); only SetStatus applies transition effects.
