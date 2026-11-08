package projects

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func real(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func docketRepo(t *testing.T) *testutil.Repo {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"docket/workflow.md": "x", "internal/x.go": "package x"})
	return r
}

func TestLocateSubdirectory(t *testing.T) {
	r := docketRepo(t)
	p, err := Locate(filepath.Join(r.Dir, "internal"), r.Env)
	if err != nil {
		t.Fatal(err)
	}
	want := real(t, r.Dir)
	if real(t, p.Path) != want || real(t, p.Worktree) != want || !p.Docket || p.Origin != nil {
		t.Fatalf("got %+v, want %s", p, want)
	}
}

func TestLocateLinkedWorktree(t *testing.T) {
	r := docketRepo(t)
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", wt, "-b", "b")
	p, err := Locate(wt, r.Env)
	if err != nil {
		t.Fatal(err)
	}
	if real(t, p.Path) != real(t, r.Dir) || real(t, p.Worktree) != real(t, wt) || !p.Docket {
		t.Fatalf("got %+v", p)
	}
}

func TestLocateOrigin(t *testing.T) {
	r := docketRepo(t)
	r.Git("remote", "add", "origin", "https://example.invalid/x.git")
	p, err := Locate(r.Dir, r.Env)
	if err != nil {
		t.Fatal(err)
	}
	if p.Origin == nil || *p.Origin != "https://example.invalid/x.git" {
		t.Fatalf("origin %v", p.Origin)
	}
}

func TestLocateNotRepository(t *testing.T) {
	r := testutil.NewRepo(t)
	plain := t.TempDir()
	file := filepath.Join(plain, "f")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{plain, filepath.Join(plain, "none"), file} {
		if _, err := Locate(d, r.Env); !errors.Is(err, ErrNotRepository) {
			t.Errorf("%s: %v", d, err)
		}
	}
}

func TestIsDocket(t *testing.T) {
	d := t.TempDir()
	if IsDocket(d) {
		t.Fatal("no file")
	}
	wf := filepath.Join(d, "docket", "workflow.md")
	if err := os.MkdirAll(wf, 0o755); err != nil {
		t.Fatal(err)
	}
	if IsDocket(d) {
		t.Fatal("folder")
	}
	os.Remove(wf)
	target := filepath.Join(d, "target")
	os.WriteFile(target, nil, 0o644)
	if err := os.Symlink(target, wf); err != nil {
		t.Fatal(err)
	}
	if IsDocket(d) {
		t.Fatal("link")
	}
	os.Remove(wf)
	os.WriteFile(wf, nil, 0o644)
	if !IsDocket(d) {
		t.Fatal("regular file")
	}
}
