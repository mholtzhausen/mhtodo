package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"mhtodo/internal/cli"
	"mhtodo/internal/core"
	"mhtodo/internal/store"
	"mhtodo/internal/update"
)

// newCLI points MHTODO_DB_PATH at a temp DB and returns (stdout, stderr, run).
func newCLI(t *testing.T) (*bytes.Buffer, *bytes.Buffer, func(args ...string) int) {
	t.Helper()
	db := filepath.Join(t.TempDir(), "mhtodo.db")
	t.Setenv("MHTODO_DB_PATH", db)

	var out, errb bytes.Buffer
	run := func(args ...string) int { return cli.Execute(args, "test", "none", &out, &errb) }
	return &out, &errb, run
}

// seedTasks inserts tasks with fixed IDs/timestamps directly via the store so
// prefix/ambiguity cases are deterministic.
func seedTasks(t *testing.T, db string, tasks ...core.Task) {
	t.Helper()
	repo, err := store.Open(db)
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()
	for _, task := range tasks {
		if err := repo.Create(ctx, task); err != nil {
			t.Fatalf("seed %s: %v", task.ID, err)
		}
	}
}

func fixedTask(id, title string, st core.Status, prog int) core.Task {
	now := time.Date(2026, 8, 19, 7, 59, 0, 0, time.UTC)
	return core.Task{ID: id, Title: title, Status: st, Progress: prog, CreatedAt: now, UpdatedAt: now}
}

func mustJSON(t *testing.T, b []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("unmarshal %s: %v", b, err)
	}
}

// --- add ----------------------------------------------------------------------

func TestAddHumanAndJSON(t *testing.T) {
	out, _, run := newCLI(t)

	if code := run("add", "Ship v0.1", "--desc", "see plan", "--feedback", "agent note"); code != 0 {
		t.Fatalf("exit %d", code)
	}
	fields := strings.Fields(out.String()) // title may contain spaces → id, status, rest...
	if len(fields) < 3 || fields[1] != "pending" || strings.Join(fields[2:], " ") != "Ship v0.1" || len(fields[0]) != 36 {
		t.Errorf("human add line wrong: %q", out.String())
	}

	out.Reset()
	if code := run("add", "JSON task", "--status", "wip", "--progress", "40", "--feedback", "ok", "--json"); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var task core.Task
	mustJSON(t, out.Bytes(), &task)
	if task.Title != "JSON task" || task.Status != core.StatusWIP || task.Progress != 40 ||
		task.Feedback != "ok" || task.CompletedAt != nil || len(task.ID) != 36 {
		t.Errorf("json add wrong: %+v", task)
	}
	if !strings.Contains(out.String(), `"created_at": "20`) { // RFC3339, second precision
		t.Errorf("timestamps not RFC3339: %s", out.String())
	}

	out.Reset()
	if code := run("-q", "add", "quiet task"); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if id := strings.TrimSpace(out.String()); len(id) != 36 {
		t.Errorf("quiet add should print only the ID: %q", out.String())
	}
}

func TestAddErrors(t *testing.T) {
	out, errb, run := newCLI(t)

	cases := []struct {
		args   []string
		code   int
		name   string // expected JSON error name ("" = plain check only)
		substr string
	}{
		{[]string{"add", "   "}, 1, "empty_title", "title must not be empty"},
		{[]string{"add", "x", "--status", "nope"}, 1, "invalid_status", `invalid status "nope"`},
		{[]string{"add", "x", "--progress", "101"}, 1, "progress_range", "out of range"},
		{[]string{"add"}, 1, "", "accepts 1 arg(s)"},
	}
	for _, tc := range cases {
		out.Reset()
		errb.Reset()
		if code := run(tc.args...); code != tc.code {
			t.Errorf("%v: exit %d, want %d (stderr: %s)", tc.args, code, tc.code, errb.String())
		}
		if got := strings.TrimSpace(errb.String()); !strings.Contains(got, "mhtodo: ") || !strings.Contains(got, tc.substr) {
			t.Errorf("%v: stderr %q missing mhtodo:/%q", tc.args, got, tc.substr)
		}

		out.Reset()
		errb.Reset()
		jsonArgs := append(append([]string{}, tc.args...), "--json")
		if code := run(jsonArgs...); code != tc.code {
			t.Errorf("%v --json: exit %d, want %d", jsonArgs, code, tc.code)
		}
		var env struct{ Error, Message string }
		mustJSON(t, errb.Bytes(), &env)
		if tc.name != "" && env.Error != tc.name {
			t.Errorf("%v --json: error name %q, want %q", jsonArgs, env.Error, tc.name)
		}
		if !strings.Contains(env.Message, tc.substr) {
			t.Errorf("%v --json: message %q missing %q", jsonArgs, env.Message, tc.substr)
		}
	}
}

