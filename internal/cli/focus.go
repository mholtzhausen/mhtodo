package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// newFocusCmd manages the Focus view pins. Bare `focus` lists the lanes.
func newFocusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "focus",
		Short: "Manage the Focus view (list, add, rm, clear); bare focus lists lanes",
		Args:  cobra.NoArgs,
		RunE:  runFocusList,
	}
	cmd.AddCommand(newFocusListCmd(), newFocusAddCmd(), newFocusRmCmd(), newFocusClearCmd())
	return cmd
}

func newFocusListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Focus lanes in order (a focused parent with a focused sub-task is hidden)",
		Args:    cobra.NoArgs,
		RunE:    runFocusList,
	}
}

func runFocusList(cmd *cobra.Command, args []string) error {
	o, err := o(cmd)
	if err != nil {
		return err
	}
	svc, closeDB, err := openService()
	if err != nil {
		return err
	}
	defer closeDB()
	lanes, err := svc.ListFocus(context.Background())
	if err != nil {
		return mapError(err)
	}
	return o.printTasks(lanes)
}

func newFocusAddCmd() *cobra.Command {
	var before string
	cmd := &cobra.Command{
		Use:   "add ID",
		Short: "Pin a task to the Focus view (or move it with --before)",
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
			t, err := svc.Focus(context.Background(), args[0], beforeRef)
			if err != nil {
				return mapError(err)
			}
			return o.printTask(t)
		},
	}
	cmd.Flags().StringVar(&before, "before", "", "place immediately before this focused task (default: append)")
	return cmd
}

func newFocusRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm ID",
		Short: "Remove a task from the Focus view",
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
			t, err := svc.Unfocus(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			return o.printTask(t)
		},
	}
}

func newFocusClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove every task from the Focus view",
		Args:  cobra.NoArgs,
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
			n, err := svc.ClearFocus(context.Background())
			if err != nil {
				return mapError(err)
			}
			if o.json {
				return o.printJSON(map[string]int{"cleared": n})
			}
			if o.quiet {
				return nil
			}
			_, err = fmt.Fprintf(o.out, "cleared %d\n", n)
			return err
		},
	}
}
