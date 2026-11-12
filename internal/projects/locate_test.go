package projects

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/testutil"
)

func TestLocateBareWorktree(t *testing.T) {
	r := docketRepo(t)
	bare := testutil.NewBare(t)
	r.Git("push", bare, "HEAD:refs/heads/trunk")
	wt := filepath.Join(t.TempDir(), "app-main")
	cmd := exec.Command("git", "-C", bare, "worktree", "add", wt, "trunk")
	cmd.Env = r.Env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	p, err := Locate(wt, r.Env)
	if err != nil {
		t.Fatal(err)
	}
	if real(t, p.Path) != real(t, wt) || !p.Docket {
		t.Fatalf("got %+v, want %s", p, wt)
	}
}

func TestOrigin(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"https://user:secret@example.com/org/app.git", "https://example.com/org/app.git"},
		{"https://ghp_token@github.com/org/app.git", "https://github.com/org/app.git"},
		{"http://u:p@example.com/app", "http://example.com/app"},
		{"git@example.com:org/app.git", "git@example.com:org/app.git"},
		{"ssh://git@example.com/app.git", "ssh://git@example.com/app.git"},
		{"https://user:p%zz@example.com/org/app.git", "https://example.com/org/app.git"},
		{"https://tok@example.com/a%zz.git", "https://example.com/a%zz.git"},
		{"https://user:p ss@example.com/app.git", "https://example.com/app.git"},
		{"https://tok@example.com/a b.git", "https://example.com/a b.git"},
		{"https://user:p@ss@example.com/app.git", "https://example.com/app.git"},
		{"HTTPS://tok@example.com/app.git", "HTTPS://example.com/app.git"},
		{"https://tok@example.com", "https://example.com"},
		{"https://example.com/path@x", "https://example.com/path@x"},
		{"https://example.com?q=a@b", "https://example.com?q=a@b"},
		{"ftp://u:p@example.com/app.git", "ftp://u:p@example.com/app.git"},
	} {
		r := docketRepo(t)
		r.Git("remote", "add", "origin", c.in)
		got, err := Origin(gitx.Git{Dir: r.Dir, Env: r.Env})
		if err != nil || got == nil || *got != c.want {
			t.Errorf("%s: got %v, %v; want %s", c.in, got, err, c.want)
		}
	}
	r := docketRepo(t)
	if got, err := Origin(gitx.Git{Dir: r.Dir, Env: r.Env}); err != nil || got != nil {
		t.Errorf("no remote: got %v, %v", got, err)
	}
}

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

func TestLocateDocketFollowsTheWorktree(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"a.txt": "x"})
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", wt, "-b", "b")
	wf := filepath.Join(wt, "docket", "workflow.md")
	if err := os.MkdirAll(filepath.Dir(wf), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wf, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Locate(wt, r.Env)
	if err != nil || !p.Docket || real(t, p.Path) != real(t, r.Dir) {
		t.Fatalf("linked: %+v, %v", p, err)
	}
	if p, err = Locate(r.Dir, r.Env); err != nil || p.Docket {
		t.Fatalf("main: %+v, %v", p, err)
	}
}

func TestLocateBrokenRepository(t *testing.T) {
	r := testutil.NewRepo(t)
	d := t.TempDir()
	gitdir := "gitdir: " + filepath.Join(d, "missing") + "\n"
	if err := os.WriteFile(filepath.Join(d, ".git"), []byte(gitdir), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Locate(d, r.Env)
	// Toplevel keeps the git failure here (a .git marker exists), so it is not ErrNotRepository.
	if err == nil || errors.Is(err, ErrNotRepository) || p != (Project{}) {
		t.Fatalf("got %+v, %v", p, err)
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
