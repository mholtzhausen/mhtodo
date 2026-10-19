package integrations

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mhtodo/internal/platform"
)

// Claude launcher kinds, in detection preference order.
const (
	LauncherHerdr          = "herdr"
	LauncherGhostty        = "ghostty"
	LauncherXTerminal      = "x-terminal-emulator"
	LauncherGnomeTerminal  = "gnome-terminal"
	claudeBinaryName       = "claude"
	herdrAgentStartRetries = 10
)

var launcherOrder = []string{LauncherHerdr, LauncherGhostty, LauncherXTerminal, LauncherGnomeTerminal}

// ClaudeLauncher starts Claude Code for a task: a herdr pane split when herdr
// is installed, otherwise a new terminal window.
type ClaudeLauncher struct {
	Kind string // one of the Launcher* constants
	Bin  string // resolved launcher executable
}

// DetectClaudeLauncher requires claude on PATH plus the first available launcher.
func DetectClaudeLauncher() (ClaudeLauncher, bool) {
	if _, err := exec.LookPath(claudeBinaryName); err != nil {
		return ClaudeLauncher{}, false
	}
	for _, kind := range launcherOrder {
		if p, err := exec.LookPath(kind); err == nil {
			return ClaudeLauncher{Kind: kind, Bin: p}, true
		}
	}
	return ClaudeLauncher{}, false
}

// ClaudeArgs is the claude argv: resume the linked session when there is one.
func ClaudeArgs(session string) []string {
	if s := strings.TrimSpace(session); s != "" {
		return []string{claudeBinaryName, "--resume", s}
	}
	return []string{claudeBinaryName}
}

var claudeProjectDirRe = regexp.MustCompile(`[^a-zA-Z0-9]`)

// ResumableSession returns session when Claude has a transcript for it under
// cwd's project dir (the only place `claude --resume` looks), else "". Stale
// or foreign todo_session values then start a fresh Claude instead of
// failing with "No conversation found".
func ResumableSession(cwd, session string) string {
	session = strings.TrimSpace(session)
	if session == "" || strings.ContainsAny(session, `/\`) {
		return ""
	}
	base := os.Getenv("CLAUDE_CONFIG_DIR")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".claude")
	}
	project := claudeProjectDirRe.ReplaceAllString(cwd, "-")
	if _, err := os.Stat(filepath.Join(base, "projects", project, session+".jsonl")); err != nil {
		return ""
	}
	return session
}

// Launch starts claude in cwd. For herdr with no session, prompt is pasted
// into Claude's input box without submitting; terminal launchers ignore it
// (the caller copies it to the clipboard instead).
func (l ClaudeLauncher) Launch(shortID, cwd, session, prompt string) error {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return fmt.Errorf("task has no working directory")
	}
	session = ResumableSession(cwd, session)
	if l.Kind == LauncherHerdr {
		return l.launchHerdr(shortID, cwd, session, prompt)
	}
	args, dir := TerminalArgs(l.Kind, l.Bin, cwd, ClaudeArgs(session))
	if args == nil {
		return fmt.Errorf("unknown claude launcher %q", l.Kind)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	if err := platform.StartDetached(cmd); err != nil {
		return fmt.Errorf("start %s: %w", l.Kind, err)
	}
	return nil
}

// TerminalArgs builds the argv (and process dir) that opens a terminal window
// in cwd running command. Returns nil for unknown kinds.
func TerminalArgs(kind, bin, cwd string, command []string) (args []string, dir string) {
	switch kind {
	case LauncherGhostty:
		return append([]string{bin, "--working-directory=" + cwd, "-e"}, command...), cwd
	case LauncherXTerminal:
		return append([]string{bin, "-e"}, command...), cwd
	case LauncherGnomeTerminal:
		return append([]string{bin, "--working-directory=" + cwd, "--"}, command...), cwd
	}
	return nil, ""
}

// HerdrAgentName is the live herdr agent name for a ticket's Claude pane.
func HerdrAgentName(shortID string) string {
	var b strings.Builder
	b.WriteString("mhtodo-")
	for _, r := range strings.ToLower(shortID) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	name := b.String()
	if len(name) > 32 {
		name = name[:32]
	}
	return name
}

func (l ClaudeLauncher) launchHerdr(shortID, cwd, session, prompt string) error {
	name := HerdrAgentName(shortID)
	// Ticket already has a live Claude pane: focus it instead of splitting again.
	if _, err := l.herdr("agent", "get", name); err == nil {
		_, err = l.herdr("agent", "focus", name)
		return err
	}
	raw, err := l.herdr("pane", "split", "--direction", "right", "--cwd", cwd, "--focus")
	if err != nil {
		return err
	}
	paneID, err := parseHerdrPaneID(raw)
	if err != nil {
		return err
	}
	start := []string{"agent", "start", name, "--kind", claudeBinaryName, "--pane", paneID}
	if s := strings.TrimSpace(session); s != "" {
		start = append(start, "--", "--resume", s)
	}
	// The new pane's shell may still be loading its rc files; agent start
	// refuses until the prompt is up, so retry briefly.
	for i := 0; ; i++ {
		if _, err = l.herdr(start...); err == nil {
			break
		}
		if i >= herdrAgentStartRetries {
			return err
		}
		time.Sleep(300 * time.Millisecond)
	}
	if strings.TrimSpace(session) != "" || strings.TrimSpace(prompt) == "" {
		return nil
	}
	_, err = l.herdr("pane", "send-text", paneID, BracketedPaste(prompt))
	return err
}

// BracketedPaste wraps text in paste markers so a multi-line prompt lands in
// Claude's input box without its newlines submitting it.
func BracketedPaste(text string) string {
	return "\x1b[200~" + text + "\x1b[201~"
}

type herdrEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// herdr runs one herdr CLI call and returns its result payload.
func (l ClaudeLauncher) herdr(args ...string) (json.RawMessage, error) {
	// herdr prints the JSON envelope on stdout for success and stderr for errors.
	out, runErr := exec.Command(l.Bin, args...).CombinedOutput()
	var env herdrEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		if runErr != nil {
			if msg := strings.TrimSpace(string(out)); msg != "" {
				return nil, fmt.Errorf("herdr %s: %s", args[0], msg)
			}
			return nil, fmt.Errorf("herdr %s: %w", args[0], runErr)
		}
		return nil, fmt.Errorf("herdr %s: parse output: %w", args[0], err)
	}
	if env.Error != nil {
		return nil, fmt.Errorf("herdr %s: %s", args[0], env.Error.Message)
	}
	if runErr != nil {
		return nil, fmt.Errorf("herdr %s: %w", args[0], runErr)
	}
	return env.Result, nil
}

func parseHerdrPaneID(result json.RawMessage) (string, error) {
	var r struct {
		Pane struct {
			PaneID string `json:"pane_id"`
		} `json:"pane"`
	}
	if err := json.Unmarshal(result, &r); err != nil || r.Pane.PaneID == "" {
		return "", fmt.Errorf("herdr pane split: no pane id in response")
	}
	return r.Pane.PaneID, nil
}
