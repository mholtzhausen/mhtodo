package cli

import (
	"context"

	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show ID",
		Aliases: []string{"get"},
		Short:   "Show one task by full ID or unique prefix (>= 4 chars)",
		Args:    cobra.ExactArgs(1),
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

			t, err := svc.Get(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			return o.printTask(t)
		},
	}
	return cmd
}
