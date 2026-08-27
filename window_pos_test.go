package main

import "testing"

func TestParseWindowPos(t *testing.T) {
	x, y, err := parseWindowPos("120,-40")
	if err != nil || x != 120 || y != -40 {
		t.Fatalf("got %d,%d err=%v", x, y, err)
	}
	if formatWindowPos(120, -40) != "120,-40" {
		t.Fatalf("format round-trip")
	}
	if _, _, err := parseWindowPos("nope"); err == nil {
		t.Fatal("expected error")
	}
	if _, _, err := parseWindowPos("1,2,3"); err == nil {
		t.Fatal("expected error")
	}
}
