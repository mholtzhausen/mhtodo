package core_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mhtodo/internal/core"
)

func TestDescribeSlackLink(t *testing.T) {
	cases := []struct {
		in, kind, ws string
		valid        bool
	}{
		{"https://acme.slack.com/archives/C0123ABC/p1712345678901234", "thread", "acme", true},
		{"https://acme.slack.com/archives/C0123ABC/p1712345678901234?thread_ts=1712345600.000100&cid=C0123ABC", "thread", "acme", true},
		{"https://acme.slack.com/archives/C0123ABC", "channel", "acme", true},
		{"https://acme.slack.com/archives/D0123ABC", "dm", "acme", true},
		{"https://app.slack.com/client/T0001/C0123ABC", "channel", "T0001", true},
		{"https://app.slack.com/client/T0001/C0123ABC/thread/C0123ABC-1712345600.000100", "thread", "T0001", true},
		{"slack://channel?team=T0001&id=C0123ABC", "channel", "T0001", true},
		{"https://example.com/archives/C1/p1", "", "", false},
		{"not a url at all", "", "", false},
		{"", "", "", false},
	}
	for _, c := range cases {
		got := core.DescribeSlackLink(c.in)
		if got.Valid != c.valid || (c.valid && (got.Kind != c.kind || got.Workspace != c.ws)) {
			t.Errorf("%q = %+v, want valid=%v kind=%q ws=%q", c.in, got, c.valid, c.kind, c.ws)
		}
		if c.valid && got.Label == "" {
			t.Errorf("%q: empty label", c.in)
		}
	}
}

func TestNormalizeSlackLinks(t *testing.T) {
	got := core.NormalizeSlackLinks([]core.SlackLink{
		{URL: " https://acme.slack.com/archives/C1/p1 "},
		{URL: ""},
		{URL: "https://ACME.slack.com/archives/C1/p1/", Description: "dup fills"},
		{URL: "https://acme.slack.com/archives/C2", Description: "  line\none "},
	})
	if len(got) != 2 || got[0].Description != "dup fills" || got[1].Description != "line one" {
		t.Fatalf("normalize = %+v", got)
	}
	if core.NormalizeSlackLinks(nil) == nil {
		t.Fatal("want non-nil empty slice")
	}
}

