package integrations

import (
	"testing"

	"mhtodo/internal/platform"
	"mhtodo/internal/settings"
)

func TestZedTicketCommand(t *testing.T) {
	t.Parallel()
	platform.SetUseSystemdScopeForTest(func() bool { return false })
	t.Cleanup(func() { platform.SetUseSystemdScopeForTest(nil) })
	c := ZedClient{Zed: settings.IntegrationConfig{
		Binary:   "/usr/bin/zed",
		EnvStart: `FOO=bar --wait`,
	}}
	got := c.TicketCommand("/home/me/proj")
	want := `FOO="bar" /usr/bin/zed --wait /home/me/proj`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestZedTicketCommandSpaces(t *testing.T) {
	t.Parallel()
	platform.SetUseSystemdScopeForTest(func() bool { return false })
	t.Cleanup(func() { platform.SetUseSystemdScopeForTest(nil) })
	c := ZedClient{Zed: settings.IntegrationConfig{Binary: "zed"}}
	got := c.TicketCommand(`/tmp/my proj`)
	want := `zed "/tmp/my proj"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
