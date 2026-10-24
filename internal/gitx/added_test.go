package gitx

import (
	"reflect"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestAddedFiles(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	r.Commit("init", map[string]string{
		"docs/superpowers/specs/old.md": "old\n",
		"elsewhere/moved.md":            "a long enough line of text\nsecond line of text\nthird line\n",
	})
	base := r.Git("rev-parse", "HEAD")
	r.Git("checkout", "-q", "-b", "topic")
	r.Commit("work", map[string]string{
		"docs/superpowers/specs/new.md":   "new\n",
		"docs/superpowers/plans/other.md": "x\n",
		"docs/superpowers/specs/old.md":   "changed\n",
		"unrelated/file.md":               "y\n",
		"docs/superpowers/specs/a b.md":   "space\n",
	})
	r.Git("mv", "elsewhere/moved.md", "docs/superpowers/specs/moved.md")
	r.Git("commit", "-q", "-m", "rename")

	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.AddedFiles(base, "docs/superpowers/specs", "docs/superpowers/plans")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"docs/superpowers/plans/other.md",
		"docs/superpowers/specs/a b.md",
		"docs/superpowers/specs/moved.md",
		"docs/superpowers/specs/new.md",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestUncommittedAdded(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	r.Commit("init", map[string]string{"docs/superpowers/specs/old.md": "old\n"})
	r.Write("docs/superpowers/specs/untracked.md", "u\n")
	r.Write("docs/superpowers/plans/with space.md", "s\n")
	r.Write("docs/superpowers/specs/staged.md", "s\n")
	r.Git("add", "docs/superpowers/specs/staged.md")
	r.Write("docs/superpowers/specs/old.md", "modified\n")
	r.Write("other/file.md", "o\n")

	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.UncommittedAdded("docs/superpowers/specs", "docs/superpowers/plans")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"docs/superpowers/plans/with space.md",
		"docs/superpowers/specs/staged.md",
		"docs/superpowers/specs/untracked.md",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
