package cli

import (
	"context"

	"github.com/spf13/cobra"
)

func newDemoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "demote ID PARENT_ID",
		Aliases: []string{"reparent"},
		Short:   "Make a task a sub-task of another top-level task (or move a sub-task to another parent)",
		Args:    cobra.ExactArgs(2),
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
			t, err := svc.SetParent(context.Background(), args[0], args[1])
			if err != nil {
				return mapError(err)
			}
			return o.printTask(t)
		},
	}
}
