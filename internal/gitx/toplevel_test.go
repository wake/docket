package gitx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestToplevelCorruptRepo(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a": "1"})
	if err := os.WriteFile(filepath.Join(r.Dir, ".git", "HEAD"), []byte("garbage\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Git{Dir: r.Dir, Env: r.Env}.Toplevel()
	if err == nil || errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v, want a non-ErrNotRepo error", err)
	}
	var ee *ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("err = %T %v, want *ExitError", err, err)
	}
}

func TestToplevelUnreadableGitDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	r := testutil.NewRepo(t)
	gitDir := filepath.Join(r.Dir, ".git")
	if err := os.Chmod(gitDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(gitDir, 0o755) })
	_, err := Git{Dir: r.Dir, Env: r.Env}.Toplevel()
	if err == nil || errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v, want a non-ErrNotRepo error", err)
	}
}

func TestToplevelGitDirSetKeepsExitError(t *testing.T) {
	env := append(testutil.Env(t), "GIT_DIR="+filepath.Join(t.TempDir(), "missing"))
	_, err := Git{Dir: t.TempDir(), Env: env}.Toplevel()
	if err == nil || errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v, want a non-ErrNotRepo error", err)
	}
}

// ceilingRepo returns a repository with an empty subdirectory two levels down.
func ceilingRepo(t *testing.T) (*testutil.Repo, string) {
	t.Helper()
	r := testutil.NewRepo(t)
	sub := filepath.Join(r.Dir, "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	return r, sub
}

func symlinkTo(t *testing.T, target string) string {
	t.Helper()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	return link
}

func TestToplevelCeilingAtRepoRoot(t *testing.T) {
	r, sub := ceilingRepo(t)
	env := append(r.Env, "GIT_CEILING_DIRECTORIES="+r.Dir)
	if _, err := (Git{Dir: sub, Env: env}).Toplevel(); !errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v, want ErrNotRepo", err)
	}
}

func TestToplevelCeilingThroughSymlink(t *testing.T) {
	r, sub := ceilingRepo(t)
	env := append(r.Env, "GIT_CEILING_DIRECTORIES="+symlinkTo(t, r.Dir))
	if _, err := (Git{Dir: sub, Env: env}).Toplevel(); !errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v, want ErrNotRepo", err)
	}
}

func TestToplevelCeilingAfterEmptyEntryIsNotResolved(t *testing.T) {
	r, sub := ceilingRepo(t)
	env := append(r.Env, "GIT_CEILING_DIRECTORIES=:"+symlinkTo(t, r.Dir))
	if _, err := (Git{Dir: sub, Env: env}).Toplevel(); err != nil {
		t.Fatalf("err = %v, want success", err)
	}
}

func TestToplevelDirEqualsCeiling(t *testing.T) {
	r, _ := ceilingRepo(t)
	env := append(r.Env, "GIT_CEILING_DIRECTORIES="+r.Dir)
	if _, err := (Git{Dir: r.Dir, Env: env}).Toplevel(); err != nil {
		t.Fatalf("err = %v, want success", err)
	}
}

func TestToplevelRelativeCeilingIgnored(t *testing.T) {
	r, sub := ceilingRepo(t)
	env := append(r.Env, "GIT_CEILING_DIRECTORIES=../..")
	if _, err := (Git{Dir: sub, Env: env}).Toplevel(); err != nil {
		t.Fatalf("err = %v, want success", err)
	}
}

func TestToplevelPhysicalParentChain(t *testing.T) {
	outer := testutil.NewRepo(t)
	elsewhere := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(elsewhere, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(outer.Dir, "link")
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Fatal(err)
	}
	if _, err := (Git{Dir: link, Env: outer.Env}).Toplevel(); !errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v, want ErrNotRepo", err)
	}
}
