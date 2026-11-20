package core_test

import (
	"context"
	"strings"
	"testing"

	"mhtodo/internal/core"
)

const sampleID = "019a3f2c-7b1e-7d40-9a55-3c2e8f10ab42"

func TestSearchIDFragment(t *testing.T) {
	t.Parallel()
	cases := []struct {
		q        string
		frag     string
		explicit bool
	}{
		{"", "", false},
		{"deploy", "", false},              // text
		{"019", "", false},                 // too short: would match the whole board
		{"019a3", "", false},               // 5 hex: still text-only
		{"019a3f", "019a3f", false},        // 6 hex: switches on
		{"  8F10AB42 ", "8f10ab42", false}, // short ID, case + whitespace
		{sampleID, strings.ReplaceAll(sampleID, "-", ""), false},
		{strings.ToUpper(strings.ReplaceAll(sampleID, "-", "")), strings.ReplaceAll(sampleID, "-", ""), false},
		{"mhtodo://task/8f10ab42", "8f10ab42", true},
		{"MHTodo Ticket: mhtodo://task/" + sampleID, strings.ReplaceAll(sampleID, "-", ""), true},
		{"fix 019a3f2c", "", false}, // mixed text is not an ID
	}
	for _, c := range cases {
		frag, explicit := core.SearchIDFragment(c.q)
		if frag != c.frag || explicit != c.explicit {
			t.Errorf("SearchIDFragment(%q) = %q,%v want %q,%v", c.q, frag, explicit, c.frag, c.explicit)
		}
	}
}

func TestExactSearchID(t *testing.T) {
	t.Parallel()
	other := "019a3f2c-7b1e-7d40-9a55-000000000001"
	ids := []string{sampleID, other}
	cases := []struct{ q, want string }{
		{"8f10ab42", sampleID},                // short ID (suffix)
		{sampleID, sampleID},                  // full UUID
		{strings.ToUpper(sampleID), sampleID}, // capitals
		{strings.ReplaceAll(sampleID, "-", ""), sampleID},
		{"mhtodo://task/8f10ab42", sampleID},
		{"MHTodo Ticket: mhtodo://task/" + sampleID, sampleID},
		{"019a3f2c", ""}, // shared prefix: ambiguous
		{"10ab42", ""},   // fragment below short-ID length: narrows only
		{"deploy", ""},
	}
	for _, c := range cases {
		if got := core.ExactSearchID(c.q, ids); got != c.want {
			t.Errorf("ExactSearchID(%q) = %q want %q", c.q, got, c.want)
		}
	}
}

func TestListWithParentsKeepsIDMatchedSubtaskParent(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	parent, err := svc.Create(ctx, core.CreateInput{Title: "Parent"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := svc.Create(ctx, core.CreateInput{Title: "Child", ParentID: parent.ID})
	if err != nil {
		t.Fatal(err)
	}
	q := core.ShortID(child.ID)

	// CLI shape: flat, only the match.
	got, err := svc.List(ctx, core.ListFilter{Search: q, IncludeHumanOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != child.ID {
		t.Fatalf("plain list = %v, want only child", titles(got))
	}

	// GUI shape: the parent comes along so the board can nest the child.
	got, err = svc.List(ctx, core.ListFilter{Search: q, IncludeHumanOnly: true, WithParents: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != child.ID || got[1].ID != parent.ID {
		t.Fatalf("WithParents list = %v, want [Child Parent]", titles(got))
	}

	// Text search never pulls parents in.
	got, err = svc.List(ctx, core.ListFilter{Search: "Child", IncludeHumanOnly: true, WithParents: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("text WithParents list = %v, want only child", titles(got))
	}
}

func titles(ts []core.Task) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Title
	}
	return out
}
