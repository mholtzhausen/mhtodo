package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

func newPriorityCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "priority ID [LEVEL]",
		Aliases: []string{"prio"},
		Short:   "Show or set a task's priority (none|low|medium|high|urgent); never reorders the board",
		Args:    cobra.RangeArgs(1, 2),
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
			ctx := context.Background()

			if len(args) == 1 {
				t, err := svc.Get(ctx, args[0])
				if err != nil {
					return mapError(err)
				}
				if o.json {
					return o.printJSON(t)
				}
				_, err = fmt.Fprintln(o.out, t.Priority)
				return err
			}
			p, err := core.ParsePriority(args[1])
			if err != nil {
				return mapError(err) // invalid_priority → exit 1
			}
			t, err := svc.SetPriority(ctx, args[0], p)
			if err != nil {
				return mapError(err)
			}
			return o.printTask(t)
		},
	}
}
