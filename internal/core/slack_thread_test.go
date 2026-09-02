package core_test

import (
	"strings"
	"testing"

	"mhtodo/internal/core"
)

func TestSlackThreadNotice(t *testing.T) {
	link := "https://example.slack.com/archives/C123/p456"
	got := core.SlackThreadNotice(link)
	want := "the primary thread on slack for communication regarding this ticket is : " + link
	if got != want {
		t.Fatalf("SlackThreadNotice = %q, want %q", got, want)
	}
	if core.SlackThreadNotice("") != "" || core.SlackThreadNotice("  ") != "" {
		t.Fatal("empty thread should yield empty notice")
	}
}

func TestFormatSlackReportWithThread(t *testing.T) {
	link := "https://example.slack.com/archives/C123/p456"
	tasks := []core.Task{
		{ID: "1", Title: "Todo one", Status: core.StatusPending, SlackThread: link},
	}
	got := core.FormatSlackReport(tasks)
	if !strings.Contains(got, core.SlackThreadNotice(link)) {
		t.Fatalf("FormatSlackReport missing thread notice:\n%s", got)
	}
}
