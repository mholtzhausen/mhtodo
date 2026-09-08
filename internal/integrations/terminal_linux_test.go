//go:build linux

package integrations

import "testing"

func TestTerminalArgsForTitle(t *testing.T) {
	got := terminalArgsFor("gnome-terminal", "echo hi", "mhtodo:sess")
	wantPrefix := []string{"--window", "--title", "mhtodo:sess", "--", "bash", "-lc", "echo hi"}
	if len(got) != len(wantPrefix) {
		t.Fatalf("got %#v", got)
	}
	for i := range wantPrefix {
		if got[i] != wantPrefix[i] {
			t.Fatalf("got %#v want %#v", got, wantPrefix)
		}
	}
	if got := terminalArgsFor("kitty", "echo hi", "mhtodo:sess"); got[0] != "--title" || got[1] != "mhtodo:sess" {
		t.Fatalf("kitty title: %#v", got)
	}
}
