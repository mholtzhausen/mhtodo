package cli

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"templates", "tpl"},
		Short:   "Manage task templates (list, search, show, create, update, rm)",
	}
	cmd.AddCommand(
		newTemplateListCmd(),
		newTemplateSearchCmd(),
		newTemplateShowCmd(),
		newTemplateCreateCmd(),
		newTemplateUpdateCmd(),
		newTemplateRmCmd(),
	)
	return cmd
}

func newTemplateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all task templates (name order)",
		Args:    cobra.NoArgs,
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
			list, err := svc.ListTemplates(context.Background())
			if err != nil {
				return mapError(err)
			}
			return o.printTemplates(list)
		},
	}
}

func newTemplateSearchCmd() *cobra.Command {
	var mode, cwd string
	cmd := &cobra.Command{
		Use:   "search [QUERY]",
		Short: "Search templates (fuzzy/regex query and/or --cwd)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			query := ""
			if len(args) == 1 {
				query = args[0]
			}
			if strings.TrimSpace(query) == "" && strings.TrimSpace(cwd) == "" {
				return usageError("template search requires a query and/or --cwd")
			}
			svc, closeDB, err := openService()
			if err != nil {
				return err
			}
			defer closeDB()
			list, err := svc.SearchTemplates(context.Background(), core.TemplateSearchFilter{
				Query: query,
				Mode:  core.TemplateSearchMode(mode),
				Cwd:   cwd,
			})
			if err != nil {
				return mapError(err)
			}
			return o.printTemplates(list)
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "fuzzy", "query match mode: fuzzy|regex (ignored when query omitted)")
	cmd.Flags().StringVar(&cwd, "cwd", "", "only templates whose cwd exactly matches this path")
	return cmd
}

func newTemplateShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "show REF",
		Aliases: []string{"get"},
		Short:   "Show one template by id or name",
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
			tpl, err := svc.GetTemplate(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			return o.printTemplate(tpl)
		},
	}
}

func newTemplateCreateCmd() *cobra.Command {
	var titlePrefix, desc, status, cwd, slackThread string
	var humanOnly, noHumanOnly bool
	var includeInReport, noIncludeInReport bool
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a task template (only passed flags become presets)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			in, err := templateInputFromFlags(cmd, args[0], titlePrefix, desc, status, cwd, slackThread,
				humanOnly, includeInReport)
			if err != nil {
				return err
			}
			svc, closeDB, err := openService()
			if err != nil {
				return err
			}
			defer closeDB()
			tpl, err := svc.CreateTemplate(context.Background(), in)
			if err != nil {
				return mapError(err)
			}
			return o.printTemplate(tpl)
		},
	}
	bindTemplatePresetFlags(cmd, &titlePrefix, &desc, &status, &cwd, &slackThread,
		&humanOnly, &noHumanOnly, &includeInReport, &noIncludeInReport)
	return cmd
}

func newTemplateUpdateCmd() *cobra.Command {
	var name, titlePrefix, desc, status, cwd, slackThread string
	var humanOnly, noHumanOnly bool
	var includeInReport, noIncludeInReport bool
	var clearTitlePrefix, clearDesc, clearStatus, clearCwd, clearSlackThread bool
	var clearHumanOnly, clearIncludeInReport bool
	cmd := &cobra.Command{
		Use:   "update REF",
		Short: "Update a template by id or name (patch; --clear-* unsets a preset)",
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

			existing, err := svc.GetTemplate(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			in := existing.AsInput()
			changed := false

			if cmd.Flags().Changed("name") {
				in.Name = name
				changed = true
			}
			patch, err := applyTemplatePresetPatches(cmd, &in, titlePrefix, desc, status, cwd, slackThread,
				humanOnly, includeInReport,
				clearTitlePrefix, clearDesc, clearStatus, clearCwd, clearSlackThread,
				clearHumanOnly, clearIncludeInReport)
			if err != nil {
				return err
			}
			changed = changed || patch
			if !changed {
				return usageError("template update requires at least one flag")
			}

			tpl, err := svc.UpdateTemplate(context.Background(), existing.ID, in)
			if err != nil {
				return mapError(err)
			}
			return o.printTemplate(tpl)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "rename the template")
	bindTemplatePresetFlags(cmd, &titlePrefix, &desc, &status, &cwd, &slackThread,
		&humanOnly, &noHumanOnly, &includeInReport, &noIncludeInReport)
	cmd.Flags().BoolVar(&clearTitlePrefix, "clear-title-prefix", false, "unset title_prefix preset")
	cmd.Flags().BoolVar(&clearDesc, "clear-desc", false, "unset description preset")
	cmd.Flags().BoolVar(&clearStatus, "clear-status", false, "unset status preset")
	cmd.Flags().BoolVar(&clearCwd, "clear-cwd", false, "unset cwd preset")
	cmd.Flags().BoolVar(&clearSlackThread, "clear-slack-thread", false, "unset slack_thread preset")
	cmd.Flags().BoolVar(&clearHumanOnly, "clear-human-only", false, "unset human_only preset")
	cmd.Flags().BoolVar(&clearIncludeInReport, "clear-include-in-report", false, "unset include_in_report preset")
	return cmd
}

func newTemplateRmCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "rm REF",
		Aliases: []string{"remove", "delete"},
		Short:   "Delete a template (non-interactive shells require --yes)",
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

			tpl, err := svc.GetTemplate(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			if !yes {
				if !stdinIsTTY() {
					return usageError("refusing to delete template %q without --yes on a non-interactive terminal", tpl.Name)
				}
				fmt.Fprintf(o.out, "Delete template %q? [y/N] ", tpl.Name)
				line, _ := bufio.NewReader(Stdin).ReadString('\n')
				if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
					_, err := fmt.Fprintln(o.out, "aborted")
					return err
				}
			}

			del, err := svc.DeleteTemplate(context.Background(), tpl.ID)
			if err != nil {
				return mapError(err)
			}
			if o.json {
				return o.printJSON(map[string]string{"id": del.ID, "name": del.Name})
			}
			_, err = fmt.Fprintln(o.out, del.ID)
			return err
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation (required on non-TTY)")
	return cmd
}

