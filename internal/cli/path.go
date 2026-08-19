package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"mhtodo/internal/store"
)

func newPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Print the database file path ($MHTODO_DB_PATH or $XDG_DATA_HOME/mhtodo/mhtodo.db)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			p := store.DBPath()
			if o.json {
				b, err := json.Marshal(p)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(o.out, string(b))
				return err
			}
			_, err = fmt.Fprintln(o.out, p)
			return err
		},
	}
	return cmd
}
