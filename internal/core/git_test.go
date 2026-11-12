package core_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"mhtodo/internal/core"
)

func TestNormalizeGitBranches(t *testing.T) {
	if got := core.NormalizeGitBranches("  feat/x \n\nmain\nfeat/x\n"); got != "feat/x\nmain" {
		t.Fatalf("NormalizeGitBranches = %q", got)
	}
	if got := core.JoinGitBranches([]string{"a", " ", " a "}); got != "a" {
		t.Fatalf("JoinGitBranches = %q", got)
	}
	if core.ParseGitBranches("") != nil {
		t.Fatal("ParseGitBranches empty should be nil")
	}
}

func TestGitRepoBranchesCreateEditRoundTrip(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	task, err := svc.Create(ctx, core.CreateInput{Title: "t", GitRepo: " org/api ", GitBranches: " feat/x \nfeat/x"})
	if err != nil {
		t.Fatal(err)
	}
	if task.GitRepo != "github.com/org/api" || task.GitBranches != "feat/x" {
		t.Fatalf("create repo=%q branches=%q", task.GitRepo, task.GitBranches)
	}
	branches := "feat/x\nfeat/y"
	got, err := svc.Edit(ctx, task.ID, core.UpdateInput{GitBranches: &branches})
	if err != nil {
		t.Fatal(err)
	}
	if got.GitRepo != "github.com/org/api" || got.GitBranches != branches || got.Status != task.Status {
		t.Fatalf("edit repo=%q branches=%q status=%s", got.GitRepo, got.GitBranches, got.Status)
	}
	empty := ""
	got, err = svc.Edit(ctx, task.ID, core.UpdateInput{GitRepo: &empty})
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := svc.Get(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.GitRepo != "" || reloaded.GitBranches != branches {
		t.Fatalf("reloaded repo=%q branches=%q", reloaded.GitRepo, reloaded.GitBranches)
	}
}

func TestNormalizeGitRepo(t *testing.T) {
	cases := map[string]string{
		"":                                      "",
		"  ":                                    "",
		"org/x":                                 "github.com/org/x",
		"https://github.com/Org/X.git":          "github.com/Org/X",
		"https://GitHub.com/org/x/":             "github.com/org/x",
		"git@github.com:org/x.git":              "github.com/org/x",
		"ssh://git@gitlab.com:22/g/sub/x.git":   "gitlab.com/g/sub/x",
		"https://user:tok@gitlab.example.com/a": "gitlab.example.com/a",
		"github.com/org/x":                      "github.com/org/x",
		"gitlab.com/g/sub/x":                    "gitlab.com/g/sub/x",
		"localhost/x":                           "localhost/x",
		"/home/me/code/repo/":                   "/home/me/code/repo",
		"~/code/repo":                           "~/code/repo",
		"./repo":                                "./repo",
		"file:///srv/git/repo.git/":             "/srv/git/repo.git",
		"single":                                "single",
	}
	for in, want := range cases {
		if got := core.NormalizeGitRepo(in); got != want {
			t.Errorf("NormalizeGitRepo(%q) = %q, want %q", in, got, want)
		}
	}
	// Idempotent.
	for _, want := range cases {
		if got := core.NormalizeGitRepo(want); got != want {
			t.Errorf("NormalizeGitRepo not idempotent on %q: %q", want, got)
		}
	}
}

func gitInit(t *testing.T, dir string, args ...[]string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	base := [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "init"},
	}
	for _, a := range append(base, args...) {
		cmd := exec.Command("git", append([]string{"-C", dir}, a...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", a, err, out)
		}
	}
}

func TestDetectGit(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	dir := t.TempDir()
	gitInit(t, dir,
		[]string{"remote", "add", "origin", "git@github.com:org/x.git"},
		[]string{"checkout", "-q", "-b", "feat/x"})

	info, err := core.DetectGitInfo(ctx, dir)
	if err != nil || info.Repo != "github.com/org/x" || info.Branch != "feat/x" {
		t.Fatalf("DetectGitInfo = %+v, %v", info, err)
	}

	task, err := svc.Create(ctx, core.CreateInput{Title: "t", Cwd: dir, GitBranches: "main"})
	if err != nil {
		t.Fatal(err)
	}
	// Empty dir → task cwd; repo set, branch appended.
	got, err := svc.DetectGit(ctx, task.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.GitRepo != "github.com/org/x" || got.GitBranches != "main\nfeat/x" {
		t.Fatalf("detect repo=%q branches=%q", got.GitRepo, got.GitBranches)
	}
	// Idempotent: no change, updated_at untouched.
	again, err := svc.DetectGit(ctx, task.ID, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !again.UpdatedAt.Equal(got.UpdatedAt) || again.GitBranches != got.GitBranches {
		t.Fatalf("second detect changed task: %+v", again)
	}

	// Different recorded repo → mismatch.
	other := "github.com/org/other"
	if _, err := svc.Edit(ctx, task.ID, core.UpdateInput{GitRepo: &other}); err != nil {
		t.Fatal(err)
	}
	var mis *core.GitRepoMismatchError
	if _, err := svc.DetectGit(ctx, task.ID, dir); !errors.As(err, &mis) {
		t.Fatalf("want GitRepoMismatchError, got %v", err)
	}
	// Case-insensitive repo match is not a mismatch.
	upper := "github.com/ORG/X"
	if _, err := svc.Edit(ctx, task.ID, core.UpdateInput{GitRepo: &upper}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DetectGit(ctx, task.ID, dir); err != nil {
		t.Fatalf("case-only difference should match: %v", err)
	}

	// Not a repo / no cwd → GitDetectError.
	var det *core.GitDetectError
	if _, err := svc.DetectGit(ctx, task.ID, t.TempDir()); !errors.As(err, &det) {
		t.Fatalf("want GitDetectError for non-repo, got %v", err)
	}
	bare, _ := svc.Create(ctx, core.CreateInput{Title: "no cwd"})
	if _, err := svc.DetectGit(ctx, bare.ID, ""); !errors.As(err, &det) {
		t.Fatalf("want GitDetectError for empty cwd, got %v", err)
	}
}

func TestDetectGitDetachedAndLocal(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	gitInit(t, dir, []string{"checkout", "-q", "--detach"})
	var det *core.GitDetectError
	if _, err := core.DetectGitInfo(ctx, dir); !errors.As(err, &det) {
		t.Fatalf("detached HEAD: want GitDetectError, got %v", err)
	}

	local := t.TempDir()
	gitInit(t, local)
	info, err := core.DetectGitInfo(ctx, local)
	if err != nil {
		t.Fatal(err)
	}
	if info.Branch != "main" || !strings.HasPrefix(info.Repo, "/") {
		t.Fatalf("local repo info = %+v", info)
	}
}

func TestListGitFilters(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, core.CreateInput{Title: "a", GitRepo: "org/x", GitBranches: "feat/x\nmain"})
	b, _ := svc.Create(ctx, core.CreateInput{Title: "b", GitRepo: "gitlab.com/g/y", GitBranches: "feat/x-2"})
	_, _ = svc.Create(ctx, core.CreateInput{Title: "c"})

	ids := func(f core.ListFilter) []string {
		t.Helper()
		ts, err := svc.List(ctx, f)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, x := range ts {
			out = append(out, x.ID)
		}
		return out
	}
	// Repo filter is normalized + case-insensitive.
	if got := ids(core.ListFilter{GitRepo: "git@GitHub.com:org/X.git"}); len(got) != 1 || got[0] != a.ID {
		t.Fatalf("repo filter = %v", got)
	}
	// Branch filter is exact (feat/x must not match feat/x-2).
	if got := ids(core.ListFilter{GitBranch: "feat/x"}); len(got) != 1 || got[0] != a.ID {
		t.Fatalf("branch filter = %v", got)
	}
	if got := ids(core.ListFilter{GitBranch: "feat/x-2"}); len(got) != 1 || got[0] != b.ID {
		t.Fatalf("branch filter 2 = %v", got)
	}
	if got := ids(core.ListFilter{GitBranch: "feat"}); len(got) != 0 {
		t.Fatalf("partial branch must not match: %v", got)
	}
	// Last-line and middle-line branches match too.
	if got := ids(core.ListFilter{GitBranch: "main", GitRepo: "org/x"}); len(got) != 1 {
		t.Fatalf("combined filter = %v", got)
	}
	// Search covers git fields.
	if got := ids(core.ListFilter{Search: "gitlab.com/g"}); len(got) != 1 || got[0] != b.ID {
		t.Fatalf("search repo = %v", got)
	}
	if got := ids(core.ListFilter{Search: "feat/x"}); len(got) != 2 {
		t.Fatalf("search branch = %v", got)
	}
}

func TestTemplateGitRepoPreset(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	repo := " git@github.com:org/x.git "
	tpl, err := svc.CreateTemplate(ctx, core.TemplateInput{Name: "x", GitRepo: &repo})
	if err != nil {
		t.Fatal(err)
	}
	if tpl.GitRepo == nil || *tpl.GitRepo != "github.com/org/x" {
		t.Fatalf("template git_repo = %v", tpl.GitRepo)
	}
	reloaded, err := svc.GetTemplate(ctx, "x")
	if err != nil || reloaded.GitRepo == nil || *reloaded.GitRepo != "github.com/org/x" {
		t.Fatalf("reloaded = %+v, %v", reloaded, err)
	}
	task, err := svc.CreateFromTemplate(ctx, "x", core.CreateInput{Title: "t"})
	if err != nil || task.GitRepo != "github.com/org/x" {
		t.Fatalf("task from template repo=%q err=%v", task.GitRepo, err)
	}
	// Full-replace update with nil clears the preset.
	in := reloaded.AsInput()
	in.GitRepo = nil
	cleared, err := svc.UpdateTemplate(ctx, "x", in)
	if err != nil || cleared.GitRepo != nil {
		t.Fatalf("cleared = %+v, %v", cleared.GitRepo, err)
	}
	reloaded, _ = svc.GetTemplate(ctx, "x")
	if reloaded.GitRepo != nil {
		t.Fatalf("reloaded after clear = %v", *reloaded.GitRepo)
	}
}
