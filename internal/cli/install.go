package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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
		integration string // bash|zsh|none; empty = prompt (TTY) or skip (non-TTY)
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install this binary into ~/.local, optionally as a service + shell helper",
		Long: `Copy the running mhtodo binary into $PREFIX (default ~/.local) with a
desktop launcher and icon — the same layout as make install.

Then, on a TTY, ask whether to:
  • install the user systemd unit (mhtodo service install)
  • install the claude.todo shell helper (mhtodo integration bash|zsh)

Non-interactive use: pass --service / --no-service and
--integration bash|zsh|none (omitted optional steps are skipped on non-TTY).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			if wantService && noService {
				return usageError("pass only one of --service or --no-service")
			}
			if integration != "" {
				switch integration {
				case "bash", "zsh", "none":
				default:
					return usageError("--integration must be bash, zsh, or none")
				}
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
			shell, err := resolveIntegrationChoice(integration, o)
			if err != nil {
				return err
			}

			result := installResult{
				Prefix:      files.Prefix,
				Executable:  files.Executable,
				Desktop:     files.Desktop,
				Icon:        files.Icon,
				Service:     false,
				Integration: shell,
				Message:     files.Message,
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

			if shell != "" && shell != "none" {
				rcName := "." + shell + "rc"
				path, err := shellRCPath(rcName)
				if err != nil {
					return &errExit{code: ExitStorage, name: "integration", msg: err.Error()}
				}
				if err := upsertShellBlock(path); err != nil {
					return &errExit{code: ExitStorage, name: "integration", msg: err.Error()}
				}
				msg := fmt.Sprintf("updated %s integration in %s", shell, path)
				result.IntegrationPath = path
				result.IntegrationMessage = msg
				result.Message = result.Message + "; " + msg
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
	cmd.Flags().StringVar(&integration, "integration", "", "bash|zsh|none — install shell helper (skip prompt)")
	return cmd
}

type installResult struct {
	Prefix             string `json:"prefix"`
	Executable         string `json:"executable"`
	Desktop            string `json:"desktop,omitempty"`
	Icon               string `json:"icon,omitempty"`
	Service            bool   `json:"service"`
	ServiceMessage     string `json:"service_message,omitempty"`
	Integration        string `json:"integration"` // bash|zsh|none|""
	IntegrationPath    string `json:"integration_path,omitempty"`
	IntegrationMessage string `json:"integration_message,omitempty"`
	Message            string `json:"message"`
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

func resolveIntegrationChoice(flag string, o opts) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if !stdinIsTTY() {
		return "none", nil
	}
	shell := detectLoginShell()
	prompt := fmt.Sprintf("Install claude.todo shell helper into ~/.%src? [y/N] ", shell)
	ok, err := askYesNo(o, prompt, false)
	if err != nil {
		return "", err
	}
	if !ok {
		return "none", nil
	}
	return shell, nil
}

func detectLoginShell() string {
	base := filepath.Base(os.Getenv("SHELL"))
	switch base {
	case "bash", "zsh":
		return base
	default:
		return "bash"
	}
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
