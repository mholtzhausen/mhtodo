// Package instance provides the single-instance lock and deep-link focus IPC
// used by the GUI and by `mhtodo open`.
package instance

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// ErrAlreadyRunning is the sentinel wrapped by AlreadyRunningError.
var ErrAlreadyRunning = errors.New("mhtodo is already running")

// AlreadyRunningError carries the holder's pid so the caller can request focus.
type AlreadyRunningError struct{ PID int }

// Error names the stable sentinel; Unwrap links it for errors.Is/As.
func (e *AlreadyRunningError) Error() string {
	return fmt.Sprintf("mhtodo is already running (pid %d)", e.PID)
}

// Unwrap returns ErrAlreadyRunning.
func (e *AlreadyRunningError) Unwrap() error { return ErrAlreadyRunning }

// lockPathFn is a test seam.
var lockPathFn = defaultLockPath

func defaultLockPath() string {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "mhtodo.lock")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("mhtodo-%d.lock", os.Getuid()))
}

// SetLockPathForTest swaps the lock path function; restore with the returned func.
func SetLockPathForTest(fn func() string) (restore func()) {
	prev := lockPathFn
	lockPathFn = fn
	return func() { lockPathFn = prev }
}

// LockPath returns the current lock file path (test/debug).
func LockPath() string { return lockPathFn() }

// Acquire writes our pid to the lock file. It returns nil on success,
// *AlreadyRunningError when a live process holds it, or an I/O error.
func Acquire() error {
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

		holder, ok := ReadLockPID(path)
		if ok && holder != os.Getpid() && PIDAlive(holder) {
			return &AlreadyRunningError{PID: holder}
		}
		os.Remove(path) // stale (dead/unknown pid or our own leftover) — retry
	}
	return fmt.Errorf("instance lock: giving up on %s", path)
}

// ReadLockPID parses a pid from the lock file.
func ReadLockPID(path string) (int, bool) {
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

// PIDAlive reports whether a process with the given pid exists. EPERM means it
// exists but belongs to another user — still alive for our purposes.
func PIDAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// Release removes the lock only if we still own it (a crash of a newer
// instance must not delete its lock).
func Release() {
	path := lockPathFn()
	if pid, ok := ReadLockPID(path); ok && pid == os.Getpid() {
		os.Remove(path)
	}
}

// RunningPID returns the live holder of the instance lock, if any.
func RunningPID() (pid int, ok bool) {
	pid, ok = ReadLockPID(lockPathFn())
	if !ok || !PIDAlive(pid) || pid == os.Getpid() {
		return 0, false
	}
	return pid, true
}
