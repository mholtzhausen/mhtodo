package cli_test

import (
	"strings"
	"testing"

	"mhtodo/internal/core"
)

func TestSlackLinkCLI(t *testing.T) {
	out, errb, run := newCLI(t)
	const (
		u1 = "https://acme.slack.com/archives/C1/p1"
		u2 = "https://acme.slack.com/archives/C2"
	)
	var tk core.Task
	if code := run("add", "T", "--slack-thread", u1, "--json"); code != 0 {
		t.Fatalf("add exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &tk)
	if tk.SlackThread != u1 || len(tk.SlackLinks) != 1 {
		t.Fatalf("add: %+v", tk)
	}
	if !strings.Contains(out.String(), `"slack_links": [`) {
		t.Fatalf("json lacks slack_links: %s", out.String())
	}

	out.Reset()
	if code := run("slack", "link", "add", tk.ID, u2, "--desc", "incident channel", "--json"); code != 0 {
		t.Fatalf("link add exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &tk)
	if len(tk.SlackLinks) != 2 || tk.SlackThread != u1 || tk.SlackLinks[1].Description != "incident channel" {
		t.Fatalf("link add: %+v", tk.SlackLinks)
	}

	out.Reset()
	if code := run("slack", "link", "list", tk.ID); code != 0 {
		t.Fatalf("list exit %d", code)
	}
	if lines := strings.Split(strings.TrimSpace(out.String()), "\n"); len(lines) != 2 ||
		!strings.HasPrefix(lines[0], "1 *") || !strings.Contains(lines[1], "incident channel") {
		t.Fatalf("list output:\n%s", out.String())
	}

	out.Reset()
	if code := run("slack", "link", "edit", tk.ID, "2", "--primary", "--json"); code != 0 {
		t.Fatalf("edit exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &tk)
	if tk.SlackThread != u2 {
		t.Fatalf("edit --primary: slack_thread = %q", tk.SlackThread)
	}

	out.Reset()
	if code := run("slack", "link", "rm", tk.ID, u2, "--json"); code != 0 {
		t.Fatalf("rm exit %d", code)
	}
	mustJSON(t, out.Bytes(), &tk)
	if tk.SlackThread != u1 || len(tk.SlackLinks) != 1 {
		t.Fatalf("rm: %+v", tk.SlackLinks)
	}

	// Errors: bad URL exits 1, unknown link exits 2, no edit fields exits 1.
	for _, c := range []struct {
		args []string
		code int
		name string
	}{
		{[]string{"slack", "link", "add", tk.ID, "https://example.com/x"}, 1, "invalid_slack_link"},
		{[]string{"slack", "link", "rm", tk.ID, "5"}, 2, "slack_link_not_found"},
		{[]string{"slack", "link", "edit", tk.ID, "1"}, 1, "no_fields"},
	} {
		errb.Reset()
		if code := run(append(c.args, "--json")...); code != c.code || !strings.Contains(errb.String(), c.name) {
			t.Errorf("%v: exit %d stderr %q, want %d %s", c.args, code, errb.String(), c.code, c.name)
		}
	}
}
