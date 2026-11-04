package gitx

import (
	"errors"
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
