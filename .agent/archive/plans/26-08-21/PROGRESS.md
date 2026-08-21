# mhtodo — progress (updated 2026-08-20)

## v0.2 — Archive (in progress, pending manual GUI check)

Done tasks now have a destination: the **archive**. Bulk-only by design — one click on the Done
column's archive button sweeps everything in it; restore individually from List → Archived.

- [x] Schema v2 migration (`archived_at` + index), idempotent, v1→v2 upgrade path tested against a real pre-v0.2 DB file
- [x] Core: `ArchiveDone()` (single-statement UPDATE…RETURNING — atomic under WAL) + `Unarchive(id)` → pending, progress 0, completed_at cleared; `ErrNotArchived` typed error
- [x] Archived tasks hidden from every default view (CLI list incl. --all, board); explicit filter only (`list --archived`, GUI chip)
- [x] CLI: `mhtodo archive`, `mhtodo unarchive ID`, `list --archived`; exit codes + JSON envelope per contract (`not_archived` → 1); golden tests green
- [x] GUI: Done-column archive button (count toast, no confirm — reversible), Archived filter chip + key `5`, detail-pane Unarchive button + archived timestamp; bindings regenerated
- [x] Parity table / plan docs / README agent contract updated; `go test -race` all green; `make build` clean
- [ ] Manual GUI verification (board sweep, archived view, unarchive round-trip)

## v0.1 — complete ✅

Status: **v0.1 complete** ✅ — all six milestones done (M0–M6). M6 shipped packaging & docs: finalized Makefile (release/install/uninstall + arm64 guard), .desktop launcher entry, app+tray icons, README with the agent contract section; `make install` smoke-tested live on this machine (tray E2E over D-Bus, close-hides-to-tray, clean quit). Everything below is copy-pasteable to Slack as-is.

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
  - [x] keyboard: `/` search · `n` new task · `esc` close · `1–4` toggle status filter · `del` delete selected (confirm dialog)
  - [x] single-instance lock `$XDG_RUNTIME_DIR/mhtodo.lock` (pid, stale-lock steal) + focus-on-relaunch via SIGUSR2; second launch exits 0 with a log line — all unit-tested in main_test.go and verified live
  - [x] window 1100×720 / min 800×560 per spec; tray wiring kept from M0 (moves to internal/tray in M4)
  - [x] `make build` = wails build (frontend + bindings + ldflags version stamp) → bin/mhtodo; `make dev` hot-reload; desktop smoke test: window renders, selftest show→hide→quit exit 0
