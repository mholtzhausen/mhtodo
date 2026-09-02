package integrations

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type herdrServerStatusJSON struct {
	Status  string `json:"status"`
	Running bool   `json:"running"`
	Socket  string `json:"socket"`
}

type herdrNotificationShow struct {
	Reason string `json:"reason"`
	Shown  bool   `json:"shown"`
}

func (c Client) serverRunning() (bool, error) {
	raw, err := c.runOutput("status", "server", "--json")
	if err != nil {
		return false, err
	}
	running, err := parseServerRunning(raw)
	if err != nil {
		return false, err
	}
	return running, nil
}

func parseServerRunning(raw []byte) (bool, error) {
	var st herdrServerStatusJSON
	if err := json.Unmarshal(raw, &st); err != nil {
		return false, fmt.Errorf("parse herdr server status: %w", err)
	}
	if st.Running {
		return true, nil
	}
	return strings.EqualFold(st.Status, "running"), nil
}

func (c Client) waitForServer(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, err := c.serverRunning()
		if err == nil && running {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("herdr server did not start within %s", timeout)
}

// herdrCommandLine builds a shell command line for the configured herdr binary.
func (c Client) herdrCommandLine(args ...string) string {
	binary := strings.TrimSpace(c.Herdr.Binary)
	if binary == "" {
		binary = "herdr"
	}
	env, prefixArgs := ParseEnvStart(c.Herdr.EnvStart)
	parts := append([]string{}, env...)
	parts = append(parts, shellWord(binary))
	for _, a := range prefixArgs {
		parts = append(parts, shellWord(a))
	}
	for _, a := range args {
		parts = append(parts, shellWord(a))
	}
	return strings.Join(parts, " ")
}

func shellWord(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\n\"'$`\\") {
		return s
	}
	return ShellDoubleQuote(s)
}

func (c Client) ensureHerdrSession() error {
	running, err := c.serverRunning()
	if err != nil {
		// Server may not be up yet; treat unreachable socket as not running.
		if !isHerdrUnreachable(err) {
			return err
		}
		running = false
	}
	if running {
		return nil
	}
	if err := launchInTerminal(c.herdrCommandLine()); err != nil {
		return fmt.Errorf("start herdr: %w", err)
	}
	return c.waitForServer(12 * time.Second)
}

func isHerdrUnreachable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connect") ||
		strings.Contains(msg, "socket") ||
		strings.Contains(msg, "no such file") ||
		strings.Contains(msg, "connection refused")
}

func isHerdrClientAlreadyAttached(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already has an attached client")
}
