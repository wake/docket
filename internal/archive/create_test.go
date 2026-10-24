package archive

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// The race between the check and the link cannot be tested
// deterministically; this pins the contract the link gives.
func TestCreateFileRefusesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "evidence.md")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := createFile(dir, "evidence.md", []byte("new"))
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("err = %v", err)
	}
	if b, _ := os.ReadFile(path); string(b) != "old" {
		t.Errorf("content %q", b)
	}
	noTempFiles(t, dir)
}

func TestCreateFileMakesANewFile(t *testing.T) {
	dir := t.TempDir()
	if err := createFile(dir, "evidence.md", []byte("new")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "evidence.md"))
	if string(b) != "new" {
		t.Errorf("content %q", b)
	}
	if fi, _ := os.Stat(filepath.Join(dir, "evidence.md")); fi.Mode().Perm() != 0o644 {
		t.Errorf("mode %v", fi.Mode())
	}
	noTempFiles(t, dir)
}
