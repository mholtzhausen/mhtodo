package cli_test

import (
	"strings"
	"testing"

	"mhtodo/internal/core"
)

func TestPriorityCLI(t *testing.T) {
	out, errb, run := newCLI(t)

	var a, b core.Task
	if code := run("add", "Alpha", "--json"); code != 0 {
		t.Fatalf("add exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &a)
	if a.Priority != core.PriorityNone {
		t.Fatalf("default priority = %q", a.Priority)
	}
	if !strings.Contains(out.String(), `"priority": "none"`) {
		t.Fatalf("json lacks priority field: %s", out.String())
	}

	out.Reset()
	if code := run("add", "Bravo", "--priority", "urgent", "--json"); code != 0 {
		t.Fatalf("add --priority exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &b)
	if b.Priority != core.PriorityUrgent {
		t.Fatalf("add priority = %q", b.Priority)
	}

	// priority ID LEVEL sets; priority ID shows.
	out.Reset()
	if code := run("priority", a.ID, "high", "--json"); code != 0 {
		t.Fatalf("priority set exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &a)
	if a.Priority != core.PriorityHigh {
		t.Fatalf("set priority = %q", a.Priority)
	}
	out.Reset()
	if code := run("priority", a.ID); code != 0 || strings.TrimSpace(out.String()) != "high" {
		t.Fatalf("priority show exit %d out %q", code, out.String())
	}

	// edit --priority.
	out.Reset()
	if code := run("edit", a.ID, "--priority", "low", "--json"); code != 0 {
		t.Fatalf("edit --priority exit %d", code)
	}
	mustJSON(t, out.Bytes(), &a)
	if a.Priority != core.PriorityLow {
		t.Fatalf("edit priority = %q", a.Priority)
	}

	// Invalid values exit 1 with invalid_priority.
	for _, args := range [][]string{
		{"priority", a.ID, "huge"},
		{"edit", a.ID, "--priority", "huge"},
		{"add", "x", "--priority", "huge"},
		{"list", "--priority", ">=nope"},
	} {
		errb.Reset()
		if code := run(append(args, "--json")...); code != 1 {
			t.Fatalf("%v exit %d, want 1", args, code)
		}
		if !strings.Contains(errb.String(), "invalid_priority") {
			t.Fatalf("%v stderr %q lacks invalid_priority", args, errb.String())
		}
	}

	// list filters: exact, comma, threshold.
	listTitles := func(args ...string) string {
		t.Helper()
		out.Reset()
		if code := run(append([]string{"list", "--json"}, args...)...); code != 0 {
			t.Fatalf("list %v exit %d: %s", args, code, errb.String())
		}
		var ts []core.Task
		mustJSON(t, out.Bytes(), &ts)
		var names []string
		for _, x := range ts {
			names = append(names, x.Title)
		}
		return strings.Join(names, ",")
	}
	if got := listTitles("--priority", "urgent"); got != "Bravo" {
		t.Fatalf("--priority urgent = %q", got)
	}
	if got := listTitles("--priority", "low,urgent"); got != "Alpha,Bravo" {
		t.Fatalf("--priority low,urgent = %q", got)
	}
	if got := listTitles("--priority", ">=medium"); got != "Bravo" {
		t.Fatalf("--priority >=medium = %q", got)
	}
	// Default board order is untouched; --sort priority is opt-in.
	if got := listTitles(); got != "Alpha,Bravo" {
		t.Fatalf("board order = %q", got)
	}
	if got := listTitles("--sort", "priority"); got != "Bravo,Alpha" {
		t.Fatalf("--sort priority = %q", got)
	}

	// Template preset + explicit flag override + clear.
	if code := run("template", "create", "incident", "--priority", "urgent"); code != 0 {
		t.Fatalf("template create exit %d: %s", code, errb.String())
	}
	var c core.Task
	out.Reset()
	if code := run("add", "Outage", "--template", "incident", "--json"); code != 0 {
		t.Fatalf("add --template exit %d", code)
	}
	mustJSON(t, out.Bytes(), &c)
	if c.Priority != core.PriorityUrgent {
		t.Fatalf("templated priority = %q", c.Priority)
	}
	out.Reset()
	if code := run("add", "Minor", "--template", "incident", "--priority", "low", "--json"); code != 0 {
		t.Fatalf("add override exit %d", code)
	}
	mustJSON(t, out.Bytes(), &c)
	if c.Priority != core.PriorityLow {
		t.Fatalf("override priority = %q", c.Priority)
	}
	if code := run("template", "update", "incident", "--priority", "high", "--clear-priority"); code != 1 {
		t.Fatalf("set+clear exit %d, want 1", code)
	}
	var tpl core.Template
	out.Reset()
	if code := run("template", "update", "incident", "--clear-priority", "--json"); code != 0 {
		t.Fatalf("clear exit %d", code)
	}
	mustJSON(t, out.Bytes(), &tpl)
	if tpl.Priority != nil {
		t.Fatalf("cleared template priority = %v", *tpl.Priority)
	}
}
