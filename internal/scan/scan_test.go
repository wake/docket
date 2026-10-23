package scan_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/testutil"
)

func yamlPath(slug string) string { return "openspec/changes/" + slug + "/.openspec.yaml" }

func newGit(r *testutil.Repo) gitx.Git { return gitx.Git{Dir: r.Dir, Env: r.Env} }

func baseRepo(t *testing.T) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"README.md": "x\n"})
	return r
}

func scanOf(t *testing.T, r *testutil.Repo) (scan.Inventory, gitx.Git) {
	t.Helper()
	g := newGit(r)
	db, err := scan.ResolveDefaultBranch(g, "")
	if err != nil {
		t.Fatalf("ResolveDefaultBranch: %v", err)
	}
	inv, err := scan.Scan(g, db)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return inv, g
}

func find(inv scan.Inventory, slug string) *scan.Change {
	active, _ := inv.Uses(slug)
	return active
}

func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestChangeOnlyInAnotherWorktree(t *testing.T) {
	a := baseRepo(t)
	bPath := filepath.Join(t.TempDir(), "b")
	b := a.AddWorktree(bPath, "other")
	b.Write(yamlPath("add-x"), "schema: spec-driven\n")
	inv, _ := scanOf(t, a)
	c := find(inv, "add-x")
	if c == nil {
		t.Fatalf("add-x not found: %+v", inv)
	}
	got := c.Sources[0]
	if got.Kind != scan.KindWorktree || realPath(t, got.Name) != realPath(t, bPath) {
		t.Fatalf("Sources[0] = %+v, want worktree %s", got, bPath)
	}
}

func TestChangeOnlyOnRemoteTrackingBranch(t *testing.T) {
	origin := baseRepo(t)
	first := origin.CloneTo(t, filepath.Join(t.TempDir(), "first"))
	second := origin.CloneTo(t, filepath.Join(t.TempDir(), "second"))
	second.Git("checkout", "-q", "-b", "feature-y")
	second.Commit("add y", map[string]string{yamlPath("add-y"): "a: b\n"})
	second.Git("push", "-q", origin.Dir, "feature-y")
	first.Git("fetch", "-q", "origin")
	inv, _ := scanOf(t, first)
	c := find(inv, "add-y")
	if c == nil || len(c.Sources) != 1 {
		t.Fatalf("add-y = %+v", c)
	}
	if c.Sources[0].Kind != scan.KindRemoteBranch || c.Sources[0].Name != "refs/remotes/origin/feature-y" {
		t.Fatalf("source = %+v", c.Sources[0])
	}
}

func TestMergedBranchSkipped(t *testing.T) {
	r := baseRepo(t)
	r.Git("checkout", "-q", "-b", "done")
	r.Commit("add", map[string]string{yamlPath("add-m"): "a: b\n"})
	r.Git("checkout", "-q", "main")
	r.Git("merge", "-q", "--ff-only", "done")
	inv, _ := scanOf(t, r)
	c := find(inv, "add-m")
	if c == nil {
		t.Fatal("add-m not found")
	}
	for _, s := range c.Sources {
		if s.Name == "refs/heads/done" {
			t.Fatalf("merged branch is a source: %+v", c.Sources)
		}
	}
}

func TestOriginHeadSet(t *testing.T) {
	origin := baseRepo(t)
	origin.Git("branch", "trunk")
	c := origin.CloneTo(t, filepath.Join(t.TempDir(), "c"))
	c.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
	db, err := scan.ResolveDefaultBranch(newGit(c), "")
	if err != nil {
		t.Fatal(err)
	}
	if db.Name != "trunk" || db.Ref != "refs/remotes/origin/trunk" || db.Commit == "" {
		t.Fatalf("db = %+v", db)
	}
}

