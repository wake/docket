package gitx

import (
	"reflect"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestCommits(t *testing.T) {
	r := testutil.NewRepo(t)
	base := r.Commit("base\n\nChange: other\nTask: 9", map[string]string{"a": "1"})
	r.Commit("first\n\nChange: add-x\nTask: 1", map[string]string{"a": "2"})
	r.Commit("second\n\nChange: add-x\nTask: 1\nTask: 2", map[string]string{"a": "3"})
	r.Commit("third", map[string]string{"a": "4"})
	r.Commit("fourth\n\nChange: add-y\ntask: 3", map[string]string{"a": "5"})
	r.Commit("fifth\n\nTask: 4\n\nbody text only", map[string]string{"a": "6"})
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.Commits(base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		Subject      string
		Change, Task []string
	}
	var rows []row
	for _, c := range got {
		if len(c.SHA) != 40 {
			t.Errorf("bad SHA %q", c.SHA)
		}
		rows = append(rows, row{c.Subject, append([]string{}, c.Change...), append([]string{}, c.Task...)})
	}
	want := []row{
		{"first", []string{"add-x"}, []string{"1"}},
		{"second", []string{"add-x"}, []string{"1", "2"}},
		{"third", []string{}, []string{}},
		{"fourth", []string{"add-y"}, []string{"3"}},
		{"fifth", []string{}, []string{}},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("got %+v\nwant %+v", rows, want)
	}
}

func TestUncommitted(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"tracked": "1", ".gitignore": "ignored\n"})
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.Uncommitted()
	if err != nil || len(got) != 0 {
		t.Fatalf("clean: %v %v", got, err)
	}
	r.Write("tracked", "2")
	r.Write("staged", "x")
	r.Git("add", "staged")
	r.Write("untracked", "y")
	r.Write("ignored", "z")
	got, err = g.Uncommitted()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"staged", "tracked", "untracked"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestShowIfExists(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"dir/f.md": "hello\n"})
	g := Git{Dir: r.Dir, Env: r.Env}
	b, ok, err := g.ShowIfExists("HEAD", "dir/f.md")
	if err != nil || !ok || string(b) != "hello\n" {
		t.Errorf("existing: %q %v %v", b, ok, err)
	}
	b, ok, err = g.ShowIfExists("HEAD", "dir/missing.md")
	if err != nil || ok || b != nil {
		t.Errorf("missing: %q %v %v", b, ok, err)
	}
	if _, _, err = g.ShowIfExists("no-such-rev", "dir/f.md"); err == nil {
		t.Error("unknown revision: want error")
	}
}
