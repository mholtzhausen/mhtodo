//go:build !linux

package platform

import "os/exec"

// SetUseSystemdScopeForTest is a no-op on non-Linux builds.
func SetUseSystemdScopeForTest(func() bool) {}

// DetachedCommandPrefix is empty on non-Linux builds.
func DetachedCommandPrefix() string { return "" }

// StartDetached starts cmd and releases it from the Go runtime (no Setsid on non-Linux).
func StartDetached(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	return nil
}
