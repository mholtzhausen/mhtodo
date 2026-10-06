package instance

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
	restore := SetLockPathForTest(func() string { return filepath.Join(dir, "mhtodo.lock") })
	t.Cleanup(restore)
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

func TestAcquireFresh(t *testing.T) {
	withLockPath(t, t.TempDir())
	if err := Acquire(); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	pid, ok := ReadLockPID(LockPath())
	if !ok || pid != os.Getpid() {
		t.Fatalf("lock file holds pid %d (ok=%v), want our own %d", pid, ok, os.Getpid())
	}
	Release()
	if _, err := os.Stat(LockPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("release did not remove lock: %v", err)
	}
}

func TestAcquireStale(t *testing.T) {
	dir := t.TempDir()
	withLockPath(t, dir)
	if err := os.WriteFile(LockPath(), []byte(fmt.Sprintf("%d\n", deadPID(t))), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Acquire(); err != nil {
		t.Fatalf("stale lock should be stolen, got: %v", err)
	}
	Release()
}

func TestAcquireLive(t *testing.T) {
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
	if err := os.WriteFile(LockPath(), []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Acquire()
	var ar *AlreadyRunningError
	if !errors.As(err, &ar) || ar.PID != cmd.Process.Pid {
		t.Fatalf("want AlreadyRunningError{PID:%d}, got: %v", cmd.Process.Pid, err)
	}
	if _, ok := ReadLockPID(LockPath()); !ok {
		t.Fatal("lock file of live holder was modified")
	}
}

func TestReleaseForeign(t *testing.T) {
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
	if err := os.WriteFile(LockPath(), []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o644); err != nil {
		t.Fatal(err)
	}
	Release() // not ours → must NOT delete
	if _, ok := ReadLockPID(LockPath()); !ok {
		t.Fatal("release removed a lock we do not own")
	}
}

func TestReadLockPIDGarbage(t *testing.T) {
	dir := t.TempDir()
	withLockPath(t, dir)
	for i, content := range []string{"", "not-a-pid\n", "-5\n", "\n"} {
		if err := os.WriteFile(LockPath(), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if pid, ok := ReadLockPID(LockPath()); ok || pid != 0 {
			t.Fatalf("case %d (%q): got (%d,%v), want (0,false)", i, content, pid, ok)
		}
	}
}

func TestRunningPID(t *testing.T) {
	dir := t.TempDir()
	withLockPath(t, dir)
	if _, ok := RunningPID(); ok {
		t.Fatal("empty lock should not report running")
	}
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()
	if err := os.WriteFile(LockPath(), []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o644); err != nil {
		t.Fatal(err)
	}
	pid, ok := RunningPID()
	if !ok || pid != cmd.Process.Pid {
		t.Fatalf("RunningPID=%d,%v want %d,true", pid, ok, cmd.Process.Pid)
	}
}
