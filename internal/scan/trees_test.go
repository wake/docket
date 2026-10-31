package scan

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/testutil"
)

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// sorted copies a list in order: with several paths ls-tree does not list in
// tree order, and nothing downstream reads the order.
func sorted(s []string) []string { s = slices.Clone(s); slices.Sort(s); return s }

// sameAsReadTree fails unless readTrees gives, for the commit, what readTree
// gives, the error text included.
func sameAsReadTree(t *testing.T, g gitx.Git, commit string) treeRead {
	t.Helper()
	var want treeRead
	want.active, want.archived, want.records, want.recErr, want.err = readTree(g, commit)
	got := readTrees(g, []string{commit, commit})[commit]
	if !reflect.DeepEqual(sorted(got.active), sorted(want.active)) || !reflect.DeepEqual(sorted(got.archived), sorted(want.archived)) ||
		!reflect.DeepEqual(sorted(got.records), sorted(want.records)) || errText(got.recErr) != errText(want.recErr) || errText(got.err) != errText(want.err) {
		t.Fatalf("readTrees = %+v\nreadTree  = %+v", got, want)
	}
	return got
}

func link(t *testing.T, r *testutil.Repo, target, rel string) {
	t.Helper()
	p := filepath.Join(r.Dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, p); err != nil {
		t.Fatal(err)
	}
}

func submodule(t *testing.T, r *testutil.Repo, rel string) {
	t.Helper()
	r.Git("update-index", "--add", "--cacheinfo", "160000,"+r.Git("rev-parse", "HEAD")+","+rel)
}

func TestReadTreesEqualsReadTree(t *testing.T) {
	rec := "docket/records/2026-10-30-fix-x.md"
	yaml := "openspec/changes/add-x/.openspec.yaml"
	arch := "openspec/changes/archive/2026-10-24-add-y/proposal.md"
	cases := map[string]func(t *testing.T, r *testutil.Repo){
		"empty":             func(t *testing.T, r *testutil.Repo) {},
		"everything":        func(t *testing.T, r *testutil.Repo) { r.Write(yaml, "a\n"); r.Write(arch, "p\n"); r.Write(rec, "x\n") },
		"only changes":      func(t *testing.T, r *testutil.Repo) { r.Write(yaml, "a\n") },
		"only archive":      func(t *testing.T, r *testutil.Repo) { r.Write(arch, "p\n") },
		"only records":      func(t *testing.T, r *testutil.Repo) { r.Write(rec, "x\n") },
		"archive is a file": func(t *testing.T, r *testutil.Repo) { r.Write(yaml, "a\n"); r.Write("openspec/changes/archive", "f\n") },
		"archive is a link": func(t *testing.T, r *testutil.Repo) {
			r.Write(yaml, "a\n")
			link(t, r, "x", "openspec/changes/archive")
		},
		"archive is submodule": func(t *testing.T, r *testutil.Repo) {
			r.Write(yaml, "a\n")
			submodule(t, r, "openspec/changes/archive")
		},
		"changes is a file":    func(t *testing.T, r *testutil.Repo) { r.Write("openspec/changes", "f\n"); r.Write(rec, "x\n") },
		"changes is a link":    func(t *testing.T, r *testutil.Repo) { link(t, r, "x", "openspec/changes"); r.Write(rec, "x\n") },
		"changes is submodule": func(t *testing.T, r *testutil.Repo) { submodule(t, r, "openspec/changes"); r.Write(rec, "x\n") },
		"openspec is a file":   func(t *testing.T, r *testutil.Repo) { r.Write("openspec", "f\n"); r.Write(rec, "x\n") },
		"openspec is a link":   func(t *testing.T, r *testutil.Repo) { link(t, r, "x", "openspec"); r.Write(rec, "x\n") },
		"docket is a file":     func(t *testing.T, r *testutil.Repo) { r.Write("docket", "f\n"); r.Write(yaml, "a\n") },
		"docket is a link":     func(t *testing.T, r *testutil.Repo) { link(t, r, "x", "docket"); r.Write(yaml, "a\n") },
		"records is a file":    func(t *testing.T, r *testutil.Repo) { r.Write("docket/records", "f\n"); r.Write(yaml, "a\n") },
		"records is a link":    func(t *testing.T, r *testutil.Repo) { link(t, r, "x", "docket/records"); r.Write(yaml, "a\n") },
		"records is submodule": func(t *testing.T, r *testutil.Repo) { submodule(t, r, "docket/records"); r.Write(yaml, "a\n") },
		"changes holds files and a link": func(t *testing.T, r *testutil.Repo) {
			r.Write(yaml, "a\n")
			r.Write("openspec/changes/notes.md", "n\n")
			link(t, r, "add-x", "openspec/changes/alias")
			submodule(t, r, "openspec/changes/mod")
		},
		"archive holds files and a link": func(t *testing.T, r *testutil.Repo) {
			r.Write(arch, "p\n")
			r.Write("openspec/changes/archive/2026-10-24-note.md", "n\n")
			link(t, r, "x", "openspec/changes/archive/2026-10-25-alias")
			submodule(t, r, "openspec/changes/archive/2026-10-26-mod")
		},
		"records hold folders, links, modes and odd names": func(t *testing.T, r *testutil.Repo) {
			r.Write(rec, "x\n")
			r.Write("docket/records/2026-10-30-fix-z.md", "z\n")
			if err := os.Chmod(filepath.Join(r.Dir, "docket/records/2026-10-30-fix-z.md"), 0o755); err != nil {
				t.Fatal(err)
			}
			r.Write("docket/records/sub/2026-10-30-nested.md", "n\n")
			r.Write("docket/records/README.md", "r\n")
			r.Write("docket/records/2026-10-30-with space.md", "s\n")
			link(t, r, "2026-10-30-fix-x.md", "docket/records/2026-10-30-link.md")
			submodule(t, r, "docket/records/2026-10-30-mod.md")
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			r := testutil.NewRepo(t)
			r.Commit("init", map[string]string{"README.md": "x\n"})
			build(t, r)
			r.Git("add", "-A")
			r.Git("commit", "-q", "--allow-empty", "-m", "b")
			sameAsReadTree(t, gitx.Git{Dir: r.Dir, Env: r.Env}, r.Git("rev-parse", "HEAD"))
		})
	}
}

