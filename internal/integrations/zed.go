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
	session := strings.TrimSpace(todoSession)
	if session == "" {
		session = core.DefaultTodoSession(shortID, title)
	}
	bin := strings.TrimSpace(c.Zed.Binary)
	if bin == "" {
		bin = "zed"
	}
	envPairs, prefixArgs := ParseEnvStart(c.Zed.EnvStart)
	args := append(append([]string{}, prefixArgs...), cwd)
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), envPairs...)
	cmd.Env = append(cmd.Env, "MHTODO_SESSION="+session)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start zed: %w", err)
	}
	_ = cmd.Process.Release()
	return nil
}
