package core

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// FormatTaskMarkdown renders a paste-ready task report: title header, meta line,
// description, sub-tasks (roots only), and activity (newest first).
func FormatTaskMarkdown(task Task, children []Task, activities []Activity) string {
	var b strings.Builder
	icon := slackStatusIcon(task.Status)
	fmt.Fprintf(&b, "## %s %s\n\n", icon, task.Title)
	fmt.Fprintf(&b, "%s · %d%% · `%s` · updated %s\n",
		task.Status, task.Progress, ShortID(task.ID), formatTaskUpdated(task.UpdatedAt))

	if notice := SlackThreadNotice(task.SlackThread); notice != "" {
		fmt.Fprintf(&b, "\n%s\n", notice)
	}

	if desc := strings.TrimSpace(task.Description); desc != "" {
		b.WriteString("\n### Description\n\n```\n")
		b.WriteString(desc)
		if !strings.HasSuffix(desc, "\n") {
			b.WriteByte('\n')
		}
		b.WriteString("```\n")
	}

	if len(children) > 0 {
		sortTaskMarkdownChildren(children)
		b.WriteString("\n### Sub-tasks\n")
		for _, c := range children {
			b.WriteByte('\n')
			fmt.Fprintf(&b, "  %s %s · %s · %d%%\n",
				slackStatusIcon(c.Status), c.Title, c.Status, c.Progress)
			if cdesc := strings.TrimSpace(c.Description); cdesc != "" {
				b.WriteString("\n  ```\n")
				for _, line := range strings.Split(cdesc, "\n") {
					b.WriteString("  ")
					b.WriteString(line)
					b.WriteByte('\n')
				}
				b.WriteString("  ```\n")
			}
		}
	}

	if len(activities) > 0 {
		childTitle := childTitleMap(children)
		b.WriteString("\n### Activity\n")
		for _, a := range activities {
			b.WriteByte('\n')
			b.WriteString("- ")
			if act := strings.TrimSpace(a.Activity); act != "" {
				fmt.Fprintf(&b, "**%s**", act)
			}
			if a.TaskID != task.ID {
				if title := childTitle[a.TaskID]; title != "" {
					fmt.Fprintf(&b, " *(%s)*", title)
				}
			}
			if comment := strings.TrimSpace(a.Comment); comment != "" {
				if strings.TrimSpace(a.Activity) != "" {
					b.WriteString(" — ")
				}
				b.WriteString(comment)
			}
		}
		b.WriteByte('\n')
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

func formatTaskUpdated(t time.Time) string {
	return t.UTC().Format("2 Jan 2006 15:04")
}

func childTitleMap(children []Task) map[string]string {
	m := make(map[string]string, len(children))
	for _, c := range children {
		m[c.ID] = c.Title
	}
	return m
}

func sortTaskMarkdownChildren(children []Task) {
	sort.SliceStable(children, func(i, j int) bool {
		a, b := children[i], children[j]
		oa, ob := statusMarkdownOrder(a.Status), statusMarkdownOrder(b.Status)
		if oa != ob {
			return oa < ob
		}
		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			return a.UpdatedAt.After(b.UpdatedAt)
		}
		return a.ID > b.ID
	})
}

func statusMarkdownOrder(st Status) int {
	switch st {
	case StatusPending:
		return 0
	case StatusWIP:
		return 1
	case StatusWaiting:
		return 2
	case StatusReview:
		return 3
	case StatusDone:
		return 4
	default:
		return 99
	}
}
