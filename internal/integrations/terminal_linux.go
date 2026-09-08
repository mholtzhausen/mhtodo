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
	return launchInTerminalPreferred("", commandLine, "")
}

// launchInTerminalPreferred prefers preferredBinary when set and found on PATH.
// windowTitle is applied when the emulator supports a title flag (empty = default).
func launchInTerminalPreferred(preferredBinary, commandLine, windowTitle string) (int, error) {
	commandLine = strings.TrimSpace(commandLine)
	if commandLine == "" {
		return 0, errors.New("empty command")
	}
	// Do not prefix with `exec`: commandLine often starts with `cd … && …` or
	// `create || resume`. `exec cd` fails (cd is a builtin), and `exec create`
	// replaces the shell so the `|| resume` fallback never runs.
	shellCmd := commandLine
	windowTitle = strings.TrimSpace(windowTitle)

	type launcher struct {
		name string
		args []string
	}
	names := []string{
		"xdg-terminal-exec",
		// --window forces a new window; otherwise gnome-terminal may only add a
		// tab to an existing (minimized / other-workspace) window.
		"gnome-terminal",
		"kgx",
		"konsole",
		"xfce4-terminal",
		"kitty",
		"alacritty",
		"wezterm",
		"xterm",
	}
	launchers := make([]launcher, 0, len(names))
	for _, name := range names {
		launchers = append(launchers, launcher{name: name, args: terminalArgsFor(name, shellCmd, windowTitle)})
	}

	if pref := strings.TrimSpace(preferredBinary); pref != "" {
		base := filepathBase(pref)
		prefArgs := terminalArgsFor(base, shellCmd, windowTitle)
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

func terminalArgsFor(name, shellCmd, windowTitle string) []string {
	title := strings.TrimSpace(windowTitle)
	switch name {
	case "gnome-terminal":
		if title != "" {
			return []string{"--window", "--title", title, "--", "bash", "-lc", shellCmd}
		}
		return []string{"--window", "--", "bash", "-lc", shellCmd}
	case "kgx":
		return []string{"--", "bash", "-lc", shellCmd}
	case "konsole":
		if title != "" {
			return []string{"-p", "tabtitle=" + title, "-e", "bash", "-lc", shellCmd}
		}
		return []string{"-e", "bash", "-lc", shellCmd}
	case "xterm":
		if title != "" {
			return []string{"-T", title, "-e", "bash", "-lc", shellCmd}
		}
		return []string{"-e", "bash", "-lc", shellCmd}
	case "xfce4-terminal":
		if title != "" {
			return []string{"--disable-server", "--title", title, "-e", "bash", "-lc", shellCmd}
		}
		return []string{"--disable-server", "-e", "bash", "-lc", shellCmd}
	case "alacritty":
		if title != "" {
			return []string{"--title", title, "-e", "bash", "-lc", shellCmd}
		}
		return []string{"-e", "bash", "-lc", shellCmd}
	case "wezterm":
		return []string{"start", "--", "bash", "-lc", shellCmd}
	case "kitty":
		if title != "" {
			return []string{"--title", title, "bash", "-lc", shellCmd}
		}
		return []string{"bash", "-lc", shellCmd}
	case "xdg-terminal-exec":
		return []string{"bash", "-lc", shellCmd}
	default:
		// Generic: many emulators accept -e; title often unsupported.
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
		if err := activateWindowForPIDWalk(pid, ""); err == nil {
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
	if err := activateWindowForPIDXdoTool(pid, ""); err == nil {
		return nil
	}
	return activateWindowForPIDWmctrl(pid, "")
}

// activateWindowForPIDPreferTitle raises a window for pid, preferring one whose
// title contains titleHint when the process owns multiple windows (e.g. gnome-terminal-server).
func activateWindowForPIDPreferTitle(pid int, titleHint string) error {
	titleHint = strings.TrimSpace(titleHint)
	if err := activateWindowForPIDXdoTool(pid, titleHint); err == nil {
		return nil
	}
	if err := activateWindowForPIDWmctrl(pid, titleHint); err == nil {
		return nil
	}
	return activateWindowForPID(pid)
}

func activateWindowForPIDXdoTool(pid int, titleHint string) error {
	path, err := exec.LookPath("xdotool")
	if err != nil {
		return err
	}
	idBytes, err := exec.Command(path, "search", "--pid", strconv.Itoa(pid)).Output()
	if err != nil || len(strings.TrimSpace(string(idBytes))) == 0 {
		return errHerdrWindowNotFound
	}
	ids := strings.Fields(strings.TrimSpace(string(idBytes)))
	if titleHint != "" && len(ids) > 1 {
		if preferred := filterWindowIDsByTitle(path, ids, titleHint); len(preferred) > 0 {
			ids = preferred
		}
	}
	for _, id := range ids {
		if err := raiseXWindow(path, id); err == nil {
			return nil
		}
	}
	return errHerdrWindowNotFound
}

func activateWindowForPIDWmctrl(pid int, titleHint string) error {
	path, err := exec.LookPath("wmctrl")
	if err != nil {
		return err
	}
	out, err := exec.Command(path, "-lp").Output()
	if err != nil {
		return err
	}
	pidStr := strconv.Itoa(pid)
	titleHint = strings.TrimSpace(titleHint)
	var fallback []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[2] != pidStr {
			continue
		}
		id := fields[0]
		if titleHint != "" && strings.Contains(line, titleHint) {
			if err := raiseWmctrlWindow(path, id); err == nil {
				return nil
			}
			continue
		}
		fallback = append(fallback, id)
	}
	for _, id := range fallback {
		if err := raiseWmctrlWindow(path, id); err == nil {
			return nil
		}
	}
	return errHerdrWindowNotFound
}

func activateWindowByTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errHerdrWindowNotFound
	}
	if err := activateWindowByTitleXdoTool(title); err == nil {
		return nil
	}
	return activateWindowByTitleWmctrl(title)
}

func activateWindowByTitleXdoTool(title string) error {
	path, err := exec.LookPath("xdotool")
	if err != nil {
		return err
	}
	// --name matches WM_NAME / title substring.
	idBytes, err := exec.Command(path, "search", "--name", title).Output()
	if err != nil || len(strings.TrimSpace(string(idBytes))) == 0 {
		return errHerdrWindowNotFound
	}
	for _, id := range strings.Fields(strings.TrimSpace(string(idBytes))) {
		if err := raiseXWindow(path, id); err == nil {
			return nil
		}
	}
	return errHerdrWindowNotFound
}

func activateWindowByTitleWmctrl(title string) error {
	path, err := exec.LookPath("wmctrl")
	if err != nil {
		return err
	}
	// -F exact title match; -a activates by title substring when -F fails.
	if err := exec.Command(path, "-F", "-a", title).Run(); err == nil {
		return nil
	}
	if err := exec.Command(path, "-a", title).Run(); err == nil {
		return nil
	}
	return errHerdrWindowNotFound
}

func filterWindowIDsByTitle(xdotoolPath string, ids []string, titleHint string) []string {
	var out []string
	for _, id := range ids {
		name, err := exec.Command(xdotoolPath, "getwindowname", id).Output()
		if err != nil {
			continue
		}
		if strings.Contains(string(name), titleHint) {
			out = append(out, id)
		}
	}
	return out
}

func raiseXWindow(xdotoolPath, id string) error {
	_ = exec.Command(xdotoolPath, "windowmap", id).Run()
	if err := exec.Command(xdotoolPath, "windowactivate", "--sync", id).Run(); err != nil {
		if err2 := exec.Command(xdotoolPath, "windowactivate", id).Run(); err2 != nil {
			return err2
		}
	}
	_ = exec.Command(xdotoolPath, "windowraise", id).Run()
	_ = exec.Command(xdotoolPath, "windowfocus", "--sync", id).Run()
	_ = exec.Command(xdotoolPath, "windowfocus", id).Run()
	return nil
}

func raiseWmctrlWindow(wmctrlPath, id string) error {
	// -R: move window to the current desktop and raise it.
	if err := exec.Command(wmctrlPath, "-i", "-R", id).Run(); err == nil {
		return nil
	}
	return exec.Command(wmctrlPath, "-i", "-a", id).Run()
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
				if err := raiseXWindow(path, id); err == nil {
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
		if err := raiseXWindow(path, id); err == nil {
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
