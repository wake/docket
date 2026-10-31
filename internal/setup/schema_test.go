package setup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/wake/docket/internal/safepath"
)

func TestSchemaDir(t *testing.T) {
	got, err := SchemaDir([]string{"XDG_DATA_HOME=/d"}, nil)
	if err != nil || got != "/d/openspec/schemas/docket" {
		t.Errorf("got %q, %v", got, err)
	}
	got, err = SchemaDir([]string{"HOME=/h"}, nil)
	if err != nil || got != "/h/.local/share/openspec/schemas/docket" {
		t.Errorf("got %q, %v", got, err)
	}
}

const twoFileHash = "4064fdd3e0cd59ca75b75ea1c50f23471e66cc72d1dfe5da6db79e5ddfb7160e"

func TestTreeHash(t *testing.T) {
	a := fstest.MapFS{"schema.yaml": {Data: []byte("name: x\n")}, "templates/a.md": {Data: []byte("# A\n")}}
	got, err := TreeHash(a)
	if err != nil || got != twoFileHash {
		t.Errorf("got %q, %v", got, err)
	}
	b := fstest.MapFS{"templates/a.md": {Data: []byte("# A\n")}, "schema.yaml": {Data: []byte("name: x\n")}}
	if got, _ := TreeHash(b); got != twoFileHash {
		t.Errorf("order changed the hash: %q", got)
	}
}

func writeTwoFiles(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	for p, c := range map[string]string{"schema.yaml": "name: x\n", "templates/a.md": "# A\n"} {
		if err := os.WriteFile(filepath.Join(dir, p), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHashDirMatchesTreeHash(t *testing.T) {
	dir := t.TempDir()
	writeTwoFiles(t, dir)
	got, err := HashDir(dir)
	if err != nil || got != twoFileHash {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestHashDirRefusesLinks(t *testing.T) {
	dir := t.TempDir()
	writeTwoFiles(t, dir)
	outside := filepath.Join(t.TempDir(), "x.md")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "templates", "a.md")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	_, err := HashDir(dir)
	var sl *safepath.SymlinkError
	if !errors.As(err, &sl) || sl.Path != "templates/a.md" {
		t.Errorf("error %v", err)
	}
}

func TestFindSchema(t *testing.T) {
	mk := func(t *testing.T, project, machine bool) (root, mdir string) {
		root, mdir = t.TempDir(), t.TempDir()
		put := func(dir string) {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "schema.yaml"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if project {
			put(filepath.Join(root, "openspec", "schemas", "docket"))
		}
		if machine {
			put(mdir)
		}
		return
	}
	t.Run("both", func(t *testing.T) {
		root, m := mk(t, true, true)
		dir, found, err := FindSchema(root, m)
		if err != nil || !found || dir != filepath.Join(root, "openspec", "schemas", "docket") {
			t.Errorf("%q %v %v", dir, found, err)
		}
	})
	t.Run("machine only", func(t *testing.T) {
		root, m := mk(t, false, true)
		dir, found, err := FindSchema(root, m)
		if err != nil || !found || dir != m {
			t.Errorf("%q %v %v", dir, found, err)
		}
	})
	t.Run("project folder without schema.yaml", func(t *testing.T) {
		root, m := mk(t, false, true)
		if err := os.MkdirAll(filepath.Join(root, "openspec", "schemas", "docket"), 0o755); err != nil {
			t.Fatal(err)
		}
		dir, found, err := FindSchema(root, m)
		if err != nil || !found || dir != m {
			t.Errorf("%q %v %v", dir, found, err)
		}
	})
	t.Run("neither", func(t *testing.T) {
		root, m := mk(t, false, false)
		if _, found, err := FindSchema(root, m); err != nil || found {
			t.Errorf("%v %v", found, err)
		}
	})
	t.Run("no machine dir", func(t *testing.T) {
		root, _ := mk(t, false, true)
		if _, found, err := FindSchema(root, ""); err != nil || found {
			t.Errorf("%v %v", found, err)
		}
	})
}
