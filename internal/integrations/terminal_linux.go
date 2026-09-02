//go:build linux

package integrations

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// launchInTerminal opens a new terminal emulator running commandLine via bash -lc.
func launchInTerminal(commandLine string) error {
	commandLine = strings.TrimSpace(commandLine)
	if commandLine == "" {
		return errors.New("empty command")
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
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("no terminal emulator found on PATH")
}

// activateHerdrWindow raises the terminal emulator hosting the Herdr TUI.
func activateHerdrWindow() error {
	if pid, ok := findHerdrTUIPID(); ok {
		for walk := pid; walk > 1; {
			if err := activateWindowForPID(walk); err == nil {
				return nil
			}
			parent, err := processParentPID(walk)
			if err != nil || parent <= 1 || parent == walk {
				break
			}
			walk = parent
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
