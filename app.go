package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"mhtodo/internal/core"
	"mhtodo/internal/globalhk"
	"mhtodo/internal/integrations"
	"mhtodo/internal/notify"
	"mhtodo/internal/platform"
	"mhtodo/internal/settings"
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
	ctx         context.Context // Wails lifecycle context; all runtime calls go through it
	svc         *core.Service
	repo        *store.TaskRepo
	visible     atomic.Bool // window visibility, self-tracked (Wails v2 emits no show/hide events)
	alwaysOnTop atomic.Bool // persisted in meta; applied via WindowSetAlwaysOnTop
	quitting    atomic.Bool // set by Quit() so OnBeforeClose allows a real exit
	notifier    *notify.Notifier
	watcher     *mhsync.Watcher // external (CLI-side) DB changes → tasks:changed
	hotkey      *globalhk.Handle

	posMu sync.Mutex
	posX  int
	posY  int
	posOK bool // true once we have a captured or loaded position

	posStop chan struct{} // stops periodic position capture
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
	a.visible.Store(!launchStartHidden)

	// Restore always-on-top from meta (default off). Applied once the window
	// exists; StartHidden builds still get the flag for the first Show.
	if v, ok, err := a.repo.GetMeta(ctx, store.MetaAlwaysOnTop); err != nil {
		log.Printf("read always_on_top: %v", err)
	} else if ok && v == "true" {
		a.alwaysOnTop.Store(true)
	}
	a.applyAlwaysOnTop()

	if v, ok, err := a.repo.GetMeta(ctx, store.MetaWindowPos); err != nil {
		log.Printf("read window_pos: %v", err)
	} else if ok {
		if x, y, perr := parseWindowPos(v); perr != nil {
			log.Printf("parse window_pos %q: %v", v, perr)
		} else {
			a.posMu.Lock()
			a.posX, a.posY, a.posOK = x, y, true
			a.posMu.Unlock()
		}
	}

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
	a.registerGlobalHotkey()
	log.Printf("mhtodo started (db %s)", store.DBPath())
}

// domReady re-applies window prefs after the window is fully up (Wails docs:
// runtime window calls are not guaranteed during OnStartup).
func (a *App) domReady(_ context.Context) {
	a.applyAlwaysOnTop()
	if a.visible.Load() {
		a.restoreWindowPos()
	}
	a.startPosCapture()
}

func (a *App) shutdown(_ context.Context) {
	a.stopPosCapture()
	if a.visible.Load() {
		a.captureWindowPos()
	}
	if a.hotkey != nil {
		a.hotkey.Close()
		a.hotkey = nil
	}
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
// hide done, board order. A nil result is normalized to an empty
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

// UpdateTask maps to CLI `edit` (title/description/feedback/progress; status goes via
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
				a.maybeCloseHerdrTabOnDone(t)
			case core.StatusWaiting:
				a.notifier.TaskWaiting(t.ID, t.Title)
			}
		}
	}
	return t, err
}

// ReorderBoardTask moves a root task within its current status column.
// beforeID empty appends to the column end.
func (a *App) ReorderBoardTask(id string, beforeID string) (core.Task, error) {
	var beforeRef *string
	if strings.TrimSpace(beforeID) != "" {
		b := strings.TrimSpace(beforeID)
		beforeRef = &b
	}
	t, err := a.svc.ReorderBoardTask(a.ctx, id, beforeRef)
	if err == nil {
		a.emitChanged(t.ID, "reorder")
	}
	return t, err
}

// Archive maps to CLI `archive ID`: archives a single done task.
func (a *App) Archive(id string) (core.Task, error) {
	t, err := a.svc.Archive(a.ctx, id)
	if err == nil {
		a.emitChanged(t.ID, "archive")
	}
	return t, err
}

// ArchiveDone maps to CLI `archive`: archives every done task at once (the
// board's Done-column button). Emits tasks:changed only when something moved,
// so an empty click is a true no-op for the UI.
func (a *App) ArchiveDone() ([]core.Task, error) {
	s, err := settings.Load(a.repo)
	if err != nil {
		return nil, err
	}
	tasks, err := a.svc.ArchiveDone(a.ctx, s.ArchiveDoneSubtasks)
	if err == nil && len(tasks) > 0 {
		a.emitChanged("", "archive")
	}
	if tasks == nil {
		tasks = []core.Task{} // Wails marshals nil slices as null; the frontend expects an array
	}
	return tasks, err
}

