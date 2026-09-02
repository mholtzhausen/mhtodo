package integrations

import "testing"

func TestParseEnvStart(t *testing.T) {
	t.Parallel()
	env, args := ParseEnvStart(`FOO=bar BAZ=qux --session work`)
	if len(env) != 2 || env[0] != "FOO=bar" || env[1] != "BAZ=qux" {
		t.Fatalf("env = %#v", env)
	}
	if len(args) != 2 || args[0] != "--session" || args[1] != "work" {
		t.Fatalf("args = %#v", args)
	}
}

func TestShellDoubleQuote(t *testing.T) {
	t.Parallel()
	if got := ShellDoubleQuote(`read todo abc and start`); got != `"read todo abc and start"` {
		t.Fatalf("got %q", got)
	}
	if got := ShellDoubleQuote(`say "hi"`); got != `"say \"hi\""` {
		t.Fatalf("got %q", got)
	}
}

func TestTicketTabLabel(t *testing.T) {
	t.Parallel()
	got := ticketTabLabel("019be00a-5f3a-7abc-8000-abc123456789", "81abc903", "Short title")
	want := "81abc903 - Short title"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	long := ticketTabLabel("ignored", "81abc903", "This is a very long ticket title that exceeds forty characters easily")
	if len([]rune(long)) > len("81abc903 - ")+40 {
		t.Fatalf("title not truncated: %q", long)
	}
}

func TestTabMatchesTicket(t *testing.T) {
	t.Parallel()
	taskA := "019be00a-5f3a-7abc-8000-abc123456789"
	taskB := "019be00a-5f3a-7abc-8000-abc123456790"
	shortA := "abc12345"
	shortB := "abc12346"

	tabA := ticketTabLabel(taskA, shortA, "Fix the bug")
	if !tabMatchesTicket(tabA, taskA, shortA) {
		t.Fatal("expected short id match")
	}
	if tabMatchesTicket(tabA, taskB, shortB) {
		t.Fatal("different short ids must not match")
	}
	legacy := taskA + "|" + tabA
	if !tabMatchesTicket(legacy, taskA, shortA) {
		t.Fatal("expected legacy embedded task id match")
	}
	if !tabMatchesTicket("abcd1234 - Fix the bug", taskA, "abcd1234") {
		t.Fatal("expected prefix match")
	}
	if !tabMatchesTicket("abcd1234", taskA, "abcd1234") {
		t.Fatal("expected exact match")
	}
	if tabMatchesTicket("abcd1234 - Fix", taskA, "abcd123") {
		t.Fatal("should not match shorter prefix")
	}
	if tabMatchesTicket("xyz - abcd1234", taskA, "abcd1234") {
		t.Fatal("should not match suffix")
	}
}
