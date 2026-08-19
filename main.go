// mhtodo — one binary, two frontends over one shared core.
//
// Dispatch (see .agent/plan/02-architecture.md):
//
//	mhtodo            → GUI (Wails app + tray)   [also: mhtodo gui]
//	mhtodo <command>  → CLI, exits when done     [agentic path]
//
// The GUI half below is the M0 spike (validated pattern: systray.Register()
// BEFORE wails.Run(), never systray.Run() — that would start a second
// gtk_main). It moves to app.go + internal/tray in M3/M4.
package main

import (
	"context"
	"embed"
	"flag"
	"log"
	"os"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"mhtodo/internal/cli"
)

// Stamped by the Makefile via -ldflags (see 06-makefile.md).
var (
	version = "dev"
	commit  = "none"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "gui" {
		runGUI() // blocks for the app's lifetime
		return
	}
	os.Exit(cli.Run(args, version, commit))
}

//go:embed frontend/index.html
var assets embed.FS

//go:embed assets/tray.png
var trayIcon []byte

type App struct {
	ctx context.Context // Wails lifecycle context; all runtime calls go through it
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }
func (a *App) shutdown(_ context.Context)  {}

// Tray callbacks — the exact functions the menu items invoke.
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

var app = &App{}

func onTrayReady() {
	systray.SetIcon(trayIcon)
	systray.SetTitle("mhtodo")
	systray.SetTooltip("mhtodo (M0 spike)")

	mShow := systray.AddMenuItem("Show", "Show the mhtodo window")
	mHide := systray.AddMenuItem("Hide", "Hide the mhtodo window")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit mhtodo")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				app.trayShow()
			case <-mHide.ClickedCh:
				app.trayHide()
			case <-mQuit.ClickedCh:
				app.trayQuit()
				return
			}
		}
	}()
	log.Println("tray ready")
}

func onTrayExit() { log.Println("tray exit") }

// runGUI is the M0-spike GUI entrypoint (M3 replaces it with app.go + Vite).
func runGUI() {
	fs := flag.NewFlagSet("gui", flag.ExitOnError)
	selftest := fs.Bool("selftest", false, "auto-run tray callbacks (show → hide → quit) for headless verification")
	fs.Parse(os.Args[1:])

	// Register the tray BEFORE Wails starts its GTK loop: gtk_init + AppIndicator are
	// created pre-loop; everything after is g_idle_add-queued onto Wails' loop.
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
		Title:  "mhtodo — M0 spike",
		Width:  960,
		Height: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind:       []interface{}{app},
	})
	if err != nil {
		log.Fatalf("wails: %v", err)
	}
}
