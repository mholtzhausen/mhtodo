package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"mhtodo/internal/update"
)

// serviceManage is the test seam for `mhtodo service …`.
var serviceManage = update.ManageService

// ServiceManageForTest swaps the service manager and returns a restore func.
func ServiceManageForTest(f func(update.ServiceOptions) (update.ServiceResult, error)) (restore func()) {
	prev := serviceManage
	serviceManage = f
	return func() { serviceManage = prev }
}

func newServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage the user systemd unit (install|stop|start|restart|uninstall)",
		Long: `Manage ~/.config/systemd/user/mhtodo.service for the running install.

  install    write/enable the user unit (ExecStart=<this binary> gui) and start it
  stop       stop the unit
  start      start the unit
  restart    restart the unit
  uninstall  disable --now, remove the unit file, daemon-reload (binary stays)

mhtodo update still detects an attached unit and restarts it after a binary swap.
Bootstrap from source remains: make service-install.`,
	}
	for _, action := range []string{"install", "stop", "start", "restart", "uninstall"} {
		cmd.AddCommand(newServiceActionCmd(action))
	}
	return cmd
}

func newServiceActionCmd(action string) *cobra.Command {
	return &cobra.Command{
		Use:   action,
		Short: serviceActionShort(action),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			act, err := update.ParseServiceAction(action)
			if err != nil {
				return usageError("%s", err.Error())
			}
			res, err := serviceManage(update.ServiceOptions{Action: act})
			if err != nil {
				return &errExit{code: ExitStorage, name: "service", msg: err.Error()}
			}
			if o.json {
				return o.printJSON(res)
			}
			if o.quiet {
				return nil
			}
			_, err = fmt.Fprintln(o.out, res.Message)
			return err
		},
	}
}

func serviceActionShort(action string) string {
	switch action {
	case "install":
		return "Install and start the user systemd unit for this binary"
	case "stop":
		return "Stop the user systemd unit"
	case "start":
		return "Start the user systemd unit"
	case "restart":
		return "Restart the user systemd unit"
	case "uninstall":
		return "Disable and remove the user systemd unit (binary stays)"
	default:
		return action
	}
}
