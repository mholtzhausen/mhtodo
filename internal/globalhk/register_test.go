//go:build linux && cgo

package globalhk

import "testing"

func TestRegisterAndClose(t *testing.T) {
	// Uncommon combo so we do not fight the product binding or DE shortcuts.
	h, err := Register([]Modifier{ModCtrl, ModShift, ModAlt}, Key(0xffc9) /* F12 */, func() {})
	if err != nil {
		t.Skipf("skip (no display or conflict): %v", err)
	}
	h.Close()
}
