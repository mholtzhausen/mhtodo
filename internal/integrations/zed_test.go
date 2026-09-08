package integrations

import (
	"testing"

	"mhtodo/internal/settings"
)

func TestZedTicketCommand(t *testing.T) {
	t.Parallel()
	c := ZedClient{Zed: settings.IntegrationConfig{
		Binary:   "/usr/bin/zed",
		EnvStart: `FOO=bar --wait`,
	}}
	got := c.TicketCommand("/home/me/proj", "81abc903", "Hello World", "")
	want := `FOO="bar" MHTODO_SESSION="81abc903-hello-world" /usr/bin/zed --wait /home/me/proj`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestZedTicketCommandExplicitSessionAndSpaces(t *testing.T) {
	t.Parallel()
	c := ZedClient{Zed: settings.IntegrationConfig{Binary: "zed"}}
	got := c.TicketCommand(`/tmp/my proj`, "81abc903", "x", "custom-session")
	want := `MHTODO_SESSION="custom-session" zed "/tmp/my proj"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
