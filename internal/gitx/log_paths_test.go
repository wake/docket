package gitx

import (
	"reflect"
	"testing"
	"time"

	"github.com/wake/docket/internal/testutil"
)

func TestCommitsTime(t *testing.T) {
	r := testutil.NewRepo(t)
	base := r.Commit("base", map[string]string{"a": "0"})
	dates := []string{"2026-01-02T03:04:05Z", "2026-01-05T00:00:00+08:00"}
	env := r.Env
	for i, d := range dates {
		r.Env = append(append([]string{}, env...), "GIT_COMMITTER_DATE="+d)
		r.Commit("c"+d, map[string]string{"a": string(rune('1' + i))})
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
