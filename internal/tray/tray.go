// Package tray wires the system tray using the M0-validated pattern:
// systray.Register() is called BEFORE wails.Run() (never systray.Run(), which
// would start a second gtk_main). Register does gtk_init + AppIndicator setup
// pre-loop; all later tray mutations queue via g_idle_add onto Wails' single
// GTK loop, so they are safe to call from any goroutine.
package tray

import (
	"log"

	"github.com/getlantern/systray"
)

// Handlers are the app's callbacks for menu actions. They run on a dedicated
// goroutine (not the GTK main thread); they must only touch Go state and Wails
// runtime APIs, which are safe from any goroutine.
type Handlers struct {
	ToggleWindow func() // Show / Hide mhtodo
	NewTask      func() // show window + open create dialog
	Quit         func() // real exit (systray.Quit + wails quit)
}

// Register installs the tray icon and menu. Must be called before wails.Run().
func Register(icon []byte, h Handlers) {
	systray.Register(func() {
		systray.SetIcon(icon)
		systray.SetTitle("mhtodo")
		systray.SetTooltip("mhtodo") // refreshed via SetTooltip as task counts change

		mToggle := systray.AddMenuItem("Show / Hide mhtodo", "Toggle the mhtodo window")
		mNew := systray.AddMenuItem("New Task", "Open a new task in the window")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit mhtodo")

		go func() {
			for {
				select {
				case <-mToggle.ClickedCh:
					h.ToggleWindow()
				case <-mNew.ClickedCh:
					h.NewTask()
				case <-mQuit.ClickedCh:
					h.Quit()
					return
				}
			}
		}()
		log.Println("tray ready")
	}, func() { log.Println("tray exit") })
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
