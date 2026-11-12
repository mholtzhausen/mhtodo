package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"
)

// ParseGitBranches splits a git_branches blob into ordered unique branch
// names, one per line. Lines are trimmed; blanks dropped; first wins on dupes.
func ParseGitBranches(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		b := strings.TrimSpace(line)
		if b == "" {
			continue
		}
		if _, ok := seen[b]; ok {
			continue
		}
		seen[b] = struct{}{}
		out = append(out, b)
	}
	return out
}

// JoinGitBranches joins branch names with newlines after normalization.
func JoinGitBranches(branches []string) string {
	return NormalizeGitBranches(strings.Join(branches, "\n"))
}

// NormalizeGitBranches canonicalizes a git_branches blob.
func NormalizeGitBranches(s string) string {
	return strings.Join(ParseGitBranches(s), "\n")
}

// NormalizeGitRepo canonicalizes a repository reference to "host/path" so the
// same repo is stored one way however it was typed or detected:
//
//	https://github.com/org/x.git    → github.com/org/x
//	git@github.com:org/x.git        → github.com/org/x
//	ssh://git@gitlab.com:22/g/sub/x → gitlab.com/g/sub/x
//	org/x                           → github.com/org/x   (owner/name shorthand)
//
// The host is lower-cased (paths keep their case). Local paths (/…, ~…, ./…,
// file://…) are kept as paths with trailing slashes removed.
func NormalizeGitRepo(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if isLocalRepoPath(s) {
		return trimPath(s)
	}
	if i := strings.Index(s, "://"); i >= 0 {
		scheme := strings.ToLower(s[:i])
		rest := s[i+3:]
		if scheme == "file" {
			return trimPath(rest)
		}
		hostEnd := strings.Index(rest, "/")
		if hostEnd < 0 {
			hostEnd = len(rest)
		}
		host, path := rest[:hostEnd], rest[hostEnd:]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		if c := strings.LastIndex(host, ":"); c >= 0 && isDigits(host[c+1:]) {
			host = host[:c] // drop :port
		}
		s = host + path
	} else if colon := strings.Index(s, ":"); colon > 0 && !strings.Contains(s[:colon], "/") {
		// scp-like: [user@]host:path
		host := s[:colon]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		s = host + "/" + strings.TrimLeft(s[colon+1:], "/")
	}
	s = strings.Trim(s, "/")
	s = strings.TrimSuffix(s, ".git")
	s = strings.Trim(s, "/")
	parts := strings.Split(s, "/")
	if len(parts) == 2 && !strings.ContainsAny(parts[0], ".:") && parts[0] != "localhost" && parts[0] != "" && parts[1] != "" {
		return "github.com/" + s
	}
	parts[0] = strings.ToLower(parts[0])
	return strings.Join(parts, "/")
}

func isLocalRepoPath(s string) bool {
	return strings.HasPrefix(s, "/") || strings.HasPrefix(s, "~") || strings.HasPrefix(s, ".")
}

func trimPath(s string) string {
	if t := strings.TrimRight(s, "/"); t != "" {
		return t
	}
	return "/"
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// --- detection ---------------------------------------------------------------

// GitDetectError is returned when the repo or branch cannot be read from a
// working tree (not a git repo, detached HEAD, git missing, no directory).
type GitDetectError struct {
	Dir    string
	Reason string
}

func (e *GitDetectError) Error() string {
	if e.Dir == "" {
		return "git detect: " + e.Reason
	}
	return fmt.Sprintf("git detect in %s: %s", e.Dir, e.Reason)
}

// GitRepoMismatchError is returned when detection finds a different repo from
// the one already recorded on the ticket (a ticket tracks a single repo).
type GitRepoMismatchError struct {
	Have  string
	Found string
}

func (e *GitRepoMismatchError) Error() string {
	return fmt.Sprintf("ticket already records git repo %q but %q was detected; change it with --git-repo first", e.Have, e.Found)
}

// GitInfo is the repository and current branch found in a working tree.
type GitInfo struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

// DetectGitInfo reads the normalized remote (origin, else the first remote,
// else the work-tree path for local-only repos) and the current branch of dir.
func DetectGitInfo(ctx context.Context, dir string) (GitInfo, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return GitInfo{}, &GitDetectError{Reason: "no directory (set the ticket's cwd or pass a directory)"}
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := runGit(ctx, dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return GitInfo{}, &GitDetectError{Dir: dir, Reason: "not a git working tree: " + err.Error()}
	}
	branch, err := runGit(ctx, dir, "branch", "--show-current")
	if err != nil {
		return GitInfo{}, &GitDetectError{Dir: dir, Reason: err.Error()}
	}
	if branch == "" {
		return GitInfo{}, &GitDetectError{Dir: dir, Reason: "detached HEAD (check out a branch first)"}
	}
	repo, err := runGit(ctx, dir, "remote", "get-url", "origin")
	if err != nil || repo == "" {
		repo = ""
		if remotes, rerr := runGit(ctx, dir, "remote"); rerr == nil && remotes != "" {
			first := strings.Fields(remotes)[0]
			repo, _ = runGit(ctx, dir, "remote", "get-url", first)
		}
	}
	if repo == "" {
		top, terr := runGit(ctx, dir, "rev-parse", "--show-toplevel")
		if terr != nil {
			return GitInfo{}, &GitDetectError{Dir: dir, Reason: terr.Error()}
		}
		repo = top
	}
	return GitInfo{Repo: NormalizeGitRepo(repo), Branch: branch}, nil
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if msg := strings.TrimSpace(stderr.String()); msg != "" {
				return "", errors.New(msg)
			}
		}
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

// DetectGit records the repo and current branch of dir on the task: the repo
// is set when empty (a different recorded repo is a GitRepoMismatchError) and
// the branch is appended when missing. An empty dir means the task's cwd.
// updated_at only bumps when something changed.
func (s *Service) DetectGit(ctx context.Context, ref, dir string) (Task, error) {
	t, err := s.Get(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	if strings.TrimSpace(dir) == "" {
		dir = t.Cwd
	}
	info, err := DetectGitInfo(ctx, dir)
	if err != nil {
		return Task{}, err
	}
	have := NormalizeGitRepo(t.GitRepo)
	if have != "" && !strings.EqualFold(have, info.Repo) {
		return Task{}, &GitRepoMismatchError{Have: have, Found: info.Repo}
	}
	branches := ParseGitBranches(t.GitBranches)
	hasBranch := slices.Contains(branches, info.Branch)
	if have != "" && hasBranch {
		return t, nil
	}
	if have == "" {
		t.GitRepo = info.Repo
	}
	if !hasBranch {
		t.GitBranches = JoinGitBranches(append(branches, info.Branch))
	}
	t.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, t); err != nil {
		return Task{}, err
	}
	return t, nil
}
