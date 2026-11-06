package conflicts

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/testutil"
)

const specPath = "openspec/changes/add-x/specs/booking/spec.md"

func gitOf(r *testutil.Repo) gitx.Git { return gitx.Git{Dir: r.Dir, Env: r.Env} }

func addWorktree(t *testing.T, r *testutil.Repo, branch string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "-b", branch, p)
	p, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, r *testutil.Repo) ([]Copy, []scan.Unreadable) {
	t.Helper()
	inv, err := scan.Changes(gitOf(r))
	if err != nil {
		t.Fatal(err)
	}
	cs, un, err := Copies(gitOf(r), inv)
	if err != nil {
		t.Fatal(err)
	}
	return cs, un
}

func TestCopiesMergesIdentical(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{specPath: "one", "openspec/changes/add-x/proposal.md": "p"})
	r.Git("branch", "feat")
	wt := addWorktree(t, r, "wtb")
	cs, un := run(t, r)
	if len(un) != 0 {
		t.Fatalf("unreadable: %v", un)
	}
	// the main checkout, the worktree, and the branches main, feat and wtb
	if len(cs) != 1 || len(cs[0].Sources) != 5 {
		t.Fatalf("copies = %+v", cs)
	}
	c := cs[0]
	if c.Slug != "add-x" || c.Folder != "openspec/changes/add-x" || string(c.Files["booking/spec.md"]) != "one" || len(c.Files) != 1 {
		t.Errorf("copy = %+v", c)
	}
	writeFile(t, wt, specPath, "two")
	cs, _ = run(t, r)
	if len(cs) != 2 || !slices.Equal(cs[1].Sources, []string{wt}) {
		t.Fatalf("copies = %+v", cs)
	}
}

func TestCopiesSkipsArchived(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"openspec/changes/archive/2026-01-02-add-x/proposal.md": "p"})
	r.Git("checkout", "-q", "-b", "old")
	r.Commit("old", map[string]string{specPath: "one"})
	r.Git("checkout", "-q", "main")
	cs, un := run(t, r)
	if len(cs) != 0 || len(un) != 0 {
		t.Errorf("copies = %+v, unreadable = %v", cs, un)
	}
}

func TestCopiesLinkedSpec(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"README.md": "x"})
	wt := addWorktree(t, r, "wtb")
	writeFile(t, wt, "elsewhere.md", "e")
	dir := filepath.Join(wt, "openspec/changes/add-y/specs/booking")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(wt, "elsewhere.md"), filepath.Join(dir, "spec.md")); err != nil {
		t.Fatal(err)
	}
	cs, un := run(t, r)
	if len(cs) != 0 || len(un) != 1 {
		t.Fatalf("copies = %+v, unreadable = %v", cs, un)
	}
	if un[0].Source != wt+":openspec/changes/add-y" || !strings.Contains(un[0].Reason, "link") {
		t.Errorf("unreadable = %+v", un[0])
	}
}

func TestCopiesLinkedSpecOnBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"README.md": "x"})
	r.Git("checkout", "-q", "-b", "lnk")
	r.Write("openspec/changes/add-z/specs/booking/real.md", "r")
	if err := os.Symlink("real.md", filepath.Join(r.Dir, "openspec/changes/add-z/specs/booking/spec.md")); err != nil {
		t.Fatal(err)
	}
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "link")
	r.Git("checkout", "-q", "main")
	cs, un := run(t, r)
	if len(cs) != 0 {
		t.Errorf("copies = %+v", cs)
	}
	if len(un) != 1 || un[0].Source != "refs/heads/lnk:openspec/changes/add-z" || !strings.Contains(un[0].Reason, "link") {
		t.Errorf("unreadable = %v", un)
	}
}

func TestCopiesLinkedSpecsDirOnBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"other/booking/spec.md": "s", "openspec/changes/add-z/proposal.md": "p"})
	r.Git("checkout", "-q", "-b", "lnk")
	if err := os.Symlink("../../../other", filepath.Join(r.Dir, "openspec/changes/add-z/specs")); err != nil {
		t.Fatal(err)
	}
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "link")
	r.Git("checkout", "-q", "main")
	cs, un := run(t, r)
	for _, c := range cs {
		if slices.Contains(c.Sources, "refs/heads/lnk") {
			t.Errorf("copy for linked branch: %+v", c)
		}
	}
	if !slices.ContainsFunc(un, func(u scan.Unreadable) bool {
		return u.Source == "refs/heads/lnk:openspec/changes/add-z" && strings.Contains(u.Reason, "openspec/changes/add-z/specs") && strings.Contains(u.Reason, "link")
	}) {
		t.Errorf("unreadable = %v", un)
	}
}

