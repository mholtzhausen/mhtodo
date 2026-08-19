package cli

import (
	"context"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

func runSetStatus(cmd *cobra.Command, ref string, st core.Status) error {
	o, err := o(cmd)
	if err != nil {
		return err
	}
	svc, closeDB, err := openService()
	if err != nil {
		return err
	}
	defer closeDB()

	t, err := svc.SetStatus(context.Background(), ref, st)
	if err != nil {
		return mapError(err)
	}
	return o.printTask(t)
}

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status ID STATUS",
		Aliases: []string{"set"},
		Short:   "Set a task's status (pending|wip|done|waiting)",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := core.ParseStatus(args[1])
			if err != nil {
				return mapError(err) // invalid_status → exit 1
			}
			return runSetStatus(cmd, args[0], st)
		},
	}
	return cmd
}

func newDoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "done ID",
		Short: "Mark a task done (shortcut for: status ID done)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetStatus(cmd, args[0], core.StatusDone)
		},
	}
	return cmd
}
