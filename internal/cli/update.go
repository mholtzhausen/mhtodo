package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"mhtodo/internal/aiskill"
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

// updateJSON is the --json envelope: binary update result plus optional skill refresh.
type updateJSON struct {
	update.Result
	SkillPath           string   `json:"skill_path,omitempty"`
	SkillAction         string   `json:"skill_action,omitempty"`
	SkillRemoved        []string `json:"skill_removed,omitempty"`
	SkillRefreshSkipped bool     `json:"skill_refresh_skipped,omitempty"`
	SkillRefreshError   string   `json:"skill_refresh_error,omitempty"`
}

func newUpdateCmd(version string) *cobra.Command {
	var checkOnly, force, noSkillRefresh bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check GitHub for a newer release and install it in place",
		Long: `Check https://github.com/mholtzhausen/mhtodo/releases for a newer
linux binary matching this machine's architecture. If one is available,
download it and replace the running install (binary, and desktop/icon when
under $PREFIX/bin/mhtodo). When a user systemd unit (mhtodo.service) is
present, stop it, rewrite the unit, and enable --now after the swap.

After a successful binary update, installs/updates the Claude mhtodo skill
(mhtodo ai) and, when claude is on PATH, runs a non-interactive skill refresh.
Progress for the skill steps is printed to the console before they start
(human output only); the Claude refresh can take up to a few minutes.
Skill refresh failures are warnings only.

Auth: set GH_TOKEN or GITHUB_TOKEN for private-repo / rate-limit headroom.
Flags: --check reports only; --force reinstalls even when already current;
--no-skill-refresh skips the optional Claude pass (skill install still runs).`,
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

			env := updateJSON{Result: res}
			human := !o.json && !o.quiet
			notify := func(msg string) {
				if !human {
					return
				}
				_, _ = fmt.Fprintln(o.out, msg)
			}

			if res.Updated && !checkOnly {
				// Print binary result first so skill steps are visibly after it.
				if human {
					if _, err = fmt.Fprintln(o.out, res.Message); err != nil {
						return err
					}
				}

				notify("Updating Claude skill…")
				skillRes, skillErr := aiskill.Install()
				if skillErr != nil {
					env.SkillRefreshError = skillErr.Error()
				} else {
					env.SkillPath = skillRes.Path
					env.SkillAction = skillRes.Action
					env.SkillRemoved = skillRes.Removed
					skipClaude := noSkillRefresh ||
						os.Getenv("MHTODO_SKIP_SKILL_REFRESH") == "1" ||
						os.Getenv("MHTODO_SKIP_SKILL_REFRESH") == "true"
					if skipClaude || !aiskill.ClaudeAvailable() {
						env.SkillRefreshSkipped = true
					} else {
						notify(fmt.Sprintf(
							"Refreshing Claude skill via claude (may take up to %s)…",
							aiskill.DefaultRefreshTimeout,
						))
						_, skipped, refreshErr := aiskill.RefreshClaude(aiskill.DefaultRefreshTimeout)
						env.SkillRefreshSkipped = skipped
						if refreshErr != nil {
							env.SkillRefreshError = refreshErr.Error()
						}
					}
				}
			}

			if o.json {
				return o.printJSON(env)
			}
			if o.quiet {
				if res.Updated {
					_, err = fmt.Fprintln(o.out, res.LatestVersion)
				}
				return err
			}
			// Binary message already printed when Updated; print it for check/no-op.
			if !(res.Updated && !checkOnly) {
				if _, err = fmt.Fprintln(o.out, res.Message); err != nil {
					return err
				}
			}
			if res.Updated && !checkOnly {
				if env.SkillPath != "" {
					msg := fmt.Sprintf("skill %s at %s", env.SkillAction, env.SkillPath)
					if len(env.SkillRemoved) == 0 {
						msg += " (nothing to clean)"
					} else {
						msg += fmt.Sprintf(" (removed %d legacy artifact(s))", len(env.SkillRemoved))
					}
					_, _ = fmt.Fprintln(o.out, msg)
				} else if env.SkillRefreshError != "" && env.SkillPath == "" {
					_, _ = fmt.Fprintln(o.out, "warning: skill install failed: "+env.SkillRefreshError)
				}
				if env.SkillPath != "" && env.SkillRefreshError != "" {
					_, _ = fmt.Fprintln(o.out, formatSkillRefreshWarn(env.SkillRefreshError))
				} else if env.SkillPath != "" && !env.SkillRefreshSkipped && env.SkillRefreshError == "" {
					_, _ = fmt.Fprintln(o.out, "skill refresh via claude: ok")
				} else if env.SkillPath != "" && env.SkillRefreshSkipped {
					_, _ = fmt.Fprintln(o.out, "skill refresh via claude: skipped")
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "only check; do not download or install")
	cmd.Flags().BoolVar(&force, "force", false, "reinstall even if already up to date")
	cmd.Flags().BoolVar(&noSkillRefresh, "no-skill-refresh", false, "skip optional claude -p skill refresh after update")
	return cmd
}
