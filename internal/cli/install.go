package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"mhtodo/internal/update"
)

// installLocal is the test seam for the folder-file copy step.
var installLocal = update.InstallLocal

// InstallLocalForTest swaps InstallLocal and returns a restore func.
func InstallLocalForTest(f func(update.LocalInstallOptions) (update.LocalInstallResult, error)) (restore func()) {
	prev := installLocal
	installLocal = f
	return func() { installLocal = prev }
}

func newInstallCmd() *cobra.Command {
	var (
		prefix      string
		wantService bool
		noService   bool
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install this binary into ~/.local, optionally as a user service",
		Long: `Copy the running mhtodo binary into $PREFIX (default ~/.local) with a
desktop launcher and icon — the same layout as make install.

Then, on a TTY, ask whether to install the user systemd unit
(mhtodo service install).

Non-interactive use: pass --service / --no-service
(omitted optional steps are skipped on non-TTY).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			if wantService && noService {
				return usageError("pass only one of --service or --no-service")
			}

			files, err := installLocal(update.LocalInstallOptions{Prefix: prefix})
			if err != nil {
				return &errExit{code: ExitStorage, name: "install", msg: err.Error()}
			}

			info := update.InstallInfoForPrefix(files.Prefix, files.UnitPath)
			doService, err := resolveServiceChoice(wantService, noService, o)
			if err != nil {
				return err
			}

			result := installResult{
				Prefix:     files.Prefix,
				Executable: files.Executable,
				Desktop:    files.Desktop,
				Icon:       files.Icon,
				Service:    false,
				Message:    files.Message,
			}

			if doService {
				svcRes, err := serviceManage(update.ServiceOptions{
					Action: update.ServiceInstall,
					Detect: func() (update.InstallInfo, error) { return info, nil },
				})
				if err != nil {
					return &errExit{code: ExitStorage, name: "service", msg: err.Error()}
				}
				result.Service = true
				result.ServiceMessage = svcRes.Message
				result.Message = files.Message + "; " + svcRes.Message
			}

			if o.json {
				return o.printJSON(result)
			}
			if o.quiet {
				_, err = fmt.Fprintln(o.out, result.Executable)
				return err
			}
			_, err = fmt.Fprintln(o.out, result.Message)
			return err
		},
	}
	cmd.Flags().StringVar(&prefix, "prefix", "", "install prefix (default: ~/.local)")
	cmd.Flags().BoolVar(&wantService, "service", false, "install user systemd unit (skip prompt)")
	cmd.Flags().BoolVar(&noService, "no-service", false, "skip systemd unit (skip prompt)")
	return cmd
}

type installResult struct {
	Prefix         string `json:"prefix"`
	Executable     string `json:"executable"`
	Desktop        string `json:"desktop,omitempty"`
	Icon           string `json:"icon,omitempty"`
	Service        bool   `json:"service"`
	ServiceMessage string `json:"service_message,omitempty"`
	Message        string `json:"message"`
}

func resolveServiceChoice(want, no bool, o opts) (bool, error) {
	if want {
		return true, nil
	}
	if no {
		return false, nil
	}
	if !stdinIsTTY() {
		return false, nil
	}
	ok, err := askYesNo(o, "Install as a user systemd service (start at login)? [y/N] ", false)
	return ok, err
}

func askYesNo(o opts, prompt string, defaultYes bool) (bool, error) {
	if _, err := fmt.Fprint(o.out, prompt); err != nil {
		return false, err
	}
	line, err := bufio.NewReader(Stdin).ReadString('\n')
	if err != nil && len(strings.TrimSpace(line)) == 0 {
		return defaultYes, nil // EOF → default
	}
	a := strings.ToLower(strings.TrimSpace(line))
	if a == "" {
		return defaultYes, nil
	}
	return a == "y" || a == "yes", nil
}
