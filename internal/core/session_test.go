package core

import "testing"

func TestDefaultTodoSession(t *testing.T) {
	t.Parallel()
	got := DefaultTodoSession("81abc903", "Short title")
	want := "81abc903-short-title"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if DefaultTodoSession("81abc903", "") != "81abc903" {
		t.Fatal("empty title should return short id only")
	}
	if DefaultTodoSession("81abc903", "Hello, World!") != "81abc903-hello-world" {
		t.Fatalf("punct slug = %q", DefaultTodoSession("81abc903", "Hello, World!"))
	}
	long := DefaultTodoSession("81abc903", "This is a very long ticket title that exceeds forty characters easily")
	prefix := "81abc903-"
	if !stringsHasPrefix(long, prefix) {
		t.Fatalf("missing prefix: %q", long)
	}
	slug := long[len(prefix):]
	if stringsContainsSpace(slug) {
		t.Fatalf("slug has spaces: %q", slug)
	}
	if runeCount(slug) > 40 {
		t.Fatalf("slug runes = %d > 40 (%q)", runeCount(slug), slug)
	}
}

func TestLegacyTodoSession(t *testing.T) {
	t.Parallel()
	if LegacyTodoSession("81abc903", "Short title") != "81abc903 - Short title" {
		t.Fatal("legacy format mismatch")
	}
}

func TestLooksLikeSessionUUID(t *testing.T) {
	t.Parallel()
	if !LooksLikeSessionUUID("019be00a-5f3a-7abc-8000-abc123456789") {
		t.Fatal("expected uuid")
	}
	if LooksLikeSessionUUID("81abc903-short-title") {
		t.Fatal("slug should not look like uuid")
	}
	if LooksLikeSessionUUID("") {
		t.Fatal("empty")
	}
}

func TestClaudeDisplayName(t *testing.T) {
	t.Parallel()
	if got := ClaudeDisplayName("81abc903", "Hello", "019be00a-5f3a-7abc-8000-abc123456789"); got != "81abc903-hello" {
		t.Fatalf("uuid session display = %q", got)
	}
	if got := ClaudeDisplayName("81abc903", "Hello", "custom-name"); got != "custom-name" {
		t.Fatalf("legacy slug display = %q", got)
	}
	if got := ClaudeDisplayName("81abc903", "Hello", ""); got != "81abc903-hello" {
		t.Fatalf("empty session display = %q", got)
	}
}

func TestEnsureClaudeSessionID(t *testing.T) {
	t.Parallel()
	const existing = "019be00a-5f3a-7abc-8000-abc123456789"
	got, gen, err := EnsureClaudeSessionID(existing)
	if err != nil || gen || got != existing {
		t.Fatalf("uuid passthrough: got=%q gen=%v err=%v", got, gen, err)
	}
	got, gen, err = EnsureClaudeSessionID("legacy-slug")
	if err != nil || !gen || !LooksLikeSessionUUID(got) {
		t.Fatalf("legacy mint: got=%q gen=%v err=%v", got, gen, err)
	}
	got, gen, err = EnsureClaudeSessionID("")
	if err != nil || !gen || !LooksLikeSessionUUID(got) {
		t.Fatalf("empty mint: got=%q gen=%v err=%v", got, gen, err)
	}
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}

func stringsContainsSpace(s string) bool {
	for _, r := range s {
		if r == ' ' {
			return true
		}
	}
	return false
}

func runeCount(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
