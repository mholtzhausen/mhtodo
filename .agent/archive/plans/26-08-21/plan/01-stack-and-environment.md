# Stack & Environment

## Chosen stack

| Layer | Choice | Why |
|---|---|---|
| Language | Go 1.25 (installed: 1.25.6) | requirement |
| GUI framework | **Wails v2** (`v2.11.x` line; CLI installed: v2.11.0) | native webview, typed bindings, dev hot-reload, packaging tooling. v3 exists but is newer with thinner docs/patterns for tray; pin v2 and note migration path later. |
| Webview backend (Linux) | WebKitGTK **4.1** via build tag `webkit2_41` | this distro only ships 4.1 (`pkg-config` shows `webkit2gtk-4.1.pc`, no 4.0). Wails v2 supports the `-tags webkit2_41` variant; bake it into every Makefile target that compiles Go. |
| Frontend | **Vite + Svelte 5 + Tailwind CSS v4** | fast DX, tiny bundle, easy "modern beautiful" dark UI. No framework lock-in beyond the frontend dir. |
| CLI parsing | `spf13/cobra` | standard; subcommand dispatch coexists cleanly with Wails (see architecture). |
| SQLite driver | **`modernc.org/sqlite`** via `database/sql` | pure Go → no cgo constraint on the DB layer, trivially testable in temp dirs. Perf is irrelevant at todo scale. (Alternative: mattn/go-sqlite3 + CGO — rejected for build friction.) |
| IDs | UUIDv7 (`google/uuid`) | time-ordered, sortable, 128-bit; supports prefix lookup. |
| System tray | `getlantern/systray` (fallback: `leaanthony/systray`) | cross-platform tray + menus; on Linux uses AppIndicator/GTK which is already linked by Wails. **Spike first — see risks.** |
| Notifications | small internal wrapper → `notify-send` via `os/exec` (freedesktop notifications) | `notify-send` confirmed present. No extra dep needed; beeep only if we later want Windows/macOS parity. |
| File watching | `fsnotify/fsnotify` | live sync of CLI→GUI changes. |
| Testing | stdlib `testing`, temp-dir DBs, golden JSON for CLI | keep it boring. |

## Machine facts (verified 2025-08)

- Go **1.25.6**, Wails CLI **v2.11.0** at `/home/nemesarial/go/bin/wails`, Node **v24.17.0**.
- `pkg-config`: `webkit2gtk-4.1` ✅, `webkit2gtk-4.0` ❌ → **all Go builds need `-tags webkit2_41`**.
  (Keep the tag in a single Makefile variable so other distros can drop it.)
- `notify-send` present at `/usr/bin/notify-send`.
- Repo is empty except `.git`; plan lives in `.agent/plan/`.

## Dependency list (go.mod, expected)

```
github.com/wailsapp/wails/v2        v2.11.x
github.com/spf13/cobra              latest
modernc.org/sqlite                  latest
github.com/google/uuid              latest (v7 support)
github.com/getlantern/systray       latest   # or leaanthony fork after M0 spike
github.com/fsnotify/fsnotify        latest
```

Frontend (`frontend/package.json`): `svelte`, `tailwindcss@4`, `vite`, `@sveltejs/vite-plugin-svelte`.
