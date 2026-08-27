package cli

import (
	"context"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

func newEditCmd() *cobra.Command {
	var title, desc, feedback string
	var progress int
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
	return cmd
}
