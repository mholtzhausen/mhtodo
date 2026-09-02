package core

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestShortID(t *testing.T) {
	t.Parallel()
	if got := ShortID(""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
	if got := ShortID("abcd"); got != "abcd" {
		t.Fatalf("short id: got %q", got)
	}

	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	full := id.String()
	head := full[:8]
	got := ShortID(full)
	if got == head {
		t.Fatalf("ShortID should not use timestamp prefix: head=%q got=%q", head, got)
	}
	compact := strings.ReplaceAll(full, "-", "")
	want := compact[len(compact)-8:]
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
