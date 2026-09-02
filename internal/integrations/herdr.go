package integrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"mhtodo/internal/settings"
)

var errHerdrWindowNotFound = errors.New("herdr window not found")

// HerdrTaskStatus is returned to the GUI for Herdr ticket integration.
type HerdrTaskStatus struct {
	Ready bool   `json:"ready"`
	Error string `json:"error,omitempty"`
}

type herdrEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type herdrWorkspace struct {
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
}

type herdrTab struct {
	TabID       string `json:"tab_id"`
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
}

type herdrPaneList struct {
	Panes []herdrPaneInfo `json:"panes"`
}

type herdrPaneInfo struct {
	PaneID string `json:"pane_id"`
	TabID  string `json:"tab_id"`
}

type herdrPaneProcessInfo struct {
	ProcessInfo struct {
		ForegroundProcesses []herdrForegroundProcess `json:"foreground_processes"`
	} `json:"process_info"`
}

type herdrForegroundProcess struct {
	Name    string   `json:"name"`
	Cmdline string   `json:"cmdline"`
	Argv    []string `json:"argv"`
}

type herdrWorkspaceList struct {
	Workspaces []herdrWorkspace `json:"workspaces"`
}

type herdrTabList struct {
	Tabs []herdrTab `json:"tabs"`
}

type herdrWorkspaceCreated struct {
	Workspace herdrWorkspace `json:"workspace"`
}

type herdrTabCreated struct {
	Tab      herdrTab  `json:"tab"`
	RootPane struct {
		PaneID string `json:"pane_id"`
	} `json:"root_pane"`
}

// Client runs herdr CLI commands using integration settings.
type Client struct {
	Herdr  settings.HerdrConfig
	Claude settings.ClaudeConfig
}

func (c Client) HerdrFound() bool {
	return settings.BinaryFound(c.Herdr.Binary)
}

func (c Client) ClaudeFound() bool {
	return settings.BinaryFound(c.Claude.Binary)
}

// TaskEligible reports whether a task meets Herdr integration preconditions.
func TaskEligible(humanOnly bool, cwd string) bool {
	return !humanOnly && strings.TrimSpace(cwd) != ""
}

// EnsureWorkspace lists workspaces and creates the configured one when missing.
func (c Client) EnsureWorkspace() (bool, error) {
	if !c.Herdr.Enabled || !c.HerdrFound() {
		return false, nil
	}
	label := c.Herdr.EffectiveSpaceName()
	if _, err := c.findWorkspace(label); err == nil {
		return true, nil
	}
	if err := c.createWorkspace(label); err != nil {
		return false, err
	}
	return true, nil
}

// OpenTicketTab focuses the configured workspace and opens or focuses a ticket tab.
// When a new tab is created and Claude integration is enabled, runs the ticket prompt.
func (c Client) OpenTicketTab(taskID, shortID, title, cwd string) error {
	if !c.Herdr.Enabled || !c.HerdrFound() {
		return fmt.Errorf("herdr integration is not available")
	}
	if err := c.ensureHerdrSession(); err != nil {
		return err
	}
	label := c.Herdr.EffectiveSpaceName()
	ws, err := c.findWorkspace(label)
	if err != nil {
		if createErr := c.createWorkspace(label); createErr != nil {
			return createErr
		}
		ws, err = c.findWorkspace(label)
		if err != nil {
			return err
		}
	}

	tabLabel := ticketTabLabel(taskID, shortID, title)
	tab, found, err := c.findTabForTicket(ws.WorkspaceID, taskID, shortID, tabLabel)
	if err != nil {
		return err
	}
	if found {
		if err := c.focusWorkspaceTab(ws.WorkspaceID, tab.TabID); err != nil {
			return err
		}
		if err := c.maybeStartClaude(ws.WorkspaceID, tab.TabID, shortID); err != nil {
			return err
		}
		return c.presentHerdrUI()
	}

	if err := c.run("workspace", "focus", ws.WorkspaceID); err != nil {
		return err
	}
	paneID, tabID, err := c.createTab(ws.WorkspaceID, tabLabel, cwd)
	if err != nil {
		return err
	}

	if err := c.focusWorkspaceTab(ws.WorkspaceID, tabID); err != nil {
		return err
	}

	if err := c.maybeStartClaudePane(paneID, shortID); err != nil {
		return err
	}
	return c.presentHerdrUI()
}

// MaybeCloseTicketTabOnDone closes the Herdr tab for a task when Claude
// close_tab_on_done is enabled. Herdr/tab errors are ignored (best effort).
func (c Client) MaybeCloseTicketTabOnDone(taskID, shortID, title string) {
	if !c.Claude.CloseTabOnDone || !c.Herdr.Enabled || !c.HerdrFound() {
		return
	}
	_ = c.CloseTicketTab(taskID, shortID, title)
}

