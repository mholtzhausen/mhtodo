// Package sync implements CLI→GUI live sync (.agent/plan/05-gui-spec.md): it
// watches the SQLite DB file for writes from other processes and emits a
// single debounced callback, which the app turns into a "tasks:changed" event.
package sync

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	_ "modernc.org/sqlite" // same pure-Go driver the store uses; registered as "sqlite"
)

const (
	pollInterval = 2 * time.Second    // safety net: fsnotify can miss WAL-only writes (risk #5)
	debounce     = 300 * time.Millisecond
)

// Watcher watches dbPath for external changes. Two sources feed one debounced
// onChange callback:
//
//   - fsnotify on the DB file and its -wal/-shm sidecars (WAL commits touch the
//     sidecars first; checkpoints rewrite the main file). This is a superset of
//     the plan's "fsnotify on the DB file" — strictly more reliable.
//   - a 2s poll of max(updated_at) over a separate connection, always running as
//     the safety net (risk register #5: fsnotify misses WAL-only writes on some
//     filesystems). It emits only when the value actually changed.
//
// The two sources can occasionally both fire for one logical change; that is
// harmless — the frontend handler is an idempotent refetch, cheap at this scale.
type Watcher struct {
	dbPath   string
	onChange func()

	fw     *fsnotify.Watcher
	pollDB *sql.DB

	mu    sync.Mutex
	timer *time.Timer

	stop chan struct{}
	once sync.Once
}

// Watch starts watching dbPath for external writes. onChange fires at most once
// per 300ms burst of changes. The caller must Close the watcher on shutdown.
func Watch(dbPath string, onChange func()) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify: %w", err)
	}

	w := &Watcher{dbPath: dbPath, onChange: onChange, fw: fw, stop: make(chan struct{})}

	// Watch the DB file + WAL sidecars if present; missing ones are fine — the
	// poll fallback covers them until they appear (and Remove events re-add).
	for _, p := range w.watchedPaths() {
		if err := fw.Add(p); err != nil && !os.IsNotExist(err) {
			fw.Close()
			return nil, fmt.Errorf("watch %s: %w", p, err)
		}
	}

	// Separate connection for the poll (WAL allows concurrent readers/writers;
	// this one only ever reads). Same DSN shape as store.Open.
	dsn := (&url.URL{Scheme: "file", Path: dbPath}).RequestURI() + "?_busy_timeout=5000"
	pollDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		fw.Close()
		return nil, fmt.Errorf("open poll connection: %w", err)
	}
	w.pollDB = pollDB

	go w.eventLoop()
	go w.pollLoop()
	return w, nil
}

func (w *Watcher) watchedPaths() []string {
	return []string{w.dbPath, w.dbPath + "-wal", w.dbPath + "-shm"}
}

// eventLoop reacts to fsnotify events. Any event pokes the debounce; removed or
// renamed files are re-added best-effort (SQLite may recreate them).
func (w *Watcher) eventLoop() {
	for {
		select {
		case <-w.stop:
			return
		case ev, ok := <-w.fw.Events:
			if !ok {
				return
			}
			w.poke()
			if ev.Op.Has(fsnotify.Remove|fsnotify.Rename) {
				p := ev.Name
				go func(p string) {
					time.Sleep(200 * time.Millisecond) // give the writer a beat to recreate it
					select {
					case <-w.stop:
						return
					default:
					}
					_ = w.fw.Add(p)
				}(p)
			}
		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			log.Printf("sync watcher error: %v", err)
		}
	}
}

// pollLoop is the always-on safety net: every 2s it reads max(updated_at) and
// pokes only when the value changed. The first read establishes a baseline
// without emitting (no spurious event at startup).
func (w *Watcher) pollLoop() {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	var last string // "" = not yet seen or empty table
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			cur, err := w.maxUpdatedAt()
			if err != nil {
				continue // transient (e.g. file mid-recreate); next tick retries
			}
			if last == "" {
				last = cur
				continue
			}
			if cur != last {
				last = cur
				w.poke()
			}
		}
	}
}

func (w *Watcher) maxUpdatedAt() (string, error) {
	var s string
	err := w.pollDB.QueryRow(`SELECT COALESCE(MAX(updated_at), '') FROM tasks`).Scan(&s)
	return s, err
}

// poke schedules a debounced onChange; rapid successive changes coalesce into a
// single callback 300ms after the last one.
func (w *Watcher) poke() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(debounce, func() {
		select {
		case <-w.stop: // closed while the timer was in flight
			return
		default:
		}
		w.onChange()
	})
}

// Close stops both loops and releases resources. Safe to call multiple times;
// no callbacks fire after it returns.
func (w *Watcher) Close() {
	w.once.Do(func() {
		close(w.stop)
		w.mu.Lock()
		if w.timer != nil {
			w.timer.Stop()
			w.timer = nil
		}
		w.mu.Unlock()
		w.fw.Close()
		w.pollDB.Close()
	})
}
