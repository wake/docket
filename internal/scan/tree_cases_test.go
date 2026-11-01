package scan_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/testutil"
)

// dropObject deletes the loose object rev names from the repository.
func dropObject(t *testing.T, r *testutil.Repo, rev string) {
	t.Helper()
	sha := r.Git("rev-parse", rev)
	obj := filepath.Join(r.Dir, ".git", "objects", sha[:2], sha[2:])
	if err := os.Chmod(obj, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
}

// fullBranch is a repository whose main holds add-keep and whose branch b
// holds a change, an archived change and a record, each folder on the way a
// tree object of its own.
func fullBranch(t *testing.T) *testutil.Repo {
	t.Helper()
	r := baseRepo(t)
	r.Commit("keep", map[string]string{yamlPath("add-keep"): "a: b\n"})
	onBranch(t, r, func() {
		r.Write(yamlPath("add-x"), "a: b\n")
		r.Write("openspec/changes/archive/2026-10-24-add-y/proposal.md", "p\n")
		r.Write("docket/records/2026-10-30-fix-x.md", "x\n")
	})
	return r
}

// A broken records folder sends the commit through the one-folder-at-a-time
// fallback; its issues are still found.
func TestBranchIssuesThroughFallback(t *testing.T) {
	r := baseRepo(t)
	onBranch(t, r, func() {
		r.Write("docket/records/2026-10-30-fix-x.md", "x\n")
		r.Write("docket/issues/2026-11-02-x.md", "x\n")
	})
	dropObject(t, r, "b:docket/records")
	inv, _ := scanOf(t, r)
	if len(inv.Unreadable) != 1 || inv.Unreadable[0].Path != "docket/records" || joined(issueSlugs(inv)) != "2026-11-02-x" {
		t.Fatalf("inv = %+v", inv)
	}
}

func joined(s []string) string { return strings.Join(s, ",") }

func archivedFolders(inv scan.Inventory) []string {
	got := []string{}
	for _, a := range inv.Archived {
		got = append(got, a.Folder)
	}
	return got
}

func TestBranchWithOneMissingObject(t *testing.T) {
	// For each missing tree: what the one Unreadable entry says (the git
	// command that failed, then text that names the failure), and what b still
	// lists. A missing records or docket tree costs the records alone.
	cases := []struct {
		rev, path, command, text string
		changes, archived        string
	}{
		{"b^{tree}", "", "git ls-tree -d --name-only -z {c} -- openspec/changes/: exit status 128", "not a tree object", "", ""},
		{"b:openspec", "", "git ls-tree -d --name-only -z {c} -- openspec/changes/: exit status 1", "Could not read {o}", "", ""},
		{"b:openspec/changes", "", "git ls-tree -d --name-only -z {c} -- openspec/changes/: exit status 1", "Could not read {o}", "", ""},
		{"b:openspec/changes/archive", "", "git ls-tree -d --name-only -z {c} -- openspec/changes/archive/: exit status 1", "Could not read {o}", "", ""},
		{"b:docket", "docket/records", "git ls-tree -r -z {c} -- docket/records: exit status 1", "Could not read {o}", "add-x", "2026-10-24-add-y"},
		{"b:docket/records", "docket/records", "git ls-tree -r -z {c} -- docket/records: exit status 1", "Could not read {o}", "add-x", "2026-10-24-add-y"},
	}
	for _, c := range cases {
		t.Run(c.rev, func(t *testing.T) {
			r := fullBranch(t)
			oid := r.Git("rev-parse", c.rev)
			commit := r.Git("rev-parse", "b")
			dropObject(t, r, c.rev)
			inv, _ := scanOf(t, r)
			// A missing docket tree costs both dated folders, each reported.
			if c.rev == "b:docket" {
				if len(inv.Unreadable) != 2 || inv.Unreadable[1].Path != "docket/issues" {
					t.Fatalf("unreadable = %+v", inv.Unreadable)
				}
				inv.Unreadable = inv.Unreadable[:1]
			}
			if len(inv.Unreadable) != 1 {
				t.Fatalf("unreadable = %+v", inv.Unreadable)
			}
			u := inv.Unreadable[0]
			sub := strings.NewReplacer("{c}", commit, "{o}", oid)
			if u.Source.Kind != scan.KindLocalBranch || u.Source.Name != "refs/heads/b" || u.Source.Commit != commit || u.Path != c.path {
				t.Fatalf("unreadable = %+v", u)
			}
			if !strings.HasPrefix(u.Reason, sub.Replace(c.command)+": ") || !strings.Contains(u.Reason, sub.Replace(c.text)) {
				t.Fatalf("reason = %q, want command %q and %q", u.Reason, sub.Replace(c.command), sub.Replace(c.text))
			}
			want := "add-keep"
			if c.changes != "" {
				want = "add-keep," + c.changes
			}
			got := []string{}
			for _, ch := range inv.Changes {
				got = append(got, ch.Slug)
			}
			if joined(got) != want || joined(archivedFolders(inv)) != c.archived {
				t.Fatalf("changes = %v, archived = %v, want %s and %q", got, archivedFolders(inv), want, c.archived)
			}
			if c.changes != "" && joined(branchChanges(inv)) != "add-keep,add-x" {
				t.Fatalf("branch changes = %v", branchChanges(inv))
			}
		})
	}
}

func TestDefaultTipWithMissingTree(t *testing.T) {
	r := fullBranch(t)
	tip := r.Git("rev-parse", "main")
	dropObject(t, r, "main^{tree}")
	inv, _ := scanOf(t, r)
	if len(inv.Unreadable) != 1 {
		t.Fatalf("unreadable = %+v", inv.Unreadable)
	}
	u := inv.Unreadable[0]
	if u.Source.Kind != scan.KindDefaultBranch || u.Source.Name != "refs/heads/main" || u.Source.Commit != tip || u.Path != "" {
		t.Fatalf("unreadable = %+v", u)
	}
	if !strings.HasPrefix(u.Reason, "git ls-tree -d --name-only -z "+tip+" -- openspec/changes/: exit status 128") || !strings.Contains(u.Reason, "not a tree object") {
		t.Fatalf("reason = %q", u.Reason)
	}
	// The other sources are still read; the default tip supplies nothing.
	if joined(branchChanges(inv)) != "add-keep,add-x" {
		t.Fatalf("inv = %+v", inv)
	}
	for _, c := range inv.Changes {
		for _, s := range c.Sources {
			if s.Kind == scan.KindDefaultBranch {
				t.Fatalf("default tip is a source: %+v", c)
			}
		}
	}
}

func TestBranchNonFolderOnTheWay(t *testing.T) {
	record := "docket/records/2026-10-30-fix-x.md"
	cases := []struct {
		name    string
		files   map[string]string
		changes string
		records string
	}{
		{"openspec is a file", map[string]string{"openspec": "f\n", record: "x\n"}, "", "2026-10-30-fix-x"},
		{"docket is a file", map[string]string{"docket": "f\n", yamlPath("add-x"): "a: b\n"}, "add-x", ""},
		{"openspec/changes is a file", map[string]string{"openspec/changes": "f\n", record: "x\n"}, "", "2026-10-30-fix-x"},
		{"archive is a file", map[string]string{"openspec/changes/archive": "f\n", yamlPath("add-x"): "a: b\n"}, "add-x", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := baseRepo(t)
			onBranch(t, r, func() {
				for rel, content := range c.files {
					r.Write(rel, content)
				}
			})
			inv, _ := scanOf(t, r)
			if len(inv.Unreadable) != 0 || len(inv.Archived) != 0 {
				t.Fatalf("inv = %+v", inv)
			}
			if joined(branchChanges(inv)) != c.changes || joined(recordSlugs(inv)) != c.records {
				t.Fatalf("changes = %v, records = %v", branchChanges(inv), recordSlugs(inv))
			}
		})
	}
}