func TestOriginHEADNotASource(t *testing.T) {
	origin := baseRepo(t)
	c := origin.CloneTo(t, filepath.Join(t.TempDir(), "c"))
	c.Commit("add", map[string]string{yamlPath("add-h"): "a: b\n"})
	inv, _ := scanOf(t, c)
	for _, ch := range inv.Changes {
		for _, s := range ch.Sources {
			if strings.HasSuffix(s.Name, "/HEAD") {
				t.Fatalf("HEAD is a source: %+v", s)
			}
		}
	}
	if len(inv.Unreadable) != 0 {
		t.Fatalf("unreadable: %+v", inv.Unreadable)
	}
}

func TestNoDefaultBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "dev")
	r.Commit("init", map[string]string{"a": "b"})
	_, err := scan.ResolveDefaultBranch(newGit(r), "")
	if !errors.Is(err, scan.ErrNoDefaultBranch) {
		t.Fatalf("err = %v", err)
	}
}

func TestDefaultBranchFlagAndMaster(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "master")
	r.Commit("init", map[string]string{"a": "b"})
	r.Git("branch", "dev")
	db, err := scan.ResolveDefaultBranch(newGit(r), "")
	if err != nil || db.Name != "master" || db.Ref != "refs/heads/master" {
		t.Fatalf("db = %+v, %v", db, err)
	}
	db, err = scan.ResolveDefaultBranch(newGit(r), "dev")
	if err != nil || db.Name != "dev" {
		t.Fatalf("flag db = %+v, %v", db, err)
	}
	if _, err = scan.ResolveDefaultBranch(newGit(r), "nope"); !errors.Is(err, scan.ErrNoDefaultBranch) {
		t.Fatalf("err = %v", err)
	}
}

func TestMergedChangeAlsoOnBranch(t *testing.T) {
	r := baseRepo(t)
	r.Commit("x", map[string]string{yamlPath("add-x"): "x-kind: feature\n"})
	r.Git("checkout", "-q", "-b", "feature")
	r.Commit("edit", map[string]string{yamlPath("add-x"): "x-kind: fix\n"})
	r.Git("checkout", "-q", "main")
	inv, g := scanOf(t, r)
	if len(inv.Changes) != 1 {
		t.Fatalf("changes = %+v", inv.Changes)
	}
	c := inv.Changes[0]
	// Sources: the primary worktree (on main), the feature branch, the default tip.
	if len(c.Sources) != 3 || c.Sources[0].Kind != scan.KindWorktree {
		t.Fatalf("sources = %+v", c.Sources)
	}
	var branch scan.Source
	for _, s := range c.Sources {
		if s.Name == "refs/heads/feature" {
			branch = s
		}
	}
	if branch.Kind != scan.KindLocalBranch {
		t.Fatalf("feature branch missing: %+v", c.Sources)
	}
	if last := c.Sources[len(c.Sources)-1]; last.Kind != scan.KindDefaultBranch {
		t.Fatalf("last = %+v", last)
	}
	b, err := scan.ReadFile(g, branch, yamlPath("add-x"))
	if err != nil || !strings.Contains(string(b), "x-kind: fix") {
		t.Fatalf("ReadFile = %q, %v", b, err)
	}
}

func TestSourceOrder(t *testing.T) {
	r := baseRepo(t)
	r.Commit("x", map[string]string{yamlPath("add-x"): "x-kind: feature\n"})
	r.Git("checkout", "-q", "-b", "feature")
	r.Commit("edit", map[string]string{yamlPath("add-x"): "x-kind: fix\n"})
	inv, _ := scanOf(t, r)
	c := find(inv, "add-x")
	kinds := []scan.Kind{}
	for _, s := range c.Sources {
		kinds = append(kinds, s.Kind)
	}
	want := []scan.Kind{scan.KindWorktree, scan.KindLocalBranch, scan.KindDefaultBranch}
	if len(kinds) != len(want) {
		t.Fatalf("kinds = %v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds = %v, want %v", kinds, want)
		}
	}
}