// CloseTicketTab closes the Herdr tab matching a ticket when one exists.
// Returns nil when Herdr is unavailable or no tab matches.
func (c Client) CloseTicketTab(taskID, shortID, title string) error {
	label := c.Herdr.EffectiveSpaceName()
	ws, err := c.findWorkspace(label)
	if err != nil {
		return nil
	}
	tabLabel := ticketTabLabel(taskID, shortID, title)
	tab, found, err := c.findTabForTicket(ws.WorkspaceID, taskID, shortID, tabLabel)
	if err != nil || !found {
		return nil
	}
	return c.run("tab", "close", tab.TabID)
}

const tabLabelTaskSep = "|"

func ticketTabLabel(_ string, shortID, title string) string {
	shortID = strings.TrimSpace(shortID)
	title = strings.TrimSpace(title)
	if title == "" {
		return shortID
	}
	if utf8.RuneCountInString(title) > 40 {
		runes := []rune(title)
		title = string(runes[:40])
	}
	return shortID + " - " + title
}

func parseTabLabel(label string) (taskID, display string, ok bool) {
	i := strings.Index(label, tabLabelTaskSep)
	if i <= 0 {
		return "", label, false
	}
	return label[:i], label[i+len(tabLabelTaskSep):], true
}

func tabMatchesTicket(tabLabel, taskID, shortID string) bool {
	shortID = strings.TrimSpace(shortID)
	if shortID == "" {
		return false
	}
	if tid, display, ok := parseTabLabel(tabLabel); ok {
		if strings.TrimSpace(taskID) != "" && tid == taskID {
			return true
		}
		tabLabel = display
	}
	if tabLabel == shortID {
		return true
	}
	return strings.HasPrefix(tabLabel, shortID+" - ")
}

// focusWorkspaceTab switches the Herdr session to the given workspace and tab.
func (c Client) focusWorkspaceTab(workspaceID, tabID string) error {
	if err := c.run("workspace", "focus", workspaceID); err != nil {
		return err
	}
	return c.run("tab", "focus", tabID)
}

func (c Client) findWorkspace(label string) (herdrWorkspace, error) {
	raw, err := c.runOutput("workspace", "list")
	if err != nil {
		return herdrWorkspace{}, err
	}
	var env herdrEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return herdrWorkspace{}, fmt.Errorf("parse workspace list: %w", err)
	}
	if env.Error != nil && env.Error.Message != "" {
		return herdrWorkspace{}, fmt.Errorf("%s", env.Error.Message)
	}
	var list herdrWorkspaceList
	if err := json.Unmarshal(env.Result, &list); err != nil {
		return herdrWorkspace{}, fmt.Errorf("parse workspace list result: %w", err)
	}
	for _, ws := range list.Workspaces {
		if ws.Label == label {
			return ws, nil
		}
	}
	return herdrWorkspace{}, fmt.Errorf("workspace %q not found", label)
}

func (c Client) createWorkspace(label string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home directory: %w", err)
	}
	return c.run("workspace", "create", "--cwd", home, "--label", label, "--no-focus")
}

func (c Client) findTabForTicket(workspaceID, taskID, shortID, label string) (herdrTab, bool, error) {
	raw, err := c.runOutput("tab", "list", "--workspace", workspaceID)
	if err != nil {
		return herdrTab{}, false, err
	}
	var env herdrEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return herdrTab{}, false, fmt.Errorf("parse tab list: %w", err)
	}
	if env.Error != nil && env.Error.Message != "" {
		return herdrTab{}, false, fmt.Errorf("%s", env.Error.Message)
	}
	var list herdrTabList
	if err := json.Unmarshal(env.Result, &list); err != nil {
		return herdrTab{}, false, fmt.Errorf("parse tab list result: %w", err)
	}
	var legacyMatch *herdrTab
	for _, tab := range list.Tabs {
		if workspaceID != "" && tab.WorkspaceID != "" && tab.WorkspaceID != workspaceID {
			continue
		}
		if tab.Label == label {
			return tab, true, nil
		}
		if tid, _, ok := parseTabLabel(tab.Label); ok {
			if tid == taskID {
				t := tab
				return t, true, nil
			}
			continue
		}
		if tabMatchesTicket(tab.Label, taskID, shortID) {
			t := tab
			if legacyMatch == nil {
				legacyMatch = &t
			}
		}
	}
	if legacyMatch != nil {
		return *legacyMatch, true, nil
	}
	return herdrTab{}, false, nil
}

