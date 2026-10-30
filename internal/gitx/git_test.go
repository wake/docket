package gitx

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
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

func TestToplevelInsideGitDir(t *testing.T) {
	r := testutil.NewRepo(t)
	g := Git{Dir: filepath.Join(r.Dir, ".git"), Env: r.Env}
	if _, err := g.Toplevel(); !errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v, want ErrNotRepo", err)
	}
}

func TestToplevelBareRepo(t *testing.T) {
	g := Git{Dir: testutil.NewBare(t), Env: testutil.Env(t)}
	if _, err := g.Toplevel(); !errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v, want ErrNotRepo", err)
	}
}

func TestToplevelLocaleIndependent(t *testing.T) {
	env := append(testutil.Env(t), "LANG=de_DE.UTF-8", "LC_ALL=de_DE.UTF-8")
	g := Git{Dir: t.TempDir(), Env: env}
	if _, err := g.Toplevel(); !errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v, want ErrNotRepo", err)
	}
}

func TestWorktreesPathWithSpace(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a": "1"})
	wt := filepath.Join(t.TempDir(), "with space", "wt")
	if err := os.MkdirAll(filepath.Dir(wt), 0o755); err != nil {
		t.Fatal(err)
	}
	r.AddWorktree(wt, "feature")
	wts, err := Git{Dir: r.Dir, Env: r.Env}.Worktrees()
	if err != nil || len(wts) != 2 {
		t.Fatalf("Worktrees = %+v, %v", wts, err)
	}
	want, _ := filepath.EvalSymlinks(wt)
	got, _ := filepath.EvalSymlinks(wts[1].Path)
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
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

func TestRefsFiltered(t *testing.T) {
	src := testutil.NewRepo(t)
	src.Commit("one", map[string]string{"a": "1"})
	src.Git("branch", "merged")
	src.Git("checkout", "-q", "-b", "ahead")
	src.Commit("two", map[string]string{"a": "2"})
	src.Git("checkout", "-q", "main")
	src.Commit("three", map[string]string{"a": "3"})
	clone := src.CloneTo(t, filepath.Join(t.TempDir(), "c"))
	g := Git{Dir: clone.Dir, Env: clone.Env}
	tip := clone.Git("rev-parse", "main")
	names := func(refs []Ref) []string {
		var n []string
		for _, r := range refs {
			n = append(n, r.Name)
		}
		return n
	}
	refs, err := g.RefsFiltered("--merged="+tip, "refs/heads", "refs/remotes")
	if err != nil {
		t.Fatalf("merged: %v", err)
	}
	// origin/HEAD is symbolic and left out
	if want := []string{"refs/heads/main", "refs/remotes/origin/main", "refs/remotes/origin/merged"}; !slices.Equal(names(refs), want) {
		t.Fatalf("merged = %v, want %v", names(refs), want)
	}
	refs, err = g.RefsFiltered("--no-merged="+tip, "refs/heads", "refs/remotes")
	if err != nil {
		t.Fatalf("no-merged: %v", err)
	}
	if want := []string{"refs/remotes/origin/ahead"}; !slices.Equal(names(refs), want) {
		t.Fatalf("no-merged = %v, want %v", names(refs), want)
	}
	if _, err := g.RefsFiltered("--merged=0123456789012345678901234567890123456789", "refs/heads"); err == nil {
		t.Fatal("want an error for a commit that does not exist")
	}
}

func TestWalkUnmerged(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a": "1"})
	r.Git("checkout", "-q", "-b", "br")
	r.Commit("two", map[string]string{"a": "2"})
	r.Commit("three", map[string]string{"a": "3"})
	sha := r.Git("rev-parse", "br~1")
	r.Git("checkout", "-q", "main")
	tip := r.Git("rev-parse", "main")
	obj := filepath.Join(r.Dir, ".git", "objects", sha[:2], sha[2:])
	if err := os.Chmod(obj, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
	g := Git{Dir: r.Dir, Env: r.Env}
	if err := g.WalkUnmerged([]string{r.Git("rev-parse", "main")}, tip); err != nil {
		t.Fatalf("intact: %v", err)
	}
	brTip := r.Git("rev-parse", "br")
	if err := g.WalkUnmerged([]string{brTip}, tip); err == nil {
		t.Fatal("want an error for a tip whose parent object is missing")
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

func TestListFiles(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write("x/a.md", "1")
	r.Write("x/deep/er/b.md", "2")
	r.Write("x/z.md", "3")
	r.Write("xx/other.md", "4")
	if err := os.Symlink("a.md", filepath.Join(r.Dir, "x", "link.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	r.Commit("one", nil)
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.ListFiles("HEAD", "x")
	want := []string{"x/a.md", "x/deep/er/b.md", "x/z.md"}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("ListFiles = %v, %v, want %v", got, err, want)
	}
	got, err = g.ListFiles("HEAD", "nope")
	if err != nil || len(got) != 0 {
		t.Fatalf("ListFiles missing = %v, %v", got, err)
	}
	if _, err := g.ListFiles("no-such-rev", "x"); err == nil {
		t.Fatal("unknown revision: want an error")
	}
	if _, err := g.ListFiles("HEAD", "xx/other.md"); err == nil {
		t.Fatal("dir that is a file: want an error")
	}
	if _, err := g.ListFiles("HEAD", "x/link.md"); err == nil {
		t.Fatal("dir that is a link: want an error")
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
