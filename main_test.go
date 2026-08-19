package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func withLockPath(t *testing.T, dir string) {
	t.Helper()
	old := lockPathFn
	lockPathFn = func() string { return filepath.Join(dir, "mhtodo.lock") }
	t.Cleanup(func() { lockPathFn = old })
}

// deadPID spawns a process and waits for it to exit, returning its (now dead) pid.
func deadPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("sleep", "0.2")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	pid := cmd.Process.Pid
	if err := cmd.Wait(); err != nil {
		t.Fatalf("wait sleep: %v", err)
	}
	return pid
}

func TestAcquireInstanceLockFresh(t *testing.T) {
	withLockPath(t, t.TempDir())
	if err := acquireInstanceLock(); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	pid, ok := readLockPID(lockPathFn())
	if !ok || pid != os.Getpid() {
		t.Fatalf("lock file holds pid %d (ok=%v), want our own %d", pid, ok, os.Getpid())
	}
	releaseInstanceLock()
	if _, err := os.Stat(lockPathFn()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("release did not remove lock: %v", err)
	}
}

func TestAcquireInstanceLockStale(t *testing.T) {
	dir := t.TempDir()
	withLockPath(t, dir)
	if err := os.WriteFile(lockPathFn(), []byte(fmt.Sprintf("%d\n", deadPID(t))), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := acquireInstanceLock(); err != nil {
		t.Fatalf("stale lock should be stolen, got: %v", err)
	}
	releaseInstanceLock()
}

func TestAcquireInstanceLockLive(t *testing.T) {
	dir := t.TempDir()
	withLockPath(t, dir)
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()
	if err := os.WriteFile(lockPathFn(), []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o644); err != nil {
		t.Fatal(err)
	}

	err := acquireInstanceLock()
	var ar *AlreadyRunningError
	if !errors.As(err, &ar) || ar.PID != cmd.Process.Pid {
		t.Fatalf("want AlreadyRunningError{PID:%d}, got: %v", cmd.Process.Pid, err)
	}
	// The live holder's lock file must be left untouched.
	if _, ok := readLockPID(lockPathFn()); !ok {
		t.Fatal("lock file of live holder was modified")
	}
}

func TestReleaseInstanceLockForeign(t *testing.T) {
	dir := t.TempDir()
	withLockPath(t, dir)
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()
	if err := os.WriteFile(lockPathFn(), []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o644); err != nil {
		t.Fatal(err)
	}
	releaseInstanceLock() // not ours → must NOT delete
	if _, ok := readLockPID(lockPathFn()); !ok {
		t.Fatal("release removed a lock we do not own")
	}
}

func TestReadLockPIDGarbage(t *testing.T) {
	dir := t.TempDir()
	withLockPath(t, dir)
	for i, content := range []string{"", "not-a-pid\n", "-5\n", "\n"} {
		if err := os.WriteFile(lockPathFn(), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if pid, ok := readLockPID(lockPathFn()); ok || pid != 0 {
			t.Fatalf("case %d (%q): got (%d,%v), want (0,false)", i, content, pid, ok)
		}
	}
}
