package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Status is a task's lifecycle state. waiting is first-class: blocked on an
// external dependency, not a flag. review sits after waiting (v0.3).
type Status string

const (
	StatusPending Status = "pending"
	StatusWIP     Status = "wip"
	StatusWaiting Status = "waiting"
	StatusReview  Status = "review"
	StatusDone    Status = "done"
)

var allStatuses = []Status{StatusPending, StatusWIP, StatusWaiting, StatusReview, StatusDone}

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
// contract — do not rename.
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Feedback    string     `json:"feedback"` // agent-authored; GUI shows only when non-empty
	Status      Status     `json:"status"`
	Progress    int        `json:"progress"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
	ArchivedAt  *time.Time `json:"archived_at"` // set on archive, cleared on unarchive (v0.2)
	ParentID    *string    `json:"parent_id"`   // nil = root; one-level children only (v0.3)
}

// CreateInput carries the fields accepted by add / CreateTask.
type CreateInput struct {
	Title       string
	Description string
	Feedback    string
	Status      Status // zero value → pending
	Progress    int    // zero value → 0
	ParentID    string // optional; empty = root. Must resolve to a root task.
}

// UpdateInput carries the optional fields accepted by edit / UpdateTask;
// nil pointers mean "leave unchanged". At least one must be set.
type UpdateInput struct {
	Title    *string
	Desc     *string
	Feedback *string
	Progress *int
}

func (in UpdateInput) hasFields() bool {
	return in.Title != nil || in.Desc != nil || in.Feedback != nil || in.Progress != nil
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
	Archived    bool   // true → archived tasks only; default false → archived tasks excluded
	RootsOnly   bool   // true → parent_id IS NULL only (v0.3)
}

// Activity is an agent/user-authored note on a task (v0.3). Not auto-logged.
// At least one of Activity/Comment must be non-empty after trim.
type Activity struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	Activity  string    `json:"activity"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// ActivityInput is accepted by AddActivity.
type ActivityInput struct {
	Activity string
	Comment  string
}

// ActivityFilter drives ListActivity. Empty TaskIDs = all tasks (still
// excludes archived tasks' activity unless IncludeArchived).
type ActivityFilter struct {
	TaskIDs         []string // resolved full IDs; empty = any non-archived task
	Limit           int      // 0 = unlimited
	IncludeArchived bool     // default false → hide activity on archived tasks
}

// --- typed errors -----------------------------------------------------------

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
	return fmt.Sprintf("invalid status %q (want pending, wip, waiting, review or done)", e.Status)
}

var ErrEmptyTitle = errors.New("title must not be empty")

// ProgressRangeError is returned for progress outside 0..100.
type ProgressRangeError struct{ Value int }

func (e *ProgressRangeError) Error() string {
	return fmt.Sprintf("progress %d out of range 0-100", e.Value)
}

var ErrNoFieldsToUpdate = errors.New("no fields to update")

// ErrNotArchived is returned when unarchive is called on a non-archived task.
var ErrNotArchived = errors.New("task is not archived")

// ErrParentIsChild is returned when --parent points at a sub-task (one level only).
var ErrParentIsChild = errors.New("parent must be a top-level task (sub-tasks cannot have children)")

// ErrEmptyActivity is returned when both activity and comment are empty.
var ErrEmptyActivity = errors.New("activity or comment is required")

// MinPrefixLen is the shortest prefix accepted for ID lookup.
const MinPrefixLen = 4
