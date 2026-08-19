// mhtodo — one binary, two frontends over one shared core.
//
// Dispatch (see .agent/plan/02-architecture.md):
//
//	mhtodo            → GUI (Wails app + tray)   [also: mhtodo gui]
//	mhtodo <command>  → CLI, exits when done     [agentic path]
//
// The GUI half is the M3 Wails app (app.go). Tray wiring still lives here —
// it moves to internal/tray in M4. Validated ordering from the M0 spike:
// systray.Register() BEFORE wails.Run(), never systray.Run() (that would start
// a second gtk_main).
package main

import (
	"embed"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"mhtodo/internal/cli"
)

// Stamped by the Makefile via -ldflags (see .agent/plan/06-makefile.md).
var (
	version = "dev"
	commit  = "none"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "gui" {
		runGUI(args) // blocks for the app's lifetime
		return
	}
	os.Exit(cli.Run(args, version, commit))
}

// NOTE (M3, 2026-08-19): this machine's Go toolchains reject //go:embed into
// string/[]byte ("imported and not used") but accept embed.FS — so all embeds
// here use embed.FS. Read the bytes out at startup instead of embedding []byte.
//go:embed assets/tray.png
var trayAssets embed.FS

var trayIcon []byte

func init() {
	b, err := trayAssets.ReadFile("assets/tray.png")
	if err != nil {
		log.Fatalf("embedded tray icon: %v", err)
	}
	trayIcon = b
}

func onTrayReady() {
	systray.SetIcon(trayIcon)
	systray.SetTitle("mhtodo")
	systray.SetTooltip("mhtodo") // M4: "mhtodo — N open tasks", refreshed on change

	mShow := systray.AddMenuItem("Show / Hide mhtodo", "Toggle the mhtodo window")
	mNew := systray.AddMenuItem("New Task", "Open a new task in the window") // M3: just shows; dialog opens in M4 tray wiring
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit mhtodo")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				if app.visible {
					app.trayHide()
				} else {
					app.trayShow()
				}
			case <-mNew.ClickedCh:
				app.trayShow() // M4: also emit an event that opens the create dialog
			case <-mQuit.ClickedCh:
				app.trayQuit()
				return
			}
		}
	}()
	log.Println("tray ready")
}

func onTrayExit() { log.Println("tray exit") }

// runGUI is the GUI entrypoint: single-instance lock, focus-on-relaunch signal,
// tray registration (before Wails), then wails.Run.
func runGUI(args []string) {
	if len(args) > 0 && args[0] == "gui" {
		args = args[1:] // flag parsing stops at non-flag args, so strip the subcommand first
	}
	fs := flag.NewFlagSet("gui", flag.ExitOnError)
	selftest := fs.Bool("selftest", false, "auto-run show → hide → quit for headless verification")
	fs.Parse(args)

	// Single instance: a second launch focuses the running one and exits.
	if err := acquireInstanceLock(); err != nil {
		var ar *AlreadyRunningError
		if errors.As(err, &ar) {
			log.Printf("mhtodo is already running (pid %d); focusing existing window", ar.PID)
			syscall.Kill(ar.PID, syscall.SIGUSR2) // best-effort focus request (never SIGUSR1 — see above)
			return
		}
		log.Fatalf("instance lock: %v", err)
	}

	// Focus-on-relaunch: a second instance signals us to show the window.
	// MUST be SIGUSR2 — WebKit/JSC installs its own C handler for signal 10
	// (SIGUSR1, "JSC_SIGNAL_FOR_GC"); sending SIGUSR1 to this process crashes it
	// with SIGSEGV during cgo execution (verified 2026-08-19).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR2)
	go func() {
		for range sigCh {
			log.Println("focus requested by second instance → showing window")
			app.trayShow()
		}
	}()

	systray.Register(onTrayReady, onTrayExit)

	if *selftest {
		// Must start BEFORE wails.Run(): that call blocks for the app's whole lifetime.
		go func() {
			for app.ctx == nil {
				time.Sleep(50 * time.Millisecond) // wait for Wails startup (GTK loop up)
			}
			time.Sleep(2 * time.Second) // let tray + window settle
			log.Println("selftest: show")
			app.trayShow()
			time.Sleep(1500 * time.Millisecond)
			log.Println("selftest: hide")
			app.trayHide()
			time.Sleep(1500 * time.Millisecond)
			log.Println("selftest: quit")
			app.trayQuit()
		}()
	}

	err := wails.Run(&options.App{
		Title:     "mhtodo",
		Width:     1100,
		Height:    720,
		MinWidth:  800,
		MinHeight: 560,
		AssetServer: &assetserver.Options{
			Assets: assets, // embedded frontend/dist; ignored under -tags dev (vite server)
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind:       []interface{}{app},
	})
	if err != nil {
		log.Fatalf("wails: %v", err)
	}
}
