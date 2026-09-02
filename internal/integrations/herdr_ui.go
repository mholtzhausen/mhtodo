package integrations

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// presentHerdrUI raises an existing Herdr window or attaches a client when needed.
func (c Client) presentHerdrUI() error {
	if err := activateHerdrWindow(); err == nil {
		return nil
	}
	attach, err := c.needsHerdrClientAttach()
	if err != nil {
		return err
	}
	if !attach {
		return nil
	}
	if err := launchInTerminal(c.herdrCommandLine("session", "attach", "default")); err != nil {
		return fmt.Errorf("open terminal for herdr: %w", err)
	}
	time.Sleep(300 * time.Millisecond)
	return nil
}

func (c Client) needsHerdrClientAttach() (bool, error) {
	reason, err := c.notificationReason()
	if err != nil {
		_, ok := findHerdrTUIPID()
		return !ok, nil
	}
	_, tuiRunning := findHerdrTUIPID()
	return needsAttachFromReason(reason, tuiRunning), nil
}

func needsAttachFromReason(reason string, tuiRunning bool) bool {
	switch reason {
	case "no_foreground_client":
		return true
	case "shown", "busy":
		return false
	default:
		return !tuiRunning
	}
}

func (c Client) notificationReason() (string, error) {
	raw, err := c.runOutput("notification", "show", "mhtodo", "--body", "")
	if err != nil {
		return "", err
	}
	var env herdrEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", fmt.Errorf("parse herdr notification: %w", err)
	}
	if env.Error != nil && env.Error.Message != "" {
		if isHerdrClientAlreadyAttached(fmt.Errorf("%s", env.Error.Message)) {
			return "busy", nil
		}
		return "", fmt.Errorf("%s", env.Error.Message)
	}
	var show herdrNotificationShow
	if err := json.Unmarshal(env.Result, &show); err != nil {
		return "", fmt.Errorf("parse herdr notification result: %w", err)
	}
	return show.Reason, nil
}

func findHerdrTUIPID() (int, bool) {
	out, err := execCommandOutput("pgrep", "-af", "herdr")
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		cmd := strings.Join(fields[1:], " ")
		if isHerdrServerProcess(cmd) || isHerdrCLIProbe(cmd) {
			continue
		}
		return pid, true
	}
	return 0, false
}

func isHerdrServerProcess(cmd string) bool {
	lower := strings.ToLower(cmd)
	return strings.Contains(lower, " server") || strings.Contains(lower, "herdr-server")
}

func isHerdrCLIProbe(cmd string) bool {
	lower := strings.ToLower(cmd)
	for _, sub := range []string{
		" workspace ", " tab ", " pane ", " notification ", " status ",
		" agent ", " api ", " session list", " session stop",
	} {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}