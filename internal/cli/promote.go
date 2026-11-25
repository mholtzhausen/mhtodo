package cli

import (
	"context"

	"github.com/spf13/cobra"
)

func newPromoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "promote ID",
		Short: "Promote a sub-task to a top-level task",
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
			t, err := svc.Promote(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			return o.printTask(t)
		},
	}
}
