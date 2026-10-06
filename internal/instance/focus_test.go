package instance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteTakeFocusRequest(t *testing.T) {
	dir := t.TempDir()
	restore := SetLockPathForTest(func() string { return filepath.Join(dir, "mhtodo.lock") })
	t.Cleanup(restore)

	if err := WriteFocusRequest("81abc903"); err != nil {
		t.Fatalf("write: %v", err)
	}
	ref, ok := TakeFocusRequest()
	if !ok || ref != "81abc903" {
		t.Fatalf("take: got (%q,%v)", ref, ok)
	}
	if _, ok := TakeFocusRequest(); ok {
		t.Fatal("second take should miss")
	}
}

func TestWriteFocusRequestEmpty(t *testing.T) {
	dir := t.TempDir()
	restore := SetLockPathForTest(func() string { return filepath.Join(dir, "mhtodo.lock") })
	t.Cleanup(restore)

	if err := WriteFocusRequest("  "); err != nil {
		t.Fatalf("write: %v", err)
	}
	ref, ok := TakeFocusRequest()
	if !ok || ref != "" {
		t.Fatalf("want empty ref, got (%q,%v)", ref, ok)
	}
	if _, err := os.Stat(focusRequestPath()); !os.IsNotExist(err) {
		t.Fatalf("focus file should be consumed: %v", err)
	}
}
