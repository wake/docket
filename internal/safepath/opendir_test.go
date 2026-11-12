package safepath

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func openRoot(t *testing.T, dir string) *os.Root {
	t.Helper()
	r, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func TestOpenDirPlainAndRoot(t *testing.T) {
	r := openRoot(t, mk(t))
	for _, rel := range []string{".", "a", "a/b"} {
		d, err := OpenDir(r, rel, false)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		d.Close()
	}
	d, _ := OpenDir(r, "a/b", false)
	defer d.Close()
	if _, err := d.Lstat("f"); err != nil {
		t.Errorf("not the folder asked for: %v", err)
	}
}

func TestOpenDirLinkInsideRootRefused(t *testing.T) {
	root := mk(t)
	r := openRoot(t, root)
	if err := os.Symlink("b", filepath.Join(root, "a/l")); err != nil { // stays inside the root
		t.Fatal(err)
	}
	for _, mkdir := range []bool{false, true} {
		_, err := OpenDir(r, "a/l/c", mkdir)
		wantLink(t, err, "a/l")
	}
	if _, err := os.Lstat(filepath.Join(root, "a/b/c")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("made a folder through the link: %v", err)
	}
}

func TestOpenDirMissing(t *testing.T) {
	root := mk(t)
	r := openRoot(t, root)
	if _, err := OpenDir(r, "a/x/y", false); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("without mkdir: %v", err)
	}
	d, err := OpenDir(r, "a/x/y", true)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()
	if fi, err := os.Lstat(filepath.Join(root, "a/x/y")); err != nil || !fi.IsDir() {
		t.Errorf("with mkdir: %v", err)
	}
}

func TestOpenDirFileInTheMiddle(t *testing.T) {
	r := openRoot(t, mk(t))
	if _, err := OpenDir(r, "a/b/f/g", true); err == nil {
		t.Error("want error")
	}
}
