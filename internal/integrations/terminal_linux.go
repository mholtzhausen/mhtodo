//go:build linux

package integrations

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// launchInTerminal opens a new terminal emulator running commandLine via bash -lc.
func launchInTerminal(commandLine string) (int, error) {
	return launchInTerminalPreferred("", commandLine)
}

// launchInTerminalPreferred prefers preferredBinary when set and found on PATH.
func launchInTerminalPreferred(preferredBinary, commandLine string) (int, error) {
	commandLine = strings.TrimSpace(commandLine)
	if commandLine == "" {
		return 0, errors.New("empty command")
	}
	shellCmd := "exec " + commandLine

	type launcher struct {
		name string
		args []string
	}
	launchers := []launcher{
		{"xdg-terminal-exec", []string{"bash", "-lc", shellCmd}},
		{"gnome-terminal", []string{"--", "bash", "-lc", shellCmd}},
		{"kgx", []string{"--", "bash", "-lc", shellCmd}},
		{"konsole", []string{"-e", "bash", "-lc", shellCmd}},
		{"xfce4-terminal", []string{"-e", "bash", "-lc", shellCmd}},
		{"kitty", []string{"bash", "-lc", shellCmd}},
		{"alacritty", []string{"-e", "bash", "-lc", shellCmd}},
		{"wezterm", []string{"start", "--", "bash", "-lc", shellCmd}},
		{"xterm", []string{"-e", "bash", "-lc", shellCmd}},
	}

	if pref := strings.TrimSpace(preferredBinary); pref != "" {
		base := filepathBase(pref)
		prefArgs := terminalArgsFor(base, shellCmd)
		// Try preferred first (by path or basename).
		ordered := make([]launcher, 0, len(launchers)+1)
		ordered = append(ordered, launcher{name: pref, args: prefArgs})
		for _, l := range launchers {
			if l.name == base || l.name == pref {
				continue
			}
			ordered = append(ordered, l)
		}
		launchers = ordered
	}

	var lastErr error
	for _, l := range launchers {
		path, err := exec.LookPath(l.name)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, l.args...)
		cmd.Env = os.Environ()
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			lastErr = err
			continue
		}
		pid := 0
		if cmd.Process != nil {
			pid = cmd.Process.Pid
		}
		// Detach: do not wait; release process so we do not leave a zombie on exit.
		go func() { _ = cmd.Wait() }()
		return pid, nil
	}
	if lastErr != nil {
		return 0, lastErr
	}
	return 0, errors.New("no terminal emulator found on PATH")
}

func terminalArgsFor(name, shellCmd string) []string {
	switch name {
	case "gnome-terminal", "kgx":
		return []string{"--", "bash", "-lc", shellCmd}
	case "konsole", "xfce4-terminal", "xterm":
		return []string{"-e", "bash", "-lc", shellCmd}
	case "alacritty":
		return []string{"-e", "bash", "-lc", shellCmd}
	case "wezterm":
		return []string{"start", "--", "bash", "-lc", shellCmd}
	case "xdg-terminal-exec", "kitty":
		return []string{"bash", "-lc", shellCmd}
	default:
		// Generic: many emulators accept -e.
		return []string{"-e", "bash", "-lc", shellCmd}
	}
}

func filepathBase(path string) string {
	path = strings.TrimSpace(path)
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

// activateHerdrWindow raises the terminal emulator hosting the Herdr TUI.
func activateHerdrWindow() error {
	if pid, ok := findHerdrTUIPID(); ok {
		if err := activateWindowForPIDWalk(pid); err == nil {
			return nil
		}
	}
	if err := activateViaWMCtrl(); err == nil {
		return nil
	}
	return activateViaXDoTool()
}

func execCommandOutput(name string, args ...string) ([]byte, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, err
	}
	return exec.Command(path, args...).Output()
}

func activateWindowForPID(pid int) error {
	path, err := exec.LookPath("xdotool")
	if err != nil {
		return err
	}
	idBytes, err := exec.Command(path, "search", "--pid", strconv.Itoa(pid)).Output()
	if err != nil || len(strings.TrimSpace(string(idBytes))) == 0 {
		return errHerdrWindowNotFound
	}
	for _, id := range strings.Fields(strings.TrimSpace(string(idBytes))) {
		if err := exec.Command(path, "windowactivate", id).Run(); err == nil {
			return nil
		}
	}
	return errHerdrWindowNotFound
}

func activateViaWMCtrl() error {
	path, err := exec.LookPath("wmctrl")
	if err != nil {
		return err
	}
	out, err := exec.Command(path, "-lx").Output()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(out), "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "herdr") &&
			!strings.Contains(lower, "gnome-terminal") &&
			!strings.Contains(lower, "konsole") &&
			!strings.Contains(lower, "kitty") &&
			!strings.Contains(lower, "alacritty") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if err := exec.Command(path, "-ia", fields[0]).Run(); err != nil {
			return err
		}
		return nil
	}
	return errHerdrWindowNotFound
}

func activateViaXDoTool() error {
	path, err := exec.LookPath("xdotool")
	if err != nil {
		return err
	}
	for _, class := range []string{"gnome-terminal", "Gnome-terminal", "konsole", "kitty", "Alacritty", "herdr"} {
		out, err := exec.Command(path, "search", "--onlyvisible", "--class", class).Output()
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			for _, id := range strings.Fields(strings.TrimSpace(string(out))) {
				if err := exec.Command(path, "windowactivate", id).Run(); err == nil {
					return nil
				}
			}
		}
	}
	out, err := exec.Command(path, "search", "--name", "herdr").Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return errHerdrWindowNotFound
	}
	for _, id := range strings.Fields(strings.TrimSpace(string(out))) {
		if err := exec.Command(path, "windowactivate", id).Run(); err == nil {
			return nil
		}
	}
	return errHerdrWindowNotFound
}

func processParentPID(pid int) (int, error) {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "PPid:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, fmt.Errorf("invalid PPid line for pid %d", pid)
		}
		return strconv.Atoi(fields[1])
	}
	return 0, fmt.Errorf("PPid not found for pid %d", pid)
}

func processSignalZero(p *os.Process) bool {
	if p == nil {
		return false
	}
	err := p.Signal(syscall.Signal(0))
	return err == nil
}

func killProcessBestEffort(pid int) error {
	if pid <= 0 {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	// Try process group first (negative PID), then the process itself.
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		_ = p.Signal(syscall.SIGTERM)
	}
	return nil
}
