# mhtodo — v0.1 progress (updated 2026-08-19)

Status: M2 done ✅ — CLI complete per 04-cli-spec.md; golden tests green (`go test ./...` + `-race`). Ready for M3 (GUI MVP). Everything below is copy-pasteable to Slack as-is.

## Milestones
- [x] **M0** Tray spike — Wails + systray coexist on this machine (gate for everything else; fallbacks not needed)
  - [x] Tray icon appears (AppIndicator), menu works
  - [x] Show/hide window from tray, clean quit
- [x] **M1** Core + store — internal/store (WAL, migrations) + internal/core (Task/Status/Service); unit tests for transitions, prefix matching, validation, migration idempotency (+ concurrent-writer test; `go test -race` green)
- [x] **M2** CLI — all commands per 04-cli-spec.md, --json everywhere, exit codes, non-TTY rm guard; golden tests incl. error paths → fully usable agentic interface with zero GUI
  - [x] add / list(ls) / show(get) / edit / status(set) / done / rm(remove) / path — cobra tree in internal/cli (zero GUI imports)
  - [x] --json + -q/--quiet on every command; JSON error envelope {"error","message"} on stderr
  - [x] exit codes 0/1/2/3 mapped from core errors (not_found, ambiguous_id → 2; validation → 1; storage → 3)
  - [x] rm guard: non-TTY requires --yes (exit 1); TTY prompt verified via pty (y/n both ways)
  - [x] golden tests: temp DB via MHTODO_DB_PATH, stdout + exit code per command incl. all error paths; `go test -race` green
  - [x] main.go dispatch: bare mhtodo / mhtodo gui → GUI (M0 spike path until M3), any subcommand → CLI; --version stamped via ldflags
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
- **M2 notes (2026-08-19):** CLI is a cobra tree in `internal/cli` with zero GUI imports — main.go dispatches bare/`gui` to the M0-spike Wails path before the CLI runs, so `go test ./...` never touches GTK. JSON error envelope codes (stable contract for README): `not_found`, `ambiguous_id`, `empty_title`, `invalid_status`, `progress_range`, `no_fields`, `usage`, `storage`. Spec ambiguity resolved: `--sort FIELD[+|-]` — the prose says "- prefix = ascending" but the syntax is a suffix; implemented as **suffix**, `-` = ascending, `+` or none = descending (documented in flag help). Service default clock now truncates to whole seconds so returned objects match persisted RFC3339 values. rm's TTY check reads package var `cli.Stdin` (test seam); non-*os.File readers are never TTYs. Cobra auto-commands trimmed: `completion` disabled, `help` hidden from the unknown-command hint but still callable. Makefile is M2-scoped (build/test/lint/fmt/tidy/path/clean); dev/fe-*/release/install land in M3/M6 per 06-makefile.md — `make build` uses plain `go build -tags "webkit2_41 production"` until the Vite frontend exists. UUIDv7 caveat: tasks created within the same second share their first ~8 hex chars, so short prefixes can be ambiguous right after a burst of adds (by design; ambiguity error lists candidates).
