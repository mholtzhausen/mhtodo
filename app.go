package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"sync/atomic"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"mhtodo/internal/core"
	"mhtodo/internal/notify"
	"mhtodo/internal/store"
	mhsync "mhtodo/internal/sync"
	"mhtodo/internal/tray"
)

//go:embed all:frontend/dist
var assets embed.FS

// App is the Wails application. It exposes exactly the core.Service surface —
// the parity contract in .agent/plan/02-architecture.md. The frontend never
// touches SQL or business rules; new capability = core method + CLI command +
// bound method here.
type App struct {
	ctx      context.Context // Wails lifecycle context; all runtime calls go through it
	svc      *core.Service
	repo     *store.TaskRepo
	visible  atomic.Bool   // window visibility, self-tracked (Wails v2 emits no show/hide events)
	quitting atomic.Bool   // set by Quit() so OnBeforeClose allows a real exit
	notifier *notify.Notifier
	watcher  *mhsync.Watcher // external (CLI-side) DB changes → tasks:changed
}

var app = &App{}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	repo, err := store.Open(store.DBPath())
	if err != nil {
		log.Fatalf("open db: %v", err) // DB failure is fatal for the GUI too
	}
	a.repo = repo
	a.svc = core.NewService(repo)
	a.notifier = notify.New()
	// Window visibility is self-tracked in showWindow/hideWindow/beforeClose:
	// Wails v2 emits no window:shown/hidden events (verified against v2.11 source),
	// and the runtime only exposes WindowIsMinimised — not hidden state.
	a.visible.Store(true) // first launch shows the window

	// Live sync (05-gui-spec.md): external writes (CLI) → same tasks:changed
	// event as local mutations. A watcher failure degrades to local-only sync;
	// the app itself is unaffected.
	if w, err := mhsync.Watch(store.DBPath(), func() {
		log.Println("external db change detected → tasks:changed")
		a.emitChanged("", "external")
	}); err != nil {
		log.Printf("start db watcher: %v (live sync degraded to local-only)", err)
	} else {
		a.watcher = w
	}
	a.refreshTooltip()
	log.Printf("mhtodo started (db %s)", store.DBPath())
}

func (a *App) shutdown(_ context.Context) {
	if a.watcher != nil {
		a.watcher.Close()
	}
	if a.repo != nil {
		a.repo.Close()
	}
	releaseInstanceLock()
}

// --- bound methods (parity contract) -----------------------------------------

// ListTasks maps to CLI `list`. Zero filter values give the CLI defaults:
// hide done, sort updated_at desc. A nil result is normalized to an empty
// slice: Wails marshals a nil Go slice as JSON null and the frontend expects
// an array (same normalization the CLI applies in printTasks).
func (a *App) ListTasks(filter core.ListFilter) ([]core.Task, error) {
	tasks, err := a.svc.List(a.ctx, filter)
	if err != nil {
		return nil, err
	}
	if tasks == nil {
		tasks = []core.Task{}
	}
	return tasks, nil
}

// GetTask maps to CLI `show`; accepts a full ID or unique prefix (same helper
// as the CLI).
func (a *App) GetTask(ref string) (core.Task, error) { return a.svc.Get(a.ctx, ref) }

// CreateTask maps to CLI `add`.
func (a *App) CreateTask(in core.CreateInput) (core.Task, error) {
	t, err := a.svc.Create(a.ctx, in)
	if err == nil {
		a.emitChanged(t.ID, "create")
	}
	return t, err
}

// UpdateTask maps to CLI `edit` (title/description/progress; status goes via
// SetStatus). Nil patch fields are left unchanged.
func (a *App) UpdateTask(id string, patch core.UpdateInput) (core.Task, error) {
	t, err := a.svc.Edit(a.ctx, id, patch)
	if err == nil {
		a.emitChanged(t.ID, "update")
	}
	return t, err
}

