package integrations

import (
	"testing"

	"mhtodo/internal/settings"
)

func TestMaybeCloseTicketTabOnDoneDisabled(t *testing.T) {
	t.Parallel()
	c := Client{
		Herdr:  settings.HerdrConfig{Enabled: true, Binary: "herdr"},
		Claude: settings.ClaudeConfig{CloseTabOnDone: false},
	}
	c.MaybeCloseTicketTabOnDone("task-id", "abcd1234", "Title") // must not panic or call herdr
}

func TestProcessIsClaude(t *testing.T) {
	t.Parallel()
	claudeProc := herdrForegroundProcess{
		Name:    "2.1.233",
		Cmdline: "/home/user/.local/share/claude/versions/2.1.233 read todo abc",
		Argv:    []string{"/home/user/.local/share/claude/versions/2.1.233", "read todo abc"},
	}
	if !processIsClaude(claudeProc, "/usr/bin/claude") {
		t.Fatal("expected claude version binary to match")
	}
	if processIsClaude(herdrForegroundProcess{Name: "bash", Cmdline: "bash"}, "/usr/bin/claude") {
		t.Fatal("bash should not match claude")
	}
}
