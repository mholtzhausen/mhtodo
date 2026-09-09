package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

func newAddCmd() *cobra.Command {
	var desc, feedback, status, parent, cwd, slackThread, todoSession, template string
	var progress int
	var humanOnly bool
	var includeInReport, noIncludeInReport bool
	cmd := &cobra.Command{
		Use:   "add TITLE",
		Short: "Create a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			svc, closeDB, err := openService()
			if err != nil {
				return err
			}
			defer closeDB()

			in := core.CreateInput{
				Title:       args[0],
				Feedback:    feedback,
				Progress:    progress,
				ParentID:    parent,
				TodoSession: todoSession,
			}

			if template != "" {
				tpl, err := svc.GetTemplate(context.Background(), template)
				if err != nil {
					return mapError(err)
				}
				// Template presets first; any flag the caller passed explicitly wins.
				in = tpl.Apply(in)
				if cmd.Flags().Changed("desc") {
					in.Description = desc
				}
				if cmd.Flags().Changed("status") {
					in.Status = core.Status(status)
				}
				if cmd.Flags().Changed("cwd") {
					in.Cwd = cwd
				}
				if cmd.Flags().Changed("slack-thread") {
					in.SlackThread = slackThread
				}
				if cmd.Flags().Changed("human-only") {
					in.HumanOnly = humanOnly
				}
				switch {
				case cmd.Flags().Changed("include-in-report"):
					v := includeInReport
					in.IncludeInReport = &v
				case cmd.Flags().Changed("no-include-in-report"):
					v := false
					in.IncludeInReport = &v
				}
			} else {
				in.Description = desc
				in.Status = core.Status(status) // "" → pending; invalid → exit 1
				in.Cwd = cwd
				in.HumanOnly = humanOnly
				in.SlackThread = slackThread
				switch {
				case cmd.Flags().Changed("include-in-report"):
					v := includeInReport
					in.IncludeInReport = &v
				case cmd.Flags().Changed("no-include-in-report"):
					v := false
					in.IncludeInReport = &v
				}
			}

			t, err := svc.Create(context.Background(), in)
			if err != nil {
				return mapError(err)
			}
			switch {
			case o.json:
				return o.printJSON(t)
			case o.quiet:
				_, err := fmt.Fprintln(o.out, t.ID) // spec: quiet add prints only the ID
				return err
			default:
				_, err = fmt.Fprintf(o.out, "%s  %s  %s\n", t.ID, t.Status, t.Title)
				return err
			}
		},
	}
	cmd.Flags().StringVar(&desc, "desc", "", "task description")
	cmd.Flags().StringVar(&feedback, "feedback", "", "agent feedback (shown in GUI when set)")
	cmd.Flags().StringVar(&status, "status", "", "initial status (pending|wip|waiting|review|done; default pending)")
	cmd.Flags().IntVar(&progress, "progress", 0, "initial progress 0-100")
	cmd.Flags().StringVar(&parent, "parent", "", "parent task ID (create as a one-level sub-task)")
	cmd.Flags().StringVar(&cwd, "cwd", "", "relevant working directory path")
	cmd.Flags().StringVar(&slackThread, "slack-thread", "", "Slack thread URL for this ticket")
	cmd.Flags().StringVar(&todoSession, "session", "", "todo session id/name (default: shortid-slugified-title)")
	cmd.Flags().StringVar(&template, "template", "", "apply a task template by id or name (CLI flags override presets)")
	cmd.Flags().BoolVar(&humanOnly, "human-only", false, "mark as human-only (excluded from default agent lists)")
	cmd.Flags().BoolVar(&includeInReport, "include-in-report", false, "include in Slack board report (default on for roots, off for sub-tasks)")
	cmd.Flags().BoolVar(&noIncludeInReport, "no-include-in-report", false, "exclude from Slack board report")
	return cmd
}
