// mhtodo — M0 spike.
//
// Validates the top project risk on this machine: a system tray (AppIndicator via
// getlantern/systray) coexisting with Wails v2's GTK main loop under webkit2gtk-4.1.
//
// Design note (why Register, not Run): systray.Run() would start a second gtk_main(),
// which is the deadlock risk this spike exists to rule out. Instead we call
// systray.Register() BEFORE wails.Run(): it does gtk_init + AppIndicator creation once,
// pre-loop, and every later tray mutation (icon, menu items, quit) is queued with
// g_idle_add onto the single GMainContext that Wails' loop iterates. One GTK main loop total.
package main

import (
	"context"
	"embed"
	"flag"
	"log"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

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
func (a *App) trayShow() { if a.ctx != nil { wruntime.WindowShow(a.ctx) } }
func (a *App) trayHide() { if a.ctx != nil { wruntime.WindowHide(a.ctx) } }
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

func main() {
	selftest := flag.Bool("selftest", false, "auto-run tray callbacks (show → hide → quit) for headless verification")
	flag.Parse()

	// Register the tray BEFORE Wails starts its GTK loop: gtk_init + AppIndicator are
	// created pre-loop; everything after is g_idle_add-queued onto Wails' loop.
	systray.Register(onTrayReady, onTrayExit)

	if *selftest {
		// Must start BEFORE wails.Run(): that call blocks for the app's whole lifetime.
		go func() {
			for app.ctx == nil {
				time.Sleep(50 * time.Millisecond) // wait for Wails startup (GTK loop up)
			}
			time.Sleep(2 * time.Second)           // let tray + window settle
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
