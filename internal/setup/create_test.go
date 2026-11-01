package setup

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFailedCreateLeavesNothingAndRerunFinishes(t *testing.T) {
	old := rename
	rename = func(string, string) error { return errors.New("rename refused") }
	t.Cleanup(func() { rename = old })
	root := t.TempDir()
	if _, err := Init(root, dirs(root), assets, false, openspecDir(t, root)); err == nil {
		t.Fatal("Init succeeded although rename failed")
	}
	first := filepath.Join(mdir(root), "schema.yaml")
	if _, err := os.Lstat(first); err == nil {
		t.Errorf("%s exists after a failed create", first)
	}
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && strings.Contains(d.Name(), ".docket-tmp-") {
			t.Errorf("temporary file left: %s", p)
		}
		return nil
	})
	rename = old
	r, err := Init(root, dirs(root), assets, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := statuses(r)[filepath.Join(mdir(root), "schema.yaml")]; got != Written {
		t.Errorf("schema.yaml = %q", got)
	}
}
