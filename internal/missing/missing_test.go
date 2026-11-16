package missing

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"syscall"
	"testing"
)

// Windows reports a file in the way like a missing folder; the cases run on every system by
// passing windows and an error that matches fs.ErrNotExist, as Windows gives it.
func TestIsOnWindows(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	notExist := &fs.PathError{Op: "open", Path: "x", Err: fs.ErrNotExist}
	fsys := os.DirFS(dir)
	fsStat := func(q string) (fs.FileInfo, error) { return fs.Stat(fsys, q) }
	for p, want := range map[string]bool{
		"f":           false, // a file read as a folder
		"f/sub":       false,
		"f/sub/x":     false,
		"missing":     true,
		"missing/sub": true,
		"d/missing":   true,
		"d/missing/x": true,
	} {
		if got := is(true, notExist, filepath.Join(dir, filepath.FromSlash(p)), os.Stat, filepath.Dir); got != want {
			t.Errorf("path %s: %v, want %v", p, got, want)
		}
		if got := is(true, notExist, p, fsStat, path.Dir); got != want {
			t.Errorf("fs %s: %v, want %v", p, got, want)
		}
	}
}

func TestIsElsewhere(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	notExist := &fs.PathError{Op: "open", Path: "x", Err: fs.ErrNotExist}
	notDir := &fs.PathError{Op: "open", Path: "x", Err: syscall.ENOTDIR}
	if errors.Is(notDir, fs.ErrNotExist) { // on Windows syscall.ENOTDIR is ERROR_PATH_NOT_FOUND
		notDir = &fs.PathError{Op: "open", Path: "x", Err: errors.New("not a directory")}
	}
	sub := filepath.Join(dir, "f", "sub")
	for _, c := range []struct {
		windows bool
		err     error
		want    bool
	}{
		{false, notExist, true}, // macOS and Linux take the error as it is
		{false, notDir, false},
		{true, notDir, false},
		{true, fs.ErrPermission, false},
		{false, nil, false},
		{true, nil, false},
	} {
		if got := is(c.windows, c.err, sub, os.Stat, filepath.Dir); got != c.want {
			t.Errorf("windows=%v %v: %v, want %v", c.windows, c.err, got, c.want)
		}
	}
}
