package gitx

import (
	"reflect"
	"strconv"
	"testing"
	"time"

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

func TestCommitsWithControlCharacters(t *testing.T) {
	r := testutil.NewRepo(t)
	base := r.Commit("base", map[string]string{"a": "1"})
	r.Commit("sub\x1fject\n\nChange: add-x\nTask: 1\x1e2", map[string]string{"a": "2"})
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.Commits(base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Subject != "sub\x1fject" ||
		!reflect.DeepEqual(got[0].Change, []string{"add-x"}) ||
		!reflect.DeepEqual(got[0].Task, []string{"1\x1e2"}) {
		t.Fatalf("got %+v", got)
	}
}

func TestCommitsZoneOutOfRange(t *testing.T) {
	r := testutil.NewRepo(t)
	base := r.Commit("base", map[string]string{"a": "1"})
	odd := "@1700000000 +2500"
	r.Env = append(r.Env, "GIT_COMMITTER_DATE="+odd, "GIT_AUTHOR_DATE="+odd)
	r.Commit("odd zone\n\nChange: add-x", map[string]string{"a": "2"})
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.Commits(base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Time.Unix() != 1700000000 ||
		!reflect.DeepEqual(got[0].Change, []string{"add-x"}) {
		t.Fatalf("got %+v", got)
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

func TestCommitsTime(t *testing.T) {
	r := testutil.NewRepo(t)
	base := r.Commit("base", map[string]string{"a": "0"})
	dates := []string{"2026-01-02T03:04:05Z", "2026-01-05T00:00:00+08:00"}
	env := r.Env
	for i, d := range dates {
		r.Env = append(append([]string{}, env...), "GIT_COMMITTER_DATE="+d)
		r.Commit("c"+d, map[string]string{"a": strconv.Itoa(i + 1)})
	}
	r.Env = env
	got, err := Git{Dir: r.Dir, Env: r.Env}.Commits(base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	for i, d := range dates {
		want, _ := time.Parse(time.RFC3339, d)
		if !got[i].Time.Equal(want) {
			t.Errorf("commit %d: time %v, want %v", i, got[i].Time, want)
		}
	}
}

func TestCommitsPaths(t *testing.T) {
	r := testutil.NewRepo(t)
	base := r.Commit("base", map[string]string{"z": "0"})
	r.Commit("one\n\nChange: add-x\nTask: 1", map[string]string{"z": "1"})
	r.Commit("two\n\nChange: add-x\nTask: 2", map[string]string{"a/f": "1"})
	r.Commit("three\n\nChange: add-x\nTask: 3", map[string]string{"a/f": "2"})
	g := Git{Dir: r.Dir, Env: r.Env}
	subjects := func(cs []Commit) []string {
		s := []string{}
		for _, c := range cs {
			s = append(s, c.Subject)
		}
		return s
	}
	got, err := g.Commits(base, "HEAD", "a")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(subjects(got), []string{"two", "three"}) ||
		!reflect.DeepEqual(got[0].Task, []string{"2"}) || !reflect.DeepEqual(got[1].Change, []string{"add-x"}) {
		t.Errorf("paths: got %+v", got)
	}
	all, err := g.Commits(base, "HEAD")
	if err != nil || len(all) != 3 {
		t.Errorf("no paths: got %+v, %v", all, err)
	}

	// A branch adding x/f merged with --no-ff after an unrelated main commit.
	mbase := r.Git("rev-parse", "HEAD")
	r.Git("checkout", "-q", "-b", "topic")
	r.Commit("branch work", map[string]string{"x/f": "1"})
	r.Git("checkout", "-q", "main")
	r.Commit("unrelated", map[string]string{"z": "2"})
	r.Git("merge", "-q", "--no-ff", "-m", "merge topic", "topic")
	got, err = g.Commits(mbase, "HEAD", "x")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"branch work", "merge topic"}; !reflect.DeepEqual(subjects(got), want) {
		t.Errorf("merge: got %v, want %v", subjects(got), want)
	}
}
