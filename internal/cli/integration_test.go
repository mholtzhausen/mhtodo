package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceOrAppendBlock(t *testing.T) {
	t.Parallel()
	block := managedBlock()
	got, err := replaceOrAppendBlock("", block)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "claude.todo()") || !strings.Contains(got, shellBlockBegin) {
		t.Fatalf("missing snippet: %q", got)
	}

	updated, err := replaceOrAppendBlock(got+"# leftover\n", block)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(updated, shellBlockBegin) != 1 {
		t.Fatalf("expected one block, got %q", updated)
	}
	if !strings.Contains(updated, "# leftover") {
		t.Fatalf("lost user content: %q", updated)
	}
}

func TestStripBlockMalformed(t *testing.T) {
	t.Parallel()
	_, _, err := stripBlock(shellBlockBegin + "\nno end\n")
	if err == nil {
		t.Fatal("expected error for missing end marker")
	}
}

func TestUpsertAndRemoveShellBlock(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(path, []byte("# existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := upsertShellBlock(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "claude.todo()") {
		t.Fatalf("upsert missing function: %s", data)
	}
	changed, err := removeShellBlock(path)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected remove to change file")
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), shellBlockBegin) {
		t.Fatalf("block still present: %s", data)
	}
	if !strings.Contains(string(data), "# existing") {
		t.Fatalf("lost user content: %s", data)
	}
}

func TestShellRCPath(t *testing.T) {
	t.Parallel()
	p, err := shellRCPath(".zshrc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, ".zshrc") {
		t.Fatalf("path = %q", p)
	}
}
