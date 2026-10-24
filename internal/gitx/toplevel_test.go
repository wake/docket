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
