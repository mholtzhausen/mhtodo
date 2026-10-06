package aiskill

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const (
	// DefaultRefreshTimeout caps the optional Claude skill-refresh pass.
	DefaultRefreshTimeout = 3 * time.Minute
	envSkipRefresh        = "MHTODO_SKIP_SKILL_REFRESH"
)

// lookPath is a test seam for exec.LookPath.
var lookPath = exec.LookPath

// LookPathForTest swaps LookPath.
func LookPathForTest(f func(string) (string, error)) (restore func()) {
	prev := lookPath
	lookPath = f
	return func() { lookPath = prev }
}

// commandContext is a test seam for exec.CommandContext.
var commandContext = exec.CommandContext

// RefreshClaude runs a non-interactive Claude pass to re-verify the skill.
// Returns ("", nil) when skipped; ("", err) on failure; (output, nil) on success.
func RefreshClaude(timeout time.Duration) (output string, skipped bool, err error) {
	if os.Getenv(envSkipRefresh) == "1" || os.Getenv(envSkipRefresh) == "true" {
		return "", true, nil
	}
	if timeout <= 0 {
		timeout = DefaultRefreshTimeout
	}
	bin, err := lookPath("claude")
	if err != nil {
		return "", true, nil
	}

	prompt := `Run ` + "`mhtodo ai`" + ` to install/update the mhtodo skill and strip old hooks/settings/integrations. Then run ` + "`mhtodo ai --check`" + ` and report the result. If ~/.agents/skills/ exists, mirror the skill there too. Do nothing else.`

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := commandContext(ctx, bin, "-p", prompt,
		"--permission-mode", "dontAsk",
		"--allowedTools",
		"Bash(mhtodo:*)",
		"Read(~/.claude/skills/**)",
		"Edit(~/.claude/skills/**)",
		"Read(~/.agents/skills/**)",
		"Edit(~/.agents/skills/**)",
		"Glob",
		"Grep",
		"--disallowedTools", "AskUserQuestion",
	)
	cmd.Env = os.Environ()
	out, runErr := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), false, fmt.Errorf("claude skill refresh timed out after %s", timeout)
	}
	if runErr != nil {
		return string(out), false, fmt.Errorf("claude skill refresh: %w\n%s", runErr, truncate(string(out), 2000))
	}
	return string(out), false, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
