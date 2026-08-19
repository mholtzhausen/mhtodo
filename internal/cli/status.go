package cli

import (
	"context"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
	"mhtodo/internal/notify"
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

// NotifyDone is the test seam for `done --notify` (cf. Stdin): golden tests
// record calls instead of exec'ing notify-send. Failures inside notify are
// logged and never fatal (spec).
var NotifyDone = func(id, title string) { notify.New().TaskDone(id, title) }

func newDoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "done ID",
		Short: "Mark a task done (shortcut for: status ID done)",
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
			ctx := context.Background()
			prev, perr := svc.Get(ctx, args[0]) // transition detection: notify only on a real change
			t, err := svc.SetStatus(ctx, args[0], core.StatusDone)
			if err != nil {
				return mapError(err)
			}
			if b, _ := cmd.Flags().GetBool("notify"); b && perr == nil && prev.Status != t.Status {
				NotifyDone(t.ID, t.Title)
			}
			return o.printTask(t)
		},
	}
	cmd.Flags().Bool("notify", false, "send a desktop notification (opt-in; the GUI always notifies on →done/→waiting)")
	return cmd
}
