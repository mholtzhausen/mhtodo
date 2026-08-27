package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "rm ID",
		Aliases: []string{"remove"},
		Short:   "Delete a task (non-interactive shells require --yes); cascades to sub-tasks",
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

			t, err := svc.Get(context.Background(), args[0])
			if err != nil {
				return mapError(err)
			}
			nChild, err := svc.CountChildren(context.Background(), t.ID)
			if err != nil {
				return mapError(err)
			}

			if !yes {
				if !stdinIsTTY() {
					return usageError("refusing to delete %s without --yes on a non-interactive terminal", shortID(t.ID))
				}
				prompt := fmt.Sprintf("Delete task %s %q? [y/N] ", shortID(t.ID), t.Title)
				if nChild > 0 {
					prompt = fmt.Sprintf("Delete task %s %q and its %d sub-task(s)? [y/N] ",
						shortID(t.ID), t.Title, nChild)
				}
				fmt.Fprint(o.out, prompt)
				line, _ := bufio.NewReader(Stdin).ReadString('\n')
				if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
					_, err := fmt.Fprintln(o.out, "aborted") // user declined: not an error
					return err
				}
			}

			del, err := svc.Delete(context.Background(), t.ID)
			if err != nil {
				return mapError(err)
			}
			if o.json {
				return o.printJSON(map[string]string{"id": del.ID}) // spec: prints the deleted task's id only
			}
			_, err = fmt.Fprintln(o.out, del.ID)
			return err
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation (required on non-TTY)")
	return cmd
}

// stdinIsTTY reports whether Stdin is an interactive terminal. Non-*os.File
// readers (the test seam) are never TTYs.
func stdinIsTTY() bool {
	f, ok := Stdin.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}
