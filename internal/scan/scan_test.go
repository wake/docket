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

func TestDanglingOriginHeadFallsBackToMain(t *testing.T) {
	r := baseRepo(t)
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/gone")
	db, err := scan.ResolveDefaultBranch(newGit(r), "")
	if err != nil || db.Name != "main" || db.Ref != "refs/heads/main" {
		t.Fatalf("db = %+v, %v", db, err)
	}
}

func TestOriginHeadOutsideOriginNamespace(t *testing.T) {
	r := baseRepo(t)
	r.Git("update-ref", "refs/remotes/upstream/main", "HEAD")
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/upstream/main")
	db, err := scan.ResolveDefaultBranch(newGit(r), "")
	if err != nil || db.Name != "main" || db.Ref != "refs/heads/main" {
		t.Fatalf("db = %+v, %v", db, err)
	}
}

func TestDanglingOriginHeadNoFallbackBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "dev")
	r.Commit("init", map[string]string{"a": "b"})
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/gone")
	if _, err := scan.ResolveDefaultBranch(newGit(r), ""); !errors.Is(err, scan.ErrNoDefaultBranch) {
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
	// Sources: the branch, the main checkout (on the default branch), the default tip.
	if len(c.Sources) != 3 || c.Sources[0].Kind != scan.KindLocalBranch || c.Sources[0].Name != "refs/heads/feature" {
		t.Fatalf("sources = %+v", c.Sources)
	}
	b, err := scan.ReadFile(g, c.Sources[0], yamlPath("add-x"))
	if err != nil || !strings.Contains(string(b), "x-kind: fix") {
		t.Fatalf("ReadFile = %q, %v", b, err)
	}
}

func TestWorktreeWithUnmergedWorkComesFirst(t *testing.T) {
	a := baseRepo(t)
	a.Commit("x", map[string]string{yamlPath("add-x"): "x-kind: feature\n"})
	bPath := filepath.Join(t.TempDir(), "b")
	b := a.AddWorktree(bPath, "b")
	b.Commit("other work", map[string]string{"other.txt": "o\n"})
	b.Write(yamlPath("add-x"), "x-kind: change-request\n")
	inv, g := scanOf(t, a)
	c := find(inv, "add-x")
	if c == nil {
		t.Fatalf("add-x not found: %+v", inv)
	}
	got := c.Sources[0]
	if got.Kind != scan.KindWorktree || realPath(t, got.Name) != realPath(t, bPath) {
		t.Fatalf("Sources[0] = %+v, want worktree %s; all %+v", got, bPath, c.Sources)
	}
	data, err := scan.ReadFile(g, got, yamlPath("add-x"))
	if err != nil || !strings.Contains(string(data), "x-kind: change-request") {
		t.Fatalf("ReadFile = %q, %v", data, err)
	}
}