func bindTemplatePresetFlags(cmd *cobra.Command, titlePrefix, desc, status, cwd, slackThread *string,
	humanOnly, noHumanOnly, includeInReport, noIncludeInReport *bool) {
	cmd.Flags().StringVar(titlePrefix, "title-prefix", "", "title prefix applied on create")
	cmd.Flags().StringVar(desc, "desc", "", "description preset")
	cmd.Flags().StringVar(status, "status", "", "status preset (pending|wip|waiting|review|done)")
	cmd.Flags().StringVar(cwd, "cwd", "", "working directory preset")
	cmd.Flags().StringVar(slackThread, "slack-thread", "", "Slack thread URL preset")
	cmd.Flags().BoolVar(humanOnly, "human-only", false, "set human_only preset to true")
	cmd.Flags().BoolVar(noHumanOnly, "no-human-only", false, "set human_only preset to false")
	cmd.Flags().BoolVar(includeInReport, "include-in-report", false, "set include_in_report preset to true")
	cmd.Flags().BoolVar(noIncludeInReport, "no-include-in-report", false, "set include_in_report preset to false")
}

// templateInputFromFlags builds a create TemplateInput from only the flags that
// were explicitly passed (omitted → unset/nil).
func templateInputFromFlags(cmd *cobra.Command, name, titlePrefix, desc, status, cwd, slackThread string,
	humanOnly, includeInReport bool) (core.TemplateInput, error) {
	in := core.TemplateInput{Name: name}
	if cmd.Flags().Changed("title-prefix") {
		v := titlePrefix
		in.TitlePrefix = &v
	}
	if cmd.Flags().Changed("desc") {
		v := desc
		in.Description = &v
	}
	if cmd.Flags().Changed("status") {
		st := core.Status(status)
		in.Status = &st
	}
	if cmd.Flags().Changed("cwd") {
		v := cwd
		in.Cwd = &v
	}
	if cmd.Flags().Changed("slack-thread") {
		v := slackThread
		in.SlackThread = &v
	}
	switch {
	case cmd.Flags().Changed("human-only") && cmd.Flags().Changed("no-human-only"):
		return in, usageError("pass only one of --human-only / --no-human-only")
	case cmd.Flags().Changed("human-only"):
		v := humanOnly
		in.HumanOnly = &v
	case cmd.Flags().Changed("no-human-only"):
		v := false
		in.HumanOnly = &v
	}
	switch {
	case cmd.Flags().Changed("include-in-report") && cmd.Flags().Changed("no-include-in-report"):
		return in, usageError("pass only one of --include-in-report / --no-include-in-report")
	case cmd.Flags().Changed("include-in-report"):
		v := includeInReport
		in.IncludeInReport = &v
	case cmd.Flags().Changed("no-include-in-report"):
		v := false
		in.IncludeInReport = &v
	}
	return in, nil
}

