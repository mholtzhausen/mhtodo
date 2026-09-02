package cli

import (
	"context"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

func newEditCmd() *cobra.Command {
	var title, desc, feedback, cwd string
	var progress int
	var humanOnly, noHumanOnly bool
	var includeInReport, noIncludeInReport bool
	cmd := &cobra.Command{
		Use:   "edit ID",
		Short: "Update a task's title/description/feedback/progress (at least one flag)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			in := core.UpdateInput{}
			if cmd.Flags().Changed("title") {
				v := title
				in.Title = &v
			}
			if cmd.Flags().Changed("desc") {
				v := desc
				in.Desc = &v
			}
			if cmd.Flags().Changed("feedback") {
				v := feedback
				in.Feedback = &v
			}
		if cmd.Flags().Changed("progress") {
			v := progress
			in.Progress = &v
		}
		if cmd.Flags().Changed("cwd") {
			v := cwd
			in.Cwd = &v
		}
		switch {
		case cmd.Flags().Changed("human-only"):
			v := humanOnly
			in.HumanOnly = &v
		case cmd.Flags().Changed("no-human-only"):
			v := false
			in.HumanOnly = &v
		case cmd.Flags().Changed("include-in-report"):
			v := includeInReport
			in.IncludeInReport = &v
		case cmd.Flags().Changed("no-include-in-report"):
			v := false
			in.IncludeInReport = &v
		}

		svc, closeDB, err := openService()
			if err != nil {
				return err
			}
			defer closeDB()

			t, err := svc.Edit(context.Background(), args[0], in)
			if err != nil {
				return mapError(err) // no flags set → ErrNoFieldsToUpdate (exit 1)
			}
			return o.printTask(t)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&desc, "desc", "", "new description")
	cmd.Flags().StringVar(&feedback, "feedback", "", "agent feedback (shown in GUI when set)")
	cmd.Flags().IntVar(&progress, "progress", 0, "new progress 0-100")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory path (empty clears)")
	cmd.Flags().BoolVar(&humanOnly, "human-only", false, "mark as human-only")
	cmd.Flags().BoolVar(&noHumanOnly, "no-human-only", false, "clear human-only flag")
	cmd.Flags().BoolVar(&includeInReport, "include-in-report", false, "include in Slack board report")
	cmd.Flags().BoolVar(&noIncludeInReport, "no-include-in-report", false, "exclude from Slack board report")
	return cmd
}
