package integrations

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mhtodo/internal/core"
	"mhtodo/internal/settings"
)

// OpenTerminalSession focuses an existing mhtodo-managed Claude terminal when
// the session process (or titled window) is still alive; otherwise launches
// Claude in a new terminal and returns the PID to persist on the task.
func (c Client) OpenTerminalSession(cwd, shortID, title, todoSession string, storedPID int) (int, error) {
	if settings.NormalizeSpawn(c.Claude.Spawn) != settings.SpawnTerminal {
		return 0, fmt.Errorf("Claude spawn mode is not terminal")
	}
	if !c.ClaudeFound() {
		return 0, fmt.Errorf("claude binary not found: %s", strings.TrimSpace(c.Claude.Binary))
	}

	session := strings.TrimSpace(todoSession)
	displayName := core.ClaudeDisplayName(shortID, title, session)
	if session == "" {
		session = displayName
	}
	winTitle := mhtodoTerminalTitle(session)

	if pid := resolveLiveSessionPID(storedPID, session); pid > 0 {
		if err := activateMHTODOTerminal(pid, winTitle); err == nil {
			return pid, nil
		}
		// Process is alive with our session but has no raisable window (e.g. a
		// Herdr pane shell left over from a prior spawn mode). Fall through and
		// open a real terminal emulator instead of silently succeeding.
	} else if err := activateWindowByTitle(winTitle); err == nil {
		// Window still up (title match) even if the stored PID went stale —
		// do not spawn a second terminal.
		if found, ok := findPIDByMHTODOSession(session, 0); ok {
			return found, nil
		}
		if storedPID > 0 {
			return storedPID, nil
		}
		return 0, nil
	}

	cmdLine := c.claudeTerminalCommandLine(cwd, shortID, title, session, displayName)
	launchPID, err := launchInTerminalPreferred(c.Terminal.Binary, cmdLine, winTitle)
	if err != nil {
		return 0, err
	}

	// Prefer a Claude process for this session, skipping the stale stored PID
	// that may still hold MHTODO_SESSION inside Herdr.
	pid := launchPID
	if found, ok := waitForSessionPID(session, storedPID, 3*time.Second); ok {
		pid = found
	} else if launchPID > 0 && processAlive(launchPID) {
		pid = launchPID
	}
	if pid > 0 || winTitle != "" {
		_ = activateMHTODOTerminal(pid, winTitle)
	}
	return pid, nil
}

// MaybeCloseTerminalSessionOnDone kills the managed terminal when
// close_tab_on_done is enabled. Returns true when the caller should clear terminal_pid.
func (c Client) MaybeCloseTerminalSessionOnDone(terminalPID int) bool {
	if !c.Claude.CloseTabOnDone {
		return false
	}
	if settings.NormalizeSpawn(c.Claude.Spawn) != settings.SpawnTerminal {
		return false
	}
	if terminalPID <= 0 {
		return true // clear stale zero-or-negative
	}
	_ = killProcessBestEffort(terminalPID)
	return true
}

// MaybeCloseTicketTabOnDone closes the Herdr tab for a task when Claude
// close_tab_on_done is enabled and spawn is herdr. Prefer MaybeCloseSessionOnDone
// when terminal_pid may need clearing.
func (c Client) MaybeCloseTicketTabOnDone(taskID, shortID, title string) {
	spawn := settings.NormalizeSpawn(c.Claude.Spawn)
	if spawn == settings.SpawnDisabled && c.Herdr.Enabled {
		// Legacy callers that only set herdr.enabled.
		spawn = settings.SpawnHerdr
	}
	if spawn != settings.SpawnHerdr || !c.Claude.CloseTabOnDone {
		return
	}
	if !c.Herdr.Enabled || !c.HerdrFound() {
		return
	}
	_ = c.CloseTicketTab(taskID, shortID, title)
}

// MaybeCloseSessionOnDone closes Herdr tab or terminal based on spawn mode.
// clearTerminalPID is invoked when a terminal PID should be cleared from the task.
func (c Client) MaybeCloseSessionOnDone(taskID, shortID, title string, terminalPID int, clearTerminalPID func()) {
	spawn := settings.NormalizeSpawn(c.Claude.Spawn)
	if spawn == settings.SpawnDisabled && c.Herdr.Enabled {
		spawn = settings.SpawnHerdr
	}
	switch spawn {
	case settings.SpawnHerdr:
		c.MaybeCloseTicketTabOnDone(taskID, shortID, title)
	case settings.SpawnTerminal:
		if c.MaybeCloseTerminalSessionOnDone(terminalPID) && clearTerminalPID != nil {
			clearTerminalPID()
		}
	}
}

