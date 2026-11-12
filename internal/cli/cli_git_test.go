package cli_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"mhtodo/internal/core"
)

func gitRepoDir(t *testing.T, remote, branch string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	steps := [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "init"},
		{"remote", "add", "origin", remote},
		{"checkout", "-q", "-b", branch},
	}
	for _, a := range steps {
		cmd := exec.Command("git", append([]string{"-C", dir}, a...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", a, err, out)
		}
	}
	return dir
}

func TestGitCLI(t *testing.T) {
	out, errb, run := newCLI(t)
	dir := gitRepoDir(t, "https://github.com/Org/X.git", "feat/cli")

	// add normalizes the repo.
	if code := run("add", "Git ticket", "--git-repo", "git@github.com:Org/X.git", "--json"); code != 0 {
		t.Fatalf("add exit %d: %s", code, errb.String())
	}
	var task core.Task
	mustJSON(t, out.Bytes(), &task)
	if task.GitRepo != "github.com/Org/X" {
		t.Fatalf("add git_repo = %q", task.GitRepo)
	}

	// edit --git-detect DIR appends the branch.
	out.Reset()
	if code := run("edit", task.ID, "--git-detect="+dir, "--json"); code != 0 {
		t.Fatalf("detect exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &task)
	if task.GitBranches != "feat/cli" {
		t.Fatalf("detect branches = %q", task.GitBranches)
	}

	// Bare --git-detect uses the process cwd; combined with another flag both apply.
	t.Chdir(dir)
	out.Reset()
	if code := run("edit", task.ID, "--git-detect", "--progress", "40", "--json"); code != 0 {
		t.Fatalf("bare detect exit %d: %s", code, errb.String())
	}
	mustJSON(t, out.Bytes(), &task)
	if task.Progress != 40 || task.GitBranches != "feat/cli" {
		t.Fatalf("bare detect: progress=%d branches=%q", task.Progress, task.GitBranches)
	}

	// list --git-current finds it; --git-branch exact; --git-repo normalized.
	for _, args := range [][]string{
		{"list", "--git-current", "--json"},
		{"list", "--git-branch", "feat/cli", "--json"},
		{"list", "--git-repo", "org/x", "--json"},
	} {
		out.Reset()
		if code := run(args...); code != 0 {
			t.Fatalf("%v exit %d: %s", args, code, errb.String())
		}
		var got []core.Task
		mustJSON(t, out.Bytes(), &got)
		if len(got) != 1 || got[0].ID != task.ID {
			t.Fatalf("%v = %d tasks", args, len(got))
		}
	}
	out.Reset()
	errb.Reset()
	if code := run("list", "--git-current", "--git-branch", "x"); code != 1 {
		t.Fatalf("--git-current with --git-branch: exit %d, want 1", code)
	}

	// Mismatch → exit 1, git_repo_mismatch.
	if code := run("edit", task.ID, "--git-repo", "other/repo"); code != 0 {
		t.Fatal("set other repo")
	}
	errb.Reset()
	if code := run("edit", task.ID, "--git-detect", "--json"); code != 1 {
		t.Fatalf("mismatch exit %d, want 1", code)
	}
	if !strings.Contains(errb.String(), `"git_repo_mismatch"`) {
		t.Fatalf("mismatch stderr = %s", errb.String())
	}

	// Non-repo → exit 1, git_detect.
	errb.Reset()
	if code := run("edit", task.ID, "--git-detect="+t.TempDir(), "--json"); code != 1 {
		t.Fatalf("non-repo exit %d, want 1", code)
	}
	if !strings.Contains(errb.String(), `"git_detect"`) {
		t.Fatalf("non-repo stderr = %s", errb.String())
	}
}

func TestTemplateGitRepoCLI(t *testing.T) {
	out, errb, run := newCLI(t)
	if code := run("template", "create", "api", "--git-repo", "org/api", "--json"); code != 0 {
		t.Fatalf("template create exit %d: %s", code, errb.String())
	}
	var tpl core.Template
	mustJSON(t, out.Bytes(), &tpl)
	if tpl.GitRepo == nil || *tpl.GitRepo != "github.com/org/api" {
		t.Fatalf("template git_repo = %v", tpl.GitRepo)
	}

	// add --template applies it; explicit --git-repo wins.
	out.Reset()
	if code := run("add", "From tpl", "--template", "api", "--json"); code != 0 {
		t.Fatalf("add --template exit %d", code)
	}
	var task core.Task
	mustJSON(t, out.Bytes(), &task)
	if task.GitRepo != "github.com/org/api" {
		t.Fatalf("templated git_repo = %q", task.GitRepo)
	}
	out.Reset()
	if code := run("add", "Override", "--template", "api", "--git-repo", "org/other", "--json"); code != 0 {
		t.Fatalf("add override exit %d", code)
	}
	mustJSON(t, out.Bytes(), &task)
	if task.GitRepo != "github.com/org/other" {
		t.Fatalf("override git_repo = %q", task.GitRepo)
	}

	// --clear-git-repo unsets; combining with --git-repo is a usage error.
	if code := run("template", "update", "api", "--git-repo", "x/y", "--clear-git-repo"); code != 1 {
		t.Fatalf("set+clear exit %d, want 1", code)
	}
	out.Reset()
	if code := run("template", "update", "api", "--clear-git-repo", "--json"); code != 0 {
		t.Fatalf("clear exit %d", code)
	}
	mustJSON(t, out.Bytes(), &tpl)
	if tpl.GitRepo != nil {
		t.Fatalf("cleared git_repo = %v", *tpl.GitRepo)
	}
}
