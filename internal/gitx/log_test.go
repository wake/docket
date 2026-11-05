package gitx

import (
	"reflect"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestChangeTrailers(t *testing.T) {
	r := testutil.NewRepo(t)
	a := r.Commit("a\n\nChange: add-x/1", map[string]string{"f": "1"})
	b := r.Commit("b", map[string]string{"f": "2"})
	c := r.Commit("c\n\nchange: add-x/2\nChange: add-y", map[string]string{"f": "3"})
	d := r.Commit("d\n\nChange: add-x/3\n\nbody only", map[string]string{"f": "4"})
	e := r.Commit("e\n\nChange: add-x/1,\n  2\nTask: 9", map[string]string{"f": "5"})
	got, err := Git{Dir: r.Dir, Env: r.Env}.ChangeTrailers("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	want := []Trailers{
		{a, []string{"add-x/1"}},
		{b, nil},
		{c, []string{"add-x/2", "add-y"}},
		{d, nil},
		{e, []string{"add-x/1, 2"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestChangeTrailersMerge(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"f": "0"})
	r.Git("checkout", "-q", "-b", "side")
	side := r.Commit("side\n\nChange: add-x/1", map[string]string{"s": "1"})
	r.Git("checkout", "-q", "main")
	main := r.Commit("main\n\nChange: add-x/2", map[string]string{"m": "1"})
	r.Git("merge", "-q", "--no-ff", "-m", "merge", "side")
	got, err := Git{Dir: r.Dir, Env: r.Env}.ChangeTrailers("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string][]string{}
	for _, tr := range got {
		seen[tr.SHA] = tr.Change
	}
	if len(got) != 4 || !reflect.DeepEqual(seen[side], []string{"add-x/1"}) || !reflect.DeepEqual(seen[main], []string{"add-x/2"}) {
		t.Errorf("got %+v", got)
	}
}

func TestChangeTrailersUnbornHead(t *testing.T) {
	r := testutil.NewRepo(t)
	if _, err := (Git{Dir: r.Dir, Env: r.Env}).ChangeTrailers("HEAD"); err == nil {
		t.Error("want git's error for an unborn HEAD")
	}
}
