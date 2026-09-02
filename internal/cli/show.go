package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var markdown bool
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

			if markdown {
				report, err := svc.TaskMarkdownReport(context.Background(), args[0])
				if err != nil {
					return mapError(err)
				}
				_, err = fmt.Fprintln(o.out, report)
				return err
			}

			t, err := svc.Get(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			return o.printTask(t)
		},
	}
	cmd.Flags().BoolVar(&markdown, "markdown", false, "print paste-ready markdown report")
	return cmd
}
