package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

func newAddCmd() *cobra.Command {
	var desc, status, parent string
	var progress int
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

			t, err := svc.Create(context.Background(), core.CreateInput{
				Title:       args[0],
				Description: desc,
				Status:      core.Status(status), // "" → pending; invalid → exit 1
				Progress:    progress,
				ParentID:    parent,
			})
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
	cmd.Flags().StringVar(&status, "status", "", "initial status (pending|wip|waiting|review|done; default pending)")
	cmd.Flags().IntVar(&progress, "progress", 0, "initial progress 0-100")
	cmd.Flags().StringVar(&parent, "parent", "", "parent task ID (create as a one-level sub-task)")
	return cmd
}
