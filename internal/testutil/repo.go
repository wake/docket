// Package testutil builds real git repositories in temporary directories for tests.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Env returns an allowlisted environment (PATH, TMPDIR and SYSTEMROOT from the
// process, no inherited GIT_* variable) plus a temporary HOME and a git
// configuration that is independent of the machine running the tests.
func Env(t testing.TB) []string {
	t.Helper()
	home := t.TempDir()
	cfg := filepath.Join(home, "gitconfig")
	content := "[user]\n\tname = Docket Test\n\temail = docket@example.invalid\n" +
		"[init]\n\tdefaultBranch = main\n[commit]\n\tgpgsign = false\n"
	if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
		t.Fatalf("write git config: %v", err)
	}
	var env []string
	for _, k := range []string{"PATH", "TMPDIR", "SYSTEMROOT"} {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return append(env, "HOME="+home, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+cfg)
}

// Repo is a git repository in a temporary directory.
type Repo struct {
	T   testing.TB
	Dir string
	Env []string
}

// NewRepo initialises a repository in a new temporary directory.
func NewRepo(t testing.TB) *Repo {
	t.Helper()
	return NewRepoAt(t, t.TempDir())
}

// NewRepoAt initialises a repository in dir.
func NewRepoAt(t testing.TB, dir string) *Repo {
	t.Helper()
	r := &Repo{T: t, Dir: dir, Env: Env(t)}
	r.Git("init", "-q")
	return r
}

// Git runs git in the repository and fails the test on error.
func (r *Repo) Git(args ...string) string {
	r.T.Helper()
	return runGit(r.T, r.Dir, r.Env, args...)
}

func runGit(t testing.TB, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimRight(string(out), "\n")
}

// Write writes a file relative to the repository root, creating directories.
func (r *Repo) Write(rel, content string) {
	r.T.Helper()
	p := filepath.Join(r.Dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.T.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.T.Fatalf("write %s: %v", rel, err)
	}
}

// Commit writes files, commits everything and returns the new commit SHA.
func (r *Repo) Commit(msg string, files map[string]string) string {
	r.T.Helper()
	for rel, content := range files {
		r.Write(rel, content)
	}
	r.Git("add", "-A")
	r.Git("commit", "-q", "--allow-empty", "-m", msg)
	return r.Git("rev-parse", "HEAD")
}

// AddWorktree adds a linked worktree at path on a new branch.
func (r *Repo) AddWorktree(path, newBranch string) *Repo {
	r.T.Helper()
	r.Git("worktree", "add", "-q", "-b", newBranch, path)
	return &Repo{T: r.T, Dir: path, Env: r.Env}
}

// NewBare creates an empty bare repository and returns its path.
func NewBare(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, Env(t), "init", "-q", "--bare")
	return dir
}

// CloneTo clones the repository into dir and returns the clone.
func (r *Repo) CloneTo(t testing.TB, dir string) *Repo {
	t.Helper()
	runGit(t, r.Dir, r.Env, "clone", "-q", r.Dir, dir)
	return &Repo{T: t, Dir: dir, Env: r.Env}
}
