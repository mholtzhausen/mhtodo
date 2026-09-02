package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newSlackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slack",
		Short: "Slack helpers",
	}
	cmd.AddCommand(newSlackReportCmd())
	return cmd
}

func newSlackReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Print a Slack-ready board summary (Completed / Todo / WIP)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			svc, close, err := openService()
			if err != nil {
				return err
			}
			defer close()

			report, err := svc.SlackReport(cmd.Context())
			if err != nil {
				return mapError(err)
			}
			if o.json {
				b, err := json.Marshal(report)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(o.out, string(b))
				return err
			}
			if report == "" {
				return nil
			}
			_, err = fmt.Fprintln(o.out, report)
			return err
		},
	}
	return cmd
}
