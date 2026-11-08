package gitx

import (
	"reflect"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

// commitAt commits with fixed author and committer dates.
func commitAt(t *testing.T, repo *testutil.Repo, date, msg string, files map[string]string) string {
	t.Helper()
	old := repo.Env
	repo.Env = append(append([]string{}, old...), "GIT_COMMITTER_DATE="+date, "GIT_AUTHOR_DATE="+date)
	defer func() { repo.Env = old }()
	return repo.Commit(msg, files)
}

func TestHistory(t *testing.T) {
	repo := testutil.NewRepo(t)
	g := Git{Dir: repo.Dir, Env: repo.Env}
	if h, err := g.History(); h != nil || err != nil {
		t.Fatalf("no revs: %v, %v", h, err)
	}
	commitAt(t, repo, "2026-11-04T10:00:00+08:00", "base", map[string]string{"a": "1"})
	repo.Git("checkout", "-q", "-b", "feat")
	commitAt(t, repo, "2026-11-05T10:00:00+02:00", "work\n\nChange: add-x/1", map[string]string{"b": "1"})
	repo.Git("checkout", "-q", "main")
	commitAt(t, repo, "2026-11-05T11:00:00+08:00", "other\n\nRecord: 2026-11-06-fix-x\n\nbody text", map[string]string{"c": "1"})
	old := repo.Env
	repo.Env = append(append([]string{}, old...), "GIT_COMMITTER_DATE=2026-11-06T09:00:00+08:00")
	repo.Git("merge", "-q", "--no-ff", "-m", "merge feat", "feat")
	repo.Env = old

	h, err := g.History("main")
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 4 {
		t.Fatalf("got %d commits: %+v", len(h), h)
	}
	by := map[string]Commit{}
	for _, c := range h {
		by[c.Subject] = c
	}
	m := by["merge feat"]
	if len(m.Parents) != 2 || m.Time != "2026-11-06T09:00:00+08:00" {
		t.Errorf("merge: %+v", m)
	}
	w := by["work"]
	if w.Time != "2026-11-05T10:00:00+02:00" || !reflect.DeepEqual(w.Trailers, []Trailer{{"Change", "add-x/1"}}) {
		t.Errorf("work: %+v", w)
	}
	if o := by["other"]; len(o.Trailers) != 0 {
		t.Errorf("other: %+v", o)
	}
	if b := by["base"]; len(b.Parents) != 0 {
		t.Errorf("base: %+v", b)
	}
}

func TestHistoryShallow(t *testing.T) {
	repo := testutil.NewRepo(t)
	commitAt(t, repo, "2026-11-04T10:00:00+08:00", "one", map[string]string{"a": "1"})
	commitAt(t, repo, "2026-11-05T10:00:00+08:00", "two", map[string]string{"a": "2"})
	dst := t.TempDir()
	repo.Git("clone", "-q", "--depth", "1", "file://"+repo.Dir, dst)
	h, err := Git{Dir: dst, Env: repo.Env}.History("HEAD")
	if err != nil || len(h) != 1 || h[0].Subject != "two" {
		t.Fatalf("%+v, %v", h, err)
	}
}

func TestTags(t *testing.T) {
	repo := testutil.NewRepo(t)
	g := Git{Dir: repo.Dir, Env: repo.Env}
	old := commitAt(t, repo, "2026-11-01T10:00:00+08:00", "old", map[string]string{"a": "1"})
	newer := commitAt(t, repo, "2026-11-03T10:00:00+08:00", "new", map[string]string{"a": "2"})
	env := repo.Env
	repo.Env = append(append([]string{}, env...), "GIT_COMMITTER_DATE=2026-11-04T10:00:00+08:00")
	repo.Git("tag", "-a", "-m", "v2", "v2", newer)
	repo.Env = env
	repo.Git("tag", "v1", old)
	blob := repo.Git("rev-parse", "HEAD:a")
	repo.Git("tag", "blobtag", blob)
	repo.Git("tag", "-a", "v-blob", blob, "-m", "x")

	tags, err := g.Tags()
	if err != nil {
		t.Fatal(err)
	}
	want := []Tag{{"v1", old, "2026-11-01T10:00:00+08:00"}, {"v2", newer, "2026-11-04T10:00:00+08:00"}}
	if !reflect.DeepEqual(tags, want) {
		t.Errorf("got %+v want %+v", tags, want)
	}
}

func TestAddedDates(t *testing.T) {
	repo := testutil.NewRepo(t)
	g := Git{Dir: repo.Dir, Env: repo.Env}
	commitAt(t, repo, "2026-11-04T10:00:00+08:00", "add", map[string]string{"docket/ideas/a.md": "a\n", "docket/ideas/c.md": "c line one\nc line two\nc line three\n", "other/x.md": "x"})
	commitAt(t, repo, "2026-11-06T10:00:00+08:00", "edit", map[string]string{"docket/ideas/a.md": "a2\n"})
	repo.Git("mv", "docket/ideas/c.md", "docket/ideas/b.md")
	commitAt(t, repo, "2026-11-07T23:30:00-05:00", "rename", nil)

	got, err := g.AddedDates("main", "docket/ideas")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"docket/ideas/a.md": "2026-11-04", "docket/ideas/b.md": "2026-11-07", "docket/ideas/c.md": "2026-11-04"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestHistoryTrailers(t *testing.T) {
	repo := testutil.NewRepo(t)
	commitAt(t, repo, "2026-11-04T10:00:00+08:00", "rec\n\nRecord: 2026-11-06-fix-x", map[string]string{"a": "1"})
	commitAt(t, repo, "2026-11-05T10:00:00+08:00", "fold\n\nChange: add-x/1,\n 2", map[string]string{"a": "2"})
	h, err := Git{Dir: repo.Dir, Env: repo.Env}.History("main")
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]Commit{}
	for _, c := range h {
		by[c.Subject] = c
	}
	if got, want := by["rec"].Trailers, []Trailer{{"Record", "2026-11-06-fix-x"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("rec: %+v", got)
	}
	if got, want := by["fold"].Trailers, []Trailer{{"Change", "add-x/1, 2"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("fold: %+v", got)
	}
}

func TestAddedDatesOldestAndCommitter(t *testing.T) {
	repo := testutil.NewRepo(t)
	g := Git{Dir: repo.Dir, Env: repo.Env}
	commitAt(t, repo, "2026-11-01T10:00:00+08:00", "add", map[string]string{"d/f.md": "f\n", "d/m.md": "m\n"})
	repo.Git("rm", "-q", "d/f.md")
	commitAt(t, repo, "2026-11-02T10:00:00+08:00", "del", nil)
	commitAt(t, repo, "2026-11-03T10:00:00+08:00", "readd", map[string]string{"d/f.md": "f\n", "d/m.md": "m2\n"})
	env := repo.Env
	repo.Env = append(append([]string{}, env...), "GIT_COMMITTER_DATE=2026-11-04T10:00:00+08:00", "GIT_AUTHOR_DATE=2026-11-09T10:00:00+08:00")
	repo.Commit("authored", map[string]string{"d/n.md": "n\n"})
	repo.Env = env

	got, err := g.AddedDates("main", "d")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"d/f.md": "2026-11-01", "d/m.md": "2026-11-01", "d/n.md": "2026-11-04"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}

	// m.md was added before the range and only modified inside it.
	got, err = g.AddedDates("main~3..main", "d")
	if err != nil {
		t.Fatal(err)
	}
	want = map[string]string{"d/f.md": "2026-11-03", "d/n.md": "2026-11-04"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("range: got %v want %v", got, want)
	}
}
