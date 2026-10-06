//go:build linux

package platform

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDetachedCommandPrefix(t *testing.T) {
	t.Parallel()
	SetUseSystemdScopeForTest(func() bool { return true })
	t.Cleanup(func() { SetUseSystemdScopeForTest(nil) })
	if p := DetachedCommandPrefix(); !strings.HasPrefix(p, "systemd-run ") {
		t.Fatalf("prefix = %q", p)
	}
}

func TestStartDetachedSystemdScopeArgs(t *testing.T) {
	SetUseSystemdScopeForTest(func() bool { return true })
	t.Cleanup(func() { SetUseSystemdScopeForTest(nil) })

	if _, err := exec.LookPath("systemd-run"); err != nil {
		t.Skip("systemd-run not on PATH")
	}

	cmd := exec.Command("true")
	if err := StartDetached(cmd); err != nil {
		t.Fatalf("StartDetached: %v", err)
	}
}
