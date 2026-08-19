package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Status is a task's lifecycle state. waiting is first-class: blocked on an
// external dependency, not a flag.
type Status string

const (
	StatusPending Status = "pending"
	StatusWIP     Status = "wip"
	StatusDone    Status = "done"
	StatusWaiting Status = "waiting"
)

var allStatuses = []Status{StatusPending, StatusWIP, StatusDone, StatusWaiting}

// ParseStatus validates a status string.
func ParseStatus(s string) (Status, error) {
	for _, st := range allStatuses {
		if s == string(st) {
			return st, nil
		}
	}
	return "", &InvalidStatusError{Status: s}
}

// Task is the canonical task object. JSON field names are a stable agent
// contract (see .agent/plan/04-cli-spec.md) — do not rename.
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	Progress    int        `json:"progress"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// CreateInput carries the fields accepted by add / CreateTask.
type CreateInput struct {
	Title       string
	Description string
	Status      Status // zero value → pending
	Progress    int    // zero value → 0
}

// UpdateInput carries the optional fields accepted by edit / UpdateTask;
// nil pointers mean "leave unchanged". At least one must be set.
type UpdateInput struct {
	Title    *string
	Desc     *string
	Progress *int
}

func (in UpdateInput) hasFields() bool {
	return in.Title != nil || in.Desc != nil || in.Progress != nil
}

// ListFilter drives list / ListTasks. Zero values give the CLI defaults:
// exclude done, sort updated_at desc, no limit.
type ListFilter struct {
	Status      Status // "" = any (subject to IncludeDone)
	Search      string // case-insensitive substring over title+description
	Limit       int    // 0 = unlimited
	Sort        string // created|updated|status|progress|title; default "updated"
	Ascending   bool   // false = descending (CLI: --sort field- for ascending)
	IncludeDone bool   // default false → done tasks are hidden unless matched by Status
}

// --- typed errors -----------------------------------------------------------
// CLI maps these to exit codes (1 validation, 2 not-found/ambiguous); the GUI
// maps them to specific toasts.

// ErrNotFound is returned when an ID or prefix matches no task.
var ErrNotFound = errors.New("task not found")

// AmbiguousIDError lists candidates so callers can show them.
type AmbiguousIDError struct {
	Ref        string
	Candidates []string // full IDs, sorted
}

func (e *AmbiguousIDError) Error() string {
	return fmt.Sprintf("ambiguous id prefix %q: matches %s", e.Ref, strings.Join(e.Candidates, ", "))
}

// InvalidStatusError is returned for an unknown status value.
type InvalidStatusError struct{ Status string }

func (e *InvalidStatusError) Error() string {
	return fmt.Sprintf("invalid status %q (want pending, wip, done or waiting)", e.Status)
}

var ErrEmptyTitle = errors.New("title must not be empty")

// ProgressRangeError is returned for progress outside 0..100.
type ProgressRangeError struct{ Value int }

func (e *ProgressRangeError) Error() string { return fmt.Sprintf("progress %d out of range 0-100", e.Value) }

var ErrNoFieldsToUpdate = errors.New("no fields to update")

// MinPrefixLen is the shortest prefix accepted for ID lookup.
const MinPrefixLen = 4
