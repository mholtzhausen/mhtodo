package integrations

import (
	"strings"
	"testing"

	"mhtodo/internal/settings"
)

func TestClaudeTerminalCommandLineResumeAndName(t *testing.T) {
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
	line := c.claudeTerminalCommandLine("/tmp/proj", "abcd1234", "My Task", "abcd1234-my-task")
	if !strings.Contains(line, "cd /tmp/proj &&") {
		t.Fatalf("missing cd: %s", line)
	}
	if !strings.Contains(line, "MHTODO_SESSION=abcd1234-my-task") {
		t.Fatalf("missing session env: %s", line)
	}
	if !strings.Contains(line, "--resume abcd1234-my-task") {
		t.Fatalf("missing resume: %s", line)
	}
	if !strings.Contains(line, "--name abcd1234-my-task") {
		t.Fatalf("missing name fallback: %s", line)
	}
	if !strings.Contains(line, `"read todo abcd1234"`) {
		t.Fatalf("missing prompt: %s", line)
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
	line := c.claudeTerminalCommandLine("", "abcd1234", "T", "sess")
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
