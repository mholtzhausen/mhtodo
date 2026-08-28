package platform

import (
	"os"
	"testing"
)

func TestPreferX11Backend(t *testing.T) {
	t.Setenv("MHTODO_WAYLAND", "")
	t.Setenv("GDK_BACKEND", "")
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("DISPLAY", ":0")
	PreferX11Backend()
	if got := os.Getenv("GDK_BACKEND"); got != "x11" {
		t.Fatalf("GDK_BACKEND = %q, want x11", got)
	}

	t.Setenv("MHTODO_WAYLAND", "1")
	t.Setenv("GDK_BACKEND", "")
	PreferX11Backend()
	if got := os.Getenv("GDK_BACKEND"); got != "" {
		t.Fatalf("opt-out: GDK_BACKEND = %q, want empty", got)
	}
}
