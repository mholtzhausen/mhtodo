package integrations

import (
	"strings"
	"testing"

	"mhtodo/internal/settings"
)

func TestClaudeTerminalCommandLineSessionIDAndResume(t *testing.T) {
	c := Client{
		Claude: settings.ClaudeConfig{
			IntegrationConfig: settings.IntegrationConfig{
				Enabled: true,
				Binary:  "/usr/bin/claude",
			},
			Spawn:        settings.SpawnTerminal,
			TicketPrompt: "read todo {{todo-hash}}",
		},
	}
	const uuid = "019be00a-5f3a-7abc-8000-abc123456789"
	line := c.claudeTerminalCommandLine("/tmp/proj", "abcd1234", "My Task", uuid, "abcd1234-my-task")
	if !strings.Contains(line, "cd /tmp/proj &&") {
		t.Fatalf("missing cd: %s", line)
	}
	if !strings.Contains(line, `MHTODO_SESSION="`+uuid+`"`) {
		t.Fatalf("missing session env: %s", line)
	}
	if !strings.Contains(line, `MHTODO_SESSION_NAME="abcd1234-my-task"`) {
		t.Fatalf("missing session name env: %s", line)
	}
	if !strings.Contains(line, "--session-id "+uuid) {
		t.Fatalf("missing session-id: %s", line)
	}
	if !strings.Contains(line, "--resume "+uuid) {
		t.Fatalf("missing resume: %s", line)
	}
	if !strings.Contains(line, "--name abcd1234-my-task") {
		t.Fatalf("missing name: %s", line)
	}
	if !strings.Contains(line, `"read todo abcd1234"`) {
		t.Fatalf("missing prompt: %s", line)
	}
	createIdx := strings.Index(line, "--session-id")
	resumeIdx := strings.Index(line, "--resume")
	if createIdx < 0 || resumeIdx < 0 || createIdx > resumeIdx {
		t.Fatalf("expected session-id before resume: %s", line)
	}
}

func TestClaudeTerminalCommandLineEnvStart(t *testing.T) {
	c := Client{
		Claude: settings.ClaudeConfig{
			IntegrationConfig: settings.IntegrationConfig{
				Binary:   "claude",
				EnvStart: "FOO=bar",
			},
			Spawn: settings.SpawnTerminal,
		},
		Terminal: settings.TerminalConfig{EnvStart: "TERM_X=1"},
	}
	line := c.claudeTerminalCommandLine("", "abcd1234", "T", "sess", "sess")
	if !strings.Contains(line, "FOO=bar") || !strings.Contains(line, "TERM_X=1") {
		t.Fatalf("env missing: %s", line)
	}
}

func TestMaybeCloseTerminalSessionOnDone(t *testing.T) {
	c := Client{Claude: settings.ClaudeConfig{Spawn: settings.SpawnTerminal, CloseTabOnDone: true}}
	if !c.MaybeCloseTerminalSessionOnDone(0) {
		t.Fatal("expected clear for zero pid")
	}
	c.Claude.CloseTabOnDone = false
	if c.MaybeCloseTerminalSessionOnDone(123) {
		t.Fatal("should not clear when close disabled")
	}
}

func TestClaudeLaunchShell(t *testing.T) {
	got := ClaudeLaunchShell("/usr/bin/claude", "019be00a-5f3a-7abc-8000-abc123456789", "abcd-name", "hi there")
	if !strings.Contains(got, "--session-id 019be00a-5f3a-7abc-8000-abc123456789") {
		t.Fatalf("missing session-id: %s", got)
	}
	if !strings.Contains(got, " || ") {
		t.Fatalf("missing fallback: %s", got)
	}
	if !strings.Contains(got, "--resume 019be00a-5f3a-7abc-8000-abc123456789") {
		t.Fatalf("missing resume: %s", got)
	}
	if !strings.Contains(got, `--name abcd-name`) {
		t.Fatalf("missing name: %s", got)
	}
}
