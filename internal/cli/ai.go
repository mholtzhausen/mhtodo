package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"mhtodo/internal/aiskill"
)

func newAICmd(_ string) *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "Install or update the Claude mhtodo skill (and strip legacy hooks/integrations)",
		Long: `Write ~/.claude/skills/mhtodo/SKILL.md from the embedded template and remove
legacy Claude Code hooks, settings entries, integration manifests, session
pointer files, and claude.todo helpers.

Use --check to report skill freshness and leftovers without writing.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			var res aiskill.Result
			if checkOnly {
				res, err = aiskill.Check()
			} else {
				res, err = aiskill.Install()
			}
			if err != nil {
				return &errExit{code: ExitStorage, name: "ai", msg: err.Error()}
			}
			if o.json {
				return o.printJSON(res)
			}
			if o.quiet {
				if !checkOnly {
					_, err = fmt.Fprintln(o.out, res.Path)
				}
				return err
			}
			_, err = fmt.Fprintln(o.out, res.Message)
			if !checkOnly && len(res.Removed) > 0 && !o.quiet {
				for _, p := range res.Removed {
					_, _ = fmt.Fprintf(o.out, "  removed: %s\n", p)
				}
			}
			if checkOnly && len(res.Leftovers) > 0 {
				for _, p := range res.Leftovers {
					_, _ = fmt.Fprintf(o.out, "  leftover: %s\n", p)
				}
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "report skill status and leftovers; do not write")
	return cmd
}

func formatSkillRefreshWarn(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		msg = "unknown error"
	}
	return "warning: skill refresh via claude failed: " + msg
}
