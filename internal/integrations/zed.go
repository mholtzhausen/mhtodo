package integrations

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"mhtodo/internal/core"
	"mhtodo/internal/settings"
)

// ZedClient launches the Zed editor for a task cwd with MHTODO_SESSION set.
type ZedClient struct {
	Zed settings.IntegrationConfig
}

func (c ZedClient) Found() bool {
	return settings.BinaryFound(c.Zed.Binary)
}

// OpenTicket opens Zed at cwd with MHTODO_SESSION=<todoSession> (plus zed.env_start).
func (c ZedClient) OpenTicket(cwd, shortID, title, todoSession string) error {
	if !c.Zed.Enabled || !c.Found() {
		return fmt.Errorf("zed integration is not available")
	}
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return fmt.Errorf("task has no working directory")
	}
	bin, args, env := c.ticketInvocation(cwd, shortID, title, todoSession)
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start zed: %w", err)
	}
	_ = cmd.Process.Release()
	return nil
}

// TicketCommand returns the shell-equivalent command OpenTicket would start
// (env assignments + binary + args), for UI tooltips.
func (c ZedClient) TicketCommand(cwd, shortID, title, todoSession string) string {
	cwd = strings.TrimSpace(cwd)
	bin, args, env := c.ticketInvocation(cwd, shortID, title, todoSession)
	parts := make([]string, 0, len(env)+1+len(args))
	for _, e := range env {
		parts = append(parts, shellEnvAssign(e))
	}
	parts = append(parts, shellWord(bin))
	for _, a := range args {
		parts = append(parts, shellWord(a))
	}
	return strings.Join(parts, " ")
}

// shellEnvAssign formats KEY=value for a POSIX shell, quoting the value.
func shellEnvAssign(kv string) string {
	i := strings.IndexByte(kv, '=')
	if i <= 0 {
		return shellWord(kv)
	}
	return kv[:i+1] + ShellDoubleQuote(kv[i+1:])
}

func (c ZedClient) ticketInvocation(cwd, shortID, title, todoSession string) (bin string, args, env []string) {
	session := strings.TrimSpace(todoSession)
	if session == "" {
		session = core.DefaultTodoSession(shortID, title)
	}
	bin = strings.TrimSpace(c.Zed.Binary)
	if bin == "" {
		bin = "zed"
	}
	envPairs, prefixArgs := ParseEnvStart(c.Zed.EnvStart)
	env = append(append([]string{}, envPairs...), "MHTODO_SESSION="+session)
	args = append(append([]string{}, prefixArgs...), cwd)
	return bin, args, env
}
