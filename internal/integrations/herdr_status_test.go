package integrations

import (
	"testing"

	"mhtodo/internal/settings"
)

func TestHerdrCommandLine(t *testing.T) {
	t.Parallel()
	c := Client{Herdr: settings.HerdrConfig{Binary: "/usr/bin/herdr", EnvStart: "HERDR=1"}}
	got := c.herdrCommandLine("session", "attach", "default")
	want := `HERDR=1 /usr/bin/herdr session attach default`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestServerRunningJSON(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"status":"running","running":true,"socket":"/tmp/herdr.sock"}`)
	running, err := parseServerRunning(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !running {
		t.Fatal("expected running")
	}
}

func TestServerRunningJSONNotRunning(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"status":"not_running","running":false,"socket":"/tmp/herdr.sock"}`)
	running, err := parseServerRunning(raw)
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("expected not running")
	}
}
