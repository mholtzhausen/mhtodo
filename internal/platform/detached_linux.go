//go:build linux

package platform

import (
	"os"
	"os/exec"
	"syscall"
)

// useSystemdScope overrides canUseSystemdScope when non-nil (tests).
var useSystemdScope func() bool

// SetUseSystemdScopeForTest forces whether StartDetached wraps with systemd-run.
// Pass nil to restore default detection.
func SetUseSystemdScopeForTest(fn func() bool) {
	useSystemdScope = fn
}

func canUseSystemdScope() bool {
	if useSystemdScope != nil {
		return useSystemdScope()
	}
	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		return false
	}
	_, err := exec.LookPath("systemd-run")
	return err == nil
}

// DetachedCommandPrefix returns a shell fragment for tooltips when StartDetached
// would wrap the command (systemd user scope, outside the service cgroup).
func DetachedCommandPrefix() string {
	if !canUseSystemdScope() {
		return ""
	}
	return "systemd-run --user --scope --collect -- "
}

// StartDetached starts cmd so it can outlive the caller. With a systemd user
// session, the command runs in a transient scope (survives mhtodo.service stop on
// update); otherwise it starts in a new session (Setsid).
func StartDetached(cmd *exec.Cmd) error {
	if canUseSystemdScope() {
		if err := startSystemdScope(cmd); err == nil {
			return nil
		}
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	return nil
}

func startSystemdScope(cmd *exec.Cmd) error {
	args := append([]string{"--user", "--scope", "--collect", "--"}, cmd.Args...)
	wrap := exec.Command("systemd-run", args...)
	wrap.Env = cmd.Env
	if wrap.Env == nil {
		wrap.Env = os.Environ()
	}
	wrap.Dir = cmd.Dir
	wrap.Stdin = nil
	wrap.Stdout = cmd.Stdout
	wrap.Stderr = cmd.Stderr
	if err := wrap.Start(); err != nil {
		return err
	}
	_ = wrap.Process.Release()
	return nil
}
