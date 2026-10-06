package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
	"mhtodo/internal/instance"
	"mhtodo/internal/platform"
)

// launchGUIFn starts a detached GUI process (bare mhtodo). Tests replace it.
var launchGUIFn = defaultLaunchGUI

// LaunchGUIForTest swaps the GUI launcher; restore with the returned func.
func LaunchGUIForTest(fn func() error) (restore func()) {
	prev := launchGUIFn
	if fn == nil {
		launchGUIFn = defaultLaunchGUI
	} else {
		launchGUIFn = fn
	}
	return func() { launchGUIFn = prev }
}

func defaultLaunchGUI() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	cmd := exec.Command(exe)
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := platform.StartDetached(cmd); err != nil {
		return fmt.Errorf("launch gui: %w", err)
	}
	return nil
}

func newOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open TARGET",
		Short: "Open the GUI focused on a task (id/prefix or mhtodo://task/…)",
		Long: `Raise the running mhtodo window and select the given task. If no instance
is running, starts the GUI. TARGET may be a task id (or unique prefix ≥ 4 chars)
or a deep link: mhtodo://task/{id}.

Desktop handlers should invoke: mhtodo open %u`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			ref, err := instance.ParseOpenTarget(args[0])
			if err != nil {
				return usageError("%v", err)
			}

			svc, closeDB, err := openService()
			if err != nil {
				return err
			}
			t, err := svc.Get(context.Background(), ref)
			closeDB()
			if err != nil {
				return mapError(err)
			}

			action, err := openTaskInGUI(t)
			if err != nil {
				return mapError(err)
			}

			if o.json {
				return o.printJSON(map[string]string{
					"id":     t.ID,
					"uri":    instance.TaskURI(core.ShortID(t.ID)),
					"action": action,
				})
			}
			if o.quiet {
				_, err = fmt.Fprintln(o.out, t.ID)
				return err
			}
			_, err = fmt.Fprintf(o.out, "%s  %s  (%s)\n", core.ShortID(t.ID), t.Title, action)
			return err
		},
	}
	return cmd
}

// openTaskInGUI focuses a running instance or launches the GUI with a pending
// focus request. Returns action "focused" or "launched".
func openTaskInGUI(t core.Task) (action string, err error) {
	id := t.ID
	if pid, ok := instance.RunningPID(); ok {
		if err := instance.RequestFocus(pid, id); err != nil {
			return "", err
		}
		return "focused", nil
	}
	if err := instance.WriteFocusRequest(id); err != nil {
		return "", err
	}
	if err := launchGUIFn(); err != nil {
		return "", err
	}
	// If a GUI was already running, the child exits after signaling — nudge again
	// so a lost race still delivers the focus file we just wrote.
	time.Sleep(150 * time.Millisecond)
	if pid, ok := instance.RunningPID(); ok {
		_ = instance.SignalFocus(pid)
	}
	return "launched", nil
}
