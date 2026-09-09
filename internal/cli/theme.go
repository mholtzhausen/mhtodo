package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

func newThemeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "theme",
		Aliases: []string{"themes"},
		Short:   "Manage GUI themes (list, search, show, create, update, rm, activate, duplicate, reset)",
	}
	cmd.AddCommand(
		newThemeListCmd(),
		newThemeSearchCmd(),
		newThemeShowCmd(),
		newThemeCreateCmd(),
		newThemeUpdateCmd(),
		newThemeRmCmd(),
		newThemeActivateCmd(),
		newThemeDuplicateCmd(),
		newThemeResetCmd(),
	)
	return cmd
}

func newThemeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all themes (name order)",
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
			list, err := svc.ListThemes(context.Background())
			if err != nil {
				return mapError(err)
			}
			return o.printThemes(list)
		},
	}
}

func newThemeSearchCmd() *cobra.Command {
	var mode string
	cmd := &cobra.Command{
		Use:   "search QUERY",
		Short: "Search themes by name (fuzzy/regex)",
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
			list, err := svc.SearchThemes(context.Background(), core.ThemeSearchFilter{
				Query: args[0],
				Mode:  core.ThemeSearchMode(mode),
			})
			if err != nil {
				return mapError(err)
			}
			return o.printThemes(list)
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "fuzzy", "query match mode: fuzzy|regex")
	return cmd
}

func newThemeShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "show REF",
		Aliases: []string{"get"},
		Short:   "Show one theme by id or name",
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
			th, err := svc.GetTheme(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			return o.printTheme(th)
		},
	}
}

func newThemeCreateCmd() *cobra.Command {
	var from string
	var activate bool
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a theme (tokens from Slate factory, or --from REF)",
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

			tokens := core.SlateFactoryTokens()
			if strings.TrimSpace(from) != "" {
				src, err := svc.GetTheme(context.Background(), from)
				if err != nil {
					return mapError(err)
				}
				tokens = src.Tokens
			}
			th, err := svc.CreateTheme(context.Background(), core.ThemeInput{
				Name:   args[0],
				Tokens: tokens,
			})
			if err != nil {
				return mapError(err)
			}
			if activate {
				th, err = svc.ActivateTheme(context.Background(), th.ID)
				if err != nil {
					return mapError(err)
				}
			}
			return o.printTheme(th)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "copy tokens from an existing theme (id or name)")
	cmd.Flags().BoolVar(&activate, "activate", false, "activate the new theme after create")
	return cmd
}

func newThemeUpdateCmd() *cobra.Command {
	var name, tokensJSON string
	var sets []string
	cmd := &cobra.Command{
		Use:   "update REF",
		Short: "Update a theme (--name, --set KEY=VALUE, and/or --tokens-json)",
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

			existing, err := svc.GetTheme(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			in := core.ThemeInput{
				Name:   existing.Name,
				Tokens: cloneThemeTokens(existing.Tokens),
			}
			changed := false

			if cmd.Flags().Changed("name") {
				in.Name = name
				changed = true
			}
			if cmd.Flags().Changed("tokens-json") {
				var patch map[string]string
				if err := json.Unmarshal([]byte(tokensJSON), &patch); err != nil {
					return usageError("invalid --tokens-json: %v", err)
				}
				for k, v := range patch {
					in.Tokens[k] = v
				}
				changed = true
			}
			for _, s := range sets {
				k, v, ok := strings.Cut(s, "=")
				if !ok || strings.TrimSpace(k) == "" {
					return usageError("invalid --set %q (want KEY=VALUE)", s)
				}
				in.Tokens[strings.TrimSpace(k)] = v
				changed = true
			}
			if !changed {
				return usageError("theme update requires at least one of --name, --set, --tokens-json")
			}

			th, err := svc.UpdateTheme(context.Background(), existing.ID, in)
			if err != nil {
				return mapError(err)
			}
			return o.printTheme(th)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "rename the theme")
	cmd.Flags().StringArrayVar(&sets, "set", nil, "set a token (KEY=VALUE; repeatable)")
	cmd.Flags().StringVar(&tokensJSON, "tokens-json", "", "merge a JSON object of tokens")
	return cmd
}

func newThemeRmCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "rm REF",
		Aliases: []string{"remove", "delete"},
		Short:   "Delete a user theme (built-ins refused; non-TTY requires --yes)",
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

			th, err := svc.GetTheme(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			if th.BuiltinKey != nil {
				return mapError(core.ErrBuiltinTheme)
			}
			if !yes {
				if !stdinIsTTY() {
					return usageError("refusing to delete theme %q without --yes on a non-interactive terminal", th.Name)
				}
				fmt.Fprintf(o.out, "Delete theme %q? [y/N] ", th.Name)
				line, _ := bufio.NewReader(Stdin).ReadString('\n')
				if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
					_, err := fmt.Fprintln(o.out, "aborted")
					return err
				}
			}

			del, err := svc.DeleteTheme(context.Background(), th.ID)
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

func newThemeActivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "activate REF",
		Short: "Set the active GUI theme",
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
			th, err := svc.ActivateTheme(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			return o.printTheme(th)
		},
	}
}

func newThemeDuplicateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "duplicate REF [NAME]",
		Short: "Copy a theme into a new user theme",
		Args:  cobra.RangeArgs(1, 2),
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
			name := ""
			if len(args) == 2 {
				name = args[1]
			}
			th, err := svc.DuplicateTheme(context.Background(), args[0], name)
			if err != nil {
				return mapError(err)
			}
			return o.printTheme(th)
		},
	}
}

func newThemeResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset REF",
		Short: "Restore factory tokens for a built-in theme",
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
			th, err := svc.ResetTheme(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			return o.printTheme(th)
		},
	}
}

func cloneThemeTokens(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (o opts) printTheme(t core.Theme) error {
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
		bk := "—"
		if t.BuiltinKey != nil {
			bk = *t.BuiltinKey
		}
		fmt.Fprintf(w, "Builtin    \t%s\n", bk)
		fmt.Fprintf(w, "Active     \t%t\n", t.Active)
		fmt.Fprintf(w, "Created    \t%s\n", absTS(t.CreatedAt))
		fmt.Fprintf(w, "Updated    \t%s\n", absTS(t.UpdatedAt))
		if err := w.Flush(); err != nil {
			return err
		}
		keys := make([]string, 0, len(t.Tokens))
		for k := range t.Tokens {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintln(o.out, "Tokens:")
		for _, k := range keys {
			fmt.Fprintf(o.out, "  %s = %s\n", k, t.Tokens[k])
		}
		return nil
	}
}

func (o opts) printThemes(list []core.Theme) error {
	if o.json {
		if list == nil {
			list = []core.Theme{}
		}
		return o.printJSON(list)
	}
	w := newTabWriter(o.out)
	for _, t := range list {
		mark := " "
		if t.Active {
			mark = "*"
		}
		bk := "user"
		if t.BuiltinKey != nil {
			bk = *t.BuiltinKey
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", mark, shortID(t.ID), t.Name, bk)
	}
	return w.Flush()
}