// Unarchive maps to CLI `unarchive`: restores an archived task to pending.
func (a *App) Unarchive(id string) (core.Task, error) {
	t, err := a.svc.Unarchive(a.ctx, id)
	if err == nil {
		a.emitChanged(t.ID, "unarchive")
	}
	return t, err
}

// DeleteTask maps to CLI `rm`; returns the deleted task. Children cascade via FK.
func (a *App) DeleteTask(id string) (core.Task, error) {
	t, err := a.svc.Delete(a.ctx, id)
	if err == nil {
		a.emitChanged(t.ID, "delete")
	}
	return t, err
}

// CountChildren maps to the cascade-confirm count before deleting a parent.
func (a *App) CountChildren(id string) (int, error) {
	return a.svc.CountChildren(a.ctx, id)
}

// AddActivity maps to CLI `activity add`.
func (a *App) AddActivity(taskID string, in core.ActivityInput) (core.Activity, error) {
	act, err := a.svc.AddActivity(a.ctx, taskID, in)
	if err == nil {
		a.emitChanged(act.TaskID, "activity")
	}
	return act, err
}

// ListActivity maps to CLI `activity list`.
func (a *App) ListActivity(filter core.ActivityFilter) ([]core.Activity, error) {
	list, err := a.svc.ListActivity(a.ctx, filter)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []core.Activity{}
	}
	return list, nil
}

// DeleteActivity maps to CLI `activity rm`.
func (a *App) DeleteActivity(id string) (core.Activity, error) {
	act, err := a.svc.DeleteActivity(a.ctx, id)
	if err == nil {
		a.emitChanged(act.TaskID, "activity")
	}
	return act, err
}

// DBPath maps to CLI `path`; shown in the GUI footer.
func (a *App) DBPath() string { return store.DBPath() }

// SlackReport maps to CLI `slack report`: a paste-ready board summary.
func (a *App) SlackReport() (string, error) {
	return a.svc.SlackReport(a.ctx)
}

// TaskMarkdownReport maps to CLI `show --markdown`: paste-ready task summary.
func (a *App) TaskMarkdownReport(ref string) (string, error) {
	return a.svc.TaskMarkdownReport(a.ctx, ref)
}

// PickDirectory opens the system folder picker and returns the chosen path, or
// "" when the user cancels. start, when a valid directory, is the initial location.
// If start is missing or invalid, or the dialog fails with that default, the picker
// opens again without a starting directory instead of surfacing an error.
func (a *App) PickDirectory(start string) (string, error) {
	opts := wruntime.OpenDialogOptions{
		Title: "Select working directory",
	}
	start = strings.TrimSpace(start)
	defaultDir := ""
	if start != "" {
		if info, err := os.Stat(start); err == nil && info.IsDir() {
			defaultDir = start
			opts.DefaultDirectory = start
		}
	}
	path, err := wruntime.OpenDirectoryDialog(a.ctx, opts)
	if err != nil && defaultDir != "" {
		opts.DefaultDirectory = ""
		return wruntime.OpenDirectoryDialog(a.ctx, opts)
	}
	return path, err
}

// GetGUISettings returns persisted GUI preferences (defaults when unset).
func (a *App) GetGUISettings() (settings.GUISettings, error) {
	return settings.Load(a.repo)
}

// SetGUISettings persists GUI preferences.
func (a *App) SetGUISettings(s settings.GUISettings) error {
	return settings.Save(s)
}

// CheckBinary reports whether path resolves to an executable (for integration UI).
func (a *App) CheckBinary(path string) bool {
	return settings.BinaryFound(path)
}

func (a *App) herdrClient() (integrations.Client, error) {
	s, err := settings.Load(a.repo)
	if err != nil {
		return integrations.Client{}, err
	}
	return integrations.Client{Herdr: s.Herdr, Claude: s.Claude}, nil
}

func (a *App) maybeCloseHerdrTabOnDone(t core.Task) {
	client, err := a.herdrClient()
	if err != nil {
		return
	}
	client.MaybeCloseTicketTabOnDone(t.ID, core.ShortID(t.ID), t.Title)
}