// recordsWithMode makes branch b, off main, hold one record whose file is
// stored in its tree with the given mode, as git mktree writes it.
func recordsWithMode(t *testing.T, r *testutil.Repo, mode string) {
	t.Helper()
	r.Write("blob.txt", "x\n")
	blob := r.Git("hash-object", "-w", "blob.txt")
	readme := r.Git("rev-parse", "main:README.md")
	records := mktree(t, r, mode+" blob "+blob+"\t2026-10-30-fix-x.md\n")
	docket := mktree(t, r, "040000 tree "+records+"\trecords\n")
	root := mktree(t, r, "100644 blob "+readme+"\tREADME.md\n040000 tree "+docket+"\tdocket\n")
	commit := r.Git("commit-tree", root, "-p", "main", "-m", "b")
	r.Git("update-ref", "refs/heads/b", commit)
}

func mktree(t *testing.T, r *testutil.Repo, lines string) string {
	t.Helper()
	cmd := exec.Command("git", "mktree")
	cmd.Dir, cmd.Env, cmd.Stdin = r.Dir, r.Env, strings.NewReader(lines)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git mktree: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestBranchRecordModes(t *testing.T) {
	// 100755 is listed; a mode git canonicalizes (100664) is read as ls-tree
	// reports it, 100644, so it is listed; a link is not.
	for mode, want := range map[string]string{"100755": "2026-10-30-fix-x", "100664": "2026-10-30-fix-x", "100644": "2026-10-30-fix-x", "120000": ""} {
		t.Run(mode, func(t *testing.T) {
			r := baseRepo(t)
			recordsWithMode(t, r, mode)
			inv, _ := scanOf(t, r)
			if joined(recordSlugs(inv)) != want || len(inv.Unreadable) != 0 {
				t.Fatalf("records = %v, inv = %+v", recordSlugs(inv), inv)
			}
		})
	}
}

func TestSameCommitSources(t *testing.T) {
	r := fullBranch(t)
	tip := r.Git("rev-parse", "b")
	r.Git("branch", "b2", "b")
	r.Git("update-ref", "refs/remotes/origin/b", tip)
	inv, _ := scanOf(t, r)
	var names []string
	for _, s := range find(inv, "add-x").Sources {
		names = append(names, string(s.Kind)+" "+s.Name+" "+s.Commit)
	}
	want := []string{
		"local-branch refs/heads/b " + tip,
		"local-branch refs/heads/b2 " + tip,
		"remote-branch refs/remotes/origin/b " + tip,
	}
	if joined(names) != joined(want) {
		t.Fatalf("sources = %v, want %v", names, want)
	}
	if rs := inv.RecordSources("2026-10-30-fix-x"); len(rs) != 3 {
		t.Fatalf("record sources = %+v", rs)
	}
	if _, arch := inv.Uses("add-y"); len(arch) != 1 || len(arch[0].Sources) != 3 {
		t.Fatalf("archived = %+v", inv.Archived)
	}
}

func TestSHA256Repository(t *testing.T) {
	r := &testutil.Repo{T: t, Dir: t.TempDir(), Env: testutil.Env(t)}
	r.Git("init", "-q", "--object-format=sha256")
	r.Commit("init", map[string]string{"README.md": "x\n"})
	onBranch(t, r, func() {
		r.Write(yamlPath("add-x"), "a: b\n")
		r.Write("openspec/changes/archive/2026-10-24-add-y/proposal.md", "p\n")
		r.Write("docket/records/2026-10-30-fix-x.md", "x\n")
	})
	inv, _ := scanOf(t, r)
	if joined(branchChanges(inv)) != "add-x" || joined(recordSlugs(inv)) != "2026-10-30-fix-x" || joined(archivedFolders(inv)) != "2026-10-24-add-y" || len(inv.Unreadable) != 0 {
		t.Fatalf("inv = %+v", inv)
	}
	if len(r.Git("rev-parse", "b")) != 64 {
		t.Fatal("not a sha256 repository")
	}
}