func applyTemplatePresetPatches(cmd *cobra.Command, in *core.TemplateInput,
	titlePrefix, desc, status, cwd, slackThread string,
	humanOnly, includeInReport bool,
	clearTitlePrefix, clearDesc, clearStatus, clearCwd, clearSlackThread,
	clearHumanOnly, clearIncludeInReport bool) (bool, error) {
	changed := false

	setStr := func(flag, clearFlag string, clear bool, setVal string, dest **string) error {
		set := cmd.Flags().Changed(flag)
		if set && clear {
			return usageError("cannot combine --%s and --%s", flag, clearFlag)
		}
		if clear {
			*dest = nil
			changed = true
			return nil
		}
		if set {
			v := setVal
			*dest = &v
			changed = true
		}
		return nil
	}

	if err := setStr("title-prefix", "clear-title-prefix", clearTitlePrefix, titlePrefix, &in.TitlePrefix); err != nil {
		return false, err
	}
	if err := setStr("desc", "clear-desc", clearDesc, desc, &in.Description); err != nil {
		return false, err
	}
	if err := setStr("cwd", "clear-cwd", clearCwd, cwd, &in.Cwd); err != nil {
		return false, err
	}
	if err := setStr("slack-thread", "clear-slack-thread", clearSlackThread, slackThread, &in.SlackThread); err != nil {
		return false, err
	}

	if cmd.Flags().Changed("status") && clearStatus {
		return false, usageError("cannot combine --status and --clear-status")
	}
	if clearStatus {
		in.Status = nil
		changed = true
	} else if cmd.Flags().Changed("status") {
		st := core.Status(status)
		in.Status = &st
		changed = true
	}

	patchBool := func(onFlag, offFlag, clearFlag string, onVal bool, clear bool, dest **bool) error {
		onSet := cmd.Flags().Changed(onFlag)
		offSet := cmd.Flags().Changed(offFlag)
		if onSet && offSet {
			return usageError("pass only one of --%s / --%s", onFlag, offFlag)
		}
		if clear && (onSet || offSet) {
			return usageError("cannot combine --%s with --%s/--%s", clearFlag, onFlag, offFlag)
		}
		if clear {
			*dest = nil
			changed = true
			return nil
		}
		if onSet {
			v := onVal
			*dest = &v
			changed = true
		} else if offSet {
			v := false
			*dest = &v
			changed = true
		}
		return nil
	}
	if err := patchBool("human-only", "no-human-only", "clear-human-only",
		humanOnly, clearHumanOnly, &in.HumanOnly); err != nil {
		return false, err
	}
	if err := patchBool("include-in-report", "no-include-in-report", "clear-include-in-report",
		includeInReport, clearIncludeInReport, &in.IncludeInReport); err != nil {
		return false, err
	}
	return changed, nil
}

func (o opts) printTemplate(t core.Template) error {
	switch {
	case o.json:
		return o.printJSON(t)
	case o.quiet:
		_, err := fmt.Fprintln(o.out, t.ID)
		return err
	default:
		w := newTabWriter(o.out)
		fmt.Fprintf(w, "ID         \t%s\n", t.ID)
		fmt.Fprintf(w, "Name       \t%s\n", t.Name)
		writeOptString(w, "Title prefix", t.TitlePrefix)
		writeOptString(w, "Description", t.Description)
		if t.Status != nil {
			fmt.Fprintf(w, "Status     \t%s\n", *t.Status)
		} else {
			fmt.Fprintf(w, "Status     \t—\n")
		}
		writeOptString(w, "Cwd", t.Cwd)
		writeOptString(w, "Slack thread", t.SlackThread)
		writeOptBool(w, "Human only", t.HumanOnly)
		writeOptBool(w, "Include in report", t.IncludeInReport)
		fmt.Fprintf(w, "Created    \t%s\n", absTS(t.CreatedAt))
		fmt.Fprintf(w, "Updated    \t%s\n", absTS(t.UpdatedAt))
		return w.Flush()
	}
}

func writeOptString(w *tabwriter.Writer, label string, v *string) {
	if v == nil {
		fmt.Fprintf(w, "%s\t—\n", label)
		return
	}
	fmt.Fprintf(w, "%s\t%s\n", label, *v)
}

func writeOptBool(w *tabwriter.Writer, label string, v *bool) {
	if v == nil {
		fmt.Fprintf(w, "%s\t—\n", label)
		return
	}
	fmt.Fprintf(w, "%s\t%t\n", label, *v)
}

func (o opts) printTemplates(list []core.Template) error {
	if o.json {
		if list == nil {
			list = []core.Template{}
		}
		return o.printJSON(list)
	}
	w := newTabWriter(o.out)
	for _, t := range list {
		cwd := "—"
		if t.Cwd != nil && *t.Cwd != "" {
			cwd = *t.Cwd
		} else if t.Cwd != nil {
			cwd = "(empty)"
		}
		st := "—"
		if t.Status != nil {
			st = string(*t.Status)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", shortID(t.ID), t.Name, st, cwd)
	}
	return w.Flush()
}
