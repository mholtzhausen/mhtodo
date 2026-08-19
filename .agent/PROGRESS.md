# mhtodo — v0.1 progress (updated 2026-08-19)

Status: M3 done ✅ — GUI MVP live: Wails + Vite/Svelte/Tailwind frontend, bound API per parity contract, single-instance lock with focus-on-relaunch; list view + new-task dialog + detail drawer = full CLI parity in UI. Verified on the desktop (screenshots). Ready for M4 (tray polish + notifications + live sync). Everything below is copy-pasteable to Slack as-is.

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
- [x] **M3** GUI MVP — Wails frontend (Vite+Svelte+Tailwind v4), app.go bindings per parity contract, single-instance lock; list view + new task dialog + detail drawer = full CLI parity in UI
  - [x] app.go: bound methods ListTasks/GetTask/CreateTask/UpdateTask/SetStatus/DeleteTask/DBPath — exactly the core.Service surface (parity table holds); every mutation emits `tasks:changed` → frontend has one refetch path
  - [x] frontend/: Vite + Svelte 5 (runes) + Tailwind v4 CSS-first; dark zinc palette, indigo accent, Inter system stack; list view with status chips / search / sort+direction (mirrors CLI `list` flags), detail drawer (title/description/status/progress/timestamps/delete-confirm), new-task dialog
  - [x] keyboard: `/` search · `n` new task · `esc` close · `1–4` toggle status filter
  - [x] single-instance lock `$XDG_RUNTIME_DIR/mhtodo.lock` (pid, stale-lock steal) + focus-on-relaunch via SIGUSR2; second launch exits 0 with a log line — all unit-tested in main_test.go and verified live
  - [x] window 1100×720 / min 800×560 per spec; tray wiring kept from M0 (moves to internal/tray in M4)
  - [x] `make build` = wails build (frontend + bindings + ldflags version stamp) → bin/mhtodo; `make dev` hot-reload; desktop smoke test: window renders, selftest show→hide→quit exit 0
- [ ] **M4** Tray + notifications + live sync — hide-to-tray close, tooltip counts, notify-send on →done/→waiting, fsnotify watcher (+2s poll fallback) → tasks:changed
- [ ] **M5** Polish — board/kanban view + column quick-add, keyboard shortcuts, styling pass, empty states, error toasts (stretch: drag-and-drop, light theme)
- [ ] **M6** Packaging & docs — Makefile finalize (arm64 guard), .desktop file, app+tray icons, README with agent contract section, make install smoke test

## Definition of done (v0.1)
- [ ] make install → launcher entry, tray icon works, close hides to tray
- [ ] Full CLI ↔ GUI parity (table in 02-architecture.md holds both ways)
- [ ] mhtodo list --json stable & documented; an agent can drive the full lifecycle via CLI alone
- [ ] make test && make lint green; DB survives concurrent CLI+GUI use

## Notes / blockers
- **M3 findings (2026-08-19):**
  - **NEVER send SIGUSR1 to a Wails/WebKit process.** WebKit/JSC installs its own C handler for signal 10 ("Overriding existing handler for signal 10 … JSC_SIGNAL_FOR_GC"); delivering SIGUSR1 crashes the app with SIGSEGV during cgo execution (reproduced). Focus-on-relaunch uses **SIGUSR2** instead.
  - **This machine's Go toolchains reject `//go:embed` into string/[]byte** ("imported and not used") while embed.FS works — reproduced across go1.24.12/1.25.5/1.25.6 with canonical examples, so it's environmental, not our code. All mhtodo embeds therefore use `embed.FS` (tray icon bytes read out at startup).
  - Go's `flag` stops parsing at the first non-flag arg → `mhtodo gui --selftest` needed a fix: runGUI now strips the `gui` subcommand before flag.Parse.
  - `wails build -o X` writes to `build/bin/X`; Makefile copies it out to bin/. wails build does support `-ldflags`, so version stamping is preserved (deviation from 06-makefile.md's plain `wails build`).
  - Frontend state lives in App.svelte runes rather than a separate lib/store module (plan layout was indicative); single refresh path via tasks:changed unchanged.
- **M0 findings (2026-08-19):** validated pattern = `systray.Register()` called *before* `wails.Run()` (never `systray.Run()` — that starts a second gtk_main). Register does gtk_init + AppIndicator pre-loop; all later tray mutations queue via g_idle_add onto Wails' single GTK loop. Show/hide/quit verified headless (`./mhtodo --selftest` → exit 0) and manually on the desktop (icon, menu, JS show/hide).
- **Wails v2 mode-tag gotcha:** raw `go build -tags webkit2_41` compiles but fails at runtime ("will not build without the correct build tags"). GUI builds need a mode tag: `wails build`/`wails dev` inject `production`/`dev` automatically; plain `go build` must use `-tags "webkit2_41 production"`. CLI-only paths (M2) never call wails.Run and are unaffected.
- Spike code lives at repo root (`main.go`, `frontend/index.html`, `assets/tray.png`) — reworked into the M3 layout; tray wiring moves to `internal/tray` in M4 using the validated Register pattern.
- **M1 design notes (2026-08-19):** `core` defines the domain types + a `TaskRepository` interface and holds all business rules (transition effects, prefix resolution ≥4 chars with sorted-candidate ambiguity error, validation); `store` implements it over SQLite (WAL + busy_timeout=5000 via DSN `_journal_mode=wal&_busy_timeout=5000`, versioned forward migrations in one tx, single-statement writes incl. `DELETE … RETURNING`). Timestamps are RFC3339 UTC strings per spec; UUIDv7 IDs. Edit never changes status (done tasks keep `completed_at`); only SetStatus applies transition effects.
- **M2 notes (2026-08-19):** CLI is a cobra tree in `internal/cli` with zero GUI imports — main.go dispatches bare/`gui` to the M0-spike Wails path before the CLI runs, so `go test ./...` never touches GTK. JSON error envelope codes (stable contract for README): `not_found`, `ambiguous_id`, `empty_title`, `invalid_status`, `progress_range`, `no_fields`, `usage`, `storage`. Spec ambiguity resolved: `--sort FIELD[+|-]` — the prose says "- prefix = ascending" but the syntax is a suffix; implemented as **suffix**, `-` = ascending, `+` or none = descending (documented in flag help). Service default clock now truncates to whole seconds so returned objects match persisted RFC3339 values. rm's TTY check reads package var `cli.Stdin` (test seam); non-*os.File readers are never TTYs. Cobra auto-commands trimmed: `completion` disabled, `help` hidden from the unknown-command hint but still callable. Makefile is M2-scoped (build/test/lint/fmt/tidy/path/clean); dev/fe-*/release/install land in M3/M6 per 06-makefile.md — `make build` uses plain `go build -tags "webkit2_41 production"` until the Vite frontend exists. UUIDv7 caveat: tasks created within the same second share their first ~8 hex chars, so short prefixes can be ambiguous right after a burst of adds (by design; ambiguity error lists candidates).