func TestCopiesOnlySpecMd(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"README.md": "x"})
	wt := addWorktree(t, r, "wtb")
	writeFile(t, wt, "openspec/changes/add-y/specs/booking/spec.md", "s")
	writeFile(t, wt, "openspec/changes/add-y/specs/booking/notes.md", "n")
	cs, un := run(t, r)
	if len(un) != 0 || len(cs) != 1 || len(cs[0].Files) != 1 || string(cs[0].Files["booking/spec.md"]) != "s" {
		t.Errorf("copies = %+v, unreadable = %v", cs, un)
	}
}

func TestCopiesOnlySpecMdOnBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{
		"openspec/changes/add-y/specs/booking/spec.md":  "s",
		"openspec/changes/add-y/specs/booking/notes.md": "n",
	})
	cs, un := run(t, r)
	if len(un) != 0 || len(cs) != 1 || len(cs[0].Files) != 1 || string(cs[0].Files["booking/spec.md"]) != "s" {
		t.Errorf("copies = %+v, unreadable = %v", cs, un)
	}
}

func TestCopiesDistinctSlugsNeverMerge(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{
		"openspec/changes/add-x/specs/booking/spec.md": "same",
		"openspec/changes/add-y/specs/booking/spec.md": "same",
		"openspec/changes/add-p/proposal.md":           "p",
		"openspec/changes/add-q/proposal.md":           "p",
	})
	cs, un := run(t, r)
	slugs := map[string]int{}
	for _, c := range cs {
		slugs[c.Slug]++
	}
	if len(un) != 0 || len(cs) != 4 || slugs["add-x"] != 1 || slugs["add-y"] != 1 || slugs["add-p"] != 1 || slugs["add-q"] != 1 {
		t.Errorf("copies = %+v, unreadable = %v", cs, un)
	}
}

func TestCopiesMissingCommit(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"README.md": "x"})
	inv := scan.Inventory{Uses: []scan.Use{{
		Slug: "add-x", Path: "openspec/changes/add-x", Source: "refs/heads/gone",
		Commit: "0000000000000000000000000000000000000000",
	}}}
	cs, un, err := Copies(gitOf(r), inv)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 0 || len(un) != 1 || un[0].Source != "refs/heads/gone:openspec/changes/add-x" {
		t.Errorf("copies = %+v, unreadable = %v", cs, un)
	}
}

func TestCopiesNoSpecs(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"openspec/changes/add-x/proposal.md": "p"})
	cs, un := run(t, r)
	if len(un) != 0 || len(cs) != 1 || len(cs[0].Files) != 0 {
		t.Errorf("copies = %+v, unreadable = %v", cs, un)
	}
}

func TestCopiesFolderGone(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"README.md": "x"})
	wt := addWorktree(t, r, "wtb")
	writeFile(t, wt, "openspec/changes/add-y/specs/booking/spec.md", "s")
	inv, err := scan.Changes(gitOf(r))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(wt, "openspec/changes/add-y")); err != nil {
		t.Fatal(err)
	}
	cs, un, err := Copies(gitOf(r), inv)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 0 || len(un) != 1 || un[0].Source != wt+":openspec/changes/add-y" || !strings.Contains(un[0].Reason, "gone") {
		t.Errorf("copies = %+v, unreadable = %v", cs, un)
	}
}

func TestCopiesLinkedSpecsDir(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"README.md": "x"})
	wt := addWorktree(t, r, "wtb")
	writeFile(t, wt, "other/booking/spec.md", "s")
	writeFile(t, wt, "openspec/changes/add-y/proposal.md", "p")
	if err := os.Symlink(filepath.Join(wt, "other"), filepath.Join(wt, "openspec/changes/add-y/specs")); err != nil {
		t.Fatal(err)
	}
	cs, un := run(t, r)
	if len(cs) != 0 || len(un) != 1 || !strings.Contains(un[0].Reason, "specs") {
		t.Errorf("copies = %+v, unreadable = %v", cs, un)
	}
}
