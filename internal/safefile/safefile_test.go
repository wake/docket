package safefile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestReadNoFollow(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.WriteFile(real, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadNoFollow(real)
	if err != nil || string(got) != "hi" {
		t.Fatalf("regular file: got %q, %v", got, err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadNoFollow(link); err == nil {
		t.Fatal("read through a link succeeded")
	}
	if _, err := ReadNoFollow(filepath.Join(dir, "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: %v", err)
	}
}

// createThrough opens name in dir with CreateFlags and reports the error.
func createThrough(t *testing.T, dir, name string) error {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	f, err := root.OpenFile(name, CreateFlags, 0o644)
	if err == nil {
		f.Close()
	}
	return err
}

func TestCreateFlagsRefuseLink(t *testing.T) {
	dir, outside := t.TempDir(), t.TempDir()
	target := filepath.Join(outside, "target")
	if err := os.WriteFile(target, []byte("kept"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "new")); err != nil {
		t.Fatal(err)
	}
	if err := createThrough(t, dir, "new"); !errors.Is(err, fs.ErrExist) {
		t.Errorf("err = %v, want fs.ErrExist", err)
	}
	if b, _ := os.ReadFile(target); string(b) != "kept" {
		t.Errorf("target now %q", b)
	}
}
