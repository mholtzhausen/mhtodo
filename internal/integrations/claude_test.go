package integrations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestClaudeArgs(t *testing.T) {
	t.Parallel()
	if got := ClaudeArgs("  "); !reflect.DeepEqual(got, []string{"claude"}) {
		t.Fatalf("no session: got %v", got)
	}
	want := []string{"claude", "--resume", "abc-123"}
	if got := ClaudeArgs(" abc-123 "); !reflect.DeepEqual(got, want) {
		t.Fatalf("session: got %v want %v", got, want)
	}
}

func TestTerminalArgs(t *testing.T) {
	t.Parallel()
	cmd := []string{"claude", "--resume", "u"}
	cases := []struct {
		kind string
		want []string
	}{
		{LauncherGhostty, []string{"/bin/ghostty", "--working-directory=/p q", "-e", "claude", "--resume", "u"}},
		{LauncherXTerminal, []string{"/bin/x-terminal-emulator", "-e", "claude", "--resume", "u"}},
		{LauncherGnomeTerminal, []string{"/bin/gnome-terminal", "--working-directory=/p q", "--", "claude", "--resume", "u"}},
	}
	for _, tc := range cases {
		got, dir := TerminalArgs(tc.kind, "/bin/"+tc.kind, "/p q", cmd)
		if !reflect.DeepEqual(got, tc.want) || dir != "/p q" {
			t.Errorf("%s: got %v dir %q want %v", tc.kind, got, dir, tc.want)
		}
	}
	if got, _ := TerminalArgs("nope", "x", "/p", cmd); got != nil {
		t.Errorf("unknown kind: got %v", got)
	}
}

func TestHerdrAgentName(t *testing.T) {
	t.Parallel()
	if got := HerdrAgentName("019A2B3C"); got != "mhtodo-019a2b3c" {
		t.Fatalf("got %q", got)
	}
}

func TestParseHerdrPaneID(t *testing.T) {
	t.Parallel()
	id, err := parseHerdrPaneID(json.RawMessage(`{"pane":{"pane_id":"wD:p2","tab_id":"wD:t1"},"type":"pane_info"}`))
	if err != nil || id != "wD:p2" {
		t.Fatalf("got %q, %v", id, err)
	}
	if _, err := parseHerdrPaneID(json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected error for missing pane id")
	}
}

func TestResumableSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	proj := filepath.Join(dir, "projects", "-mnt-code-my-app")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "abc.jsonl"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ResumableSession("/mnt/code/my.app", " abc "); got != "abc" {
		t.Fatalf("existing transcript: got %q", got)
	}
	if got := ResumableSession("/mnt/code/my.app", "missing"); got != "" {
		t.Fatalf("missing transcript: got %q", got)
	}
	if got := ResumableSession("/mnt/other", "abc"); got != "" {
		t.Fatalf("other project: got %q", got)
	}
}
