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

func TestWindowPosLooksValid(t *testing.T) {
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("GDK_BACKEND", "wayland")
	t.Setenv("MHTODO_WAYLAND", "1")
	if windowPosLooksValid(0, 0) {
		t.Fatal("expected (0,0) invalid on native wayland")
	}
	if !windowPosLooksValid(10, 0) {
		t.Fatal("expected non-zero valid")
	}

	t.Setenv("MHTODO_WAYLAND", "")
	t.Setenv("GDK_BACKEND", "x11")
	if !windowPosLooksValid(0, 0) {
		t.Fatal("expected (0,0) valid on x11")
	}
}
