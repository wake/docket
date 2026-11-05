package gitx

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestReadBlobs(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"a.txt": "hello\r\nno newline", "d/x": "x", "b.txt": "bee\n", "e.txt": ""})
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.ReadBlobs(c, []string{"missing", "a.txt", "d", "b.txt", "nope/x", "e.txt", "bad\nname"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]byte{"a.txt": []byte("hello\r\nno newline"), "b.txt": []byte("bee\n"), "e.txt": {}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestReadBlobsAfterNonBlobAndEmpty(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"d/x": "x", "b.txt": "bee\n"})
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.ReadBlobs(c, []string{"d", "b.txt"})
	if err != nil || string(got["b.txt"]) != "bee\n" || len(got) != 1 {
		t.Errorf("got %q %v", got, err)
	}
	got, err = (Git{Dir: "/nonexistent"}).ReadBlobs(c, nil)
	if err != nil || len(got) != 0 {
		t.Errorf("empty paths: %v %v", got, err)
	}
}

func TestReadBlobsBadCommit(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a": "1"})
	got, err := (Git{Dir: r.Dir, Env: r.Env}).ReadBlobs("0000000000000000000000000000000000000000", []string{"a"})
	if err != nil || len(got) != 0 {
		t.Errorf("got %v %v", got, err)
	}
}

func TestReadBlobsMissingPathWithSpace(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"b.txt": "bee\n"})
	got, err := (Git{Dir: r.Dir, Env: r.Env}).ReadBlobs(c, []string{"docs/my file.md", "b.txt"})
	if err != nil || !reflect.DeepEqual(got, map[string][]byte{"b.txt": []byte("bee\n")}) {
		t.Errorf("got %q %v", got, err)
	}
}

func TestReadBlobsNewlinePathNotAsked(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"b.txt": "bee\n"})
	got, err := (Git{Dir: r.Dir, Env: r.Env}).ReadBlobs(c, []string{"x\nb.txt", "b.txt"})
	if err != nil || !reflect.DeepEqual(got, map[string][]byte{"b.txt": []byte("bee\n")}) {
		t.Errorf("got %q %v", got, err)
	}
}

func remove(t *testing.T, r *testutil.Repo, rel string) {
	t.Helper()
	if err := os.Remove(filepath.Join(r.Dir, rel)); err != nil {
		t.Fatal(err)
	}
}

func TestAddedSince(t *testing.T) {
	r := testutil.NewRepo(t)
	base := r.Commit("base", map[string]string{
		"docs/old.md": "1", "docs/gone.md": "1", ".gitignore": "*.log\n", "other/o.md": "1",
	})
	r.Commit("branch", map[string]string{"docs/old.md": "2", "docs/committed.md": "1", "docs/del.md": "1", "docs/sub/b.md": "1"})
	r.Write("docs/staged.md", "1")
	r.Git("add", "docs/staged.md")
	r.Write("docs/staged-del.md", "1")
	r.Git("add", "docs/staged-del.md")
	r.Write("docs/untracked.md", "1")
	r.Write("docs/with space.md", "1")
	r.Write("docs/ignored.log", "1")
	r.Write("other/new.md", "1")
	remove(t, r, "docs/staged-del.md")
	remove(t, r, "docs/del.md")
	remove(t, r, "docs/gone.md")
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.AddedSince(base, "docs")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"docs/committed.md", "docs/staged.md", "docs/sub/b.md", "docs/untracked.md", "docs/with space.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
	got, err = g.AddedSince(base, "docs/sub", "other")
	if err != nil || !reflect.DeepEqual(got, []string{"docs/sub/b.md", "other/new.md"}) {
		t.Errorf("two dirs: %q %v", got, err)
	}
	got, err = (Git{Dir: "/nonexistent"}).AddedSince(base)
	if err != nil || len(got) != 0 {
		t.Errorf("no dirs: %q %v", got, err)
	}
}
