package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

// newSlackLinkCmd manages a ticket's Slack conversations (v21). The first
// link is the primary thread and is what `slack_thread` reports.
func newSlackLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "link",
		Aliases: []string{"links"},
		Short:   "List / add / edit / remove a ticket's Slack conversation links (first = primary)",
	}
	cmd.AddCommand(newSlackLinkListCmd(), newSlackLinkAddCmd(), newSlackLinkEditCmd(), newSlackLinkRmCmd())
	return cmd
}

func withSvc(cmd *cobra.Command, fn func(o opts, svc *core.Service, ctx context.Context) error) error {
	o, err := o(cmd)
	if err != nil {
		return err
	}
	svc, closeDB, err := openService()
	if err != nil {
		return err
	}
	defer closeDB()
	return fn(o, svc, context.Background())
}

func printSlackLinks(o opts, t core.Task) error {
	if o.json {
		return o.printJSON(t.SlackLinks)
	}
	for i, l := range t.SlackLinks {
		info := core.DescribeSlackLink(l.URL)
		label := info.Label
		if !info.Valid {
			label = "link"
		}
		prim := ""
		if i == 0 {
			prim = " *"
		}
		line := fmt.Sprintf("%d%s\t%s\t%s", i+1, prim, label, l.URL)
		if l.Description != "" {
			line += "\t" + l.Description
		}
		if _, err := fmt.Fprintln(o.out, line); err != nil {
			return err
		}
	}
	return nil
}

func newSlackLinkListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list ID",
		Aliases: []string{"ls"},
		Short:   "List Slack links (index, * = primary, kind, url, description); --json prints the slack_links array",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSvc(cmd, func(o opts, svc *core.Service, ctx context.Context) error {
				t, err := svc.Get(ctx, args[0])
				if err != nil {
					return mapError(err)
				}
				return printSlackLinks(o, t)
			})
		},
	}
}

func newSlackLinkAddCmd() *cobra.Command {
	var desc string
	var primary bool
	cmd := &cobra.Command{
		Use:   "add ID URL",
		Short: "Attach a Slack conversation (https://….slack.com/… or slack://…); re-adding a linked URL updates it",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSvc(cmd, func(o opts, svc *core.Service, ctx context.Context) error {
				t, err := svc.AddSlackLink(ctx, args[0], core.SlackLink{URL: args[1], Description: desc}, primary)
				if err != nil {
					return mapError(err)
				}
				return o.printTask(t)
			})
		},
	}
	cmd.Flags().StringVar(&desc, "desc", "", "optional description shown with the link")
	cmd.Flags().BoolVar(&primary, "primary", false, "make it the primary thread (first; mirrored in slack_thread)")
	return cmd
}

func newSlackLinkEditCmd() *cobra.Command {
	var desc string
	var primary bool
	cmd := &cobra.Command{
		Use:   "edit ID LINK",
		Short: "Change a link's description and/or make it primary (LINK = URL or 1-based index)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSvc(cmd, func(o opts, svc *core.Service, ctx context.Context) error {
				var d *string
				if cmd.Flags().Changed("desc") {
					d = &desc
				}
				if d == nil && !primary {
					return &errExit{code: ExitUsage, name: "no_fields", msg: "pass --desc and/or --primary"}
				}
				t, err := svc.EditSlackLink(ctx, args[0], args[1], d, primary)
				if err != nil {
					return mapError(err)
				}
				return o.printTask(t)
			})
		},
	}
	cmd.Flags().StringVar(&desc, "desc", "", `new description ("" clears)`)
	cmd.Flags().BoolVar(&primary, "primary", false, "make it the primary thread")
	return cmd
}

func newSlackLinkRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm ID LINK",
		Short: "Remove a link (LINK = URL or 1-based index); removing the primary promotes the next",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSvc(cmd, func(o opts, svc *core.Service, ctx context.Context) error {
				t, err := svc.RemoveSlackLink(ctx, args[0], args[1])
				if err != nil {
					return mapError(err)
				}
				return o.printTask(t)
			})
		},
	}
}
