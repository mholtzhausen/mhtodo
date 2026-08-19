package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

var sortFields = []string{"created", "updated", "status", "progress", "title"}

// parseSort parses --sort FIELD[+|-]. Per spec, a "-" suffix means ascending;
// "+" or no suffix means descending (the default).
func parseSort(s string) (field string, ascending bool, err error) {
	field = s
	switch {
	case strings.HasSuffix(field, "-"):
		field = strings.TrimSuffix(field, "-")
		ascending = true
	case strings.HasSuffix(field, "+"):
		field = strings.TrimSuffix(field, "+")
	}
	for _, f := range sortFields {
		if field == f {
			return field, ascending, nil
		}
	}
	return "", false, usageError("invalid --sort field %q (want one of: %s, with optional + or - direction)", s, strings.Join(sortFields, ", "))
}

func newListCmd() *cobra.Command {
	var status, search string
	var limit int
	var sort string
	var all bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List tasks (default: excludes done, updated_at desc)",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			field, ascending, err := parseSort(sort)
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

			tasks, err := svc.List(context.Background(), core.ListFilter{
				Status:      core.Status(status), // "" = any (subject to --all); invalid → exit 1
				Search:      search,
				Limit:       limit,
				Sort:        field,
				Ascending:   ascending,
				IncludeDone: all || status != "",
			})
			if err != nil {
				return mapError(err)
			}
			return o.printTasks(tasks)
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "filter by status (pending|wip|done|waiting)")
	cmd.Flags().StringVar(&search, "search", "", "case-insensitive substring over title + description")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results (0 = unlimited)")
	cmd.Flags().StringVar(&sort, "sort", "updated", "sort field: created|updated|status|progress|title; suffix - ascending, + or none descending")
	cmd.Flags().BoolVar(&all, "all", false, "include done tasks")
	return cmd
}
