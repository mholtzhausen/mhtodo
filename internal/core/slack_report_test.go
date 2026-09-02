package core_test

import (
	"context"
	"strings"
	"testing"

	"mhtodo/internal/core"
)

func TestFormatSlackReport(t *testing.T) {
	r1, r2, r3 := 1.0, 2.0, 3.0
	tasks := []core.Task{
		{ID: "1", Title: "Pending B", Status: core.StatusPending, BoardRank: &r2},
		{ID: "2", Title: "Pending A", Status: core.StatusPending, BoardRank: &r1},
		{ID: "3", Title: "WIP task", Status: core.StatusWIP, BoardRank: &r1},
		{ID: "4", Title: "Waiting task", Status: core.StatusWaiting, BoardRank: &r2},
		{ID: "5", Title: "Review task", Status: core.StatusReview, BoardRank: &r3},
		{ID: "6", Title: "Done B", Status: core.StatusDone, BoardRank: &r2},
		{ID: "7", Title: "Done A", Status: core.StatusDone, BoardRank: &r1},
	}

	got := core.FormatSlackReport(tasks)
	want := strings.Join([]string{
		"Completed ✓",
		"  ✓ Done A",
		"  ✓ Done B",
		"",
		"Todo ○",
		"  ○ Pending A",
		"  ○ Pending B",
		"",
		"WIP ◐",
		"  ◐ WIP task",
		"  ◷ Waiting task",
		"  ◎ Review task",
	}, "\n")
	if got != want {
		t.Errorf("FormatSlackReport:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestSlackReportService(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	parent, err := svc.Create(ctx, core.CreateInput{Title: "Parent todo"})
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []core.CreateInput{
		{Title: "Todo one"},
		{Title: "Active", Status: core.StatusWIP},
		{Title: "Blocked", Status: core.StatusWaiting},
		{Title: "Shipped", Status: core.StatusDone},
	} {
		if _, err := svc.Create(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Create(ctx, core.CreateInput{Title: "Sub step", ParentID: parent.ID}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.SlackReport(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Completed ✓", "Todo ○", "WIP ◐", "Shipped", "Todo one", "Active", "Blocked", "Parent todo"} {
		if !strings.Contains(got, want) {
			t.Errorf("report missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Sub step") {
		t.Error("sub-task should be excluded from slack report")
	}
}
