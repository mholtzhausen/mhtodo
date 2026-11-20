package cli_test

import (
	"strings"
	"testing"

	"mhtodo/internal/core"
)

// list --search also matches ticket IDs (fragments, short IDs, UUIDs, links)
// without changing the JSON shape or text-search results.
func TestListSearchByTicketID(t *testing.T) {
	out, errb, run := newCLI(t)

	add := func(args ...string) core.Task {
		t.Helper()
		out.Reset()
		if code := run(append([]string{"add"}, append(args, "--json")...)...); code != 0 {
			t.Fatalf("add %v exit %d: %s", args, code, errb.String())
		}
		var tk core.Task
		mustJSON(t, out.Bytes(), &tk)
		return tk
	}
	alpha := add("Alpha")
	add("Facade work")
	sub := add("Sub step", "--parent", alpha.ID)

	search := func(q string) []string {
		t.Helper()
		out.Reset()
		if code := run("list", "--search", q, "--json"); code != 0 {
			t.Fatalf("list --search %q exit %d: %s", q, code, errb.String())
		}
		var ts []core.Task
		mustJSON(t, out.Bytes(), &ts)
		ids := make([]string, len(ts))
		for i, tk := range ts {
			ids[i] = tk.ID
		}
		return ids
	}
	only := func(q, want string) {
		t.Helper()
		if got := search(q); len(got) != 1 || got[0] != want {
			t.Errorf("--search %q = %v, want [%s]", q, got, want)
		}
	}

	dashless := strings.ReplaceAll(alpha.ID, "-", "")
	only(core.ShortID(alpha.ID), alpha.ID)
	only(alpha.ID, alpha.ID)
	only(dashless, alpha.ID)
	only(strings.ToUpper(alpha.ID), alpha.ID)
	only("mhtodo://task/"+core.ShortID(alpha.ID), alpha.ID)
	only("MHTodo Ticket: mhtodo://task/"+alpha.ID, alpha.ID)
	// A sub-task's ID returns just the sub-task: the CLI stays flat.
	only(core.ShortID(sub.ID), sub.ID)

	// Below 6 hex chars the query is text-only: the shared UUIDv7 prefix
	// must not flood results.
	if got := search(alpha.ID[:4]); len(got) != 0 {
		t.Errorf("--search %q (too short) = %v, want none", alpha.ID[:4], got)
	}
	// An all-hex word still finds its text match.
	if got := search("facade"); len(got) != 1 {
		t.Errorf("--search facade = %v, want the Facade ticket", got)
	}

	// JSON shape unchanged: still an array of task objects with the usual fields.
	out.Reset()
	run("list", "--search", core.ShortID(alpha.ID), "--json")
	for _, field := range []string{`"id"`, `"title"`, `"status"`, `"parent_id"`} {
		if !strings.Contains(out.String(), field) {
			t.Errorf("json lacks %s: %s", field, out.String())
		}
	}
}