// SetStatus maps to CLI `status` / `done`. Notifies on real →done/→waiting
// transitions (05-gui-spec.md); no-op re-sets and plain edits stay silent.
func (a *App) SetStatus(id string, status core.Status) (core.Task, error) {
	prev, perr := a.svc.Get(a.ctx, id) // old status for transition detection
	t, err := a.svc.SetStatus(a.ctx, id, status)
	if err == nil {
		a.emitChanged(t.ID, "status")
		if perr == nil && prev.Status != t.Status {
			switch t.Status {
			case core.StatusDone:
				a.notifier.TaskDone(t.ID, t.Title)
			case core.StatusWaiting:
				a.notifier.TaskWaiting(t.ID, t.Title)
			}
		}
	}
	return t, err
}

// DeleteTask maps to CLI `rm`; returns the deleted task.
func (a *App) DeleteTask(id string) (core.Task, error) {
	t, err := a.svc.Delete(a.ctx, id)
	if err == nil {
		a.emitChanged(t.ID, "delete")
	}
	return t, err
}

// DBPath maps to CLI `path`; shown in the GUI footer.
func (a *App) DBPath() string { return store.DBPath() }

// emitChanged is the single refresh path for the frontend: every local
// mutation emits tasks:changed; the external watcher (internal/sync) emits the
// same event on foreign writes, so the UI has exactly one refetch handler. It
// also refreshes the tray tooltip count — both paths go through here.
func (a *App) emitChanged(id, op string) {
	if a.ctx == nil {
		return
	}
	wruntime.EventsEmit(a.ctx, "tasks:changed", map[string]string{"id": id, "op": op})
	a.refreshTooltip()
}

// refreshTooltip updates the tray count (open = not done), refreshed on every
// change per 05-gui-spec.md. Two channels: the spec tooltip text (effective on
// Windows/macOS; getlantern/systray's Linux AppIndicator backend ignores it)
// and a compact label — XAyatanaLabel on this machine, visible in Cinnamon
// when tray labels are enabled. Transient DB errors are ignored; the next
// change retries.
func (a *App) refreshTooltip() {
	n, err := a.svc.CountOpen(a.ctx)
	if err != nil {
		return
	}
	tray.SetTooltip(fmt.Sprintf("mhtodo — %d open tasks", n))
	label := "mhtodo"
	if n > 0 {
		label = fmt.Sprintf("mhtodo (%d)", n)
	}
	tray.SetLabel(label)
}

// --- window lifecycle ---------------------------------------------------------

func (a *App) showWindow() {
	if a.ctx != nil {
		wruntime.WindowShow(a.ctx)
		a.visible.Store(true)
	}
}
func (a *App) hideWindow() {
	if a.ctx != nil {
		wruntime.WindowHide(a.ctx)
		a.visible.Store(false)
	}
}

// Quit is the real exit path: tray "Quit" menu item or Ctrl+Q in the window.
// Sets quitting first so OnBeforeClose allows the close instead of hiding to
// tray, then removes the indicator and quits Wails on the shared GTK loop.
func (a *App) Quit() {
	a.quitting.Store(true)
	tray.Quit() // queues indicator removal + gtk_main_quit on the shared loop
	if a.ctx != nil {
		wruntime.Quit(a.ctx)
	}
}

// beforeClose implements hide-to-tray. NOTE: Wails routes EVERY quit path
// through here — window X button, wruntime.Quit(), and even SIGINT/SIGTERM
// (Frontend.Quit) — so returning true would swallow signals too. The signal
// handler in main.go sets quitting first; tray Quit / Ctrl+Q do the same.
func (a *App) beforeClose(_ context.Context) bool {
	if a.quitting.Load() {
		return false // allow real quit
	}
	wruntime.WindowHide(a.ctx)
	a.visible.Store(false)
	return true
}

// openNewTaskFromTray is the tray "New Task" action: show the window and ask
// the frontend to open the create dialog.
func (a *App) openNewTaskFromTray() {
	a.showWindow()
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "tray:new-task")
	}
}

// Bound to JS for manual testing from the window itself.
func (a *App) ShowWindow() error { a.showWindow(); return nil }
func (a *App) HideWindow() error { a.hideWindow(); return nil }
