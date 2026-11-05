package gitx

import (
	"reflect"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestUncommittedUnder(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{".gitignore": "*.log\n", "x/t.md": "1\n", "y/k.md": "1\n"})
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.UncommittedUnder("x")
	if err != nil || len(got) != 0 {
		t.Fatalf("clean: got %v, %v", got, err)
	}
	r.Write("x/t.md", "2\n")
	r.Write("x/s.md", "s\n")
	r.Git("add", "x/s.md")
	r.Write("x/new/a.md", "a\n")
	r.Write("x/new/b.md", "b\n")
	r.Write("x/i.log", "ignored\n")
	r.Write("y/u.md", "u\n")
	got, err = g.UncommittedUnder("x")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"x/new/a.md", "x/new/b.md", "x/s.md", "x/t.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

const oddName = "a b\"c\nd.md"

func TestUncommittedUnderOddName(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"x/t.md": "1\n"})
	r.Write("x/"+oddName, "o\n")
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.UncommittedUnder()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"x/" + oddName}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStatusRecords(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"t.md": "1\n"})
	r.Write("t.md", "2\n")
	r.Write("staged.md", "s\n")
	r.Git("add", "staged.md")
	r.Write("u/new.md", "n\n")
	r.Write(oddName, "o\n")
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.StatusRecords()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"t.md": " M", "staged.md": "A ", "u/new.md": "??", oddName: "??"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
