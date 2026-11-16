package gitx

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestPathSpelledAsWindowsDoes(t *testing.T) {
	dir, empty := t.TempDir(), t.TempDir()
	writeStub(t, filepath.Join(dir, "git"))
	if out, err := (Git{Dir: t.TempDir(), Env: []string{"Path=" + dir}}).Run("anything"); err != nil || out != "stub-marker" {
		t.Errorf("Path: %q %v", out, err)
	}
	if out, err := (Git{Dir: t.TempDir(), Env: []string{"PATH=" + empty, "Path=" + dir}}).Run("anything"); err != nil || out != "stub-marker" {
		t.Errorf("last spelling holds git: %q %v", out, err)
	}
	if _, err := (Git{Dir: t.TempDir(), Env: []string{"Path=" + dir, "PATH=" + empty}}).Run("anything"); !errors.Is(err, ErrGitMissing) {
		t.Errorf("last spelling has no git: %v", err)
	}
}

func TestPathsFromGitWithBackslashes(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a": "1"})
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "-b", "other", wt)
	top, err := Git{Dir: wt, Env: r.Env}.Toplevel()
	if err != nil {
		t.Fatal(err)
	}
	wts, err := Git{Dir: r.Dir, Env: r.Env}.Worktrees()
	if err != nil || len(wts) != 2 {
		t.Fatalf("%v %v", wts, err)
	}
	for _, p := range []string{top, wts[0].Path, wts[1].Path} {
		if strings.Contains(p, "/") {
			t.Errorf("%q holds a forward slash", p)
		}
	}
	if top != wts[1].Path {
		t.Errorf("toplevel %q, worktree list %q", top, wts[1].Path)
	}
	if want, _ := filepath.EvalSymlinks(wt); top != want {
		t.Errorf("toplevel %q, want %q", top, want)
	}
}
