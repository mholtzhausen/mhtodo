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
	got := c.TicketCommand("/home/me/proj", "81abc903", "Hello World", "", "")
	want := `FOO="bar" MHTODO_SESSION="81abc903-hello-world" MHTODO_SESSION_NAME="81abc903-hello-world" /usr/bin/zed --wait /home/me/proj`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestZedTicketCommandExplicitSessionAndSpaces(t *testing.T) {
	t.Parallel()
	c := ZedClient{Zed: settings.IntegrationConfig{Binary: "zed"}}
	got := c.TicketCommand(`/tmp/my proj`, "81abc903", "x", "custom-session", "custom-session")
	want := `MHTODO_SESSION="custom-session" MHTODO_SESSION_NAME="custom-session" zed "/tmp/my proj"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestZedTicketCommandUUIDSession(t *testing.T) {
	t.Parallel()
	c := ZedClient{Zed: settings.IntegrationConfig{Binary: "zed"}}
	const uuid = "019be00a-5f3a-7abc-8000-abc123456789"
	got := c.TicketCommand("/tmp/p", "81abc903", "Hello", uuid, "81abc903-hello")
	want := `MHTODO_SESSION="` + uuid + `" MHTODO_SESSION_NAME="81abc903-hello" zed /tmp/p`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
