package cli

import (
	"context"

	"github.com/spf13/cobra"
)

func newReorderCmd() *cobra.Command {
	var before string
	cmd := &cobra.Command{
		Use:   "reorder ID",
		Short: "Move a root task within its board column",
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

			var beforeRef *string
			if before != "" {
				beforeRef = &before
			}
			t, err := svc.ReorderBoardTask(context.Background(), args[0], beforeRef)
			if err != nil {
				return mapError(err)
			}
			return o.printTask(t)
		},
	}
	cmd.Flags().StringVar(&before, "before", "", "insert immediately before this root task (default: append to column end)")
	return cmd
}
