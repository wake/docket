package gitx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestToplevelOutsideRepo(t *testing.T) {
	g := Git{Dir: t.TempDir(), Env: testutil.Env(t)}
	if _, err := g.Toplevel(); !errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v, want ErrNotRepo", err)
	}
}

func TestToplevelInRepo(t *testing.T) {
	r := testutil.NewRepo(t)
	g := Git{Dir: r.Dir, Env: r.Env}
	top, err := g.Toplevel()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(r.Dir)
	got, _ := filepath.EvalSymlinks(top)
	if got != want {
		t.Fatalf("toplevel = %q, want %q", got, want)
	}
}

func TestGitMissing(t *testing.T) {
	g := Git{Dir: t.TempDir(), Env: []string{"PATH="}}
	if _, err := g.Toplevel(); !errors.Is(err, ErrGitMissing) {
		t.Fatalf("err = %v, want ErrGitMissing", err)
	}
}

func TestHeadUnborn(t *testing.T) {
	r := testutil.NewRepo(t)
	g := Git{Dir: r.Dir, Env: r.Env}
	sha, ok, err := g.Head()
	if err != nil || ok || sha != "" {
		t.Fatalf("Head = %q, %v, %v; want unborn", sha, ok, err)
	}
	want := r.Commit("one", map[string]string{"a": "1"})
	sha, ok, err = g.Head()
	if err != nil || !ok || sha != want {
		t.Fatalf("Head = %q, %v, %v; want %q", sha, ok, err, want)
	}
}

func TestBranchDetached(t *testing.T) {
	r := testutil.NewRepo(t)
	sha := r.Commit("one", map[string]string{"a": "1"})
	g := Git{Dir: r.Dir, Env: r.Env}
	name, ok, err := g.Branch()
	if err != nil || !ok || name != "main" {
		t.Fatalf("Branch = %q, %v, %v; want main", name, ok, err)
	}
	r.Git("checkout", "-q", "--detach", sha)
	name, ok, err = g.Branch()
	if err != nil || ok || name != "" {
		t.Fatalf("Branch = %q, %v, %v; want detached", name, ok, err)
	}
}

func TestRemoteURLMissing(t *testing.T) {
	r := testutil.NewRepo(t)
	g := Git{Dir: r.Dir, Env: r.Env}
	if u, ok, err := g.RemoteURL("origin"); err != nil || ok || u != "" {
		t.Fatalf("RemoteURL = %q, %v, %v; want missing", u, ok, err)
	}
	r.Git("remote", "add", "origin", "https://example.invalid/x.git")
	u, ok, err := g.RemoteURL("origin")
	if err != nil || !ok || u != "https://example.invalid/x.git" {
		t.Fatalf("RemoteURL = %q, %v, %v", u, ok, err)
	}
}

func TestRefsSkipsSymbolic(t *testing.T) {
	src := testutil.NewRepo(t)
	sha := src.Commit("one", map[string]string{"a": "1"})
	clone := src.CloneTo(t, filepath.Join(t.TempDir(), "c"))
	g := Git{Dir: clone.Dir, Env: clone.Env}
	if ok, err := g.RefExists("refs/remotes/origin/HEAD"); err != nil || !ok {
		t.Fatalf("origin/HEAD should exist in a clone: %v, %v", ok, err)
	}
	refs, err := g.Refs("refs/remotes")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Name != "refs/remotes/origin/main" || refs[0].SHA != sha {
		t.Fatalf("refs = %+v", refs)
	}
	target, ok, err := g.SymbolicRef("refs/remotes/origin/HEAD")
	if err != nil || !ok || target != "refs/remotes/origin/main" {
		t.Fatalf("SymbolicRef = %q, %v, %v", target, ok, err)
	}
}

func TestWorktreesPrunable(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a": "1"})
	wt := filepath.Join(t.TempDir(), "wt")
	r.AddWorktree(wt, "feature")
	g := Git{Dir: r.Dir, Env: r.Env}
	wts, err := g.Worktrees()
	if err != nil || len(wts) != 2 {
		t.Fatalf("Worktrees = %+v, %v", wts, err)
	}
	if wts[1].Branch != "refs/heads/feature" || wts[1].Prunable || wts[1].Head == "" {
		t.Fatalf("linked worktree = %+v", wts[1])
	}
	if err := os.RemoveAll(wt); err != nil {
		t.Fatal(err)
	}
	wts, err = g.Worktrees()
	if err != nil || len(wts) != 2 || !wts[1].Prunable {
		t.Fatalf("Worktrees after delete = %+v, %v", wts, err)
	}
}

func TestListDirsMissing(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"x/a/f": "1", "x/b/f": "2", "x/file": "3"})
	g := Git{Dir: r.Dir, Env: r.Env}
	dirs, err := g.ListDirs("HEAD", "nope")
	if err != nil || len(dirs) != 0 {
		t.Fatalf("ListDirs missing = %v, %v", dirs, err)
	}
	dirs, err = g.ListDirs("HEAD", "x")
	if err != nil || len(dirs) != 2 || dirs[0] != "a" || dirs[1] != "b" {
		t.Fatalf("ListDirs = %v, %v", dirs, err)
	}
}

func TestShow(t *testing.T) {
	r := testutil.NewRepo(t)
	first := r.Commit("one", map[string]string{"d/f.txt": "old\n\n"})
	r.Commit("two", map[string]string{"d/f.txt": "new"})
	g := Git{Dir: r.Dir, Env: r.Env}
	b, err := g.Show(first, "d/f.txt")
	if err != nil || string(b) != "old\n\n" {
		t.Fatalf("Show = %q, %v", b, err)
	}
	var xe *ExitError
	if _, err := g.Show(first, "missing"); !errors.As(err, &xe) {
		t.Fatalf("Show missing err = %v, want *ExitError", err)
	}
}

func TestMergeBaseUnrelated(t *testing.T) {
	r := testutil.NewRepo(t)
	base := r.Commit("root", map[string]string{"a": "1"})
	r.Git("checkout", "-q", "--orphan", "other")
	other := r.Commit("other root", map[string]string{"b": "2"})
	g := Git{Dir: r.Dir, Env: r.Env}
	if sha, ok, err := g.MergeBase(base, other); err != nil || ok || sha != "" {
		t.Fatalf("MergeBase = %q, %v, %v; want none", sha, ok, err)
	}
	r.Git("checkout", "-q", "main")
	tip := r.Commit("two", map[string]string{"a": "2"})
	sha, ok, err := g.MergeBase(base, tip)
	if err != nil || !ok || sha != base {
		t.Fatalf("MergeBase = %q, %v, %v; want %q", sha, ok, err, base)
	}
	if yes, err := g.IsAncestor(base, tip); err != nil || !yes {
		t.Fatalf("IsAncestor(base, tip) = %v, %v", yes, err)
	}
	if yes, err := g.IsAncestor(tip, base); err != nil || yes {
		t.Fatalf("IsAncestor(tip, base) = %v, %v", yes, err)
	}
}
