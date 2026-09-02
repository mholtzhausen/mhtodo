package cli

import (
	"context"

	"github.com/spf13/cobra"

	"mhtodo/internal/settings"
)

// archive / unarchive (v0.2). Archive is bulk-only by design: the GUI button
// sits on the Done column header and archives everything in it; there is no
// per-task archive command. Unarchive restores a task to pending with
// progress reset (core.Unarchive owns that rule).

func newArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive all done tasks (reversible via unarchive)",
		Long: `Archives every task currently in the done status. Archived tasks disappear
from default lists and the board; list them with: mhtodo list --archived`,
		Args: cobra.NoArgs,
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

			cfg, err := settings.Load(nil)
			if err != nil {
				return err
			}
			tasks, err := svc.ArchiveDone(context.Background(), cfg.ArchiveDoneSubtasks)
			if err != nil {
				return mapError(err)
			}
			return o.printTasks(tasks) // empty → no output, exit 0
		},
	}
	return cmd
}

func newUnarchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unarchive ID",
		Short: "Restore an archived task (goes back to pending, progress reset)",
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

			t, err := svc.Unarchive(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			return o.printTask(t)
		},
	}
	return cmd
}