// --- list ---------------------------------------------------------------------

func TestListFiltersAndSort(t *testing.T) {
	out, _, run := newCLI(t)

	if code := run("add", "Alpha"); code != 0 {
		t.Fatal(code)
	}
	out.Reset()
	if code := run("add", "Beta", "--status", "done"); code != 0 {
		t.Fatal(code)
	}
	idBeta := strings.Fields(out.String())[0]
	out.Reset()

	// Default excludes done.
	var tasks []core.Task
	if code := run("list", "--json"); code != 0 {
		t.Fatalf("exit %d", code)
	}
	mustJSON(t, out.Bytes(), &tasks)
	if len(tasks) != 1 || tasks[0].Title != "Alpha" {
		t.Errorf("default list should show only Alpha: %+v", tasks)
	}

	// --all includes done.
	out.Reset()
	run("list", "--all", "--json")
	mustJSON(t, out.Bytes(), &tasks)
	if len(tasks) != 2 {
		t.Errorf("--all should show both: %+v", tasks)
	}

	// --status filter (done is visible when explicitly requested).
	out.Reset()
	run("list", "--status", "done", "--json")
	mustJSON(t, out.Bytes(), &tasks)
	if len(tasks) != 1 || tasks[0].ID != idBeta {
		t.Errorf("--status done wrong: %+v", tasks)
	}

	// --search is case-insensitive over title+description.
	out.Reset()
	run("list", "--search", "ALPHA", "--json")
	mustJSON(t, out.Bytes(), &tasks)
	if len(tasks) != 1 || tasks[0].Title != "Alpha" {
		t.Errorf("--search wrong: %+v", tasks)
	}

	// --limit.
	out.Reset()
	run("list", "--all", "--limit", "1", "--json")
	mustJSON(t, out.Bytes(), &tasks)
	if len(tasks) != 1 {
		t.Errorf("--limit 1 wrong: %+v", tasks)
	}

	// --sort title- (ascending per spec).
	out.Reset()
	run("list", "--all", "--sort", "title-", "--json")
	mustJSON(t, out.Bytes(), &tasks)
	if len(tasks) != 2 || tasks[0].Title != "Alpha" || tasks[1].Title != "Beta" {
		t.Errorf("--sort title- wrong: %+v", tasks)
	}

	// Human format: aligned columns ID(8) STATUS PROG UPDATED(rel+abs) TITLE.
	out.Reset()
	run("list")
	line := strings.TrimSpace(out.String())
	re := regexp.MustCompile(`^\S{8}\s+pending\s+0%\s+(?:just now|\d+[mhd] ago) \(20\d\d-\d\d-\d\d \d\d:\d\d\)\s+Alpha$`)
	if !re.MatchString(line) {
		t.Errorf("human list line wrong: %q", line)
	}

	// Invalid sort field → exit 1.
	errb := new(bytes.Buffer)
	code := cli.Execute([]string{"list", "--sort", "bogus"}, "test", "none", out, errb)
	if code != 1 || !strings.Contains(errb.String(), `invalid --sort field`) {
		t.Errorf("bad sort: exit %d stderr %q", code, errb.String())
	}

	// Empty list → empty human output, [] in JSON.
	out2, _, run2 := newCLI(t)
	if code := run2("list"); code != 0 || out2.Len() != 0 {
		t.Errorf("empty list: exit %d out %q", code, out2.String())
	}
	out2.Reset()
	run2("list", "--json")
	if s := strings.TrimSpace(out2.String()); s != "[]" {
		t.Errorf("empty json list should be []: %q", s)
	}
}

