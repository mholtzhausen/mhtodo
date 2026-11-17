package core_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"mhtodo/internal/core"
)

func TestParsePriorityFilter(t *testing.T) {
	P := func(ps ...core.Priority) []core.Priority { return ps }
	cases := []struct {
		in   string
		want []core.Priority
	}{
		{"", nil},
		{"high", P(core.PriorityHigh)},
		{"URGENT, high", P(core.PriorityHigh, core.PriorityUrgent)},
		{">=high", P(core.PriorityHigh, core.PriorityUrgent)},
		{">medium", P(core.PriorityHigh, core.PriorityUrgent)},
		{"<=low", P(core.PriorityNone, core.PriorityLow)},
		{"<low,urgent", P(core.PriorityNone, core.PriorityUrgent)},
		{">urgent", []core.Priority{}},
	}
	for _, c := range cases {
		got, err := core.ParsePriorityFilter(c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q = %v, want %v", c.in, got, c.want)
		}
	}
	for _, bad := range []string{"hgh", ">=", "high,nope", "=>high"} {
		var ipe *core.InvalidPriorityError
		if _, err := core.ParsePriorityFilter(bad); !errors.As(err, &ipe) {
			t.Errorf("%q: err = %v, want InvalidPriorityError", bad, err)
		}
	}
}

func TestPriorityLifecycle(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	a, err := svc.Create(ctx, core.CreateInput{Title: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Priority != core.PriorityNone {
		t.Fatalf("default priority = %q", a.Priority)
	}
	b, err := svc.Create(ctx, core.CreateInput{Title: "B", Priority: core.PriorityUrgent})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, core.CreateInput{Title: "bad", Priority: "huge"}); err == nil {
		t.Fatal("invalid priority accepted on create")
	}

	// Sub-tasks do not inherit the parent's priority.
	child, err := svc.Create(ctx, core.CreateInput{Title: "child", ParentID: b.ID})
	if err != nil {
		t.Fatal(err)
	}
	if child.Priority != core.PriorityNone {
		t.Fatalf("child inherited priority %q", child.Priority)
	}

	// Setting priority never changes board_rank (manual order wins).
	rankBefore := *a.BoardRank
	a2, err := svc.SetPriority(ctx, a.ID, core.PriorityHigh)
	if err != nil {
		t.Fatal(err)
	}
	if a2.Priority != core.PriorityHigh || *a2.BoardRank != rankBefore {
		t.Fatalf("after SetPriority: priority=%q rank=%v (was %v)", a2.Priority, *a2.BoardRank, rankBefore)
	}
	got, _ := svc.Get(ctx, a.ID)
	if got.Priority != core.PriorityHigh {
		t.Fatalf("persisted priority = %q", got.Priority)
	}

	// Default board order is unaffected by priority.
	list, err := svc.List(ctx, core.ListFilter{RootsOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != a.ID || list[1].ID != b.ID {
		t.Fatalf("board order changed: %v", ids(list))
	}

	// Filter and opt-in sort.
	urgent, _ := core.ParsePriorityFilter("urgent")
	list, _ = svc.List(ctx, core.ListFilter{Priorities: urgent})
	if len(list) != 1 || list[0].ID != b.ID {
		t.Fatalf("filter urgent = %v", ids(list))
	}
	none, _ := core.ParsePriorityFilter(">urgent")
	list, _ = svc.List(ctx, core.ListFilter{Priorities: none})
	if len(list) != 0 {
		t.Fatalf(">urgent matched %v", ids(list))
	}
	list, _ = svc.List(ctx, core.ListFilter{Sort: "priority"})
	if list[0].ID != b.ID || list[1].ID != a.ID {
		t.Fatalf("priority sort = %v", ids(list))
	}

	// Clearing.
	a3, err := svc.SetPriority(ctx, a.ID, core.PriorityNone)
	if err != nil || a3.Priority != core.PriorityNone {
		t.Fatalf("clear: %v %q", err, a3.Priority)
	}
}

func TestTemplatePriorityPreset(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	p := core.Priority("URGENT")
	tpl, err := svc.CreateTemplate(ctx, core.TemplateInput{Name: "incident", Priority: &p})
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Priority == nil || *tpl.Priority != core.PriorityUrgent {
		t.Fatalf("template priority = %v", tpl.Priority)
	}
	got, _ := svc.GetTemplate(ctx, "incident")
	in := got.Apply(core.CreateInput{Title: "outage"})
	task, err := svc.Create(ctx, in)
	if err != nil || task.Priority != core.PriorityUrgent {
		t.Fatalf("templated task priority = %q (%v)", task.Priority, err)
	}
	bad := core.Priority("loud")
	if _, err := svc.CreateTemplate(ctx, core.TemplateInput{Name: "bad", Priority: &bad}); err == nil {
		t.Fatal("invalid template priority accepted")
	}
}

func ids(ts []core.Task) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Title
	}
	return out
}
