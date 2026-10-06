package aiskill_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mhtodo/internal/aiskill"
)

func useTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	restore := aiskill.HomeDirForTest(func() (string, error) { return home, nil })
	t.Cleanup(restore)
	return home
}

func TestInstallAndCheck(t *testing.T) {
	home := useTempHome(t)

	res, err := aiskill.Install()
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != "installed" {
		t.Fatalf("action = %q", res.Action)
	}
	wantPath := filepath.Join(home, ".claude", "skills", "mhtodo", "SKILL.md")
	if res.Path != wantPath {
		t.Fatalf("path = %q want %q", res.Path, wantPath)
	}
	raw, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Hard rule") || !strings.Contains(string(raw), "mhtodo") {
		t.Fatalf("skill content missing workflows")
	}

	res2, err := aiskill.Install()
	if err != nil {
		t.Fatal(err)
	}
	if res2.Action != "updated" {
		t.Fatalf("second install action = %q", res2.Action)
	}

	chk, err := aiskill.Check()
	if err != nil {
		t.Fatal(err)
	}
	if !chk.SkillPresent || chk.SkillStale || len(chk.Leftovers) != 0 {
		t.Fatalf("check: %+v", chk)
	}
}

func TestStripLegacy(t *testing.T) {
	home := useTempHome(t)

	hooks := filepath.Join(home, ".claude", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	hookFile := filepath.Join(hooks, "mhtodo-reminder.sh")
	if err := os.WriteFile(hookFile, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(home, ".claude", "settings.json")
	cfg := map[string]any{
		"hooks": map[string]any{
			"UserPromptSubmit": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{"type": "command", "command": "~/.claude/hooks/mhtodo-reminder.sh"},
					},
				},
				map[string]any{
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo keep-me"},
					},
				},
			},
			"Stop": []any{
				map[string]any{"type": "command", "command": "~/.claude/hooks/mhtodo-stop.sh"},
			},
		},
		"other": "keep",
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(settings, b, 0o644); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(home, ".local", "state", "mhtodo-agent")
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "integration.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "abc-session"), []byte("task-id\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	todoHelper := filepath.Join(bin, "claude.todo")
	if err := os.WriteFile(todoHelper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := aiskill.Install()
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(res.Removed, "\n")
	for _, want := range []string{hookFile, "settings.json", "integration.json", "abc-session", "claude.todo"} {
		if !strings.Contains(joined, want) {
			t.Errorf("removed missing %q; got %v", want, res.Removed)
		}
	}
	if _, err := os.Stat(hookFile); !os.IsNotExist(err) {
		t.Errorf("hook still present")
	}
	raw, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "mhtodo") {
		t.Errorf("settings still mention mhtodo: %s", raw)
	}
	if !strings.Contains(string(raw), "keep-me") || !strings.Contains(string(raw), `"other"`) {
		t.Errorf("settings lost unrelated config: %s", raw)
	}
	if _, err := os.Stat(todoHelper); !os.IsNotExist(err) {
		t.Errorf("claude.todo still present")
	}
}

func TestRefreshSkippedWithoutClaude(t *testing.T) {
	restore := aiskill.LookPathForTest(func(string) (string, error) {
		return "", os.ErrNotExist
	})
	t.Cleanup(restore)
	out, skipped, err := aiskill.RefreshClaude(0)
	if err != nil || !skipped || out != "" {
		t.Fatalf("out=%q skipped=%v err=%v", out, skipped, err)
	}
}

func TestRefreshSkippedEnv(t *testing.T) {
	t.Setenv("MHTODO_SKIP_SKILL_REFRESH", "1")
	restore := aiskill.LookPathForTest(func(string) (string, error) {
		return "/usr/bin/claude", nil
	})
	t.Cleanup(restore)
	_, skipped, err := aiskill.RefreshClaude(0)
	if err != nil || !skipped {
		t.Fatalf("skipped=%v err=%v", skipped, err)
	}
}
