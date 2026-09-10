package traymenu

import "testing"

func TestFormatAttentionLabel(t *testing.T) {
	counts := map[string]int{"waiting": 2, "review": 1, "wip": 3}
	got := FormatAttentionLabel([]string{"waiting", "review"}, counts)
	want := "2 waiting, 1 review"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	got = FormatAttentionLabel([]string{"review"}, map[string]int{"review": 2})
	if got != "2 reviews" {
		t.Fatalf("plural review: got %q", got)
	}
	if FormatAttentionLabel([]string{"waiting"}, map[string]int{"waiting": 0}) != "" {
		t.Fatal("expected empty when all zero")
	}
}

func TestFormatTrayLabel(t *testing.T) {
	if got := FormatTrayLabel("2 waiting, 1 review", 9); got != "mhtodo · 2 waiting, 1 review" {
		t.Fatalf("attention: %q", got)
	}
	if got := FormatTrayLabel("", 7); got != "mhtodo (7)" {
		t.Fatalf("open: %q", got)
	}
	if got := FormatTrayLabel("  ", 0); got != "mhtodo" {
		t.Fatalf("idle: %q", got)
	}
}

func TestTruncateTitle(t *testing.T) {
	if got := TruncateTitle("short", 48); got != "short" {
		t.Fatalf("short: %q", got)
	}
	long := stringsRepeat("a", 50)
	got := TruncateTitle(long, 48)
	if len([]rune(got)) != 48 || !stringsHasSuffix(got, "…") {
		t.Fatalf("truncate: %q (runes=%d)", got, len([]rune(got)))
	}
}

func TestBuildSections(t *testing.T) {
	tasks := map[string][]TaskRef{
		"waiting": {{ID: "1", Title: "A"}, {ID: "2", Title: "B"}, {ID: "3", Title: "C"}},
		"review":  {{ID: "4", Title: "D"}},
	}
	secs := BuildSections(
		[]string{"waiting", "review", "waiting"},
		map[string]int{"waiting": 5, "review": 1},
		tasks,
		2,
	)
	if len(secs) != 2 {
		t.Fatalf("len=%d, want 2 (dedupe)", len(secs))
	}
	if secs[0].Status != "waiting" || secs[0].Count != 5 || len(secs[0].Items) != 2 {
		t.Fatalf("waiting section: %+v", secs[0])
	}
	if secs[0].Label != "Waiting (5)" {
		t.Fatalf("label=%q", secs[0].Label)
	}
	if secs[1].Status != "review" || secs[1].Count != 1 {
		t.Fatalf("review section: %+v", secs[1])
	}
}

func TestClampMaxItems(t *testing.T) {
	if ClampMaxItems(0) != DefaultMaxItemsPerStatus {
		t.Fatal("0 → default")
	}
	if ClampMaxItems(100) != MaxItemsHardCap {
		t.Fatal("over cap")
	}
	if ClampMaxItems(7) != 7 {
		t.Fatal("passthrough")
	}
}

func stringsRepeat(s string, n int) string {
	b := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}

func stringsHasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}
