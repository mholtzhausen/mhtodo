package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// InServiceCgroup reports whether this process runs inside mhtodo.service's
// cgroup (e.g. `mhtodo update` spawned by the GUI's Install button while the
// GUI runs as the user unit). Such an updater must not run inline: stopping
// the unit SIGTERMs the whole cgroup, killing the updater before it swaps the
// binary or restarts the unit.
func InServiceCgroup() bool {
	b, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return false
	}
	return serviceInCgroup(string(b))
}

// serviceInCgroup parses /proc/self/cgroup content (v1 or v2 lines of
// "id:controllers:path").
func serviceInCgroup(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 3)
		if len(parts) != 3 {
			continue
		}
		if strings.HasSuffix(parts[2], "/"+ServiceUnit) {
			return true
		}
	}
	return false
}

// handoffEnv is forwarded into the transient unit; systemd-run otherwise only
// passes the user manager's environment.
var handoffEnv = []string{
	"PATH", "HOME", "GH_TOKEN", "GITHUB_TOKEN", "MHTODO_SKIP_SKILL_REFRESH",
	"MHTODO_DB_PATH", "XDG_DATA_HOME", "XDG_CONFIG_HOME",
}

// HandoffArgs builds the systemd-run argv that re-runs `exe args…` in its own
// transient user unit, outside mhtodo.service's cgroup. No --wait/--pipe: the
// caller is about to be killed by the unit stop and must not take it along.
func HandoffArgs(unit, exe string, args []string, getenv func(string) string) []string {
	out := []string{"--user", "--collect", "--quiet", "--unit=" + unit}
	for _, k := range handoffEnv {
		if v := getenv(k); v != "" {
			out = append(out, "--setenv="+k+"="+v)
		}
	}
	out = append(out, exe)
	return append(out, args...)
}

// Handoff starts `<this binary> args…` via systemd-run and returns the
// transient unit name. The started process runs independently of the caller.
func Handoff(args []string) (string, error) {
	sr, err := exec.LookPath("systemd-run")
	if err != nil {
		return "", fmt.Errorf("systemd-run not found: %w", err)
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	unit := fmt.Sprintf("%s-update-%d", AppName, time.Now().Unix())
	out, err := exec.Command(sr, HandoffArgs(unit, exe, args, os.Getenv)...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("systemd-run: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return unit, nil
}
