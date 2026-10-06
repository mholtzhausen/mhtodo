package core_test

import (
	"testing"

	"mhtodo/internal/core"
)

func TestParseJoinNormalizePRURLs(t *testing.T) {
	raw := "  https://a/1  \n\nhttps://b/2\nhttps://a/1\n  \nhttps://c/3\n"
	got := core.ParsePRURLs(raw)
	want := []string{"https://a/1", "https://b/2", "https://c/3"}
	if len(got) != len(want) {
		t.Fatalf("ParsePRURLs len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParsePRURLs[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	joined := core.NormalizePRURL(raw)
	if joined != "https://a/1\nhttps://b/2\nhttps://c/3" {
		t.Fatalf("NormalizePRURL = %q", joined)
	}
	if core.JoinPRURLs([]string{" https://x ", "", "https://x", "https://y"}) != "https://x\nhttps://y" {
		t.Fatalf("JoinPRURLs = %q", core.JoinPRURLs([]string{" https://x ", "", "https://x", "https://y"}))
	}
	if core.NormalizePRURL("") != "" || core.NormalizePRURL("  \n  ") != "" {
		t.Fatal("empty input should normalize to empty")
	}
	if core.ParsePRURLs("") != nil {
		t.Fatalf("ParsePRURLs empty = %#v, want nil", core.ParsePRURLs(""))
	}
}
