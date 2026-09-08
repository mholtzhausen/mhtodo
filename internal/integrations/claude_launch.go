package integrations

import (
	"strings"

	"mhtodo/internal/core"
)

// ClaudeLaunchShell builds a POSIX shell fragment that creates-or-resumes a
// Claude session by UUID:
//
//	bin --session-id UUID --name NAME PROMPT || bin --resume UUID --name NAME PROMPT
//
// --session-id errors when the session already exists; --resume opens that
// session. Using --resume alone with a missing name opens Claude's search TUI
// (no non-zero exit), which broke the old resume||name fallback.
func ClaudeLaunchShell(bin, sessionUUID, displayName, prompt string) string {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		bin = "claude"
	}
	sessionUUID = strings.TrimSpace(sessionUUID)
	displayName = strings.TrimSpace(displayName)
	quotedPrompt := ShellDoubleQuote(prompt)
	createCmd := strings.Join([]string{
		shellWord(bin), "--session-id", shellWord(sessionUUID),
		"--name", shellWord(displayName), quotedPrompt,
	}, " ")
	resumeCmd := strings.Join([]string{
		shellWord(bin), "--resume", shellWord(sessionUUID),
		"--name", shellWord(displayName), quotedPrompt,
	}, " ")
	return createCmd + " || " + resumeCmd
}

// ClaudeSessionEnv returns MHTODO_SESSION / MHTODO_SESSION_NAME assignments for
// spawn backends and shell helpers.
func ClaudeSessionEnv(sessionUUID, displayName string) []string {
	out := make([]string, 0, 2)
	if s := strings.TrimSpace(sessionUUID); s != "" {
		out = append(out, "MHTODO_SESSION="+s)
	}
	if n := strings.TrimSpace(displayName); n != "" {
		out = append(out, "MHTODO_SESSION_NAME="+n)
	}
	return out
}

// ResolveClaudeSession picks the UUID and --name for a launch. Callers that
// receive generated=true should persist sessionUUID onto the task.
func ResolveClaudeSession(shortID, title, todoSession string) (sessionUUID, displayName string, generated bool, err error) {
	displayName = core.ClaudeDisplayName(shortID, title, todoSession)
	sessionUUID, generated, err = core.EnsureClaudeSessionID(todoSession)
	return sessionUUID, displayName, generated, err
}
