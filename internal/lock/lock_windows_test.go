package lock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTempFolderUnusable(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-folder")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMP", file)
	t.Setenv("TEMP", file)
	unlock, err := Dir(t.TempDir(), ".")
	if err == nil || unlock != nil {
		t.Fatalf("err = %v, unlock = %v; want an error and no unlock", err, unlock != nil)
	}
	if want := filepath.Join(file, "docket"); !strings.Contains(err.Error(), want) {
		t.Errorf("%q does not name %s", err, want)
	}
}

func TestLockFileStays(t *testing.T) {
	t.Setenv("TMP", t.TempDir())
	unlock, err := Dir(t.TempDir(), ".")
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	entries, err := os.ReadDir(filepath.Join(os.TempDir(), "docket"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("%v %v", entries, err)
	}
	if fi, err := entries[0].Info(); err != nil || fi.Size() != 0 || !strings.HasSuffix(fi.Name(), ".lock") {
		t.Errorf("%v %v", fi, err)
	}
}
