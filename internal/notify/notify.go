// Package notify sends desktop notifications via notify-send (see
// .agent/plan/05-gui-spec.md). Failures are logged, never fatal — a missing
// display or D-Bus session must not break task mutations.
package notify

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
)

const appName = "mhtodo" // -a flag: groups notifications per app in the shell

// dedupeWindow suppresses repeat notifications for the same id+status.
const dedupeWindow = 60 * time.Second

// Notifier is a small wrapper around notify-send with per-id+status dedupe.
// State is per-instance: the GUI keeps one long-lived Notifier; CLI invocations
// are short-lived, so cross-process dedupe is not expected (and not possible).
type Notifier struct {
	mu   sync.Mutex
	last map[string]time.Time // key id+"\x00"+status → last sent time

	now func() time.Time           // test seam
	run func(args ...string) error // test seam; defaults to exec.Command("notify-send", args...)
}

// New returns a Notifier that shells out to notify-send.
func New() *Notifier {
	return &Notifier{
		last: make(map[string]time.Time),
		now:  time.Now,
		run: func(args ...string) error { return exec.Command("notify-send", args...).Run() },
	}
}

// TaskDone notifies a →done transition ("Task completed: <title>").
func (n *Notifier) TaskDone(id, title string) {
	n.send(id, "done", fmt.Sprintf("Task completed: %s", title), id)
}

// TaskWaiting notifies a →waiting transition ("Waiting on external: <title>").
func (n *Notifier) TaskWaiting(id, title string) {
	n.send(id, "waiting", fmt.Sprintf("Waiting on external: %s", title), id)
}

// send dedupes per id+status within 60s, then fires notify-send with the app
// name constant for grouping. The body carries the task ID so the user can
// `mhtodo show <id>` from a terminal. Errors are logged and never returned.
func (n *Notifier) send(id, status, summary, body string) {
	key := id + "\x00" + status
	now := n.now()

	n.mu.Lock()
	if last, ok := n.last[key]; ok && now.Sub(last) < dedupeWindow {
		n.mu.Unlock()
		return // deduped
	}
	n.last[key] = now
	n.mu.Unlock()

	if err := n.run("-a", appName, "-i", "dialog-task", summary, body); err != nil {
		log.Printf("notify: %v (non-fatal)", err)
	}
}
