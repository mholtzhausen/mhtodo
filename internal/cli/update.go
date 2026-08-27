package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"mhtodo/internal/update"
)

// updateRun is the test seam for `mhtodo update` (swap in tests).
var updateRun = update.Run

// UpdateRunForTest swaps the update runner and returns a restore func.
func UpdateRunForTest(f func(update.Options) (update.Result, error)) (restore func()) {
	prev := updateRun
	updateRun = f
	return func() { updateRun = prev }
}

func newUpdateCmd(version string) *cobra.Command {
	var checkOnly, force bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check GitHub for a newer release and install it in place",
		Long: `Check https://github.com/mholtzhausen/mhtodo/releases for a newer
linux binary matching this machine's architecture. If one is available,
download it and replace the running install (binary, and desktop/icon when
under $PREFIX/bin/mhtodo). When a user systemd unit (mhtodo.service) is
present, stop it, rewrite the unit, and enable --now after the swap.

Auth: set GH_TOKEN or GITHUB_TOKEN for private-repo / rate-limit headroom.
Flags: --check reports only; --force reinstalls even when already current.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			res, err := updateRun(update.Options{
				CurrentVersion: version,
				CheckOnly:      checkOnly,
				Force:          force,
			})
			if err != nil {
				return &errExit{code: ExitStorage, name: "update", msg: err.Error()}
			}
			if o.json {
				return o.printJSON(res)
			}
			if o.quiet {
				if res.Updated {
					_, err = fmt.Fprintln(o.out, res.LatestVersion)
				}
				return err
			}
			_, err = fmt.Fprintln(o.out, res.Message)
			return err
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "only check; do not download or install")
	cmd.Flags().BoolVar(&force, "force", false, "reinstall even if already up to date")
	return cmd
}
