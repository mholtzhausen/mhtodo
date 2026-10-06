package instance

import "testing"

func TestTaskURI(t *testing.T) {
	got := TaskURI("81abc903")
	if got != "mhtodo://task/81abc903" {
		t.Fatalf("TaskURI: %q", got)
	}
}

func TestParseOpenTarget(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"81abc903", "81abc903"},
		{"  abcd1234  ", "abcd1234"},
		{"mhtodo://task/81abc903", "81abc903"},
		{"mhtodo://task/81abc903/", "81abc903"},
		{"MHTODO://Task/AbCd1234", "AbCd1234"},
		{"mhtodo:task/81abc903", "81abc903"},
		{"mhtodo://81abc903", "81abc903"},
	}
	for _, tc := range cases {
		got, err := ParseOpenTarget(tc.in)
		if err != nil {
			t.Errorf("ParseOpenTarget(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseOpenTarget(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseOpenTargetErrors(t *testing.T) {
	for _, in := range []string{"", "   ", "http://task/x", "mhtodo://task/", "mhtodo://"} {
		if _, err := ParseOpenTarget(in); err == nil {
			t.Errorf("ParseOpenTarget(%q): want error", in)
		}
	}
}
