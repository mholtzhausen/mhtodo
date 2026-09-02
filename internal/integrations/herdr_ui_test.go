package integrations

import "testing"

func TestNeedsAttachFromReason(t *testing.T) {
	t.Parallel()
	cases := []struct {
		reason     string
		tuiRunning bool
		want       bool
	}{
		{"no_foreground_client", false, true},
		{"no_foreground_client", true, true},
		{"disabled", false, true},
		{"disabled", true, false},
		{"shown", false, false},
		{"busy", true, false},
	}
	for _, tc := range cases {
		if got := needsAttachFromReason(tc.reason, tc.tuiRunning); got != tc.want {
			t.Fatalf("reason=%q tui=%v: got %v want %v", tc.reason, tc.tuiRunning, got, tc.want)
		}
	}
}

func TestIsHerdrCLIProbe(t *testing.T) {
	t.Parallel()
	if !isHerdrCLIProbe("/usr/bin/herdr workspace list") {
		t.Fatal("expected workspace list to be a probe")
	}
	if isHerdrCLIProbe("/usr/bin/herdr session attach default") {
		t.Fatal("expected session attach to be a TUI")
	}
	if isHerdrCLIProbe("/usr/bin/herdr") {
		t.Fatal("bare herdr should be a TUI")
	}
}
