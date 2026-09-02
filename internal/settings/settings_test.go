package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"mhtodo/internal/store"
)

func TestLoadSaveYAMLRoundTrip(t *testing.T) {
	path := configPathIn(t, "config.yml")

	want := Default()
	want.DefaultCwd = "/tmp/proj"
	want.DefaultHumanOnly = true
	want.ArchiveDoneSubtasks = true
	want.Claude.Enabled = true
	want.Claude.Binary = "/usr/bin/claude"
	want.Claude.EnvStart = "ANTHROPIC_API_KEY=..."
	want.Claude.TicketPrompt = "read todo {{todo-hash}}"
	want.Herdr.Enabled = true
	want.Herdr.Binary = "/usr/bin/herdr"
	want.Herdr.EnvStart = "HERDR=1"
	want.Herdr.SpaceName = "my-space"

	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("round-trip mismatch:\nwant %+v\ngot  %+v", want, got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "default_cwd:") {
		t.Fatalf("expected yaml config at %s, got:\n%s", path, data)
	}
}

func TestLoadMissingAutodetectsAndWritesConfig(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	claude := filepath.Join(bin, "claude")
	if err := os.WriteFile(claude, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "config.yml")
	t.Setenv("MHTODO_CONFIG_PATH", path)
	t.Setenv("PATH", bin)

	got, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Claude.Enabled {
		t.Error("expected claude enabled after autodetect")
	}
	if got.Claude.Binary != claude {
		t.Errorf("claude binary = %q, want %q", got.Claude.Binary, claude)
	}
	if got.Herdr.Enabled {
		t.Error("herdr should stay disabled when not on PATH")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cf configFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		t.Fatal(err)
	}
	if cf.Claude.UserSet {
		t.Error("autodetected integration should not be marked user_set")
	}
}

func TestAutodetectSkipsUserSetIntegrations(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	claude := filepath.Join(bin, "claude")
	if err := os.WriteFile(claude, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "config.yml")
	t.Setenv("MHTODO_CONFIG_PATH", path)
	t.Setenv("PATH", bin)

	s := Default()
	s.Claude.Enabled = false
	s.Claude.Binary = "disabled-by-user"
	if err := Save(s); err != nil {
		t.Fatal(err)
	}

	got, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Claude.Enabled {
		t.Error("user-set claude integration should not be auto-enabled")
	}
	if got.Claude.Binary != "disabled-by-user" {
		t.Errorf("binary = %q, want disabled-by-user", got.Claude.Binary)
	}
}

func TestMigrateFromMeta(t *testing.T) {
	repo := openTestRepo(t)
	ctx := t.Context()
	legacy := Default()
	legacy.DefaultCwd = "/legacy"
	if err := repo.SetMeta(ctx, MetaGUISettings, `{"default_cwd":"/legacy"}`); err != nil {
		t.Fatal(err)
	}

	path := configPathIn(t, "config.yml")
	got, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got.DefaultCwd != "/legacy" {
		t.Fatalf("default_cwd = %q, want /legacy", got.DefaultCwd)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected migrated config at %s: %v", path, err)
	}
}

func TestBinaryFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	exe := filepath.Join(dir, "tool")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	noExec := filepath.Join(dir, "plain")
	if err := os.WriteFile(noExec, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !BinaryFound(exe) {
		t.Error("expected executable file to be found")
	}
	if BinaryFound(noExec) {
		t.Error("expected non-executable file to be missing")
	}
	if BinaryFound("") {
		t.Error("empty path should be false")
	}
	if !BinaryFound("sh") {
		t.Error("expected sh on PATH")
	}
}

func TestExpandBinaryNameOnLoad(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	claude := filepath.Join(bin, "claude")
	if err := os.WriteFile(claude, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "config.yml")
	t.Setenv("MHTODO_CONFIG_PATH", path)
	t.Setenv("PATH", bin)

	if err := os.WriteFile(path, []byte("claude:\n  enabled: true\n  binary: claude\nherdr:\n  binary: herdr\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Claude.Binary != claude {
		t.Errorf("claude binary = %q, want full path %q", got.Claude.Binary, claude)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), claude) {
		t.Fatalf("expected expanded path in config:\n%s", data)
	}
}

func TestOptionalIntegrationFieldsOmittedFromYAML(t *testing.T) {
	path := configPathIn(t, "config.yml")
	s := Default()
	if err := Save(s); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, key := range []string{"ticket_prompt:", "env_start:", "space_name:"} {
		if strings.Contains(body, key) {
			t.Errorf("expected empty optional field omitted from config, found %q in:\n%s", key, body)
		}
	}
}

func TestEffectiveIntegrationDefaults(t *testing.T) {
	var c ClaudeConfig
	if got := c.EffectiveTicketPrompt(); got != DefaultClaudeTicketPrompt {
		t.Fatalf("EffectiveTicketPrompt = %q, want default", got)
	}
	var h HerdrConfig
	if got := h.EffectiveSpaceName(); got != DefaultHerdrSpaceName {
		t.Fatalf("EffectiveSpaceName = %q, want %q", got, DefaultHerdrSpaceName)
	}
}

func TestStripStoredIntegrationDefaults(t *testing.T) {
	cf := configFile{
		Claude: claudeFile{TicketPrompt: DefaultClaudeTicketPrompt},
		Herdr:  herdrFile{SpaceName: DefaultHerdrSpaceName},
	}
	stripStoredIntegrationDefaults(&cf)
	if cf.Claude.TicketPrompt != "" || cf.Herdr.SpaceName != "" {
		t.Fatalf("stripStoredIntegrationDefaults: %+v", cf)
	}
}

func TestResolveBinary(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "herdr")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	p, ok := resolveBinary("herdr")
	if !ok || p != exe {
		t.Fatalf("resolveBinary = (%q, %v), want (%q, true)", p, ok, exe)
	}
}

func configPathIn(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	t.Setenv("MHTODO_CONFIG_PATH", path)
	return path
}

func openTestRepo(t *testing.T) *store.TaskRepo {
	t.Helper()
	db := filepath.Join(t.TempDir(), "test.db")
	repo, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}