func (c Client) createTab(workspaceID, label, cwd string) (paneID, tabID string, err error) {
	args := []string{
		"tab", "create",
		"--workspace", workspaceID,
		"--cwd", strings.TrimSpace(cwd),
		"--label", label,
		"--focus",
	}
	for _, env := range claudeTabEnv(c.Claude) {
		args = append(args, "--env", env)
	}
	raw, err := c.runOutput(args...)
	if err != nil {
		return "", "", err
	}
	var env herdrEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", "", fmt.Errorf("parse tab create: %w", err)
	}
	if env.Error != nil && env.Error.Message != "" {
		return "", "", fmt.Errorf("%s", env.Error.Message)
	}
	var created herdrTabCreated
	if err := json.Unmarshal(env.Result, &created); err != nil {
		return "", "", fmt.Errorf("parse tab create result: %w", err)
	}
	return created.RootPane.PaneID, created.Tab.TabID, nil
}

func claudeTabEnv(claude settings.ClaudeConfig) []string {
	env, _ := ParseEnvStart(claude.EnvStart)
	return env
}

func (c Client) maybeStartClaude(workspaceID, tabID, shortID string) error {
	paneID, err := c.findTabRootPane(workspaceID, tabID)
	if err != nil {
		return err
	}
	return c.maybeStartClaudePane(paneID, shortID)
}

func (c Client) maybeStartClaudePane(paneID, shortID string) error {
	if !c.Claude.Enabled || !c.ClaudeFound() || paneID == "" {
		return nil
	}
	running, err := c.paneRunningClaude(paneID)
	if err != nil {
		return err
	}
	if running {
		return nil
	}
	time.Sleep(200 * time.Millisecond)
	if err := c.runClaudeInPane(paneID, c.ticketPrompt(shortID)); err != nil {
		return fmt.Errorf("claude in herdr tab: %w", err)
	}
	return nil
}

func (c Client) ticketPrompt(shortID string) string {
	prompt := c.Claude.EffectiveTicketPrompt()
	return strings.ReplaceAll(prompt, "{{todo-hash}}", shortID)
}

func (c Client) findTabRootPane(workspaceID, tabID string) (string, error) {
	raw, err := c.runOutput("pane", "list", "--workspace", workspaceID)
	if err != nil {
		return "", err
	}
	var env herdrEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", fmt.Errorf("parse pane list: %w", err)
	}
	if env.Error != nil && env.Error.Message != "" {
		return "", fmt.Errorf("%s", env.Error.Message)
	}
	var list herdrPaneList
	if err := json.Unmarshal(env.Result, &list); err != nil {
		return "", fmt.Errorf("parse pane list result: %w", err)
	}
	for _, pane := range list.Panes {
		if pane.TabID == tabID {
			return pane.PaneID, nil
		}
	}
	return "", fmt.Errorf("pane for tab %q not found", tabID)
}

func (c Client) paneRunningClaude(paneID string) (bool, error) {
	raw, err := c.runOutput("pane", "process-info", "--pane", paneID)
	if err != nil {
		return false, err
	}
	var env herdrEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return false, fmt.Errorf("parse pane process info: %w", err)
	}
	if env.Error != nil && env.Error.Message != "" {
		return false, fmt.Errorf("%s", env.Error.Message)
	}
	var info herdrPaneProcessInfo
	if err := json.Unmarshal(env.Result, &info); err != nil {
		return false, fmt.Errorf("parse pane process info result: %w", err)
	}
	for _, proc := range info.ProcessInfo.ForegroundProcesses {
		if processIsClaude(proc, c.Claude.Binary) {
			return true, nil
		}
	}
	return false, nil
}

func processIsClaude(proc herdrForegroundProcess, claudeBinary string) bool {
	bin := strings.ToLower(filepath.Base(strings.TrimSpace(claudeBinary)))
	if bin == "" {
		bin = "claude"
	}
	if strings.Contains(strings.ToLower(proc.Name), "claude") {
		return true
	}
	fields := []string{proc.Cmdline}
	if len(proc.Argv) > 0 {
		fields = append(fields, proc.Argv[0], strings.Join(proc.Argv, " "))
	}
	for _, field := range fields {
		lower := strings.ToLower(strings.TrimSpace(field))
		if lower == "" {
			continue
		}
		if strings.Contains(lower, "claude") || strings.Contains(lower, bin) {
			return true
		}
	}
	return false
}

func (c Client) runClaudeInPane(paneID, prompt string) error {
	_, prefixArgs := ParseEnvStart(c.Claude.EnvStart)
	args := append([]string{"pane", "run", paneID}, prefixArgs...)
	args = append(args, c.Claude.Binary, ShellDoubleQuote(prompt))
	return c.run(args...)
}

func (c Client) run(args ...string) error {
	_, err := c.runOutput(args...)
	return err
}

func (c Client) runOutput(args ...string) ([]byte, error) {
	binary := strings.TrimSpace(c.Herdr.Binary)
	if binary == "" {
		binary = "herdr"
	}
	env, prefixArgs := ParseEnvStart(c.Herdr.EnvStart)
	allArgs := append(prefixArgs, args...)
	cmd := exec.Command(binary, allArgs...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return out, nil
}
