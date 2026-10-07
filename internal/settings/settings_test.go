package settings

import (
	"os"
	"path/filepath"
	"reflect"
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
	want.DefaultIncludeInReport = false
	want.ArchiveDoneSubtasks = true
	want.StartHidden = true
	want.Notifications.TrayLabelStatuses = []string{"waiting"}
	want.Notifications.TrayMenuStatuses = []string{"waiting", "review", "wip"}
	want.Notifications.MaxItemsPerStatus = 7
	want.Notifications.NotifySendWIP = true
	want.Notifications.NotifySendWaiting = false
	want.Notifications.NotifySendReview = false
	want.Notifications.NotifySendDone = true
	want.Notifications.PanelAttnStatuses = []string{"wip", "review"}
	want.Notifications.PanelAttnIntervalSec = 8
	want.Notifications.PanelAttnIntensity = 70
	want.Zed.Enabled = false
	want.Zed.Binary = "/usr/bin/zed"
	want.Zed.EnvStart = "FOO=1"

	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch:\nwant %+v\ngot  %+v", want, got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "default_cwd:") {
		t.Fatalf("expected yaml config at %s, got:\n%s", path, data)
	}
	if !strings.Contains(string(data), "notifications:") {
		t.Fatalf("expected notifications section in yaml:\n%s", data)
	}
}

func TestNotificationsDefaultsWhenAbsent(t *testing.T) {
	path := configPathIn(t, "config.yml")
	if err := os.WriteFile(path, []byte("start_hidden: false\nzed:\n  binary: zed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	def := Default().Notifications
	if !reflect.DeepEqual(got.Notifications.TrayLabelStatuses, def.TrayLabelStatuses) {
		t.Fatalf("label statuses = %v, want %v", got.Notifications.TrayLabelStatuses, def.TrayLabelStatuses)
	}
	if got.Notifications.MaxItemsPerStatus != 10 {
		t.Fatalf("max items = %d", got.Notifications.MaxItemsPerStatus)
	}
	if got.Notifications.NotifySendWaiting || got.Notifications.NotifySendDone || got.Notifications.NotifySendWIP {
		t.Fatal("notify-send wip/waiting/done defaults should be false")
	}
	if !got.Notifications.NotifySendReview {
		t.Fatal("notify-send review default should be true")
	}
	defPanel := Default().Notifications.PanelAttnStatuses
	if !reflect.DeepEqual(got.Notifications.PanelAttnStatuses, defPanel) {
		t.Fatalf("panel attn statuses = %v, want %v", got.Notifications.PanelAttnStatuses, defPanel)
	}
	if got.Notifications.PanelAttnIntervalSec != 5 {
		t.Fatalf("panel attn interval = %d, want 5", got.Notifications.PanelAttnIntervalSec)
	}
	if got.Notifications.PanelAttnIntensity != 40 {
		t.Fatalf("panel attn intensity = %d, want 40", got.Notifications.PanelAttnIntensity)
	}
}

func TestPanelAttnEmptyListRoundTrip(t *testing.T) {
	_ = configPathIn(t, "config.yml")
	want := Default()
	want.Notifications.PanelAttnStatuses = []string{}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Notifications.PanelAttnStatuses) != 0 {
		t.Fatalf("empty panel attn should round-trip, got %v", got.Notifications.PanelAttnStatuses)
	}
}

func TestLoadMissingAutodetectsAndWritesConfig(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	zed := filepath.Join(bin, "zed")
	if err := os.WriteFile(zed, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "config.yml")
	t.Setenv("MHTODO_CONFIG_PATH", path)
	t.Setenv("PATH", bin)

	got, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Zed.Enabled {
		t.Error("expected zed enabled after autodetect")
	}
	if got.Zed.Binary != zed {
		t.Errorf("zed binary = %q, want %q", got.Zed.Binary, zed)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cf configFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		t.Fatal(err)
	}
	if cf.Zed.UserSet {
		t.Error("autodetected integration should not be marked user_set")
	}
}

func TestAutodetectSkipsUserSetIntegrations(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	zed := filepath.Join(bin, "zed")
	if err := os.WriteFile(zed, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "config.yml")
	t.Setenv("MHTODO_CONFIG_PATH", path)
	t.Setenv("PATH", bin)

	s := Default()
	s.Zed.Enabled = false
	s.Zed.Binary = "disabled-by-user"
	if err := Save(s); err != nil {
		t.Fatal(err)
	}

	got, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Zed.Enabled {
		t.Error("user-set zed integration should not be auto-enabled")
	}
	if got.Zed.Binary != "disabled-by-user" {
		t.Errorf("binary = %q, want disabled-by-user", got.Zed.Binary)
	}
}

func TestMigrateFromMeta(t *testing.T) {
	repo := openTestRepo(t)
	ctx := t.Context()
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
	zed := filepath.Join(bin, "zed")
	if err := os.WriteFile(zed, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "config.yml")
	t.Setenv("MHTODO_CONFIG_PATH", path)
	t.Setenv("PATH", bin)

	if err := os.WriteFile(path, []byte("zed:\n  enabled: true\n  binary: zed\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Zed.Binary != zed {
		t.Errorf("zed binary = %q, want full path %q", got.Zed.Binary, zed)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), zed) {
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
	for _, key := range []string{"env_start:", "claude:", "herdr:", "terminal:"} {
		if strings.Contains(body, key) {
			t.Errorf("expected field omitted from config, found %q in:\n%s", key, body)
		}
	}
}

func TestResolveBinary(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "zed")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	p, ok := resolveBinary("zed")
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
