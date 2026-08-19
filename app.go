package main

import (
	"context"
	"embed"
	"log"

	"github.com/getlantern/systray"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"mhtodo/internal/core"
	"mhtodo/internal/store"
)

//go:embed all:frontend/dist
var assets embed.FS

// App is the Wails application. It exposes exactly the core.Service surface —
// the parity contract in .agent/plan/02-architecture.md. The frontend never
// touches SQL or business rules; new capability = core method + CLI command +
// bound method here.
type App struct {
	ctx     context.Context // Wails lifecycle context; all runtime calls go through it
	svc     *core.Service
	repo    *store.TaskRepo
	visible bool // window visibility, tracked via window:shown/hidden events (tray toggle)
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
	wruntime.EventsOn(ctx, "window:shown", func(_ ...interface{}) { a.visible = true })
	wruntime.EventsOn(ctx, "window:hidden", func(_ ...interface{}) { a.visible = false })
	log.Printf("mhtodo started (db %s)", store.DBPath())
}

func (a *App) shutdown(_ context.Context) {
	if a.repo != nil {
		a.repo.Close()
	}
	releaseInstanceLock()
}

// --- bound methods (parity contract) -----------------------------------------

// ListTasks maps to CLI `list`. Zero filter values give the CLI defaults:
// hide done, sort updated_at desc.
func (a *App) ListTasks(filter core.ListFilter) ([]core.Task, error) {
	return a.svc.List(a.ctx, filter)
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

// SetStatus maps to CLI `status` / `done`. M4 adds notify-send on done/waiting.
func (a *App) SetStatus(id string, status core.Status) (core.Task, error) {
	t, err := a.svc.SetStatus(a.ctx, id, status)
	if err == nil {
		a.emitChanged(t.ID, "status")
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
// mutation emits tasks:changed; M4's external watcher emits the same event on
// foreign writes, so the UI has exactly one refetch handler.
func (a *App) emitChanged(id, op string) {
	if a.ctx == nil {
		return
	}
	wruntime.EventsEmit(a.ctx, "tasks:changed", map[string]string{"id": id, "op": op})
}

// --- tray callbacks (M0-validated wiring; moves to internal/tray in M4) ------

func (a *App) trayShow() {
	if a.ctx != nil {
		wruntime.WindowShow(a.ctx)
	}
}
func (a *App) trayHide() {
	if a.ctx != nil {
		wruntime.WindowHide(a.ctx)
	}
}
func (a *App) trayQuit() {
	systray.Quit() // queues indicator removal + gtk_main_quit on the shared loop
	if a.ctx != nil {
		wruntime.Quit(a.ctx)
	}
}

// Bound to JS for manual testing from the window itself.
func (a *App) ShowWindow() error { a.trayShow(); return nil }
func (a *App) HideWindow() error { a.trayHide(); return nil }
