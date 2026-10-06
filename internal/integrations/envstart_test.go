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

func TestShellWord(t *testing.T) {
	t.Parallel()
	if got := shellWord("plain"); got != "plain" {
		t.Fatalf("got %q", got)
	}
	if got := shellWord("/tmp/my proj"); got != `"/tmp/my proj"` {
		t.Fatalf("got %q", got)
	}
}
