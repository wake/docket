package scan

import (
	"bytes"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/record"
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
	want := readTree(g, commit)
	got := readTrees(g, []string{commit, commit})[commit]
	if !reflect.DeepEqual(sorted(got.active), sorted(want.active)) || !reflect.DeepEqual(sorted(got.archived), sorted(want.archived)) ||
		!reflect.DeepEqual(sorted(got.dated[record.Dir]), sorted(want.dated[record.Dir])) || errText(got.datedErr[record.Dir]) != errText(want.datedErr[record.Dir]) || errText(got.err) != errText(want.err) {
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

// rawEntry is one entry of a tree written byte by byte, so it can hold what
// git's own checks refuse: a slash in a name, or two entries with one name.
type rawEntry struct{ mode, name, oid string }

// rawTree writes a tree object holding the entries as given and returns its id.
func rawTree(t *testing.T, r *testutil.Repo, es ...rawEntry) string {
	t.Helper()
	var buf bytes.Buffer
	for _, e := range es {
		raw, err := hex.DecodeString(e.oid)
		if err != nil {
			t.Fatal(err)
		}
		buf.WriteString(e.mode + " " + e.name + "\x00")
		buf.Write(raw)
	}
	cmd := exec.Command("git", "hash-object", "-t", "tree", "--literally", "-w", "--stdin")
	cmd.Dir, cmd.Env, cmd.Stdin = r.Dir, r.Env, &buf
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("hash-object: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// commitOf commits the tree on its own branch and returns the commit.
func commitOf(t *testing.T, r *testutil.Repo, tree string) string {
	t.Helper()
	c := r.Git("commit-tree", tree, "-m", "raw")
	r.Git("update-ref", "refs/heads/raw", c)
	return c
}

// TestReadTreesMalformedTreesEqualReadTree: trees git never writes, but a
// damaged repository can hold. Every case must come out as readTree has it.
func TestReadTreesMalformedTreesEqualReadTree(t *testing.T) {
	cases := map[string]func(t *testing.T, r *testutil.Repo) string{
		"changes holds an entry named add/x": func(t *testing.T, r *testutil.Repo) string {
			sub := rawTree(t, r, rawEntry{"100644", "a.md", blob(t, r, "a\n")})
			changes := rawTree(t, r, rawEntry{"40000", "add-y", sub}, rawEntry{"40000", "add/x", sub})
			return commitOf(t, r, rawTree(t, r, rawEntry{"40000", "openspec", rawTree(t, r, rawEntry{"40000", "changes", changes})}))
		},
		"archive holds an entry named 2026-01-01-add-x/y": func(t *testing.T, r *testutil.Repo) string {
			sub := rawTree(t, r, rawEntry{"100644", "a.md", blob(t, r, "a\n")})
			archive := rawTree(t, r, rawEntry{"40000", "2026-01-01-add-z", sub}, rawEntry{"40000", "2026-01-01-add-x/y", sub})
			changes := rawTree(t, r, rawEntry{"40000", "archive", archive}, rawEntry{"40000", "add-y", sub})
			return commitOf(t, r, rawTree(t, r, rawEntry{"40000", "openspec", rawTree(t, r, rawEntry{"40000", "changes", changes})}))
		},
		"records hold a blob named 2026-10-30-x/y.md": func(t *testing.T, r *testutil.Repo) string {
			b := blob(t, r, "x\n")
			records := rawTree(t, r, rawEntry{"100644", "2026-10-30-fix-a.md", b}, rawEntry{"100644", "2026-10-30-x/y.md", b})
			return commitOf(t, r, rawTree(t, r, rawEntry{"40000", "docket", rawTree(t, r, rawEntry{"40000", "records", records})}))
		},
		"root holds two openspec entries": func(t *testing.T, r *testutil.Repo) string {
			sub := rawTree(t, r, rawEntry{"100644", "a.md", blob(t, r, "a\n")})
			changes := rawTree(t, r, rawEntry{"40000", "add-y", sub})
			os := rawTree(t, r, rawEntry{"40000", "changes", changes})
			return commitOf(t, r, rawTree(t, r, rawEntry{"100644", "openspec", blob(t, r, "f\n")}, rawEntry{"40000", "openspec", os}))
		},
		"openspec holds two changes entries": func(t *testing.T, r *testutil.Repo) string {
			sub := rawTree(t, r, rawEntry{"100644", "a.md", blob(t, r, "a\n")})
			c1 := rawTree(t, r, rawEntry{"40000", "add-one", sub})
			c2 := rawTree(t, r, rawEntry{"40000", "add-two", sub})
			os := rawTree(t, r, rawEntry{"40000", "changes", c1}, rawEntry{"40000", "changes", c2})
			return commitOf(t, r, rawTree(t, r, rawEntry{"40000", "openspec", os}))
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			r := testutil.NewRepo(t)
			r.Commit("init", map[string]string{"README.md": "x\n"})
			c := build(t, r)
			got := sameAsReadTree(t, gitx.Git{Dir: r.Dir, Env: r.Env}, c)
			t.Logf("readTree: %+v", got)
		})
	}
}

func blob(t *testing.T, r *testutil.Repo, content string) string {
	t.Helper()
	cmd := exec.Command("git", "hash-object", "-w", "--stdin")
	cmd.Dir, cmd.Env, cmd.Stdin = r.Dir, r.Env, strings.NewReader(content)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}