func TestReadTreesMissingObjectEqualsReadTree(t *testing.T) {
	for _, rev := range []string{"HEAD^{tree}", "HEAD:openspec", "HEAD:openspec/changes", "HEAD:openspec/changes/archive", "HEAD:docket", "HEAD:docket/records"} {
		t.Run(rev, func(t *testing.T) {
			r := testutil.NewRepo(t)
			r.Commit("b", map[string]string{
				"openspec/changes/add-x/.openspec.yaml":                 "a\n",
				"openspec/changes/archive/2026-10-24-add-y/proposal.md": "p\n",
				"docket/records/2026-10-30-fix-x.md":                    "x\n",
			})
			sha := r.Git("rev-parse", rev)
			commit := r.Git("rev-parse", "HEAD")
			obj := filepath.Join(r.Dir, ".git", "objects", sha[:2], sha[2:])
			if err := os.Chmod(obj, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(obj); err != nil {
				t.Fatal(err)
			}
			got := sameAsReadTree(t, gitx.Git{Dir: r.Dir, Env: r.Env}, commit)
			if rev != "HEAD:docket" && rev != "HEAD:docket/records" && got.err == nil {
				t.Fatalf("no error for a missing %s", rev)
			}
		})
	}
}

func TestReadTreesUnknownCommitAndBatchFailure(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("b", map[string]string{"openspec/changes/add-x/.openspec.yaml": "a\n"})
	commit := r.Git("rev-parse", "HEAD")
	g := gitx.Git{Dir: r.Dir, Env: r.Env}
	sameAsReadTree(t, g, "0123456789012345678901234567890123456789")
	// A directory that is no repository: every batch fails, and every
	// commit is left to readTree, whose error is the same.
	bad := gitx.Git{Dir: t.TempDir(), Env: r.Env}
	sameAsReadTree(t, bad, commit)
}
