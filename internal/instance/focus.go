package instance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// focusRequestTTL drops deep-link requests that were never consumed (crash,
// abandoned launch). Wall-clock mtime is acceptable for this soft expiry.
const focusRequestTTL = 60 * time.Second

// focusRequestPath sits next to the lock file so both share XDG_RUNTIME_DIR.
func focusRequestPath() string {
	lock := lockPathFn()
	return filepath.Join(filepath.Dir(lock), "mhtodo.focus")
}

// WriteFocusRequest stores a task ref (or empty for show-window-only) for the
// running GUI to consume on SIGUSR2 / startup.
func WriteFocusRequest(ref string) error {
	path := focusRequestPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("focus request dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".mhtodo.focus.*")
	if err != nil {
		return fmt.Errorf("focus request temp: %w", err)
	}
	tmpName := tmp.Name()
	payload := strings.TrimSpace(ref) + "\n"
	if _, err := tmp.WriteString(payload); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("focus request write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("focus request close: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("focus request rename: %w", err)
	}
	return nil
}

// TakeFocusRequest reads and removes the pending focus request.
// ok is false when no file exists or the request is older than focusRequestTTL.
func TakeFocusRequest() (ref string, ok bool) {
	path := focusRequestPath()
	fi, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	if time.Since(fi.ModTime()) > focusRequestTTL {
		os.Remove(path)
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	os.Remove(path)
	return strings.TrimSpace(string(data)), true
}

// SignalFocus asks a running instance to show (and optionally focus a task
// previously written with WriteFocusRequest). MUST be SIGUSR2 — WebKit/JSC
// owns SIGUSR1 ("JSC_SIGNAL_FOR_GC"); SIGUSR1 crashes the GUI.
func SignalFocus(pid int) error {
	if err := syscall.Kill(pid, syscall.SIGUSR2); err != nil {
		return fmt.Errorf("signal focus to pid %d: %w", pid, err)
	}
	return nil
}

// RequestFocus writes the focus ref and signals a live instance.
func RequestFocus(pid int, ref string) error {
	if err := WriteFocusRequest(ref); err != nil {
		return err
	}
	return SignalFocus(pid)
}
