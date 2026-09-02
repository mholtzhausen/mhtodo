package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// slackStatusIcon is the per-task marker in the Slack board report.
func slackStatusIcon(st Status) string {
	switch st {
	case StatusPending:
		return "○"
	case StatusWIP:
		return "◐"
	case StatusWaiting:
		return "◷"
	case StatusReview:
		return "◎"
	case StatusDone:
		return "✓"
	default:
		return "?"
	}
}

type slackSection struct {
	title string
	icon  string
	order int
	tasks []Task
}

// FormatSlackReport renders root tasks into the Slack paste format: Completed,
// Todo, then WIP (wip + waiting + review grouped under one heading). Tasks
// should already be in board order; this re-buckets by section.
func FormatSlackReport(tasks []Task) string {
	var completed, todo, wip []Task
	for _, t := range tasks {
		switch t.Status {
		case StatusDone:
			completed = append(completed, t)
		case StatusPending:
			todo = append(todo, t)
		case StatusWIP, StatusWaiting, StatusReview:
			wip = append(wip, t)
		}
	}
	sortSlackTasks(completed)
	sortSlackTasks(todo)
	sortSlackWIPTasks(wip)

	sections := []slackSection{
		{title: "Completed", icon: "✓", order: 0, tasks: completed},
		{title: "Todo", icon: "○", order: 1, tasks: todo},
		{title: "WIP", icon: "◐", order: 2, tasks: wip},
	}

	var b strings.Builder
	first := true
	for _, sec := range sections {
		if len(sec.tasks) == 0 {
			continue
		}
		if !first {
			b.WriteByte('\n')
		}
		first = false
		fmt.Fprintf(&b, "%s %s\n", sec.title, sec.icon)
		for _, t := range sec.tasks {
			fmt.Fprintf(&b, "  %s %s\n", slackStatusIcon(t.Status), t.Title)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func sortSlackTasks(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		return slackBoardLess(tasks[i], tasks[j])
	})
}

// sortSlackWIPTasks keeps wip → waiting → review, then board rank within each.
func sortSlackWIPTasks(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i], tasks[j]
		oa, ob := slackWIPOrder(a.Status), slackWIPOrder(b.Status)
		if oa != ob {
			return oa < ob
		}
		return slackBoardLess(a, b)
	})
}

func slackWIPOrder(st Status) int {
	switch st {
	case StatusWIP:
		return 0
	case StatusWaiting:
		return 1
	case StatusReview:
		return 2
	default:
		return 99
	}
}

func slackBoardLess(a, b Task) bool {
	ra, rb := slackRank(a), slackRank(b)
	if ra != rb {
		return ra < rb
	}
	if !a.UpdatedAt.Equal(b.UpdatedAt) {
		return a.UpdatedAt.After(b.UpdatedAt)
	}
	return a.ID > b.ID
}

func slackRank(t Task) float64 {
	if t.BoardRank != nil {
		return *t.BoardRank
	}
	return 1e18
}

// SlackReport returns a Slack-ready board summary of non-archived root tasks
// (includes done and human-only tasks).
func (s *Service) SlackReport(ctx context.Context) (string, error) {
	tasks, err := s.List(ctx, ListFilter{
		Sort:             "board",
		IncludeDone:      true,
		RootsOnly:        true,
		IncludeHumanOnly: true,
	})
	if err != nil {
		return "", err
	}
	filtered := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		if t.IncludeInReport {
			filtered = append(filtered, t)
		}
	}
	return FormatSlackReport(filtered), nil
}
