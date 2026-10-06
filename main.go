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
	"mhtodo/internal/instance"
	"mhtodo/internal/platform"
	"mhtodo/internal/settings"
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
	// Desktop MimeType handler: Exec=mhtodo %u passes mhtodo://task/… as argv[1].
	if hasMhtodoScheme(args[0]) {
		os.Exit(cli.Run(append([]string{"open"}, args...), version, commit))
		return
	}
	os.Exit(cli.Run(args, version, commit))
}

// hasMhtodoScheme reports whether s starts with "mhtodo:" (case-insensitive).
func hasMhtodoScheme(s string) bool {
	const prefix = "mhtodo:"
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != prefix[i] {
			return false
		}
	}
	return true
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

// launchStartHidden is resolved once in runGUI from config (see resolveLaunchStartHidden).
var launchStartHidden bool

func resolveLaunchStartHidden() bool {
	s, err := settings.Load(nil)
	if err != nil {
		return startHidden
	}
	return s.StartHidden
}

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

	launchStartHidden = resolveLaunchStartHidden()
	// startHidden is the build-tag const (true under make dev). Distinct from the
	// user StartHidden setting: deep-link SIGUSR2 must still work when the user
	// prefers launch-to-tray, but must stay off under wails hot-reload.
	devRebuild := startHidden

	// Single instance: a second launch focuses the running one and exits.
	// Under -tags dev (make dev), skip focus signaling — wails generate module
	// and hot-reload rebuilds spawn short-lived second processes that would
	// otherwise pop the hidden window on every rebuild.
	// A pending mhtodo.focus file (from `mhtodo open`) is left in place so the
	// running instance can TakeFocusRequest on SIGUSR2.
	if err := instance.Acquire(); err != nil {
		var ar *instance.AlreadyRunningError
		if errors.As(err, &ar) {
			if devRebuild {
				log.Printf("mhtodo is already running (pid %d); exiting quietly (dev)", ar.PID)
			} else {
				log.Printf("mhtodo is already running (pid %d); focusing existing window", ar.PID)
				// Do NOT clear mhtodo.focus here: `mhtodo open` may have just written a
				// task id and then launched us; wiping would drop the deep link.
				// Stale requests expire in TakeFocusRequest (mtime TTL).
				_ = instance.SignalFocus(ar.PID) // best-effort; never SIGUSR1 — see SignalFocus
			}
			return
		}
		log.Fatalf("instance lock: %v", err)
	}

	// Focus-on-relaunch / deep link: a second instance (or `mhtodo open`) signals
	// SIGUSR2. MUST be SIGUSR2 — WebKit/JSC owns SIGUSR1 ("JSC_SIGNAL_FOR_GC");
	// SIGUSR1 crashes the GUI (verified 2026-08-19).
	// Not registered under make dev: rebuild spawns would steal focus.
	if !devRebuild {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGUSR2)
		go func() {
			for range sigCh {
				log.Println("focus requested by second instance")
				app.applyPendingFocus()
			}
		}()
	}

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
		NewTask:             app.openNewTaskFromTray,
		NewTaskFromTemplate: app.openNewTaskFromTemplateFromTray,
		OpenSettings:        app.openSettingsFromTray,
		FocusTask:           app.openFocusTaskFromTray,
		Quit:                app.Quit,
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
		Frameless:   true, // custom chrome: drag + dblclick-maximize on app header
		StartHidden: launchStartHidden, // user setting in config.yml (General → Start hidden)
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
