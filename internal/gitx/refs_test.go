package gitx

import (
	"slices"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestRefsSkipsSymbolicRefs(t *testing.T) {
	r := testutil.NewRepo(t)
	c := r.Commit("one", map[string]string{"a": "1"})
	r.Git("branch", "other")
	r.Git("update-ref", "refs/remotes/origin/main", c)
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	r.Git("tag", "v1")
	refs, err := Git{Dir: r.Dir, Env: r.Env}.Refs()
	if err != nil {
		t.Fatal(err)
	}
	want := []Ref{
		{"refs/heads/main", c}, {"refs/heads/other", c}, {"refs/remotes/origin/main", c},
	}
	if !slices.Equal(refs, want) {
		t.Errorf("refs = %v, want %v", refs, want)
	}
}

func TestTreeDirsOnlyTrees(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write("openspec/changes/add-x/p.md", "x")
	r.Write("openspec/changes/a-file", "x")
	r.Write("openspec/changes/archive/2026-01-02-old/p.md", "x")
	r.Git("update-index", "--add", "--cacheinfo", "160000,1111111111111111111111111111111111111111,openspec/changes/sub")
	c := r.Commit("one", nil)
	g := Git{Dir: r.Dir, Env: r.Env}
	got, err := g.TreeDirs(c, "openspec/changes/", "openspec/changes/archive/")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"openspec/changes/add-x", "openspec/changes/archive/2026-01-02-old"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	none, err := g.TreeDirs(c, "nope/")
	if err != nil || len(none) != 0 {
		t.Errorf("missing path: %v, %v", none, err)
	}
	if _, err := g.TreeDirs("0000000000000000000000000000000000000000", "openspec/"); err == nil {
		t.Error("bad commit: want error")
	}
}
