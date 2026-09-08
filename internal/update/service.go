package update

import (
	"fmt"
	"os"
)

// ServiceAction is a lifecycle verb for the user systemd unit.
type ServiceAction string

const (
	ServiceInstall   ServiceAction = "install"
	ServiceStop      ServiceAction = "stop"
	ServiceStart     ServiceAction = "start"
	ServiceRestart   ServiceAction = "restart"
	ServiceUninstall ServiceAction = "uninstall"
)

// ServiceOptions controls ManageService.
type ServiceOptions struct {
	Action  ServiceAction
	Detect  func() (InstallInfo, error) // nil → DetectInstall
	Service *ServiceOps                 // nil → DefaultServiceOps
}

// ServiceResult is the --json envelope for `mhtodo service …`.
type ServiceResult struct {
	Action     string `json:"action"`
	Unit       string `json:"unit"`
	UnitPath   string `json:"unit_path"`
	Executable string `json:"executable"`
	Installed  bool   `json:"installed"`
	Message    string `json:"message"`
}

// ParseServiceAction maps a CLI verb to ServiceAction.
func ParseServiceAction(s string) (ServiceAction, error) {
	switch ServiceAction(s) {
	case ServiceInstall, ServiceStop, ServiceStart, ServiceRestart, ServiceUninstall:
		return ServiceAction(s), nil
	default:
		return "", fmt.Errorf("unknown service action %q (want install|stop|start|restart|uninstall)", s)
	}
}

// ManageService runs one user-unit lifecycle action for this install.
func ManageService(opts ServiceOptions) (ServiceResult, error) {
	detect := opts.Detect
	if detect == nil {
		detect = DetectInstall
	}
	info, err := detect()
	if err != nil {
		return ServiceResult{}, err
	}

	ops := DefaultServiceOps()
	if opts.Service != nil {
		ops = *opts.Service
	}

	res := ServiceResult{
		Action:     string(opts.Action),
		Unit:       ServiceUnit,
		UnitPath:   info.UnitPath,
		Executable: info.Executable,
		Installed:  unitFilePresent(info.UnitPath),
	}

	switch opts.Action {
	case ServiceInstall:
		if IsEphemeralInstall(info.Executable) {
			return res, fmt.Errorf("refusing to install service for ephemeral binary %s (install with make install / install.sh first)", info.Executable)
		}
		if err := ReinstallService(ops, info); err != nil {
			return res, err
		}
		res.Installed = true
		res.Message = fmt.Sprintf("installed and started %s", ServiceUnit)
	case ServiceStop:
		if err := requireUnit(info.UnitPath); err != nil {
			return res, err
		}
		if ops.Stop == nil {
			return res, fmt.Errorf("stop not configured")
		}
		if err := ops.Stop(); err != nil {
			return res, err
		}
		res.Message = fmt.Sprintf("stopped %s", ServiceUnit)
	case ServiceStart:
		if err := requireUnit(info.UnitPath); err != nil {
			return res, err
		}
		if ops.ImportEnv != nil {
			_ = ops.ImportEnv()
		}
		if ops.Start == nil {
			return res, fmt.Errorf("start not configured")
		}
		if err := ops.Start(); err != nil {
			return res, err
		}
		res.Message = fmt.Sprintf("started %s", ServiceUnit)
	case ServiceRestart:
		if err := requireUnit(info.UnitPath); err != nil {
			return res, err
		}
		if ops.ImportEnv != nil {
			_ = ops.ImportEnv()
		}
		if ops.Restart == nil {
			return res, fmt.Errorf("restart not configured")
		}
		if err := ops.Restart(); err != nil {
			return res, err
		}
		res.Message = fmt.Sprintf("restarted %s", ServiceUnit)
	case ServiceUninstall:
		if !unitFilePresent(info.UnitPath) {
			res.Installed = false
			res.Message = fmt.Sprintf("%s is not installed", ServiceUnit)
			return res, nil
		}
		if err := UninstallService(ops, info); err != nil {
			return res, err
		}
		res.Installed = false
		res.Message = fmt.Sprintf("uninstalled %s", ServiceUnit)
	default:
		return res, fmt.Errorf("unknown service action %q", opts.Action)
	}
	return res, nil
}

// UninstallService disables the user unit, removes the unit file, and reloads.
func UninstallService(ops ServiceOps, info InstallInfo) error {
	if ops.DisableNow != nil {
		_ = ops.DisableNow()
	}
	if ops.RemoveUnit != nil {
		if err := ops.RemoveUnit(info.UnitPath); err != nil {
			return fmt.Errorf("remove unit: %w", err)
		}
	}
	if ops.DaemonReload != nil {
		if err := ops.DaemonReload(); err != nil {
			return err
		}
	}
	return nil
}

func unitFilePresent(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func requireUnit(path string) error {
	if unitFilePresent(path) {
		return nil
	}
	return fmt.Errorf("%s is not installed (run: mhtodo service install)", ServiceUnit)
}
