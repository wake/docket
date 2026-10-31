package gitx

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/wake/docket/internal/testutil"
)

const recA = "docket/records/a.md"

func addRepo(t *testing.T) (*testutil.Repo, Git) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	r.Commit("init", map[string]string{"README.md": "x\n"})
	return r, Git{Dir: r.Dir, Env: r.Env}
}

func TestAdditionsFirstParent(t *testing.T) {
	r, g := addRepo(t)
	r.Git("checkout", "-q", "-b", "topic")
	r.Commit("add a", map[string]string{recA: "a\n"})
	r.Git("checkout", "-q", "main")
	r.Git("merge", "-q", "--no-ff", "-m", "merge", "topic")
	merge := r.Git("rev-parse", "HEAD")
	got, err := g.Additions("main", "", "docket/records")
	if err != nil || len(got) != 1 || got[0].Path != recA || got[0].SHA != merge || got[0].Time.IsZero() {
		t.Fatalf("got %+v, err %v", got, err)
	}
	if got, err := g.Additions("main", merge, "docket/records"); err != nil || len(got) != 0 {
		t.Fatalf("since: got %+v, err %v", got, err)
	}
}

func TestAdditionsFastForward(t *testing.T) {
	r, g := addRepo(t)
	r.Git("checkout", "-q", "-b", "topic")
	sha := r.Commit("add a", map[string]string{recA: "a\n"})
	r.Git("checkout", "-q", "main")
	r.Git("merge", "-q", "--ff-only", "topic")
	got, err := g.Additions("main", "", "docket/records")
	if err != nil || len(got) != 1 || got[0].SHA != sha {
		t.Fatalf("got %+v, err %v", got, err)
	}
}

func TestAdditionsNewestFirst(t *testing.T) {
	r, g := addRepo(t)
	first := r.Commit("add", map[string]string{recA: "a\n"})
	r.Git("rm", "-q", recA)
	r.Git("commit", "-q", "-m", "rm")
	second := r.Commit("add again", map[string]string{recA: "b\n"})
	got, err := g.Additions("main", "", "docket/records")
	if err != nil || len(got) != 2 || got[0].SHA != second || got[1].SHA != first {
		t.Fatalf("got %+v, err %v", got, err)
	}
}

func TestAdditionsRootCommit(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	r.Git("config", "log.showRoot", "false")
	root := r.Commit("root", map[string]string{recA: "a\n"})
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.Additions("main", "", "docket/records")
	if err != nil || len(got) != 1 || got[0].SHA != root {
		t.Fatalf("got %+v, err %v", got, err)
	}
}

func TestTagsByCreationDate(t *testing.T) {
	r, g := addRepo(t)
	at := func(date string, args ...string) {
		saved := r.Env
		r.Env = append(append([]string{}, saved...), "GIT_COMMITTER_DATE="+date)
		defer func() { r.Env = saved }()
		r.Git(args...)
	}
	// A lightweight tag takes the date of its commit, an annotated one its own.
	at("2026-01-02T00:00:00Z", "commit", "-q", "--allow-empty", "-m", "c2")
	c2 := r.Git("rev-parse", "HEAD")
	r.Git("tag", "v2")
	at("2026-01-03T00:00:00Z", "commit", "-q", "--allow-empty", "-m", "c3")
	c3 := r.Git("rev-parse", "HEAD")
	r.Git("tag", "b")
	r.Git("tag", "a")
	at("2026-01-01T00:00:00Z", "tag", "-a", "-m", "v1", "v1", c2)
	r.Git("tag", "tree", c3+"^{tree}")
	got, err := g.Tags()
	want := []Ref{{"refs/tags/v1", c2}, {"refs/tags/v2", c2}, {"refs/tags/a", c3}, {"refs/tags/b", c3}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, err %v", got, err)
	}
}

func TestTagsMissingObject(t *testing.T) {
	r, g := addRepo(t)
	sha := r.Commit("only for tag", map[string]string{"t.txt": "t"})
	r.Git("tag", "lost")
	r.Git("reset", "-q", "--hard", "HEAD~1")
	obj := filepath.Join(r.Dir, ".git", "objects", sha[:2], sha[2:])
	if err := os.Chmod(obj, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Tags(); err == nil {
		t.Fatal("no error")
	}
}

func TestAdditionsKeepsTheCommittersOffset(t *testing.T) {
	r, g := addRepo(t)
	const date = "2026-10-30T20:00:00+08:00"
	r.Env = append(r.Env, "GIT_COMMITTER_DATE="+date)
	r.Commit("add a", map[string]string{recA: "a\n"})
	got, err := g.Additions("main", "", "docket/records")
	if err != nil || len(got) != 1 || got[0].Time.Format(time.RFC3339) != date {
		t.Fatalf("got %+v, err %v", got, err)
	}
}