// EnsureHerdrReady ensures the configured Herdr workspace exists when Herdr
// integration is enabled and the binary is found (no task required).
func (a *App) EnsureHerdrReady() (integrations.HerdrTaskStatus, error) {
	client, err := a.herdrClient()
	if err != nil {
		return integrations.HerdrTaskStatus{}, err
	}
	if !client.Herdr.Enabled || !client.HerdrFound() {
		return integrations.HerdrTaskStatus{}, nil
	}
	ready, err := client.EnsureWorkspace()
	if err != nil {
		return integrations.HerdrTaskStatus{Error: err.Error()}, nil
	}
	return integrations.HerdrTaskStatus{Ready: ready}, nil
}

// EnsureHerdrWorkspaceForTask ensures the configured Herdr workspace exists when
// the task is eligible (Herdr enabled, cwd set, not human-only).
func (a *App) EnsureHerdrWorkspaceForTask(ref string) (integrations.HerdrTaskStatus, error) {
	client, err := a.herdrClient()
	if err != nil {
		return integrations.HerdrTaskStatus{}, err
	}
	if !client.Herdr.Enabled || !client.HerdrFound() {
		return integrations.HerdrTaskStatus{}, nil
	}
	t, err := a.svc.Get(a.ctx, ref)
	if err != nil {
		return integrations.HerdrTaskStatus{}, err
	}
	if !integrations.TaskEligible(t.HumanOnly, t.Cwd, client.Claude.RequireCwd) {
		return integrations.HerdrTaskStatus{}, nil
	}
	ready, err := client.EnsureWorkspace()
	if err != nil {
		return integrations.HerdrTaskStatus{Error: err.Error()}, nil
	}
	return integrations.HerdrTaskStatus{Ready: ready}, nil
}

// OpenHerdrTicket opens or focuses a Herdr tab for the task and optionally
// starts Claude with the configured ticket prompt on a new tab.
func (a *App) OpenHerdrTicket(ref string) error {
	client, err := a.herdrClient()
	if err != nil {
		return err
	}
	t, err := a.svc.Get(a.ctx, ref)
	if err != nil {
		return err
	}
	if !integrations.TaskEligible(t.HumanOnly, t.Cwd, client.Claude.RequireCwd) {
		if client.Claude.RequireCwd {
			return fmt.Errorf("task is not eligible for Herdr (needs cwd and must not be human-only)")
		}
		return fmt.Errorf("task is not eligible for Herdr (must not be human-only)")
	}
	shortID := core.ShortID(t.ID)
	return client.OpenTicketTab(t.ID, shortID, t.Title, t.Cwd)
}

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
	if a.ctx == nil {
		return
	}
	// Place before show so the WM maps at the remembered spot (avoids a jump
	// from the default/centered position). Re-apply after show for WMs that
	// ignore moves on unmapped windows.
	a.restoreWindowPos()
	// Raise + focus: unminimise, show/present, then nudge z-order. When the
	// user preference is off, a brief always-on-top pulse helps stubborn WMs
	// bring the window forward without leaving it pinned.
	wruntime.WindowUnminimise(a.ctx)
	wruntime.WindowShow(a.ctx)
	a.restoreWindowPos()
	if a.alwaysOnTop.Load() {
		wruntime.WindowSetAlwaysOnTop(a.ctx, true)
	} else {
		wruntime.WindowSetAlwaysOnTop(a.ctx, true)
		wruntime.WindowSetAlwaysOnTop(a.ctx, false)
	}
	a.visible.Store(true)
	a.startPosCapture()
	// Some WMs apply the final frame only after the first map; nudge once more.
	time.AfterFunc(150*time.Millisecond, func() {
		if a.visible.Load() {
			a.restoreWindowPos()
		}
	})
}

func (a *App) hideWindow() {
	if a.ctx == nil {
		return
	}
	a.stopPosCapture()
	a.captureWindowPos() // must run while still mapped
	wruntime.WindowHide(a.ctx)
	a.visible.Store(false)
}