- [x] **M4** Tray + notifications + live sync — hide-to-tray close, open-task count in tray, notify-send on →done/→waiting, fsnotify watcher (+2s poll fallback) → tasks:changed
  - [x] internal/tray: M0-validated Register pattern (before wails.Run), menu Show/Hide · New Task · Quit; handlers run off the GTK loop via channels
  - [x] hide-to-tray close: OnBeforeClose hides instead of quitting; real exit only via tray Quit / Ctrl+Q / SIGINT/SIGTERM (signal handler sets quitting first — Wails routes its own signal path through OnBeforeClose, which would otherwise swallow Ctrl+C forever)
  - [x] tray count: "mhtodo — N open tasks" tooltip + compact XAyatanaLabel "mhtodo (N)" refreshed on every change (local or external); verified live over D-Bus (add → "mhtodo (1)", done → "mhtodo")
  - [x] internal/notify: notify-send `-a mhtodo -i dialog-task` wrapper, 60s dedupe per id+status, failures logged never fatal; GUI fires on real →done/→waiting transitions only; CLI opt-in `mhtodo done ID --notify` (same transition semantics)
  - [x] internal/sync: fsnotify on .db + -wal/-shm sidecars AND always-on 2s max(updated_at) poll (risk #5 safety net), 300ms debounce → tasks:changed; unit tests incl. fallback-with-fsnotify-dead path, `go test -race` green
  - [x] tray "New Task" shows window + emits tray:new-task → frontend opens create dialog; Ctrl+Q bound to Quit()
  - [x] E2E verified headless via dbusmenu Event over D-Bus: Show/Hide toggles both ways (window map state), New Task re-shows, Quit exits clean; single-instance relaunch still exit-0 + focus
- [x] **M5** Polish — board/kanban view + column quick-add, keyboard shortcuts, styling pass, empty states, error toasts; stretch drag-and-drop done too (light theme deferred)
  - [x] Board view (default): four kanban columns pending|wip|waiting|done with live counts; cards show title / progress bar / relative updated time; per-column + button opens the new-task dialog preset to that column's status; board always shows all statuses incl. done, list keeps its own filters
  - [x] Board/List toggle in header (+ `b`/`l` keys), choice persisted across launches (localStorage); footer shortcut legend updated
  - [x] drag-and-drop (stretch): native HTML5 DnD, zero deps — drop a card on another column → api.setStatus (same bound method as CLI `status`; Go notifies + emits tasks:changed, frontend refetches through its single refresh path); own-column drop is a no-op; dragged-card opacity/rotate + indigo drop-target highlight (≤150ms); post-drag click suppressed so drags never open the drawer. Verified live with synthesized xdotool drags (pending→wip and wip→empty column) confirmed in DB
  - [x] error toasts: rose-tinted alert on every failure path (load, drawer edits, dialog submit, board drop), single showToast auto-dismiss path
  - [x] empty states: board "no tasks yet" / search-no-match + per-column dashed placeholder; list distinguishes "no tasks yet" from "filters match nothing"
  - [x] styling pass: dark zinc/indigo palette throughout, card hover lift, drawer/dialog fly-in (150ms), thin scrollbars; `npm run build` + `make build` clean, `go test -race ./...` green
- [x] **M6** Packaging & docs — Makefile finalize (arm64 guard), .desktop file, app+tray icons, README with agent contract section, make install smoke test
  - [x] Makefile finalized: `release` cross-builds linux tarballs → dist/ (staged under mhtodo_0.1.0/); **arm64 guard** — without aarch64-linux-gnu-gcc it warns and ships amd64 only (verified on this machine, which lacks the toolchain); `install`/`uninstall` into $PREFIX (~/.local): binary + .desktop + hicolor icon + update-desktop-database; version stamping refactored to one STAMP var shared by build/release
  - [x] packaging/mhtodo.desktop: Terminal=false, Exec=mhtodo, Icon=mhtodo — passes desktop-file-validate; installed entry appears in the launcher menu
  - [x] icons: assets/icon.png (512×512 app icon, indigo rounded square + white check matching the GUI palette) → .desktop/hicolor/tarball AND wired as the Wails window icon via options.Linux.Icon (embedded embed.FS, same pattern as tray); assets/tray.png replaced with a 48px template-style badge — verified byte-identical in the live SNI item
  - [x] README.md: install (source + tarball), full CLI reference from the real binary (commands/flags/exit codes/error envelope/canonical JSON object/examples), **agent contract stability** section, GUI/tray/notification/live-sync behavior, data & concurrency notes, parity table, dev notes (TAGS + mode-tag gotcha)
  - [x] make install smoke test on this machine: installed files verified; live E2E of the installed binary — SNI tray item registered with new icon, Show/Hide toggles both ways over dbusmenu D-Bus clicks, **WM close request hides to tray** (window unmaps, process + indicator survive), Quit exits clean and releases the instance lock; SIGTERM quit path re-verified
  - [x] make test && make lint green after all changes; gofmt clean

## Definition of done (v0.1) — all met ✅
- [x] make install → launcher entry, tray icon works, close hides to tray (M6 live E2E: SNI registered with new icon, dbusmenu Show/Hide both ways, WM-close unmaps window while process + indicator survive, Quit clean)
- [x] Full CLI ↔ GUI parity (table in 02-architecture.md holds both ways; restated in README)
- [x] mhtodo list --json stable & documented; an agent can drive the full lifecycle via CLI alone (README "CLI reference (agent contract)" incl. exit codes, error envelope, canonical object, examples)
- [x] make test && make lint green; DB survives concurrent CLI+GUI use

## Notes / blockers
- **M6 findings (2026-08-19):**
  - **This machine's GTK does not export `_NET_WM_ICON` for pixbuf-based window icons** — even a direct `gdk_window_set_icon_list()` call on a realized+shown window writes no X property at all (verified with xprop -spy + minimal C tests; zenity/Zed get theirs via other means). Wails' `options.Linux.Icon` is wired correctly in main.go and works on standard distro builds; here the taskbar falls back to .desktop lookup, which resolves our installed hicolor icon via mhtodo.desktop (Icon=mhtodo). Also: mhtodo's window did not appear in this machine's Cinnamon panel window list regardless of workspace — user panel-config quirk, not an app defect (window has correct WM properties: NORMAL type, proper class/name).
  - **gdbus call gotchas for D-Bus E2E:** `v`-typed params need a wrapped annotated variant (`'<@i 0>'` works; bare `{}`, `@a{sv}{}`, and `0` all fail to parse); args starting with `-` (e.g. dbusmenu GetLayout's recursionDepth -1) must follow `--`. Cinnamon's StatusNotifierWatcher exposes the property as **RegisteredStatusNotifierItems** (not the spec name RegisteredItems).
  - arm64 cross-build guard verified: without gcc-aarch64-linux-gnu, `make release` warns and ships amd64-only; tarball layout mhtodo_0.1.0/{mhtodo,mhtodo.desktop,icon.png,README.md}, extracted binary smoke-tested (version stamp + CLI add --json).
- **M5 findings (2026-08-19):**
  - Svelte 5's template parser rejects mixing `??` and `||` in one expression — parenthesize.
  - HTML5 drag-and-drop works as-is in WebKitGTK, including drags synthesized with xdotool (mousemove/mousedown/move/up) — no special handling needed for headless verification; drops land through the normal api.setStatus path so DB state is ground truth.
- **M4 findings (2026-08-19):**
  - **Wails v2 routes SIGINT/SIGTERM through OnBeforeClose** (`Frontend.Quit` calls it before `mainWindow.Quit`). A hide-to-tray handler that returns true therefore swallows Ctrl+C/kill — the process hangs forever in "Shutting down...". Fix: our own signal handler sets `quitting` first, so whichever path reaches OnBeforeClose last allows the exit. Verified: SIGTERM now exits clean and releases the instance lock.
  - **getlantern/systray's Linux AppIndicator backend is a no-op for SetTooltip** (libappindicator has no tooltip API; the C shim just frees the string). The count therefore also goes to XAyatanaLabel via `tray.SetLabel` ("mhtodo (N)", visible in Cinnamon when tray labels are enabled); the spec tooltip call stays for Windows/macOS. Verified live by reading SNI properties over D-Bus.
  - **Wails v2 emits no window:shown/window:hidden events** (M3's EventsOn tracking was dead code — `visible` stayed false, so the tray toggle could never hide). Visibility is now self-tracked with atomic.Bool in showWindow/hideWindow/beforeClose; all app paths go through those three methods.
  - **Watcher watches .db + -wal/-shm** (superset of the plan's "fsnotify on the DB file" — WAL commits touch sidecars first). Both sources can fire for one logical change (fsnotify immediate, poll at next tick); harmless by design since the frontend handler is an idempotent refetch. Poll baseline is established on its first tick → no spurious event at startup; a write in that first 2s window is only seen if fsnotify also missed it (negligible double-failure case).
  - **Tray E2E testing without a human:** SNI menu items are clickable over D-Bus — `com.canonical.dbusmenu.GetLayout` + `Event(id, "clicked", ...)` on the item's Menu objectpath. That's how Show/Hide/New Task/Quit were verified headless (window map state via xwininfo).
  - **This machine's bash is broken for function definitions** (`f() { : }` fails to parse in files and `bash -c`, even byte-perfect; dash too; zsh fine) — environmental, like the Go embed anomaly. Use zsh or flat scripts for shell test harnesses.
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


## Design refresh — "Slate" (2026-08-20)
Post-v0.1 GUI polish. Explored three directions as single-page mockups in `tmp/design/` (01-slate, 02-paper, 03-ember; `build_mockups.py` regenerates them). **Slate chosen and retrofitted to the app.**
- Palette: near-black zinc-950 → layered mid-dark slate (canvas `#252b37`, chrome `#1f242f`, column `#2c3340`, card `#343c4d`, elevated `#3d465a`) so surface steps, borders and shadows read; brighter periwinkle accent `#7b8cff`; status legend pending `#a3adbf` / wip `#7b8cff` / waiting `#e8ab4a` / done `#4cc48e`
- Radii: 12px/8px → 6px cards, 4px controls (was "too round and generic")
- Board: cards now carry a 2px status-colored left edge (dot + bar + edge = one legend); count badges, sharper column headers
- Forms: native status `<select>` → `StatusPicker.svelte` chips (drawer + new-task dialog); native progress number input → `ProgressControl.svelte` (styled slider + −/+ stepper, commits on release = one API call, same semantics)
- Restyled: App shell (underline Board/List tabs, search icon, task count), Board, List, FilterBar (styled sort select + chevron), TaskDetail, NewTaskDialog
- Tokenized in `frontend/src/styles.css` (`@theme` → `bg-canvas`, `text-ink`, `border-line-soft`, `bg-st-wip/20` …); new components: `StatusPicker.svelte`, `ProgressControl.svelte`
- Font fix: brand name and other ≥600 weights rendered blurry in Wails but crisp in-browser — machine only has Roboto 400/500 installed, so weight 600 was faux-bold synthesis (WebKitGTK's FreeType synthetic embolden is fuzzy at small sizes; Chromium's stroke expansion isn't). Fixed by self-hosting Inter: `frontend/src/assets/fonts/InterVariable.ttf` (variable, wght 100–900, SIL OFL — see OFL.txt) + `@font-face` in styles.css. Every CSS weight is now a native instance in both engines; font ships inside the binary via go:embed so no target machine needs any system fonts.
- Verified: `npm run build` clean, `make test` green, live E2E on desktop (board / detail drawer / new-task modal all render per mockup)
- Alternatives kept in `tmp/` (gitignored) if we ever want to switch: Paper = light theme, Ember = warm amber/sharp
