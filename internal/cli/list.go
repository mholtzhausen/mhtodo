package cli

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

var sortFields = []string{"board", "created", "updated", "status", "progress", "title"}

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
	var status, search, gitRepo, gitBranch string
	var limit int
	var sort string
	var all, archived, roots, humanOnly, gitCurrent bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List tasks (default: excludes done, board order)",
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
			if gitCurrent {
				if cmd.Flags().Changed("git-repo") || cmd.Flags().Changed("git-branch") {
					return usageError("--git-current cannot be combined with --git-repo / --git-branch")
				}
				wd, err := os.Getwd()
				if err != nil {
					return usageError("--git-current: %v", err)
				}
				info, err := core.DetectGitInfo(context.Background(), wd)
				if err != nil {
					return mapError(err)
				}
				gitRepo, gitBranch = info.Repo, info.Branch
			}
			svc, closeDB, err := openService()
			if err != nil {
				return err
			}
			defer closeDB()

			tasks, err := svc.List(context.Background(), core.ListFilter{
				Status:           core.Status(status), // "" = any (subject to --all); invalid → exit 1
				Search:           search,
				Limit:            limit,
				Sort:             field,
				Ascending:        ascending,
				IncludeDone:      all || status != "",
				Archived:         archived, // --archived shows only archived tasks (incl. done)
				RootsOnly:        roots,
				IncludeHumanOnly: humanOnly,
				GitRepo:          gitRepo,
				GitBranch:        gitBranch,
			})
			if err != nil {
				return mapError(err)
			}
			return o.printTasks(tasks)
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "filter by status (icebox|pending|wip|waiting|review|pr|done)")
	cmd.Flags().StringVar(&search, "search", "", "case-insensitive substring over title + description + git repo/branches")
	cmd.Flags().StringVar(&gitRepo, "git-repo", "", "only tasks recording this git repo (any URL form; normalized)")
	cmd.Flags().StringVar(&gitBranch, "git-branch", "", "only tasks listing this exact git branch")
	cmd.Flags().BoolVar(&gitCurrent, "git-current", false, "only tasks recording the current directory's repo + branch")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results (0 = unlimited)")
	cmd.Flags().StringVar(&sort, "sort", "board", "sort field: board|created|updated|status|progress|title; suffix - ascending, + or none descending")
	cmd.Flags().BoolVar(&all, "all", false, "include done tasks")
	cmd.Flags().BoolVar(&archived, "archived", false, "show archived tasks only (default lists hide them)")
	cmd.Flags().BoolVar(&roots, "roots", false, "show top-level tasks only (exclude sub-tasks)")
	cmd.Flags().BoolVar(&humanOnly, "human-only", false, "include human-only tasks (excluded by default)")
	return cmd
}
