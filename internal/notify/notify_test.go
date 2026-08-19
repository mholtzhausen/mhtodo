package notify

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// recordingNotifier builds a Notifier with injected clock + runner. It returns
// the notifier and an advance function that moves the fake clock forward.
func recordingNotifier(t *testing.T, calls *[]string, fail bool) (*Notifier, func(d time.Duration)) {
	t.Helper()
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	n := &Notifier{
		last: make(map[string]time.Time),
		now:  func() time.Time { return now },
		run: func(args ...string) error {
			if fail {
				return errors.New("no display")
			}
			mu.Lock()
			*calls = append(*calls, args[4]) // args: -a mhtodo -i dialog-task <summary> <body>
			mu.Unlock()
			return nil
		},
	}
	return n, func(d time.Duration) { now = now.Add(d) }
}

func TestDedupeSameIDAndStatus(t *testing.T) {
	var calls []string
	n, advance := recordingNotifier(t, &calls, false)

	n.TaskDone("id1", "Ship v0.1")
	n.TaskDone("id1", "Ship v0.1") // within 60s → suppressed
	if len(calls) != 1 {
		t.Fatalf("want 1 call after immediate repeat, got %d: %v", len(calls), calls)
	}

	advance(59 * time.Second)
	n.TaskDone("id1", "Ship v0.1") // still within window → suppressed
	if len(calls) != 1 {
		t.Fatalf("want 1 call at +59s, got %d: %v", len(calls), calls)
	}

	advance(2 * time.Second)       // total +61s
	n.TaskDone("id1", "Ship v0.1") // window elapsed → sent
	if len(calls) != 2 {
		t.Fatalf("want 2 calls at +61s, got %d: %v", len(calls), calls)
	}
}

func TestDedupeIsPerStatus(t *testing.T) {
	var calls []string
	n, _ := recordingNotifier(t, &calls, false)

	n.TaskDone("id1", "A")
	n.TaskWaiting("id1", "A") // different status → not deduped
	if len(calls) != 2 || calls[0] != "Task completed: A" || calls[1] != "Waiting on external: A" {
		t.Fatalf("want both summaries in order, got %v", calls)
	}

	n.TaskDone("id2", "B") // different id → not deduped
	if len(calls) != 3 {
		t.Fatalf("want 3 calls, got %d: %v", len(calls), calls)
	}
}

func TestNotifySendArgs(t *testing.T) {
	var args []string
	n := &Notifier{last: map[string]time.Time{}, now: time.Now, run: func(a ...string) error {
		args = append(args, a...)
		return nil
	}}
	n.TaskDone("abc", "My task")
	want := []string{"-a", appName, "-i", "dialog-task", "Task completed: My task", "abc"}
	if len(args) != len(want) {
		t.Fatalf("args %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestFailureIsNotFatal(t *testing.T) {
	var calls []string
	n, _ := recordingNotifier(t, &calls, true) // runner always fails
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("notify-send failure must not panic: %v", r)
		}
	}()
	n.TaskDone("id1", "A")
	n.TaskWaiting("id2", "B")
	if len(calls) != 0 {
		t.Fatalf("no successful sends expected, got %v", calls)
	}
	// A failed send still counts toward the dedupe window (avoids log spam).
	var calls2 []string
	n2, _ := recordingNotifier(t, &calls2, true)
	n2.TaskDone("id1", "A")
	n2.TaskDone("id1", "A") // suppressed even though both failed
	if len(calls2) != 0 {
		t.Fatalf("failed sends should still dedupe, got %v", calls2)
	}
}

func TestConcurrentSends(t *testing.T) {
	var mu sync.Mutex
	var n int
	nf := &Notifier{
		last: make(map[string]time.Time),
		now:  time.Now,
		run: func(args ...string) error {
			mu.Lock()
			n++
			mu.Unlock()
			return nil
		},
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nf.TaskDone("same-id", "T") // all deduped to one send
		}(i)
	}
	wg.Wait()
	if n != 1 {
		t.Fatalf("want exactly 1 send from 50 concurrent calls, got %d", n)
	}
}