func TestReorderCLI(t *testing.T) {
	out, _, run := newCLI(t)
	if code := run("add", "Alpha"); code != 0 {
		t.Fatal(code)
	}
	idA := strings.Fields(out.String())[0]
	out.Reset()
	if code := run("add", "Beta"); code != 0 {
		t.Fatal(code)
	}
	out.Reset()
	if code := run("add", "Gamma"); code != 0 {
		t.Fatal(code)
	}
	idC := strings.Fields(out.String())[0]
	out.Reset()

	if code := run("reorder", idC, "--before", idA, "--json"); code != 0 {
		t.Fatalf("reorder exit %d", code)
	}
	var moved core.Task
	mustJSON(t, out.Bytes(), &moved)
	if moved.BoardRank == nil {
		t.Fatalf("reorder json missing board_rank: %+v", moved)
	}

	out.Reset()
	run("list", "--json")
	var tasks []core.Task
	mustJSON(t, out.Bytes(), &tasks)
	if len(tasks) != 3 || tasks[0].Title != "Gamma" || tasks[1].Title != "Alpha" || tasks[2].Title != "Beta" {
		t.Errorf("after reorder --before first: %+v", tasks)
	}
}

// --- show ---------------------------------------------------------------------

func TestShow(t *testing.T) {
	out, errb, run := newCLI(t)
	db := os.Getenv("MHTODO_DB_PATH")
	seedTasks(t, db,
		fixedTask("aaaa1111-0000-7000-8000-000000000001", "First", core.StatusWIP, 40),
		fixedTask("bbbb2222-0000-7000-8000-000000000002", "Second", core.StatusPending, 0),
	)

	// Full ID → JSON canonical shape.
	var task core.Task
	if code := run("show", "aaaa1111-0000-7000-8000-000000000001", "--json"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &task)
	if task.Title != "First" || task.Status != core.StatusWIP || task.Progress != 40 {
		t.Errorf("show wrong: %+v", task)
	}

	// Unique prefix (>= 4 chars).
	out.Reset()
	if code := run("show", "bbbb"); code != 0 {
		t.Fatalf("prefix exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "Second") || !strings.Contains(out.String(), "bbbb2222-0000-7000-8000-000000000002") {
		t.Errorf("human show wrong:\n%s", out.String())
	}

	// Too-short ref → not found (exit 2).
	out.Reset()
	errb.Reset()
	if code := run("show", "abc"); code != cli.ExitNotFound {
		t.Errorf("short ref: exit %d, want 2 (%s)", code, errb.String())
	}
	var env struct{ Error string }
	errb.Reset()
	if code := run("show", "abc", "--json"); code != cli.ExitNotFound {
		t.Errorf("short ref json: exit %d", code)
	}
	mustJSON(t, errb.Bytes(), &env)
	if env.Error != "not_found" {
		t.Errorf("error name %q, want not_found", env.Error)
	}

	// Ambiguous prefix → exit 2 with candidates.
	out.Reset()
	errb.Reset()
	seedTasks(t, db, fixedTask("aaaa9999-0000-7000-8000-000000000003", "Third", core.StatusPending, 0))
	if code := run("show", "aaaa"); code != cli.ExitNotFound {
		t.Errorf("ambiguous: exit %d, want 2 (%s)", code, errb.String())
	}
	if !strings.Contains(errb.String(), "ambiguous") ||
		!strings.Contains(errb.String(), "aaaa1111-0000-7000-8000-000000000001") {
		t.Errorf("ambiguous stderr missing candidates: %s", errb.String())
	}
	errb.Reset()
	run("show", "aaaa", "--json")
	mustJSON(t, errb.Bytes(), &env)
	if env.Error != "ambiguous_id" {
		t.Errorf("error name %q, want ambiguous_id", env.Error)
	}

	// Unknown ID → exit 2.
	errb.Reset()
	if code := run("show", "zzzz9999-0000-7000-8000-000000000004"); code != cli.ExitNotFound {
		t.Errorf("unknown: exit %d, want 2 (%s)", code, errb.String())
	}
}

// --- edit ---------------------------------------------------------------------

func TestEdit(t *testing.T) {
	out, errb, run := newCLI(t)
	id := strings.Fields(runMust(out, run, "add", "Original"))[0]
	out.Reset()

	// No flags → exit 1 no_fields.
	errb.Reset()
	if code := run("edit", id); code != 1 {
		t.Fatalf("no fields: exit %d (%s)", code, errb.String())
	}
	var env struct{ Error string }
	errb.Reset()
	run("edit", id, "--json")
	mustJSON(t, errb.Bytes(), &env)
	if env.Error != "no_fields" {
		t.Errorf("error name %q, want no_fields", env.Error)
	}

	// Title + progress update (human block), then the same via --json.
	out.Reset()
	errb.Reset()
	if code := run("edit", id, "--title", "Renamed", "--progress", "55"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "Renamed") || !strings.Contains(out.String(), "55%") {
		t.Errorf("human edit block wrong:\n%s", out.String())
	}

	var task core.Task
	out.Reset()
	errb.Reset()
	if code := run("edit", id, "--title", "Renamed2", "--progress", "60", "--feedback", "looks good", "--json"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &task)
	if task.Title != "Renamed2" || task.Progress != 60 || task.Feedback != "looks good" || task.Status != core.StatusPending {
		t.Errorf("edit wrong: %+v", task)
	}

	// Validation + not-found paths.
	errb.Reset()
	if code := run("edit", id, "--progress", "-5"); code != 1 {
		t.Errorf("bad progress: exit %d (%s)", code, errb.String())
	}
	errb.Reset()
	if code := run("edit", "zzzz9999-0000-7000-8000-000000000004", "--title", "x"); code != cli.ExitNotFound {
		t.Errorf("unknown id: exit %d (%s)", code, errb.String())
	}
}

// --- status / done --------------------------------------------------------------

func TestStatusAndDone(t *testing.T) {
	out, errb, run := newCLI(t)
	id := strings.Fields(runMust(out, run, "add", "Task"))[0]
	out.Reset()

	var task core.Task
	if code := run("status", id, "wip", "--json"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &task)
	if task.Status != core.StatusWIP || task.CompletedAt != nil {
		t.Errorf("→wip wrong: %+v", task)
	}

	out.Reset()
	if code := run("done", id, "--json"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &task)
	if task.Status != core.StatusDone || task.Progress != 100 || task.CompletedAt == nil {
		t.Errorf("→done effects missing: %+v", task)
	}

	out.Reset()
	if code := run("status", id, "waiting", "--json"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &task)
	if task.Status != core.StatusWaiting || task.CompletedAt != nil {
		t.Errorf("done→waiting should clear completed_at: %+v", task)
	}

	// Invalid status → exit 1.
	errb.Reset()
	if code := run("status", id, "blocked"); code != 1 {
		t.Errorf("bad status: exit %d (%s)", code, errb.String())
	}
	var env struct{ Error string }
	errb.Reset()
	run("status", id, "blocked", "--json")
	mustJSON(t, errb.Bytes(), &env)
	if env.Error != "invalid_status" {
		t.Errorf("error name %q, want invalid_status", env.Error)
	}

	// Unknown ID → exit 2.
	errb.Reset()
	if code := run("done", "zzzz9999-0000-7000-8000-000000000004"); code != cli.ExitNotFound {
		t.Errorf("unknown id: exit %d (%s)", code, errb.String())
	}
}

func TestDoneNotifyFlag(t *testing.T) {
	out, errb, run := newCLI(t)
	id := strings.Fields(runMust(out, run, "add", "Notify me"))[0]
	out.Reset()

	// Record notifications instead of exec'ing notify-send (test seam).
	var notified []string
	orig := cli.NotifyDone
	cli.NotifyDone = func(id, title string) { notified = append(notified, id+"|"+title) }
	defer func() { cli.NotifyDone = orig }()

	if code := run("done", id, "--notify"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if len(notified) != 1 || notified[0] != id+"|Notify me" {
		t.Errorf("--notify should fire once for the transition, got %v", notified)
	}

	out.Reset()
	notified = nil
	// Already done → no-op re-set must not notify (matches GUI transition semantics).
	if code := run("done", id, "--notify"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if len(notified) != 0 {
		t.Errorf("no-op re-set should not notify, got %v", notified)
	}

	out.Reset()
	notified = nil
	// Without --notify → never notifies.
	runMust(out, run, "status", id, "wip")
	if code := run("done", id); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if len(notified) != 0 {
		t.Errorf("plain done should not notify, got %v", notified)
	}

	out.Reset()
	notified = nil
	// --notify must not change the printed object or exit code (agent contract).
	runMust(out, run, "status", id, "wip")
	if code := run("done", id); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	plain := out.String()
	out.Reset()
	notified = nil
	runMust(out, run, "status", id, "wip")
	if code := run("done", id, "--notify"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if strings.TrimSpace(plain) != strings.TrimSpace(out.String()) {
		t.Errorf("--notify changed output:\n%s\nvs\n%s", plain, out.String())
	}
	if len(notified) != 1 {
		t.Errorf("expected one notification, got %v", notified)
	}
}

// --- rm --------------------------------------------------------------------------

func TestRm(t *testing.T) {
	out, errb, run := newCLI(t)
	id := strings.Fields(runMust(out, run, "add", "Doomed"))[0]
	out.Reset()

	// Non-TTY without --yes → exit 1 (the agent guard).
	t.Cleanup(func() { cli.Stdin = os.Stdin })
	cli.Stdin = strings.NewReader("") // never a TTY
	errb.Reset()
	if code := run("rm", id); code != 1 {
		t.Fatalf("non-tty rm: exit %d (%s)", code, errb.String())
	}
	if !strings.Contains(errb.String(), "--yes") {
		t.Errorf("stderr should mention --yes: %s", errb.String())
	}

	// With --yes → prints the id only; task is gone.
	out.Reset()
	errb.Reset()
	if code := run("rm", id, "--yes"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if got := strings.TrimSpace(out.String()); got != id {
		t.Errorf("rm should print only the id: %q", out.String())
	}
	errb.Reset()
	if code := run("show", id); code != cli.ExitNotFound {
		t.Errorf("task should be deleted: exit %d (%s)", code, errb.String())
	}

	// JSON mode → {"id": ...}.
	out2, _, run2 := newCLI(t)
	id2 := strings.Fields(runMust(out2, run2, "add", "Doomed2"))[0]
	out2.Reset()
	if code := run2("rm", id2, "--yes", "--json"); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var env struct{ ID string }
	mustJSON(t, out2.Bytes(), &env)
	if env.ID != id2 {
		t.Errorf("json rm wrong: %+v", env)
	}

	// Unknown ID → exit 2.
	errb.Reset()
	if code := run("rm", "zzzz9999-0000-7000-8000-000000000004"); code != cli.ExitNotFound {
		t.Errorf("unknown id: exit %d (%s)", code, errb.String())
	}
}

// --- path / misc -------------------------------------------------------------------

func TestPath(t *testing.T) {
	out, _, run := newCLI(t)
	want := os.Getenv("MHTODO_DB_PATH")
	if code := run("path"); code != 0 || strings.TrimSpace(out.String()) != want {
		t.Errorf("path: exit %d out %q want %q", code, out.String(), want)
	}
	out.Reset()
	run("path", "--json")
	var s string
	mustJSON(t, out.Bytes(), &s)
	if s != want {
		t.Errorf("json path wrong: %q", s)
	}
}

func TestAI(t *testing.T) {
	fixed := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	prev := cli.AINowForTest(func() time.Time { return fixed })
	t.Cleanup(prev)

	out, _, run := newCLI(t)
	db := os.Getenv("MHTODO_DB_PATH")

	if code := run("ai"); code != 0 {
		t.Fatalf("ai: exit %d", code)
	}
	body := out.String()
	for _, want := range []string{
		"mhtodo — agent integration instructions",
		"Integration contract version: 6",
		"mhtodo binary version:        test",
		"Database:                     " + db,
		"Generated:                    2026-08-27T12:00:00Z",
		"pending|wip|waiting|review|done",
		"board|created|updated|status|progress|title",
		"v6  Task-picker turns",
		"AskUserQuestion",
		"v5  Sub-tasks are a mandatory step plan",
		"v4  Activity labels",
		"--feedback",
		"Markdown fields",
		"Task Picked Up",
		"mhtodo ai", // listed in §2 CLI surface
	} {
		if !strings.Contains(body, want) {
			t.Errorf("ai output missing %q", want)
		}
	}
	if strings.Contains(body, "{{") {
		t.Errorf("uninterpolated placeholder left in output")
	}

	out.Reset()
	if code := run("ai", "--json"); code != 0 {
		t.Fatalf("ai --json: exit %d", code)
	}
	var doc struct {
		IntegrationVersion int    `json:"integration_version"`
		MhtodoVersion      string `json:"mhtodo_version"`
		DBPath             string `json:"db_path"`
		Generated          string `json:"generated"`
		Content            string `json:"content"`
	}
	mustJSON(t, out.Bytes(), &doc)
	if doc.IntegrationVersion != 6 || doc.MhtodoVersion != "test" || doc.DBPath != db ||
		doc.Generated != "2026-08-27T12:00:00Z" || !strings.Contains(doc.Content, "agent integration") {
		t.Errorf("ai --json envelope wrong: %+v", doc)
	}
}

func TestUpdateCheckJSON(t *testing.T) {
	prev := cli.UpdateRunForTest(func(opts update.Options) (update.Result, error) {
		if !opts.CheckOnly || opts.CurrentVersion != "test" {
			t.Fatalf("unexpected opts: %+v", opts)
		}
		return update.Result{
			CurrentVersion: "test",
			LatestVersion:  "9.9.9",
			UpToDate:       false,
			CheckOnly:      true,
			InstallPath:    "/tmp/mhtodo",
			Service:        true,
			Message:        "update available: vtest → v9.9.9",
		}, nil
	})
	defer prev()

	out, errb, run := newCLI(t)
	if code := run("update", "--check", "--json"); code != 0 {
		t.Fatalf("exit %d (%s)", code, errb.String())
	}
	var res update.Result
	mustJSON(t, out.Bytes(), &res)
	if res.LatestVersion != "9.9.9" || res.UpToDate || !res.Service {
		t.Fatalf("json: %+v", res)
	}

	out.Reset()
	if code := run("update", "--check"); code != 0 {
		t.Fatalf("human: exit %d", code)
	}
	if !strings.Contains(out.String(), "update available") {
		t.Errorf("human: %q", out.String())
	}
}

func TestUnknownCommandAndVersion(t *testing.T) {
	out, errb, run := newCLI(t)

	errb.Reset()
	if code := run("frobnicate"); code != 1 {
		t.Fatalf("unknown command: exit %d (%s)", code, errb.String())
	}
	got := errb.String()
	if !strings.Contains(got, `mhtodo: unknown command "frobnicate"`) || !strings.Contains(got, "available commands:") {
		t.Errorf("unknown command stderr wrong: %q", got)
	}
	if !strings.Contains(got, "update") {
		t.Errorf("available commands missing update: %q", got)
	}

	out.Reset()
	errb.Reset()
	if code := run("--version"); code != 0 {
		t.Fatalf("--version: exit %d (%s)", code, errb.String())
	}
	if !strings.Contains(out.String(), "mhtodo version test") {
		t.Errorf("--version output wrong: %q", out.String())
	}

	// Bad flag value → usage error exit 1.
	errb.Reset()
	if code := run("add", "x", "--progress", "notanint"); code != 1 {
		t.Fatalf("bad int flag: exit %d (%s)", code, errb.String())
	}
	if !strings.Contains(errb.String(), "mhtodo:") {
		t.Errorf("stderr format wrong: %q", errb.String())
	}
}

// runMust runs a command that must succeed and returns its stdout.
func runMust(out *bytes.Buffer, run func(args ...string) int, args ...string) string {
	if code := run(args...); code != 0 {
		panic("run failed: " + strings.Join(args, " "))
	}
	s := out.String()
	out.Reset()
	return s
}

// --- archive / unarchive (v0.2) ----------------------------------------------

func TestArchiveAndUnarchive(t *testing.T) {
	out, errb, run := newCLI(t)

	if code := run("add", "Finished thing"); code != 0 {
		t.Fatal(code)
	}
	out.Reset()
	if code := run("add", "Done one", "--status", "done"); code != 0 {
		t.Fatal(code)
	}
	id1 := strings.Fields(out.String())[0]
	out.Reset()
	if code := run("add", "Done two", "--status", "done"); code != 0 {
		t.Fatal(code)
	}
	id2 := strings.Fields(out.String())[0]

	// archive → JSON array of exactly the done tasks, each with archived_at set.
	out.Reset()
	if code := run("archive", "--json"); code != 0 {
		t.Fatalf("exit %d (stderr: %s)", code, errb.String())
	}
	var archived []core.Task
	mustJSON(t, out.Bytes(), &archived)
	if len(archived) != 2 {
		t.Fatalf("archive returned %d tasks, want 2: %+v", len(archived), archived)
	}
	gotIDs := map[string]bool{archived[0].ID: true, archived[1].ID: true}
	if !gotIDs[id1] || !gotIDs[id2] {
		t.Errorf("archive returned wrong ids: %v (want %s + %s)", gotIDs, id1, id2)
	}
	for _, tsk := range archived {
		if tsk.ArchivedAt == nil || tsk.Status != core.StatusDone {
			t.Errorf("archived task wrong: %+v", tsk)
		}
	}

	// Default list and --all now hide the archived tasks; only "Finished thing" shows.
	out.Reset()
	run("list", "--all", "--json")
	mustJSON(t, out.Bytes(), &archived)
	if len(archived) != 1 || archived[0].Title != "Finished thing" {
		t.Errorf("--all should hide archived: %+v", archived)
	}

	// list --archived shows exactly the two.
	out.Reset()
	if code := run("list", "--archived", "--json"); code != 0 {
		t.Fatalf("exit %d", code)
	}
	mustJSON(t, out.Bytes(), &archived)
	if len(archived) != 2 {
		t.Errorf("--archived wrong: %+v", archived)
	}

	// unarchive by prefix → pending, progress reset.
	out.Reset()
	if code := run("unarchive", id1, "--json"); code != 0 {
		t.Fatalf("exit %d (stderr: %s)", code, errb.String())
	}
	var task core.Task
	mustJSON(t, out.Bytes(), &task)
	if task.ID != id1 || task.Status != core.StatusPending || task.Progress != 0 ||
		task.CompletedAt != nil || task.ArchivedAt != nil {
		t.Errorf("unarchived task wrong: %+v", task)
	}

	// It is back in the default list (order is updated_at desc — check as a set).
	out.Reset()
	run("list", "--json")
	mustJSON(t, out.Bytes(), &archived)
	if len(archived) != 2 {
		t.Fatalf("default list after unarchive = %d tasks: %+v", len(archived), archived)
	}
	titles := map[string]bool{archived[0].Title: true, archived[1].Title: true}
	if !titles["Finished thing"] || !titles["Done one"] {
		t.Errorf("default list after unarchive wrong titles: %v", titles)
	}

	// unarchive on a non-archived task (id1 was restored to pending above) →
	// exit 1, not_archived.
	out.Reset()
	errb.Reset()
	if code := run("unarchive", id1); code != 1 {
		t.Fatalf("exit %d, want 1 (stderr: %s)", code, errb.String())
	}
	if got := strings.TrimSpace(errb.String()); !strings.Contains(got, "not archived") {
		t.Errorf("stderr missing 'not archived': %q", got)
	}

	out.Reset()
	errb.Reset()
	if code := run("unarchive", id1, "--json"); code != 1 {
		t.Fatalf("--json exit %d, want 1", code)
	}
	var env struct{ Error, Message string }
	mustJSON(t, errb.Bytes(), &env)
	if env.Error != "not_archived" {
		t.Errorf("error name = %q, want not_archived", env.Error)
	}

	// archive with nothing done → empty output, exit 0.
	out.Reset()
	errb.Reset()
	if code := run("archive"); code != 0 {
		t.Fatalf("exit %d (stderr: %s)", code, errb.String())
	}
	if got := strings.TrimSpace(out.String()); got != "" {
		t.Errorf("empty archive should print nothing: %q", got)
	}

	// unarchive unknown id → exit 2.
	out.Reset()
	errb.Reset()
	if code := run("unarchive", "zzzz9999"); code != 2 {
		t.Fatalf("exit %d, want 2 (stderr: %s)", code, errb.String())
	}
}

func TestParentAndReviewAndActivity(t *testing.T) {
	out, errb, run := newCLI(t)
	db := os.Getenv("MHTODO_DB_PATH")

	parentID := "aaaa1111-0000-7000-8000-000000000001"
	childID := "bbbb2222-0000-7000-8000-000000000001"
	seedTasks(t, db,
		fixedTask(parentID, "Parent", core.StatusPending, 0),
		func() core.Task {
			t := fixedTask(childID, "Child", core.StatusPending, 0)
			pid := parentID
			t.ParentID = &pid
			return t
		}(),
	)

	if code := run("add", "Grand", "--parent", childID, "--json"); code != 1 {
		t.Fatalf("nest under child exit %d want 1 (%s)", code, errb.String())
	}
	errb.Reset()
	out.Reset()

	if code := run("status", parentID, "review", "--json"); code != 0 {
		t.Fatalf("review: %d %s", code, errb.String())
	}
	var reviewed core.Task
	mustJSON(t, out.Bytes(), &reviewed)
	if reviewed.Status != core.StatusReview {
		t.Fatalf("status: %+v", reviewed)
	}
	out.Reset()

	if code := run("list", "--roots", "--json"); code != 0 {
		t.Fatalf("roots: %d", code)
	}
	var roots []core.Task
	mustJSON(t, out.Bytes(), &roots)
	for _, tsk := range roots {
		if tsk.ParentID != nil {
			t.Fatalf("--roots returned child: %+v", tsk)
		}
	}
	out.Reset()

	if code := run("activity", "add", parentID, "--activity", "Agent did X", "--comment", "note", "--json"); code != 0 {
		t.Fatalf("activity add: %d %s", code, errb.String())
	}
	var act core.Activity
	mustJSON(t, out.Bytes(), &act)
	if act.Activity != "Agent did X" || act.Comment != "note" {
		t.Fatalf("activity: %+v", act)
	}
	out.Reset()

	if code := run("activity", "list", "--json"); code != 0 {
		t.Fatalf("activity list: %d", code)
	}
	var acts []core.Activity
	mustJSON(t, out.Bytes(), &acts)
	if len(acts) != 1 {
		t.Fatalf("want 1 activity, got %d", len(acts))
	}
	out.Reset()

	if code := run("activity", "rm", act.ID, "--yes", "--json"); code != 0 {
		t.Fatalf("activity rm: %d %s", code, errb.String())
	}

	if code := run("rm", parentID, "--yes"); code != 0 {
		t.Fatalf("rm parent: %d %s", code, errb.String())
	}
	if code := run("show", childID); code != 2 {
		t.Fatalf("child should be gone: exit %d", code)
	}
}