func TestWorktreeWithFailedAncestryCheckIsUnreadable(t *testing.T) {
	a := baseRepo(t)
	a.Commit("keep", map[string]string{yamlPath("add-keep"): "a: b\n"})
	bPath := filepath.Join(t.TempDir(), "b")
	b := a.AddWorktree(bPath, "w")
	sha := b.Commit("extra", map[string]string{"extra.txt": "e\n"})
	b.Write(yamlPath("add-w"), "a: b\n")
	obj := filepath.Join(a.Dir, ".git", "objects", sha[:2], sha[2:])
	if err := os.Chmod(obj, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
	inv, _ := scanOf(t, a)
	var got bool
	for _, u := range inv.Unreadable {
		if u.Source.Kind == scan.KindWorktree && realPath(t, u.Source.Name) == realPath(t, bPath) && u.Reason != "" {
			got = true
		}
	}
	if !got {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
	if c := find(inv, "add-w"); c != nil {
		t.Fatalf("changes of an unreadable worktree were added: %+v", c)
	}
	if find(inv, "add-keep") == nil {
		t.Fatal("other sources lost")
	}
}

func TestSourceOrder(t *testing.T) {
	r := baseRepo(t)
	r.Commit("x", map[string]string{yamlPath("add-x"): "x-kind: feature\n"})
	mPath := filepath.Join(t.TempDir(), "m")
	r.AddWorktree(mPath, "merged-wt") // HEAD is the default tip: contained
	r.Git("checkout", "-q", "-b", "feature")
	r.Commit("edit", map[string]string{yamlPath("add-x"): "x-kind: fix\n"})
	// The primary worktree is on feature (unmerged work).
	inv, _ := scanOf(t, r)
	c := find(inv, "add-x")
	if c == nil {
		t.Fatalf("add-x not found: %+v", inv)
	}
	want := []scan.Source{
		{Kind: scan.KindWorktree, Name: realPath(t, r.Dir)},
		{Kind: scan.KindLocalBranch, Name: "refs/heads/feature"},
		{Kind: scan.KindWorktree, Name: realPath(t, mPath)},
		{Kind: scan.KindDefaultBranch, Name: "refs/heads/main"},
	}
	if len(c.Sources) != len(want) {
		t.Fatalf("sources = %+v", c.Sources)
	}
	for i, w := range want {
		got := c.Sources[i]
		name := got.Name
		if got.Kind == scan.KindWorktree {
			name = realPath(t, name)
		}
		if got.Kind != w.Kind || name != w.Name {
			t.Fatalf("sources[%d] = %+v, want %+v (all %+v)", i, got, w, c.Sources)
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

func TestReadFileRejectsNonLocalPath(t *testing.T) {
	r := baseRepo(t)
	r.Write(yamlPath("add-w"), "hello\n")
	if err := os.WriteFile(filepath.Join(filepath.Dir(r.Dir), "x"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inv, g := scanOf(t, r)
	c := find(inv, "add-w")
	if c == nil {
		t.Fatalf("inv = %+v", inv)
	}
	dbSrc := inv.Changes[0].Sources[len(c.Sources)-1]
	for _, src := range []scan.Source{c.Sources[0], dbSrc} {
		for _, rel := range []string{"../x", "a/../../x", filepath.Join(r.Dir, "README.md"), ""} {
			if b, err := scan.ReadFile(g, src, rel); err == nil {
				t.Fatalf("ReadFile(%s, %q) = %q, want error", src.Kind, rel, b)
			}
		}
	}
}

func TestReadFileWorktreeSymlinks(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := baseRepo(t)
	r.Write(yamlPath("add-w"), "hello\n")
	if err := os.Symlink(outside, filepath.Join(r.Dir, "leak")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink("README.md", filepath.Join(r.Dir, "inner")); err != nil {
		t.Fatal(err)
	}
	inv, g := scanOf(t, r)
	c := find(inv, "add-w")
	if c == nil || c.Sources[0].Kind != scan.KindWorktree {
		t.Fatalf("c = %+v", c)
	}
	if b, err := scan.ReadFile(g, c.Sources[0], "leak"); err == nil {
		t.Fatalf("symlink out of the worktree read: %q", b)
	}
	b, err := scan.ReadFile(g, c.Sources[0], "inner")
	if err != nil || string(b) != "x\n" {
		t.Fatalf("inner symlink = %q, %v", b, err)
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

func recordSlugs(inv scan.Inventory) []string {
	slugs := []string{}
	for _, r := range inv.Records {
		slugs = append(slugs, r.Slug)
	}
	return slugs
}

func TestScanNoRecordsFolder(t *testing.T) {
	inv, _ := scanOf(t, baseRepo(t))
	if inv.Records == nil || len(inv.Records) != 0 {
		t.Fatalf("records = %#v, want empty and not nil", inv.Records)
	}
	if len(inv.Unreadable) != 0 {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
}

func TestScanRecordsUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("mode 000 does not stop root")
	}
	a := baseRepo(t)
	bPath := filepath.Join(t.TempDir(), "b")
	b := a.AddWorktree(bPath, "other")
	b.Write(yamlPath("add-x"), "schema: spec-driven\n")
	b.Write("docket/records/2026-10-30-fix-x.md", "x\n")
	dir := filepath.Join(bPath, "docket/records")
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	inv, _ := scanOf(t, a)
	var got []scan.Unreadable
	for _, u := range inv.Unreadable {
		if u.Source.Kind == scan.KindWorktree && realPath(t, u.Source.Name) == realPath(t, bPath) {
			got = append(got, u)
		}
	}
	if len(got) != 1 || got[0].Path != "docket/records" {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
	if find(inv, "add-x") == nil {
		t.Fatal("add-x lost")
	}
}

func TestWorktreeListedOnce(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("mode 000 does not stop root")
	}
	a := baseRepo(t)
	bPath := filepath.Join(t.TempDir(), "b")
	b := a.AddWorktree(bPath, "w")
	sha := b.Commit("extra", map[string]string{"docket/records/2026-11-01-fix-x.md": "x\n"})
	obj := filepath.Join(a.Dir, ".git", "objects", sha[:2], sha[2:])
	if err := os.Chmod(obj, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(bPath, "docket/records")
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	inv, _ := scanOf(t, a)
	var got []scan.Unreadable
	for _, u := range inv.Unreadable {
		if u.Source.Kind == scan.KindWorktree && realPath(t, u.Source.Name) == realPath(t, bPath) {
			got = append(got, u)
		}
	}
	if len(got) != 1 || got[0].Path != "" {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
}

func TestRecordOnABranch(t *testing.T) {
	a := baseRepo(t)
	a.Git("checkout", "-q", "-b", "fix-x")
	a.Commit("rec", map[string]string{"docket/records/2026-11-01-fix-x.md": "x\n"})
	a.Git("checkout", "-q", "main")
	bPath := filepath.Join(t.TempDir(), "b")
	a.Git("worktree", "add", "-q", bPath, "fix-x")
	inv, _ := scanOf(t, a)
	if len(inv.Records) != 1 || inv.Records[0].Slug != "2026-11-01-fix-x" {
		t.Fatalf("records = %+v", inv.Records)
	}
	s := inv.Records[0].Sources
	if len(s) != 2 || s[0].Kind != scan.KindWorktree || realPath(t, s[0].Name) != realPath(t, bPath) || s[1].Name != "refs/heads/fix-x" {
		t.Fatalf("sources = %+v", s)
	}
}

func TestScanRecordsNotAFolder(t *testing.T) {
	r := baseRepo(t)
	r.Git("checkout", "-q", "-b", "b")
	r.Commit("file", map[string]string{"docket/records": "not a folder\n"})
	r.Git("checkout", "-q", "main")
	inv, _ := scanOf(t, r)
	if len(inv.Unreadable) != 1 || inv.Unreadable[0].Source.Name != "refs/heads/b" || inv.Unreadable[0].Path != "docket/records" {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
}

func TestScanListsRecords(t *testing.T) {
	r := baseRepo(t)
	r.Commit("z", map[string]string{"docket/records/2026-10-01-fix-z.md": "z\n"})
	r.Git("checkout", "-q", "-b", "b")
	r.Commit("y", map[string]string{
		"docket/records/2026-10-29-fix-y.md":               "y\n",
		"docket/records/sub/2026-10-29-deep.md":            "d\n",
		"docket/records/2026-10-29-dir/2026-10-29-deep.md": "d\n",
	})
	r.Git("checkout", "-q", "main")
	r.Write("docket/records/2026-10-30-fix-x.md", "x\n")
	r.Write("docket/records/2026-10-30-dir/2026-10-30-deep.md", "d\n")
	r.Write("docket/records/README.md", "r\n")
	r.Write("docket/records/2026-10-30-notes.txt", "n\n")
	if err := os.Symlink("2026-10-30-fix-x.md", filepath.Join(r.Dir, "docket/records/2026-10-30-link.md")); err != nil {
		t.Fatal(err)
	}
	inv, _ := scanOf(t, r)
	want := []string{"2026-10-01-fix-z", "2026-10-29-fix-y", "2026-10-30-fix-x"}
	if got := recordSlugs(inv); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("records = %v, want %v", got, want)
	}
	if s := inv.RecordSources("2026-10-30-fix-x"); len(s) != 1 || s[0].Kind != scan.KindWorktree {
		t.Errorf("fix-x sources = %+v", s)
	}
	if s := inv.RecordSources("2026-10-29-fix-y"); len(s) != 1 || s[0].Name != "refs/heads/b" {
		t.Errorf("fix-y sources = %+v", s)
	}
	hasDefault := false
	for _, s := range inv.RecordSources("2026-10-01-fix-z") {
		hasDefault = hasDefault || s.Kind == scan.KindDefaultBranch
	}
	if !hasDefault {
		t.Errorf("fix-z sources lack the default branch")
	}
	if s := inv.RecordSources("2026-10-30-none"); s != nil {
		t.Errorf("none sources = %+v, want nil", s)
	}
	if len(inv.Unreadable) != 0 {
		t.Errorf("unreadable = %+v", inv.Unreadable)
	}
}

// onBranch commits build's changes on a new branch b and returns to main, so
// that b is read as a branch source.
func onBranch(t *testing.T, r *testutil.Repo, build func()) {
	t.Helper()
	r.Git("checkout", "-q", "-b", "b")
	build()
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "b")
	r.Git("checkout", "-q", "main")
}

// branchChanges returns the slugs of the changes that b alone holds.
func branchChanges(inv scan.Inventory) []string {
	slugs := []string{}
	for _, c := range inv.Changes {
		for _, s := range c.Sources {
			if s.Name == "refs/heads/b" {
				slugs = append(slugs, c.Slug)
			}
		}
	}
	return slugs
}

func recordsErrors(inv scan.Inventory) []scan.Unreadable {
	var got []scan.Unreadable
	for _, u := range inv.Unreadable {
		if u.Path == "docket/records" {
			got = append(got, u)
		}
	}
	return got
}

func TestBranchTreeActiveAndArchived(t *testing.T) {
	r := baseRepo(t)
	onBranch(t, r, func() {
		r.Write(yamlPath("add-x"), "a: b\n")
		r.Write("openspec/changes/archive/2026-10-24-add-y/proposal.md", "p\n")
		r.Write("openspec/changes/notes.md", "n\n")
	})
	inv, _ := scanOf(t, r)
	if got := branchChanges(inv); strings.Join(got, ",") != "add-x" {
		t.Fatalf("changes = %v, inv = %+v", got, inv)
	}
	if find(inv, "archive") != nil || find(inv, "notes.md") != nil {
		t.Fatalf("archive folder or file counted as a change: %+v", inv.Changes)
	}
	if _, arch := inv.Uses("add-y"); len(arch) != 1 || arch[0].Folder != "2026-10-24-add-y" {
		t.Fatalf("archived = %+v", inv.Archived)
	}
	if len(inv.Unreadable) != 0 {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
}

func TestBranchTreeArchiveEmptyOrMissing(t *testing.T) {
	r := baseRepo(t)
	onBranch(t, r, func() { r.Write(yamlPath("add-x"), "a: b\n") })
	inv, _ := scanOf(t, r)
	if got := branchChanges(inv); strings.Join(got, ",") != "add-x" || len(inv.Archived) != 0 || len(inv.Unreadable) != 0 {
		t.Fatalf("missing archive: inv = %+v", inv)
	}
	// An archive that is a file is no folder: nothing archived.
	r2 := baseRepo(t)
	onBranch(t, r2, func() {
		r2.Write(yamlPath("add-x"), "a: b\n")
		r2.Write("openspec/changes/archive", "file\n")
	})
	inv, _ = scanOf(t, r2)
	if len(inv.Archived) != 0 || find(inv, "archive") != nil {
		t.Fatalf("file archive: inv = %+v", inv)
	}
	if got := branchChanges(inv); strings.Join(got, ",") != "add-x" {
		t.Fatalf("file archive: changes = %v", got)
	}
}

func TestBranchTreeNoChangesFolder(t *testing.T) {
	r := baseRepo(t)
	onBranch(t, r, func() {
		r.Write("openspec/changes/archive/2026-10-24-add-y/proposal.md", "p\n")
	})
	inv, _ := scanOf(t, r)
	if len(inv.Changes) != 0 || len(inv.Archived) != 1 || len(inv.Unreadable) != 0 {
		t.Fatalf("inv = %+v", inv)
	}
}

func TestBranchRecordsLinkIsNotAFolder(t *testing.T) {
	r := baseRepo(t)
	onBranch(t, r, func() {
		r.Write(yamlPath("add-x"), "a: b\n")
		if err := os.MkdirAll(filepath.Join(r.Dir, "docket"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("elsewhere", filepath.Join(r.Dir, "docket/records")); err != nil {
			t.Fatal(err)
		}
	})
	inv, _ := scanOf(t, r)
	got := recordsErrors(inv)
	if len(got) != 1 || got[0].Source.Name != "refs/heads/b" || got[0].Reason != "docket/records is not a folder at "+r.Git("rev-parse", "b") {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
	if len(inv.Unreadable) != 1 || strings.Join(branchChanges(inv), ",") != "add-x" {
		t.Fatalf("inv = %+v", inv)
	}
}

func TestBranchRecordsSubmoduleIsNotAFolder(t *testing.T) {
	r := baseRepo(t)
	r.Git("checkout", "-q", "-b", "b")
	r.Write(yamlPath("add-x"), "a: b\n")
	r.Git("add", "-A")
	r.Git("update-index", "--add", "--cacheinfo", "160000,"+r.Git("rev-parse", "HEAD")+",docket/records")
	r.Git("commit", "-q", "-m", "b")
	r.Git("checkout", "-q", "main")
	inv, _ := scanOf(t, r)
	got := recordsErrors(inv)
	if len(got) != 1 || got[0].Reason != "docket/records is not a folder at "+r.Git("rev-parse", "b") {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
	if strings.Join(branchChanges(inv), ",") != "add-x" {
		t.Fatalf("changes = %v", branchChanges(inv))
	}
}

func TestBranchRecordsFileReason(t *testing.T) {
	r := baseRepo(t)
	onBranch(t, r, func() {
		r.Write(yamlPath("add-x"), "a: b\n")
		r.Write("docket/records", "file\n")
	})
	inv, _ := scanOf(t, r)
	got := recordsErrors(inv)
	if len(got) != 1 || got[0].Reason != "docket/records is not a folder at "+r.Git("rev-parse", "b") {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
	if strings.Join(branchChanges(inv), ",") != "add-x" {
		t.Fatalf("changes = %v", branchChanges(inv))
	}
}

func TestBranchRecordLinkNotListed(t *testing.T) {
	r := baseRepo(t)
	onBranch(t, r, func() {
		r.Write("docket/records/2026-10-30-fix-x.md", "x\n")
		if err := os.Symlink("2026-10-30-fix-x.md", filepath.Join(r.Dir, "docket/records/2026-10-30-link.md")); err != nil {
			t.Fatal(err)
		}
		r.Write("docket/records/2026-10-30-fix-z.md", "z\n")
		if err := os.Chmod(filepath.Join(r.Dir, "docket/records/2026-10-30-fix-z.md"), 0o755); err != nil {
			t.Fatal(err)
		}
	})
	inv, _ := scanOf(t, r)
	want := []string{"2026-10-30-fix-x", "2026-10-30-fix-z"}
	if got := recordSlugs(inv); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("records = %v, want %v", got, want)
	}
	if len(inv.Unreadable) != 0 {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
}

func TestBranchRecordsTreeMissingIsRecordsErrorAlone(t *testing.T) {
	r := baseRepo(t)
	onBranch(t, r, func() {
		r.Write(yamlPath("add-x"), "a: b\n")
		r.Write("docket/records/2026-10-30-fix-x.md", "x\n")
	})
	sha := r.Git("rev-parse", "b:docket/records")
	obj := filepath.Join(r.Dir, ".git", "objects", sha[:2], sha[2:])
	if err := os.Chmod(obj, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
	inv, _ := scanOf(t, r)
	got := recordsErrors(inv)
	if len(got) != 1 || got[0].Source.Name != "refs/heads/b" || got[0].Reason == "" {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
	if len(inv.Unreadable) != 1 || strings.Join(branchChanges(inv), ",") != "add-x" {
		t.Fatalf("inv = %+v", inv)
	}
}

func TestBranchFileInArchiveIsNotArchived(t *testing.T) {
	r := baseRepo(t)
	onBranch(t, r, func() {
		r.Write(yamlPath("add-keep"), "a: b\n")
		r.Write("openspec/changes/archive/2026-01-01-add-x.md", "n\n")
		r.Write("openspec/changes/archive/2026-01-01-add-x", "n\n")
	})
	inv, _ := scanOf(t, r)
	if len(inv.Archived) != 0 {
		t.Fatalf("archived = %+v", inv.Archived)
	}
	if _, arch := inv.Uses("add-x"); len(arch) != 0 {
		t.Fatalf("add-x archived = %+v", arch)
	}
	if strings.Join(branchChanges(inv), ",") != "add-keep" || len(inv.Unreadable) != 0 {
		t.Fatalf("inv = %+v", inv)
	}
}