// captureWindowPos reads the current position into memory and persists it.
func (a *App) captureWindowPos() {
	if a.ctx == nil {
		return
	}
	x, y := wruntime.WindowGetPosition(a.ctx)
	if !windowPosLooksValid(x, y) {
		return
	}
	a.posMu.Lock()
	a.posX, a.posY, a.posOK = x, y, true
	a.posMu.Unlock()
	a.persistWindowPos(x, y)
}

// windowPosLooksValid rejects bogus (0,0) reads on native Wayland where GTK
// cannot report global coordinates — overwriting a good saved position.
func windowPosLooksValid(x, y int) bool {
	if x != 0 || y != 0 {
		return true
	}
	if os.Getenv("MHTODO_WAYLAND") == "1" || os.Getenv("GDK_BACKEND") == "wayland" {
		if os.Getenv("XDG_SESSION_TYPE") == "wayland" {
			return false
		}
	}
	return true
}

func (a *App) startPosCapture() {
	a.stopPosCapture()
	a.posStop = make(chan struct{})
	stop := a.posStop
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if a.visible.Load() {
					a.captureWindowPos()
				}
			}
		}
	}()
}

func (a *App) stopPosCapture() {
	if a.posStop == nil {
		return
	}
	select {
	case <-a.posStop:
	default:
		close(a.posStop)
	}
	a.posStop = nil
}

func (a *App) restoreWindowPos() {
	if a.ctx == nil {
		return
	}
	a.posMu.Lock()
	ok, x, y := a.posOK, a.posX, a.posY
	a.posMu.Unlock()
	if !ok {
		return
	}
	wruntime.WindowSetPosition(a.ctx, x, y)
}

func (a *App) persistWindowPos(x, y int) {
	if a.repo == nil {
		return
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := a.repo.SetMeta(ctx, store.MetaWindowPos, formatWindowPos(x, y)); err != nil {
		log.Printf("save window_pos: %v", err)
	}
}

func formatWindowPos(x, y int) string { return fmt.Sprintf("%d,%d", x, y) }

func parseWindowPos(s string) (x, y int, err error) {
	parts := strings.Split(strings.TrimSpace(s), ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("want x,y")
	}
	x, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	y, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func (a *App) toggleWindow() {
	if a.visible.Load() {
		a.hideWindow()
	} else {
		a.showWindow()
	}
}

func (a *App) applyAlwaysOnTop() {
	if a.ctx == nil {
		return
	}
	wruntime.WindowSetAlwaysOnTop(a.ctx, a.alwaysOnTop.Load())
}

func (a *App) registerGlobalHotkey() {
	h, err := globalhk.Register(
		[]globalhk.Modifier{globalhk.ModCtrl, globalhk.ModShift, globalhk.ModAlt},
		globalhk.KeyT,
		a.toggleWindow,
	)
	if err != nil {
		log.Printf("global hotkey (Ctrl+Shift+Alt+T): %v", err)
		return
	}
	a.hotkey = h
	platform.OnResume(func() {
		if a.hotkey == nil {
			return
		}
		if err := a.hotkey.Regrab(); err != nil {
			log.Printf("global hotkey re-grab after resume: %v", err)
		}
	})
}

// Quit is the real exit path: tray "Quit" menu item or Ctrl+Q in the window.
// Sets quitting first so OnBeforeClose allows the close instead of hiding to
// tray, then removes the indicator and quits Wails on the shared GTK loop.
func (a *App) Quit() {
	a.quitting.Store(true)
	if a.hotkey != nil {
		a.hotkey.Close()
		a.hotkey = nil
	}
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
	a.hideWindow() // captures position, then hides to tray
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

// GetAlwaysOnTop returns the persisted always-on-top preference.
func (a *App) GetAlwaysOnTop() bool { return a.alwaysOnTop.Load() }

// SetAlwaysOnTop updates the preference, applies it to the window, and stores
// it in the DB meta table for future launches.
func (a *App) SetAlwaysOnTop(on bool) error {
	a.alwaysOnTop.Store(on)
	a.applyAlwaysOnTop()
	if a.repo == nil {
		return nil
	}
	val := "false"
	if on {
		val = "true"
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// Bound a short timeout so a stuck DB cannot hang the UI toggle.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return a.repo.SetMeta(ctx, store.MetaAlwaysOnTop, val)
}