func TestReadFileWorktree(t *testing.T) {
	r := baseRepo(t)
	r.Write(yamlPath("add-w"), "hello\n")
	inv, g := scanOf(t, r)
	c := find(inv, "add-w")
	if c == nil || c.Sources[0].Kind != scan.KindWorktree {
		t.Fatalf("c = %+v", c)
	}
	b, err := scan.ReadFile(g, c.Sources[0], yamlPath("add-w"))
	if err != nil || string(b) != "hello\n" {
		t.Fatalf("ReadFile = %q, %v", b, err)
	}
}

func TestUnreadableBranchTree(t *testing.T) {
	r := baseRepo(t)
	r.Commit("keep", map[string]string{yamlPath("add-keep"): "a: b\n"})
	r.Git("checkout", "-q", "-b", "broken")
	r.Commit("z", map[string]string{yamlPath("add-z"): "a: b\n"})
	sha := r.Git("rev-parse", "broken:openspec/changes")
	r.Git("checkout", "-q", "main")
	obj := filepath.Join(r.Dir, ".git", "objects", sha[:2], sha[2:])
	if err := os.Chmod(obj, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
	inv, _ := scanOf(t, r)
	var got bool
	for _, u := range inv.Unreadable {
		if u.Source.Name == "refs/heads/broken" && u.Reason != "" {
			got = true
		}
	}
	if !got {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
	if find(inv, "add-keep") == nil {
		t.Fatal("other sources lost")
	}
}

func TestArchivedSlugFound(t *testing.T) {
	r := baseRepo(t)
	r.Commit("arch", map[string]string{
		"openspec/changes/archive/2026-10-24-add-x/proposal.md": "p\n",
		"openspec/changes/archive/old-thing/proposal.md":        "p\n",
	})
	inv, _ := scanOf(t, r)
	_, arch := inv.Uses("add-x")
	if len(arch) != 1 || arch[0].Folder != "2026-10-24-add-x" || arch[0].Slug != "add-x" {
		t.Fatalf("archived = %+v", inv.Archived)
	}
	if len(inv.Archived) != 1 || len(inv.Changes) != 0 {
		t.Fatalf("inv = %+v", inv)
	}
}

func TestWorktreeOnUnmountedVolume(t *testing.T) {
	a := baseRepo(t)
	a.Commit("c", map[string]string{yamlPath("add-keep"): "a: b\n"})
	bPath := filepath.Join(t.TempDir(), "b")
	a.AddWorktree(bPath, "other")
	if err := os.RemoveAll(bPath); err != nil {
		t.Fatal(err)
	}
	inv, _ := scanOf(t, a)
	var got bool
	for _, u := range inv.Unreadable {
		if u.Source.Kind == scan.KindWorktree && strings.HasSuffix(u.Source.Name, "/b") && u.Reason != "" {
			got = true
		}
	}
	if !got {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
	if find(inv, "add-keep") == nil {
		t.Fatal("other sources lost")
	}
}

func TestScanOffline(t *testing.T) {
	origin := baseRepo(t)
	c := origin.CloneTo(t, filepath.Join(t.TempDir(), "c"))
	c.Git("remote", "set-url", "origin", "ssh://invalid.invalid/x.git")
	inv, _ := scanOf(t, c)
	if len(inv.Unreadable) != 0 {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
}

func TestPathsWithSpaces(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "with space")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	r := testutil.NewRepoAt(t, dir)
	r.Commit("init", map[string]string{"a": "b"})
	w := r.AddWorktree(filepath.Join(t.TempDir(), "another dir"), "wt")
	w.Write(yamlPath("add-s"), "a: b\n")
	inv, g := scanOf(t, r)
	c := find(inv, "add-s")
	if c == nil {
		t.Fatalf("inv = %+v", inv)
	}
	if _, err := scan.ReadFile(g, c.Sources[0], yamlPath("add-s")); err != nil {
		t.Fatal(err)
	}
}
