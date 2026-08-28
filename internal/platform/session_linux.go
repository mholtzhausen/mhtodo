//go:build linux

package platform

import "os"

// PreferX11Backend sets GDK_BACKEND=x11 on Wayland sessions when the user has
// not opted into native Wayland. Ubuntu 24 defaults to Wayland where Wails'
// gtk_window_get_position often returns (0,0) and global XGrabKey grabs are
// dropped after lock/suspend. XWayland restores reliable coordinates and hotkeys.
//
// Opt out with MHTODO_WAYLAND=1. Override backend explicitly with GDK_BACKEND.
func PreferX11Backend() {
	if os.Getenv("MHTODO_WAYLAND") == "1" {
		return
	}
	if os.Getenv("GDK_BACKEND") != "" {
		return
	}
	if os.Getenv("XDG_SESSION_TYPE") != "wayland" {
		return
	}
	if os.Getenv("DISPLAY") == "" {
		return
	}
	_ = os.Setenv("GDK_BACKEND", "x11")
}
