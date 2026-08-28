// mhtodo — one binary, two frontends over one shared core.
//
// Dispatch (see .agent/plan/02-architecture.md):
//
//	mhtodo            → GUI (Wails app + tray)   [also: mhtodo gui]
//	mhtodo <command>  → CLI, exits when done     [agentic path]
//
// The GUI half is the Wails app (app.go) with tray wiring in internal/tray.
// Validated ordering from the M0 spike: systray.Register() BEFORE wails.Run(),
// never systray.Run() (that would start a second gtk_main).
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

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"

	"mhtodo/internal/cli"
	"mhtodo/internal/platform"
	"mhtodo/internal/tray"
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
//
//go:embed assets/icon.png assets/tray.png
var uiAssets embed.FS

var (
	appIcon  []byte // window/taskbar icon → options.Linux.Icon
	trayIcon []byte // AppIndicator tray icon
)

func init() {
	var err error
	if appIcon, err = uiAssets.ReadFile("assets/icon.png"); err != nil {
		log.Fatalf("embedded app icon: %v", err)
	}
	if trayIcon, err = uiAssets.ReadFile("assets/tray.png"); err != nil {
		log.Fatalf("embedded tray icon: %v", err)
	}
}

// Tray wiring lives in internal/tray (M0-validated Register pattern); the app
// provides the handlers. Tooltip counts are refreshed by App.emitChanged via
// tray.SetTooltip.

// runGUI is the GUI entrypoint: single-instance lock, focus-on-relaunch signal,
// tray registration (before Wails), then wails.Run.
func runGUI(args []string) {
	platform.PreferX11Backend()

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
			app.showWindow()
		}
	}()

	// SIGINT/SIGTERM must really quit — Wails routes its own signal handling
	// through OnBeforeClose, which would otherwise hide-to-tray and swallow the
	// signal (Ctrl+C / kill would hang forever). Our handler sets quitting
	// first, so whichever path reaches OnBeforeClose last allows the exit.
	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range termCh {
			log.Println("termination signal received → quitting")
			app.Quit()
		}
	}()

	// M0-validated ordering: Register BEFORE wails.Run (see internal/tray).
	tray.Register(trayIcon, tray.Handlers{
		ToggleWindow: func() {
			if app.visible.Load() {
				app.hideWindow()
			} else {
				app.showWindow()
			}
		},
		NewTask: app.openNewTaskFromTray,
		Quit:    app.Quit,
	})

	if *selftest {
		// Must start BEFORE wails.Run(): that call blocks for the app's whole lifetime.
		go func() {
			for app.ctx == nil {
				time.Sleep(50 * time.Millisecond) // wait for Wails startup (GTK loop up)
			}
			time.Sleep(2 * time.Second) // let tray + window settle
			log.Println("selftest: show")
			app.showWindow()
			time.Sleep(1500 * time.Millisecond)
			log.Println("selftest: hide")
			app.hideWindow()
			time.Sleep(1500 * time.Millisecond)
			log.Println("selftest: quit")
			app.Quit()
		}()
	}

	err := wails.Run(&options.App{
		Title:       "mhtodo",
		Width:       1100,
		Height:      720,
		MinWidth:    800,
		MinHeight:   560,
		StartHidden: startHidden, // true under -tags dev (make dev); tray can still show
		Linux: &linux.Options{
			Icon: appIcon, // window/taskbar icon (M6)
		},
		AssetServer: &assetserver.Options{
			Assets: assets, // embedded frontend/dist; ignored under -tags dev (vite server)
		},
		OnStartup:     app.startup,
		OnDomReady:    app.domReady,
		OnShutdown:    app.shutdown,
		OnBeforeClose: app.beforeClose, // hide-to-tray unless Quit() was called first
		Bind:          []interface{}{app},
	})
	if err != nil {
		log.Fatalf("wails: %v", err)
	}
}
