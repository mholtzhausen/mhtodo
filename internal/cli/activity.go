package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

func newActivityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activity",
		Short: "Agent/user-authored activity on a task (add, list, rm)",
	}
	cmd.AddCommand(newActivityAddCmd(), newActivityListCmd(), newActivityRmCmd())
	return cmd
}

func newActivityAddCmd() *cobra.Command {
	var activity, comment string
	cmd := &cobra.Command{
		Use:   "add ID",
		Short: "Add an activity entry to a task (at least one of --activity/--comment)",
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
			a, err := svc.AddActivity(context.Background(), args[0], core.ActivityInput{
				Activity: activity,
				Comment:  comment,
			})
			if err != nil {
				return mapError(err)
			}
			if o.json {
				return o.printJSON(a)
			}
			_, err = fmt.Fprintf(o.out, "%s  %s\n", a.ID, a.Activity)
			return err
		},
	}
	cmd.Flags().StringVar(&activity, "activity", "", "activity summary (agent announcement)")
	cmd.Flags().StringVar(&comment, "comment", "", "optional comment body")
	return cmd
}

func newActivityListCmd() *cobra.Command {
	var limit int
	var tasks []string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List activity entries (newest first; non-archived tasks by default)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			if limit < 0 {
				return usageError("--limit must be >= 0")
			}
			svc, closeDB, err := openService()
			if err != nil {
				return err
			}
			defer closeDB()
			list, err := svc.ListActivity(context.Background(), core.ActivityFilter{
				TaskIDs: tasks,
				Limit:   limit,
			})
			if err != nil {
				return mapError(err)
			}
			if o.json {
				if list == nil {
					list = []core.Activity{}
				}
				return o.printJSON(list)
			}
			for _, a := range list {
				line := fmt.Sprintf("%s  %s  %s", shortID(a.ID), shortID(a.TaskID), a.Activity)
				if a.Comment != "" {
					line += "  — " + a.Comment
				}
				if _, err := fmt.Fprintln(o.out, line); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&tasks, "task", nil, "filter by task ID (repeatable)")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results (0 = unlimited)")
	return cmd
}

func newActivityRmCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "rm ID",
		Short: "Delete an activity entry (non-TTY requires --yes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			if !yes && !stdinIsTTY() {
				return usageError("refusing to delete activity %s without --yes on a non-interactive terminal", args[0])
			}
			svc, closeDB, err := openService()
			if err != nil {
				return err
			}
			defer closeDB()
			del, err := svc.DeleteActivity(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			if o.json {
				return o.printJSON(map[string]string{"id": del.ID})
			}
			_, err = fmt.Fprintln(o.out, del.ID)
			return err
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation (required on non-TTY)")
	return cmd
}
