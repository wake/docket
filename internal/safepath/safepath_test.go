package safepath

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func mk(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a/b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a/b/f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func wantLink(t *testing.T, err error, path string) {
	t.Helper()
	var se *SymlinkError
	if !errors.As(err, &se) || se.Path != path {
		t.Fatalf("err = %v, want SymlinkError at %q", err, path)
	}
}

func TestPlainPathPasses(t *testing.T) {
	root := mk(t)
	if err := Check(root, "a/b/f"); err != nil {
		t.Fatal(err)
	}
}

func TestLinkAtFirstMiddleAndLast(t *testing.T) {
	outside := t.TempDir()
	root := mk(t)
	if err := os.Symlink(outside, filepath.Join(root, "l1")); err != nil {
		t.Fatal(err)
	}
	wantLink(t, Check(root, "l1/x/y"), "l1")
	if err := os.Symlink(outside, filepath.Join(root, "a/l2")); err != nil {
		t.Fatal(err)
	}
	wantLink(t, Check(root, "a/l2/y"), "a/l2")
	if err := os.Symlink(filepath.Join(root, "a/b/f"), filepath.Join(root, "a/b/l3")); err != nil {
		t.Fatal(err)
	}
	wantLink(t, Check(root, "a/b/l3"), "a/b/l3")
}

func TestNonDirectoryMiddle(t *testing.T) {
	root := mk(t)
	err := Check(root, "a/b/f/g")
	var se *SymlinkError
	if err == nil || errors.As(err, &se) {
		t.Fatalf("err = %v", err)
	}
}

func TestMissingComponentEndsWalk(t *testing.T) {
	root := mk(t)
	if err := Check(root, "a/zzz/q/r"); err != nil {
		t.Fatal(err)
	}
}

func TestNonLocalRejected(t *testing.T) {
	root := mk(t)
	for _, rel := range []string{"../x", "/abs", "a/../../x", ""} {
		if err := Check(root, rel); err == nil {
			t.Errorf("%q accepted", rel)
		}
	}
}
