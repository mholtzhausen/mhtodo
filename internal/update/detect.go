package update

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	AppName     = "mhtodo"
	OwnerRepo   = "mholtzhausen/mhtodo"
	ServiceUnit = "mhtodo.service"
)

// InstallInfo describes where this binary lives and whether a user systemd
// unit is attached to *this* install.
type InstallInfo struct {
	// Executable is the resolved path of the running binary.
	Executable string
	// Prefix is the install prefix when layout is $PREFIX/bin/mhtodo
	// (e.g. ~/.local). Empty when the binary is not under a standard layout.
	Prefix string
	// HasService is true when the user unit's ExecStart points at Executable.
	HasService bool
	// UnitPath is ~/.config/systemd/user/mhtodo.service (even if absent).
	UnitPath string
	Arch     string // amd64 | arm64
}

// DetectInstall inspects the running binary and optional systemd user unit.
func DetectInstall() (InstallInfo, error) {
	exe, err := os.Executable()
	if err != nil {
		return InstallInfo{}, fmt.Errorf("resolve executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return InstallInfo{}, fmt.Errorf("resolve executable symlink: %w", err)
	}
	info := InstallInfo{
		Executable: exe,
		Arch:       goarch(),
		UnitPath:   filepath.Join(userHome(), ".config", "systemd", "user", ServiceUnit),
	}
	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "bin" && filepath.Base(exe) == AppName {
		info.Prefix = filepath.Dir(dir)
	}
	info.HasService = serviceAttached(info.UnitPath, exe)
	return info, nil
}

func goarch() string {
	switch runtime.GOARCH {
	case "amd64", "arm64":
		return runtime.GOARCH
	default:
		return runtime.GOARCH
	}
}

func userHome() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return h
	}
	return os.Getenv("HOME")
}

// serviceAttached is true only when the unit file's ExecStart binary resolves
// to the same path as exe — so a stray unit does not hijack `go run` / temp builds.
func serviceAttached(unitPath, exe string) bool {
	body, err := os.ReadFile(unitPath)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ExecStart=") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "ExecStart="))
		if len(fields) == 0 {
			return false
		}
		unitExe := fields[0]
		if resolved, err := filepath.EvalSymlinks(unitExe); err == nil {
			unitExe = resolved
		}
		return unitExe == exe
	}
	return false
}

// IsEphemeralInstall reports throwaway build paths (go run / go test) that
// should not be self-updated.
func IsEphemeralInstall(exe string) bool {
	return strings.Contains(exe, "go-build")
}

// DesktopPath / IconPath return the standard XDG paths under prefix.
func DesktopPath(prefix string) string {
	return filepath.Join(prefix, "share", "applications", AppName+".desktop")
}

func IconPath(prefix string) string {
	return filepath.Join(prefix, "share", "icons", "hicolor", "512x512", "apps", AppName+".png")
}
