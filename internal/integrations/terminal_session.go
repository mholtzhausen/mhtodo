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
// storedPID is still alive; otherwise launches Claude in a new terminal and
// returns the PID to persist on the task.
func (c Client) OpenTerminalSession(cwd, shortID, title, todoSession string, storedPID int) (int, error) {
	if settings.NormalizeSpawn(c.Claude.Spawn) != settings.SpawnTerminal {
		return 0, fmt.Errorf("Claude spawn mode is not terminal")
	}
	if !c.ClaudeFound() {
		return 0, fmt.Errorf("claude binary not found: %s", strings.TrimSpace(c.Claude.Binary))
	}

	session := strings.TrimSpace(todoSession)
	if session == "" {
		session = core.DefaultTodoSession(shortID, title)
	}

	if storedPID > 0 && processAlive(storedPID) {
		if err := activateWindowForPIDWalk(storedPID); err == nil {
			return storedPID, nil
		}
		// Window focus failed; still treat as alive and return the PID.
		return storedPID, nil
	}

	cmdLine := c.claudeTerminalCommandLine(cwd, shortID, title, session)
	launchPID, err := launchInTerminalPreferred(c.Terminal.Binary, cmdLine)
	if err != nil {
		return 0, err
	}

	if pid, ok := waitForSessionPID(session, 3*time.Second); ok {
		return pid, nil
	}
	if launchPID > 0 && processAlive(launchPID) {
		return launchPID, nil
	}
	return launchPID, nil
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

func (c Client) claudeTerminalCommandLine(cwd, shortID, title, session string) string {
	prompt := c.ticketPrompt(shortID)
	bin := strings.TrimSpace(c.Claude.Binary)
	if bin == "" {
		bin = "claude"
	}
	displayName := session
	if core.LooksLikeSessionUUID(session) || session == "" {
		displayName = core.DefaultTodoSession(shortID, title)
	}

	claudeEnv, claudePrefix := ParseEnvStart(c.Claude.EnvStart)
	termEnv, termPrefix := ParseEnvStart(c.Terminal.EnvStart)

	var parts []string
	if d := strings.TrimSpace(cwd); d != "" {
		parts = append(parts, "cd", shellWord(d), "&&")
	}
	parts = append(parts, "MHTODO_SESSION="+shellWord(session))
	for _, e := range append(append([]string{}, termEnv...), claudeEnv...) {
		parts = append(parts, shellWord(e))
	}
	for _, a := range append(append([]string{}, termPrefix...), claudePrefix...) {
		parts = append(parts, shellWord(a))
	}

	quotedPrompt := ShellDoubleQuote(prompt)
	resumeCmd := strings.Join([]string{shellWord(bin), "--resume", shellWord(session), quotedPrompt}, " ")
	nameCmd := strings.Join([]string{shellWord(bin), "--name", shellWord(displayName), quotedPrompt}, " ")
	if session != "" {
		parts = append(parts, "("+resumeCmd+" || "+nameCmd+")")
	} else {
		parts = append(parts, nameCmd)
	}
	return strings.Join(parts, " ")
}

func activateWindowForPIDWalk(pid int) error {
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

func waitForSessionPID(session string, timeout time.Duration) (int, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if pid, ok := findPIDByMHTODOSession(session); ok {
			return pid, true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return 0, false
}

func findPIDByMHTODOSession(session string) (int, bool) {
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
		if err != nil || pid <= 1 || pid == self {
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
