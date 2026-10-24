package conflicts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/testutil"
)

func specPath(slug, capability string) string {
	return "openspec/changes/" + slug + "/specs/" + capability + "/spec.md"
}

func modifies(name string) string { return section("MODIFIED", name) }

func baseRepo(t *testing.T) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"README.md": "x\n"})
	return r
}

func scanAndLoad(t *testing.T, r *testutil.Repo) ([]Change, []scan.Unreadable) {
	t.Helper()
	g := gitx.Git{Dir: r.Dir, Env: r.Env}
	db, err := scan.ResolveDefaultBranch(g, "")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := scan.Scan(g, db)
	if err != nil {
		t.Fatal(err)
	}
	changes, unread, err := Load(g, inv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return changes, unread
}

func bySlug(changes []Change, slug string) *Change {
	for i := range changes {
		if changes[i].Slug == slug {
			return &changes[i]
		}
	}
	return nil
}

func TestOverlapWithAnotherWorktree(t *testing.T) {
	a := baseRepo(t)
	a.Git("checkout", "-q", "-b", "feat-x")
	a.Commit("x", map[string]string{specPath("add-x", "booking"): modifies("Cancel booking")})
	bPath := filepath.Join(t.TempDir(), "b")
	b := a.AddWorktree(bPath, "feat-y")
	b.Write(specPath("add-y", "booking"), modifies("Cancel booking"))

	changes, unread := scanAndLoad(t, a)
	if len(unread) != 0 {
		t.Fatalf("unreadable = %+v", unread)
	}
	overlaps := Find(changes)
	if len(overlaps) != 1 {
		t.Fatalf("overlaps = %+v", overlaps)
	}
	for _, e := range overlaps[0].Changes {
		if e.Slug == "add-y" {
			real, _ := filepath.EvalSymlinks(bPath)
			got, _ := filepath.EvalSymlinks(e.Sources[0].Name)
			if e.Sources[0].Kind != scan.KindWorktree || got != real {
				t.Fatalf("add-y source = %+v, want worktree %s", e.Sources[0], bPath)
			}
			return
		}
	}
	t.Fatalf("no add-y entry: %+v", overlaps[0])
}

func TestArchivedChangeLeftOnABranch(t *testing.T) {
	r := baseRepo(t)
	r.Git("checkout", "-q", "-b", "old")
	r.Commit("x", map[string]string{specPath("add-x", "booking"): modifies("Cancel booking")})
	r.Git("checkout", "-q", "main")
	r.Commit("archive", map[string]string{"openspec/changes/archive/2026-01-01-add-x/proposal.md": "p\n"})
	r.Commit("y", map[string]string{specPath("add-y", "booking"): modifies("Cancel booking")})

	changes, _ := scanAndLoad(t, r)
	if bySlug(changes, "add-x") != nil {
		t.Fatalf("archived add-x loaded: %+v", changes)
	}
	if bySlug(changes, "add-y") == nil {
		t.Fatalf("add-y missing: %+v", changes)
	}
	if o := Find(changes); len(o) != 0 {
		t.Fatalf("overlaps = %+v", o)
	}
}

func TestSourcesListed(t *testing.T) {
	origin := baseRepo(t)
	first := origin.CloneTo(t, filepath.Join(t.TempDir(), "first"))
	second := origin.CloneTo(t, filepath.Join(t.TempDir(), "second"))
	second.Git("checkout", "-q", "-b", "add-y")
	second.Commit("y", map[string]string{specPath("add-y", "booking"): modifies("Cancel booking")})
	second.Git("push", "-q", origin.Dir, "add-y")
	first.Git("fetch", "-q", "origin")
	b := first.AddWorktree(filepath.Join(t.TempDir(), "b"), "wt")
	b.Write(specPath("add-y", "booking"), modifies("Cancel booking"))

	changes, _ := scanAndLoad(t, first)
	c := bySlug(changes, "add-y")
	if c == nil || len(c.Sources) != 2 {
		t.Fatalf("add-y = %+v", c)
	}
	kinds := map[scan.Kind]bool{}
	for _, s := range c.Sources {
		if s.Commit == "" {
			t.Fatalf("source without commit: %+v", s)
		}
		kinds[s.Kind] = true
	}
	if !kinds[scan.KindWorktree] || !kinds[scan.KindRemoteBranch] {
		t.Fatalf("sources = %+v", c.Sources)
	}
}

func TestSameSlugInManySources(t *testing.T) {
	origin := baseRepo(t)
	first := origin.CloneTo(t, filepath.Join(t.TempDir(), "first"))
	first.Git("checkout", "-q", "-b", "feat")
	first.Commit("y", map[string]string{specPath("add-y", "booking"): modifies("Cancel booking")})
	first.Git("push", "-q", origin.Dir, "feat")
	first.Git("fetch", "-q", "origin")
	first.AddWorktree(filepath.Join(t.TempDir(), "b"), "wt")
	first.Write(specPath("add-y", "booking"), modifies("Cancel booking")+modifies("Other thing"))

	changes, _ := scanAndLoad(t, first)
	n := 0
	for _, c := range changes {
		if c.Slug == "add-y" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("add-y loaded %d times", n)
	}
	c := bySlug(changes, "add-y")
	if len(c.Deltas["booking"].Modified) != 2 {
		t.Fatalf("content not from the first source: %+v", c)
	}
	if o := Find(changes); len(o) != 0 {
		t.Fatalf("overlaps = %+v", o)
	}
}

func TestChangeWithoutSpecs(t *testing.T) {
	r := baseRepo(t)
	r.Write("openspec/changes/add-z/.openspec.yaml", "schema: x\n")
	changes, unread := scanAndLoad(t, r)
	c := bySlug(changes, "add-z")
	if c == nil || c.Deltas == nil || len(c.Deltas) != 0 || len(unread) != 0 {
		t.Fatalf("add-z = %+v unread = %+v", c, unread)
	}
}

func skipIfRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("mode 000 is readable as root")
	}
}

func TestUnreadableDeltaSpec(t *testing.T) {
	skipIfRoot(t)
	r := baseRepo(t)
	r.Write(specPath("add-x", "booking"), modifies("Cancel booking"))
	r.Write(specPath("add-x", "billing"), modifies("Pay"))
	bad := filepath.Join(r.Dir, specPath("add-x", "booking"))
	if err := os.Chmod(bad, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(bad, 0o644) })
	changes, unread := scanAndLoad(t, r)
	if len(unread) != 1 || unread[0].Path != specPath("add-x", "booking") {
		t.Fatalf("unreadable = %+v", unread)
	}
	c := bySlug(changes, "add-x")
	if c == nil || len(c.Deltas) != 1 || len(c.Deltas["billing"].Modified) != 1 {
		t.Fatalf("add-x = %+v", c)
	}
}

func TestUnreadableOtherFileIgnored(t *testing.T) {
	skipIfRoot(t)
	r := baseRepo(t)
	r.Write(specPath("add-x", "booking"), modifies("Cancel booking"))
	r.Write("openspec/changes/add-x/specs/booking/notes.md", "n\n")
	bad := filepath.Join(r.Dir, "openspec/changes/add-x/specs/booking/notes.md")
	if err := os.Chmod(bad, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(bad, 0o644) })
	changes, unread := scanAndLoad(t, r)
	if len(unread) != 0 {
		t.Fatalf("unreadable = %+v", unread)
	}
	if c := bySlug(changes, "add-x"); c == nil || len(c.Deltas) != 1 {
		t.Fatalf("add-x = %+v", c)
	}
}
