# Milestones & Risks

## Build order (each milestone ends with a working, testable increment)

### M0 — Tray spike (½ day) ⚠ do first
Tiny Wails v2 app + `getlantern/systray.Run` in a goroutine on **this machine** (`-tags webkit2_41`).
Verify: tray icon appears (AppIndicator), menu works, show/hide window from menu, clean quit.
If it deadlocks or double-inits GTK → switch to `leaanthony/systray`; if that also fails → plan B is a
minimal D-Bus AppIndicator client via `godbus` (more work, still feasible). **Nothing else starts until M0 passes.**

### M1 — Core + store (≈1 day)
- `internal/store`: Open/WAL/migrations/repo; `internal/core`: Task/Status/Service with invariants.
- Unit tests: transitions, prefix matching, validation, migration idempotency (temp-dir DBs).

### M2 — CLI (≈1 day)
- All commands per 04-cli-spec.md, `--json`, exit codes, non-TTY rm guard.
- Golden tests: temp DB via `MHTODO_DB_PATH`, assert stdout + exit code for each command incl. error paths.
- Deliverable: fully usable agentic interface with zero GUI.

### M3 — GUI MVP (≈1–2 days)
- `wails init` frontend (Vite+Svelte+Tailwind v4), app.go bindings per parity contract, single-instance lock.
- List view + New Task dialog + Detail drawer → **full CLI parity in the UI**. Board view can land here or M5.

### M4 — Tray + notifications + live sync (≈1 day)
- Wire `internal/tray` (validated pattern from M0), hide-to-tray close, tooltip counts.
- `internal/notify` triggers; `internal/sync` watcher → `tasks:changed`.
- Manual test matrix: CLI edit while GUI open/hidden; done/waiting notifications; single-instance relaunch.

### M5 — Polish (≈1 day)
- Board/kanban view + column quick-add, keyboard shortcuts, styling pass, empty states, error toasts.
- Stretch: drag-and-drop between columns, light theme toggle.

### M6 — Packaging & docs (½–1 day)
- Finalize Makefile (arm64 guard), `.desktop` file, icons (app + tray template), README with the
  **agent contract** section (CLI reference + JSON shapes), `make install` smoke test on this machine.

## Risk register

| # | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| 1 | systray + Wails dual GTK main loop misbehaves on Linux (deadlock, icon never appears) | Medium | High — tray is a hard requirement | M0 spike before any other work; fallbacks ranked: leaanthony/systray → godbus AppIndicator client. |
| 2 | webkit2gtk version mismatch breaks builds on other machines | Certain (this box needs the tag) | Low | `TAGS` variable in Makefile, documented override; CI matrix later if desired. |
| 3 | Concurrent CLI+GUI writes corrupt/lock DB | Low | Medium | WAL + busy_timeout=5000 from day one; single-statement transactions only. |
| 4 | Wails v2 long-term maintenance (v3 is the active line) | Medium | Low-medium | Pin v2.x deliberately; core/store/cli are framework-agnostic so a v3 port touches app.go + build config only. |
| 5 | fsnotify misses WAL-only writes on some filesystems | Low | Low | 2s `max(updated_at)` poll fallback is always running as the safety net. |
| 6 | Scope creep (DnD, markdown rendering, themes) | Medium | Low | Explicitly stretch goals in M5; v1 ships without them. |

## Definition of done (v0.1)

- `make install` on this machine → app appears in the launcher, tray icon works, close hides to tray.
- Every CLI command has a GUI equivalent and vice versa (parity table in 02-architecture.md holds).
- `mhtodo list --json` output is stable-documented; an agent can drive the full lifecycle using only the CLI.
- `make test && make lint` green; DB survives concurrent CLI+GUI use.
