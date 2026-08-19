package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Single-instance lock (.agent/plan/05-gui-spec.md): a pid file in
// $XDG_RUNTIME_DIR/mhtodo.lock with stale-lock detection. A second launch
// focuses the running instance (SIGUSR1 → WindowShow) and exits.

var ErrAlreadyRunning = errors.New("mhtodo is already running")

// AlreadyRunningError carries the holder's pid so the caller can request focus.
type AlreadyRunningError struct{ PID int }

// Error names the stable sentinel; Unwrap links it for errors.Is/As.
func (e *AlreadyRunningError) Error() string {
	return fmt.Sprintf("mhtodo is already running (pid %d)", e.PID)
}
func (e *AlreadyRunningError) Unwrap() error { return ErrAlreadyRunning }

// lockPathFn is a test seam.
var lockPathFn = instanceLockPath

func instanceLockPath() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "mhtodo.lock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("mhtodo-%d.lock", os.Getuid()))
}

// acquireInstanceLock writes our pid to the lock file. It returns nil on
// success, *AlreadyRunningError when a live process holds it, or an I/O error.
func acquireInstanceLock() error {
	path := lockPathFn()
	for attempt := 0; attempt < 3; attempt++ {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, werr := fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return werr
		}
		if !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("instance lock: %w", err)
		}

		holder, ok := readLockPID(path)
		if ok && holder != os.Getpid() && pidAlive(holder) {
			return &AlreadyRunningError{PID: holder}
		}
		os.Remove(path) // stale (dead/unknown pid or our own leftover) — retry
	}
	return fmt.Errorf("instance lock: giving up on %s", path)
}

func readLockPID(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var pid int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid); err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

// pidAlive reports whether a process with the given pid exists. EPERM means it
// exists but belongs to another user — still alive for our purposes.
func pidAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// releaseInstanceLock removes the lock only if we still own it (a crash of a
// newer instance must not delete its lock).
func releaseInstanceLock() {
	path := lockPathFn()
	if pid, ok := readLockPID(path); ok && pid == os.Getpid() {
		os.Remove(path)
	}
}