func (c Client) claudeTerminalCommandLine(cwd, shortID, title, sessionUUID, displayName string) string {
	prompt := c.ticketPrompt(shortID)
	bin := strings.TrimSpace(c.Claude.Binary)
	if bin == "" {
		bin = "claude"
	}
	sessionUUID = strings.TrimSpace(sessionUUID)
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = core.ClaudeDisplayName(shortID, title, sessionUUID)
	}
	if sessionUUID == "" {
		sessionUUID = displayName
	}

	claudeEnv, claudePrefix := ParseEnvStart(c.Claude.EnvStart)
	termEnv, termPrefix := ParseEnvStart(c.Terminal.EnvStart)

	var parts []string
	if winTitle := mhtodoTerminalTitle(sessionUUID); winTitle != "" {
		// OSC 0 so emulators without a --title flag still get a searchable name.
		// Use a quoted escape sequence (not raw ESC/BEL bytes) for a safe bash -lc line.
		parts = append(parts, "printf", ShellDoubleQuote(`\033]0;`+winTitle+`\007`), ";")
	}
	if d := strings.TrimSpace(cwd); d != "" {
		parts = append(parts, "cd", shellWord(d), "&&")
	}
	for _, e := range ClaudeSessionEnv(sessionUUID, displayName) {
		parts = append(parts, shellEnvAssign(e))
	}
	for _, e := range append(append([]string{}, termEnv...), claudeEnv...) {
		parts = append(parts, shellWord(e))
	}
	for _, a := range append(append([]string{}, termPrefix...), claudePrefix...) {
		parts = append(parts, shellWord(a))
	}

	parts = append(parts, ClaudeLaunchShell(bin, sessionUUID, displayName, prompt))
	return strings.Join(parts, " ")
}

// mhtodoTerminalTitle is the WM title used to find an existing Claude terminal.
func mhtodoTerminalTitle(session string) string {
	session = strings.TrimSpace(session)
	if session == "" {
		return ""
	}
	return "mhtodo:" + session
}

// activateMHTODOTerminal raises the Claude terminal by title (precise) then PID walk.
func activateMHTODOTerminal(pid int, winTitle string) error {
	winTitle = strings.TrimSpace(winTitle)
	if winTitle != "" {
		if err := activateWindowByTitle(winTitle); err == nil {
			return nil
		}
	}
	if pid > 0 {
		if err := activateWindowForPIDWalk(pid, winTitle); err == nil {
			return nil
		}
	}
	return errHerdrWindowNotFound
}

func resolveLiveSessionPID(storedPID int, session string) int {
	session = strings.TrimSpace(session)
	if storedPID > 0 && processAlive(storedPID) && processHasMHTODOSession(storedPID, session) {
		return storedPID
	}
	if found, ok := findPIDByMHTODOSession(session, 0); ok {
		return found
	}
	return 0
}

func activateWindowForPIDWalk(pid int, titleHint string) error {
	self := os.Getpid()
	for walk := pid; walk > 1 && walk != self; {
		if err := activateWindowForPIDPreferTitle(walk, titleHint); err == nil {
			return nil
		}
		parent, err := processParentPID(walk)
		if err != nil || parent <= 1 || parent == walk || parent == self {
			break
		}
		walk = parent
	}
	return errHerdrWindowNotFound
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return processSignalZero(p)
}

func processHasMHTODOSession(pid int, session string) bool {
	session = strings.TrimSpace(session)
	if pid <= 0 || session == "" {
		return false
	}
	env, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if err != nil {
		return false
	}
	return environContains(env, "MHTODO_SESSION="+session)
}

func waitForSessionPID(session string, excludePID int, timeout time.Duration) (int, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if pid, ok := findPIDByMHTODOSession(session, excludePID); ok {
			return pid, true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return 0, false
}

func findPIDByMHTODOSession(session string, excludePID int) (int, bool) {
	session = strings.TrimSpace(session)
	if session == "" {
		return 0, false
	}
	want := "MHTODO_SESSION=" + session
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, false
	}
	self := os.Getpid()
	var best int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 1 || pid == self || pid == excludePID {
			continue
		}
		env, err := os.ReadFile(filepath.Join("/proc", e.Name(), "environ"))
		if err != nil {
			continue
		}
		if !environContains(env, want) {
			continue
		}
		// Prefer a process that looks like Claude when several match.
		if processCmdlineLooksLikeClaude(pid) {
			return pid, true
		}
		if best == 0 {
			best = pid
		}
	}
	if best > 0 {
		return best, true
	}
	return 0, false
}

func environContains(environ []byte, want string) bool {
	for _, kv := range strings.Split(string(environ), "\x00") {
		if kv == want {
			return true
		}
	}
	return false
}

func processCmdlineLooksLikeClaude(pid int) bool {
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return false
	}
	cmd := strings.ToLower(strings.ReplaceAll(string(raw), "\x00", " "))
	return strings.Contains(cmd, "claude")
}
