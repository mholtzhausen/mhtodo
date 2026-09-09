package core

import "testing"

func TestFuzzyScoreBasics(t *testing.T) {
	if fuzzyScore("elog", "Event Logger") < 0 {
		t.Fatal("expected subsequence match")
	}
	if fuzzyScore("xyz", "Event Logger") >= 0 {
		t.Fatal("expected no match")
	}
	if fuzzyScore("Event", "Event Logger") <= fuzzyScore("Evt", "Event Logger") {
		t.Fatal("contiguous / denser match should score higher")
	}
}
