package core_test

import (
	"strings"
	"testing"
	"time"

	"mhtodo/internal/core"
)

func TestFormatTaskMarkdown(t *testing.T) {
	updated := time.Date(2026, 9, 2, 12, 59, 0, 0, time.UTC)
	root := core.Task{
		ID:          "01900000-0000-7000-8000-000000000001",
		Title:       "Replace car-port shading fabric",
		Description: "# Car-port shading replacement\n\nNeed replacement fabric.",
		Status:      core.StatusWIP,
		Progress:    35,
		UpdatedAt:   updated,
	}
	childDone := core.Task{
		ID:          "01900000-0000-7000-8000-000000000002",
		Title:       "Measure car-port opening",
		Description: "Measured 6.18 m × 3.08 m.",
		Status:      core.StatusDone,
		Progress:    100,
		UpdatedAt:   updated,
	}
	childWIP := core.Task{
		ID:    "01900000-0000-7000-8000-000000000003",
		Title: "Request quotes from installers",
		Status: core.StatusWIP,
		Progress: 60,
		UpdatedAt: updated,
	}
	activities := []core.Activity{
		{
			TaskID:   root.ID,
			Activity: "Fabric Shortlist",
			Comment:  "ShadePro sample book: Sandstone 320.",
			CreatedAt: updated,
		},
		{
			TaskID:   childWIP.ID,
			Activity: "Quotes Requested",
			Comment:  "Emailed ShadePro and SunStop.",
			CreatedAt: updated,
		},
	}

	got := core.FormatTaskMarkdown(root, []core.Task{childDone, childWIP}, activities)

	for _, want := range []string{
		"## ◐ Replace car-port shading fabric",
		"wip · 35% · `00000001` · updated 2 Sep 2026 12:59",
		"### Description",
		"# Car-port shading replacement",
		"### Sub-tasks",
		"✓ Measure car-port opening · done · 100%",
		"Measured 6.18 m × 3.08 m.",
		"◐ Request quotes from installers · wip · 60%",
		"### Activity",
		"**Fabric Shortlist** — ShadePro sample book: Sandstone 320.",
		"**Quotes Requested** *(Request quotes from installers)* — Emailed ShadePro and SunStop.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("FormatTaskMarkdown missing %q:\n%s", want, got)
		}
	}
}