func TestSlackLinksLifecycle(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	const (
		u1 = "https://acme.slack.com/archives/C1/p1"
		u2 = "https://acme.slack.com/archives/C2"
		u3 = "https://acme.slack.com/archives/C3/p3"
	)

	// Legacy create path: SlackThread becomes link #1.
	tk, err := svc.Create(ctx, core.CreateInput{Title: "T", SlackThread: u1})
	if err != nil {
		t.Fatal(err)
	}
	if tk.SlackThread != u1 || len(tk.SlackLinks) != 1 || tk.SlackLinks[0].URL != u1 {
		t.Fatalf("create: %+v", tk.SlackLinks)
	}
	plain, _ := svc.Create(ctx, core.CreateInput{Title: "no links"})
	if plain.SlackLinks == nil || len(plain.SlackLinks) != 0 || plain.SlackThread != "" {
		t.Fatalf("empty: %#v", plain.SlackLinks)
	}

	// Add appends; descriptions travel with links; slack_thread stays the primary.
	tk, err = svc.AddSlackLink(ctx, tk.ID, core.SlackLink{URL: u2, Description: "incident channel"}, false)
	if err != nil {
		t.Fatal(err)
	}
	tk, _ = svc.AddSlackLink(ctx, tk.ID, core.SlackLink{URL: u3, Description: "design review"}, false)
	if len(tk.SlackLinks) != 3 || tk.SlackThread != u1 || tk.SlackLinks[2].Description != "design review" {
		t.Fatalf("add: %+v", tk.SlackLinks)
	}
	// Re-adding an existing URL updates instead of duplicating.
	tk, _ = svc.AddSlackLink(ctx, tk.ID, core.SlackLink{URL: u2 + "/", Description: "incidents"}, true)
	if len(tk.SlackLinks) != 3 || tk.SlackThread != u2 || tk.SlackLinks[0].Description != "incidents" {
		t.Fatalf("re-add primary: %+v", tk.SlackLinks)
	}
	// Invalid URLs are refused.
	var isl *core.InvalidSlackLinkError
	if _, err := svc.AddSlackLink(ctx, tk.ID, core.SlackLink{URL: "https://example.com/x"}, false); !errors.As(err, &isl) {
		t.Fatalf("invalid add err = %v", err)
	}

	// Legacy edit --slack-thread: an existing URL is promoted, nothing dropped.
	s := u3
	tk, _ = svc.Edit(ctx, tk.ID, core.UpdateInput{SlackThread: &s})
	if len(tk.SlackLinks) != 3 || tk.SlackThread != u3 || tk.SlackLinks[0].Description != "design review" {
		t.Fatalf("edit promote: %+v", tk.SlackLinks)
	}
	// edit --slack-thread "" removes the primary; the next is promoted.
	empty := ""
	tk, _ = svc.Edit(ctx, tk.ID, core.UpdateInput{SlackThread: &empty})
	if len(tk.SlackLinks) != 2 || tk.SlackThread != u2 {
		t.Fatalf("edit clear: %+v", tk.SlackLinks)
	}

	// Edit description by index; remove by URL.
	d := "pager"
	tk, err = svc.EditSlackLink(ctx, tk.ID, "2", &d, true)
	if err != nil || tk.SlackThread != u1 || tk.SlackLinks[0].Description != "pager" {
		t.Fatalf("edit link: %+v %v", tk.SlackLinks, err)
	}
	tk, _ = svc.RemoveSlackLink(ctx, tk.ID, u1)
	if len(tk.SlackLinks) != 1 || tk.SlackThread != u2 {
		t.Fatalf("rm: %+v", tk.SlackLinks)
	}
	var nf *core.SlackLinkNotFoundError
	if _, err := svc.RemoveSlackLink(ctx, tk.ID, "9"); !errors.As(err, &nf) {
		t.Fatalf("rm missing err = %v", err)
	}

	// Sub-tasks seed only what they are given (the GUI passes the parent's primary).
	child, _ := svc.Create(ctx, core.CreateInput{Title: "c", ParentID: tk.ID, SlackThread: tk.SlackThread})
	if len(child.SlackLinks) != 1 {
		t.Fatalf("child links = %+v", child.SlackLinks)
	}
}

func TestSetSlackLinksKeepsLegacyNonSlackURL(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	legacy := "https://teams.example.com/thread/1" // pre-v21 slack_thread had no validation
	tk, _ := svc.Create(ctx, core.CreateInput{Title: "T", SlackThread: legacy})
	slack := "https://acme.slack.com/archives/C9"
	tk, err := svc.SetSlackLinks(ctx, tk.ID, []core.SlackLink{{URL: slack, Description: "new"}, {URL: legacy, Description: "old"}})
	if err != nil {
		t.Fatalf("reorder with legacy url: %v", err)
	}
	if tk.SlackThread != slack || tk.SlackLinks[1].Description != "old" {
		t.Fatalf("set: %+v", tk.SlackLinks)
	}
}

func TestSlackLinksNotice(t *testing.T) {
	tk := core.Task{SlackLinks: []core.SlackLink{
		{URL: "https://a.slack.com/archives/C1/p1", Description: "main"},
		{URL: "https://a.slack.com/archives/C2", Description: "incidents"},
		{URL: "https://a.slack.com/archives/C3"},
	}}
	got := core.SlackLinksNotice(tk)
	for _, want := range []string{
		core.SlackThreadNotice("https://a.slack.com/archives/C1/p1") + " (main)",
		"other slack conversations for this ticket:",
		"- https://a.slack.com/archives/C2 — incidents",
		"- https://a.slack.com/archives/C3\n",
	} {
		if !strings.Contains(got+"\n", want) {
			t.Errorf("notice missing %q:\n%s", want, got)
		}
	}
	if core.SlackLinksNotice(core.Task{}) != "" {
		t.Error("no links → empty notice")
	}
}
