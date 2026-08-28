//go:build linux

package platform

import (
	"log"

	"github.com/godbus/dbus/v5"
)

// OnResume calls fn after the system wakes from suspend. Requires a system D-Bus
// session (typical desktop). Failures are logged once and ignored.
func OnResume(fn func()) {
	if fn == nil {
		return
	}
	go func() {
		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			log.Printf("resume listener: system bus: %v", err)
			return
		}
		defer conn.Close()

		match := "type='signal',interface='org.freedesktop.login1.Manager',member='PrepareForSleep'"
		if err := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, match).Err; err != nil {
			log.Printf("resume listener: AddMatch: %v", err)
			return
		}

		c := make(chan *dbus.Signal, 8)
		conn.Signal(c)
		for sig := range c {
			if sig == nil || sig.Name != "org.freedesktop.login1.Manager.PrepareForSleep" {
				continue
			}
			if len(sig.Body) < 1 {
				continue
			}
			sleeping, ok := sig.Body[0].(bool)
			if ok && !sleeping {
				fn()
			}
		}
	}()
}
