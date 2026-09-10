// Package tray wires the system tray using the M0-validated pattern:
// systray.Register() is called BEFORE wails.Run() (never systray.Run(), which
// would start a second gtk_main). Register does gtk_init + AppIndicator setup
// pre-loop; all later tray mutations queue via g_idle_add onto Wails' single
// GTK loop, so they are safe to call from any goroutine.
package tray

import (
	"log"
	"sync"

	"github.com/getlantern/systray"

	"mhtodo/internal/traymenu"
)

// Handlers are the app's callbacks for menu actions. They run on a dedicated
// goroutine (not the GTK main thread); they must only touch Go state and Wails
// runtime APIs, which are safe from any goroutine.
type Handlers struct {
	ToggleWindow        func() // Show / Hide mhtodo
	NewTask             func() // show window + open create dialog
	NewTaskFromTemplate func() // show window + open create dialog with the template picker
	OpenSettings        func() // show window + open Settings
	FocusTask           func(id string) // show window + focus task from a status submenu
	Quit                func()          // real exit (systray.Quit + wails quit)
}

// statusKeys is the fixed set of submenu roots (slots preallocated once).
var statusKeys = []string{"pending", "wip", "waiting", "review", "done"}

type statusSlot struct {
	parent *systray.MenuItem
	empty  *systray.MenuItem
	items  []*systray.MenuItem
}

var (
	menuMu     sync.Mutex
	ready      bool
	focusTask  func(string)
	statusMenu map[string]*statusSlot
	slotTaskID map[string][]string // status → per-slot task id
)

// Register installs the tray icon and menu. Must be called before wails.Run().
func Register(icon []byte, h Handlers) {
	focusTask = h.FocusTask
	systray.Register(func() {
		systray.SetIcon(icon)
		systray.SetTitle("mhtodo")
		systray.SetTooltip("mhtodo") // refreshed via SetTooltip as task counts change

		mToggle := systray.AddMenuItem("Show / Hide mhtodo", "Toggle the mhtodo window")
		mNew := systray.AddMenuItem("New Task", "Open a new task in the window")
		mNewFromTpl := systray.AddMenuItem("New Task from Template", "Open a new task and pick a template")
		mSettings := systray.AddMenuItem("Settings", "Open Settings")
		systray.AddSeparator()

		statusMenu = make(map[string]*statusSlot, len(statusKeys))
		slotTaskID = make(map[string][]string, len(statusKeys))
		for _, st := range statusKeys {
			parent := systray.AddMenuItem(traymenu.StatusMenuTitle(st, 0), "")
			parent.Hide()
			empty := parent.AddSubMenuItem("No tasks", "")
			empty.Disable()
			items := make([]*systray.MenuItem, traymenu.MaxItemsHardCap)
			ids := make([]string, traymenu.MaxItemsHardCap)
			for i := 0; i < traymenu.MaxItemsHardCap; i++ {
				it := parent.AddSubMenuItem("…", "")
				it.Hide()
				items[i] = it
				st, i := st, i
				go func() {
					for range it.ClickedCh {
						menuMu.Lock()
						id := ""
						if slotTaskID != nil {
							if row := slotTaskID[st]; i < len(row) {
								id = row[i]
							}
						}
						fn := focusTask
						menuMu.Unlock()
						if id != "" && fn != nil {
							fn(id)
						}
					}
				}()
			}
			statusMenu[st] = &statusSlot{parent: parent, empty: empty, items: items}
			slotTaskID[st] = ids
		}

		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit mhtodo")

		menuMu.Lock()
		ready = true
		menuMu.Unlock()

		go func() {
			for {
				select {
				case <-mToggle.ClickedCh:
					h.ToggleWindow()
				case <-mNew.ClickedCh:
					h.NewTask()
				case <-mNewFromTpl.ClickedCh:
					if h.NewTaskFromTemplate != nil {
						h.NewTaskFromTemplate()
					}
				case <-mSettings.ClickedCh:
					if h.OpenSettings != nil {
						h.OpenSettings()
					}
				case <-mQuit.ClickedCh:
					h.Quit()
					return
				}
			}
		}()
		log.Println("tray ready")
	}, func() { log.Println("tray exit") })
}

// UpdateStatusMenus applies the board section: which status roots are visible
// and which task rows fill the fixed slots. Safe from any goroutine after Register.
func UpdateStatusMenus(sections []traymenu.StatusSection) {
	menuMu.Lock()
	defer menuMu.Unlock()
	if !ready || statusMenu == nil {
		return
	}

	visible := make(map[string]traymenu.StatusSection, len(sections))
	for _, sec := range sections {
		visible[sec.Status] = sec
	}

	for _, st := range statusKeys {
		slot := statusMenu[st]
		if slot == nil {
			continue
		}
		sec, ok := visible[st]
		if !ok || !sec.Visible {
			slot.parent.Hide()
			continue
		}
		slot.parent.SetTitle(sec.Label)
		slot.parent.Show()

		n := len(sec.Items)
		if n > len(slot.items) {
			n = len(slot.items)
		}
		ids := slotTaskID[st]
		for i := 0; i < len(slot.items); i++ {
			if i < n {
				title := traymenu.TruncateTitle(sec.Items[i].Title, traymenu.DefaultTitleTruncate)
				if title == "" {
					title = "(untitled)"
				}
				slot.items[i].SetTitle(title)
				tip := sec.Items[i].Title
				if id := sec.Items[i].ID; id != "" {
					if tip != "" {
						tip = tip + "\n" + id
					} else {
						tip = id
					}
				}
				slot.items[i].SetTooltip(tip)
				slot.items[i].Enable()
				slot.items[i].Show()
				if i < len(ids) {
					ids[i] = sec.Items[i].ID
				}
			} else {
				slot.items[i].Hide()
				if i < len(ids) {
					ids[i] = ""
				}
			}
		}
		if n == 0 {
			slot.empty.Show()
		} else {
			slot.empty.Hide()
		}
	}
}

// SetTooltip updates the tray tooltip ("mhtodo — N open tasks"). Safe to call
// from any goroutine after Register (queued onto the GTK loop).
// NOTE: on Linux the getlantern/systray AppIndicator backend is a no-op for
// tooltips (libappindicator has no tooltip API) — see SetLabel for the channel
// that actually shows on this machine.
func SetTooltip(s string) { systray.SetTooltip(s) }

// SetLabel updates the text shown next to the tray icon: XAyatanaLabel on
// Linux AppIndicator (visible in Cinnamon when "show label" is enabled for the
// icon), window title elsewhere. This is where the open-task count lands on
// this machine, since SetTooltip is a no-op there.
func SetLabel(s string) { systray.SetTitle(s) }

// Quit removes the indicator and queues gtk_main_quit on the shared loop.
// Call it together with wruntime.Quit for a real exit.
func Quit() { systray.Quit() }
