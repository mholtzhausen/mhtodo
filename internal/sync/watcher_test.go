package sync

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"mhtodo/internal/core"
	"mhtodo/internal/store"
)

// externalWrite opens a second connection to the same DB (simulating a CLI
// process) and updates a task.
func externalWrite(t *testing.T, db string, id string) {
	t.Helper()
	repo, err := store.Open(db)
	if err != nil {
		t.Fatalf("open second conn: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()
	task, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	task.Title = "edited externally"
	task.UpdatedAt = time.Now().UTC().Truncate(time.Second).Add(1 * time.Second) // ensure max(updated_at) moves
	if err := repo.Update(ctx, task); err != nil {
		t.Fatalf("external update: %v", err)
	}
}

func TestExternalWriteTriggersCallback(t *testing.T) {
	db := filepath.Join(t.TempDir(), "mhtodo.db")
	repo, err := store.Open(db)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()
	task, err := core.NewService(repo).Create(ctx, core.CreateInput{Title: "watch me"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	var count atomic.Int32
	w, err := Watch(db, func() { count.Add(1) })
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	defer w.Close()

	time.Sleep(50 * time.Millisecond) // let the poll loop establish its baseline
	externalWrite(t, db, task.ID)

	deadline := time.Now().Add(3 * time.Second) // fsnotify path is fast; 2s poll is the backstop
	for count.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if count.Load() == 0 {
		t.Fatal("no callback after external write within 3s")
	}
}

func TestDebounceCoalescesRapidPokes(t *testing.T) {
	// No real DB needed: poke() is the unit under test. Watch tolerates a
	// missing file (fsnotify skips it, the poll connection is lazy).
	var count atomic.Int32
	w, err := Watch(filepath.Join(t.TempDir(), "missing.db"), func() { count.Add(1) })
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	defer w.Close()

	w.poke()
	w.poke() // 0ms apart → must coalesce into one callback
	time.Sleep(debounce + 250*time.Millisecond)
	if got := count.Load(); got != 1 {
		t.Fatalf("want exactly 1 debounced callback, got %d", got)
	}

	// A later poke fires again (debounce resets per burst).
	w.poke()
	time.Sleep(debounce + 250*time.Millisecond)
	if got := count.Load(); got != 2 {
		t.Fatalf("want 2 callbacks after second burst, got %d", got)
	}
}

func TestCloseStopsCallbacks(t *testing.T) {
	var count atomic.Int32
	w, err := Watch(filepath.Join(t.TempDir(), "missing.db"), func() { count.Add(1) })
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	w.Close()
	w.poke() // timer fires after close → must be suppressed by the stop channel
	time.Sleep(debounce + 250*time.Millisecond)
	if got := count.Load(); got != 0 {
		t.Fatalf("no callbacks expected after Close, got %d", got)
	}
	w.Close() // idempotent
}

func TestPollFallsBackWhenFsnotifyMisses(t *testing.T) {
	// Simulate fsnotify blindness by closing the file watcher directly; only
	// the 2s poll should then deliver the change.
	db := filepath.Join(t.TempDir(), "mhtodo.db")
	repo, err := store.Open(db)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()
	task, err := core.NewService(repo).Create(ctx, core.CreateInput{Title: "poll me"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	var count atomic.Int32
	w, err := Watch(db, func() { count.Add(1) })
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	defer w.Close()
	w.fw.Close() // kill fsnotify; poll loop keeps running

	time.Sleep(2500 * time.Millisecond) // let the first poll tick establish its baseline
	externalWrite(t, db, task.ID)

	deadline := time.Now().Add(4 * time.Second) // ≤2s to next tick + 300ms debounce + slack
	for count.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if count.Load() == 0 {
		t.Fatal("poll fallback did not deliver the change within 4s")
	}
}
