package update

import (
	"slices"
	"testing"
)

func TestServiceInCgroup(t *testing.T) {
	cases := map[string]bool{
		"0::/user.slice/user-1000.slice/user@1000.service/app.slice/mhtodo.service\n":                        true,
		"0::/user.slice/user-1000.slice/user@1000.service/app.slice/mhtodo-update-1.service\n":               false,
		"0::/user.slice/user-1000.slice/user@1000.service/app.slice/app-ghostty-surface-transient-1.scope\n": false,
		"12:pids:/user.slice\n1:name=systemd:/user.slice/user@1000.service/mhtodo.service\n":                 true,
		"": false,
	}
	for in, want := range cases {
		if got := serviceInCgroup(in); got != want {
			t.Errorf("serviceInCgroup(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestHandoffArgs(t *testing.T) {
	env := map[string]string{"PATH": "/bin", "GH_TOKEN": "tok"}
	got := HandoffArgs("mhtodo-update-1", "/x/mhtodo", []string{"update", "--force"},
		func(k string) string { return env[k] })
	want := []string{"--user", "--collect", "--quiet", "--unit=mhtodo-update-1",
		"--setenv=PATH=/bin", "--setenv=GH_TOKEN=tok", "/x/mhtodo", "update", "--force"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
