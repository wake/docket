package scan

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/testutil"
)

func gitOf(r *testutil.Repo) gitx.Git { return gitx.Git{Dir: r.Dir, Env: r.Env} }

func realPath(t *testing.T, p string) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func addWorktree(t *testing.T, r *testutil.Repo, branch string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "-b", branch, p)
	return realPath(t, p)
}

func write(t *testing.T, root, rel string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// commitRef creates ref (a full name) at a new commit holding files, leaving
// the repository on main.
func commitRef(r *testutil.Repo, ref string, files map[string]string) string {
	r.Git("checkout", "-q", "-b", "tmp-scan", "main")
	c := r.Commit("c "+ref, files)
	r.Git("update-ref", ref, c)
	r.Git("checkout", "-q", "main")
	r.Git("branch", "-q", "-D", "tmp-scan")
	return c
}

func TestChangesRecordsCommit(t *testing.T) {
	r := newRepo(t)
	c := r.Commit("c", map[string]string{"openspec/changes/add-x/proposal.md": "x"})
	wt := addWorktree(t, r, "wtb")
	write(t, wt, "openspec/changes/add-w/proposal.md")
	inv, err := Changes(gitOf(r))
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, u := range inv.Uses {
		switch {
		case u.Source == "refs/heads/main" && u.Slug == "add-x":
			seen++
			if u.Commit != c {
				t.Errorf("branch Commit = %q, want %q", u.Commit, c)
			}
		case u.Source == wt && u.Slug == "add-w":
			seen++
			if u.Commit != "" {
				t.Errorf("worktree Commit = %q", u.Commit)
			}
		}
	}
	if seen != 2 {
		t.Errorf("uses = %v", inv.Uses)
	}
}

func newRepo(t *testing.T) *testutil.Repo {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"README.md": "x"})
	return r
}

func scanOK(t *testing.T, r *testutil.Repo) Inventory {
	t.Helper()
	inv, err := Changes(gitOf(r))
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

// Scenario: Change only in another worktree
func TestChangeOnlyInAnotherWorktree(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	write(t, b, "openspec/changes/add-x/proposal.md")
	inv := scanOK(t, r)
	want := Use{Slug: "add-x", Path: "openspec/changes/add-x", Source: b}
	if !slices.Contains(inv.Uses, want) || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

// Scenario: Change only on a remote-tracking branch
func TestChangeOnlyOnRemoteTrackingBranch(t *testing.T) {
	r := newRepo(t)
	c := commitRef(r, "refs/remotes/origin/feature-y", map[string]string{"openspec/changes/add-y/p.md": "x"})
	inv := scanOK(t, r)
	want := []Use{{Slug: "add-y", Path: "openspec/changes/add-y", Source: "refs/remotes/origin/feature-y", Commit: c}}
	if !slices.Equal(inv.Uses, want) {
		t.Errorf("got %+v", inv.Uses)
	}
}

// Scenario: Archived slug found
func TestArchivedSlugFound(t *testing.T) {
	r := newRepo(t)
	c := commitRef(r, "refs/heads/old", map[string]string{"openspec/changes/archive/2026-10-24-add-x/p.md": "x"})
	inv := scanOK(t, r)
	want := []Use{{Slug: "add-x", Archived: true, Path: "openspec/changes/archive/2026-10-24-add-x", Source: "refs/heads/old", Commit: c}}
	if !slices.Equal(inv.Uses, want) {
		t.Errorf("got %+v", inv.Uses)
	}
}

// Scenario: Project without changes
func TestProjectWithoutChanges(t *testing.T) {
	r := newRepo(t)
	addWorktree(t, r, "b")
	inv := scanOK(t, r)
	if len(inv.Uses) != 0 || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

// Scenario: Worktree on an unmounted volume
func TestWorktreeOnUnmountedVolume(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	if err := os.RemoveAll(b); err != nil {
		t.Fatal(err)
	}
	inv := scanOK(t, r)
	if len(inv.Unreadable) != 1 || inv.Unreadable[0].Source != b ||
		!strings.Contains(inv.Unreadable[0].Reason, "git worktree prune") {
		t.Errorf("got %+v", inv.Unreadable)
	}
}

// Scenario: Linked changes folder in another worktree
func TestLinkedChangesFolderInAnotherWorktree(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	outside := t.TempDir()
	write(t, outside, "leaked/p.md")
	if err := os.MkdirAll(filepath.Join(b, "openspec"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(b, "openspec/changes")); err != nil {
		t.Fatal(err)
	}
	inv := scanOK(t, r)
	if len(inv.Uses) != 0 || len(inv.Unreadable) != 1 || inv.Unreadable[0].Source != b ||
		!strings.Contains(inv.Unreadable[0].Reason, "openspec/changes is a symbolic link") {
		t.Errorf("got %+v", inv)
	}
}

// Scenario: Scan offline
func TestScanOffline(t *testing.T) {
	r := newRepo(t)
	r.Git("remote", "add", "origin", "/nonexistent")
	commitRef(r, "refs/remotes/origin/feature-y", map[string]string{"openspec/changes/add-y/p.md": "x"})
	inv := scanOK(t, r)
	if len(inv.Uses) != 1 {
		t.Errorf("got %+v", inv)
	}
}

func TestBareEntryAndOriginHeadAddNoSource(t *testing.T) {
	r := newRepo(t)
	commitRef(r, "refs/heads/f", map[string]string{"openspec/changes/add-x/p.md": "x"})
	bare := filepath.Join(t.TempDir(), "bare.git")
	r.Git("clone", "-q", "--bare", r.Dir, bare)
	inv, err := Changes(gitx.Git{Dir: bare, Env: r.Env})
	if err != nil {
		t.Fatal(err)
	}
	write(t, bare, "openspec/changes/add-y/p.md")
	inv, err = Changes(gitx.Git{Dir: bare, Env: r.Env})
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Unreadable) != 0 || len(inv.Uses) != 1 || inv.Uses[0].Source != "refs/heads/f" {
		t.Errorf("bare: %+v", inv)
	}
	r.Git("update-ref", "refs/remotes/origin/main", "main")
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	inv = scanOK(t, r)
	for _, u := range inv.Uses {
		if u.Source == "refs/remotes/origin/HEAD" {
			t.Errorf("symbolic ref used: %+v", u)
		}
	}
}

func TestTwoRefsAtOneCommitAreBothSources(t *testing.T) {
	r := newRepo(t)
	c := commitRef(r, "refs/heads/a", map[string]string{"openspec/changes/add-x/p.md": "x"})
	r.Git("update-ref", "refs/remotes/origin/a", c)
	inv := scanOK(t, r)
	var srcs []string
	for _, u := range inv.Uses {
		srcs = append(srcs, u.Source)
	}
	if !slices.Equal(srcs, []string{"refs/heads/a", "refs/remotes/origin/a"}) {
		t.Errorf("sources = %v", srcs)
	}
}

func TestNamesThatAreNotUses(t *testing.T) {
	r := newRepo(t)
	commitRef(r, "refs/heads/f", map[string]string{
		"openspec/changes/archive/not-a-date/p.md":       "x",
		"openspec/changes/archive/2026-01-01-/p.md":      "x",
		"openspec/changes/archive/2026-01-02-ok/p.md":    "x",
		"openspec/changes/a-file":                        "x",
		"openspec/changes/archive/2026-01-03-a-file-too": "x",
	})
	b := addWorktree(t, r, "b")
	write(t, b, "openspec/changes/a-file")
	write(t, b, "openspec/changes/archive/not-a-date/p.md")
	write(t, b, "openspec/changes/archive/2026-01-02-ok/p.md")
	if err := os.Symlink(t.TempDir(), filepath.Join(b, "openspec/changes/linked")); err != nil {
		t.Fatal(err)
	}
	inv := scanOK(t, r)
	var got []string
	for _, u := range inv.Uses {
		got = append(got, u.Slug+"@"+u.Source)
	}
	want := []string{"ok@" + b, "ok@refs/heads/f"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFileLinkOrSubmoduleAtChangePathIsNoUse(t *testing.T) {
	r := newRepo(t)
	r.Git("checkout", "-q", "-b", "odd")
	r.Write("openspec/changes/a-file", "x")
	if err := os.Symlink("README.md", filepath.Join(r.Dir, "openspec/changes/a-link")); err != nil {
		t.Fatal(err)
	}
	r.Git("add", "-A")
	r.Git("update-index", "--add", "--cacheinfo", "160000,1111111111111111111111111111111111111111,openspec/changes/sub")
	r.Git("commit", "-q", "-m", "odd")
	if !strings.Contains(r.Git("ls-tree", "HEAD", "openspec/changes/sub"), "commit") {
		t.Fatal("the commit does not hold the gitlink")
	}
	r.Git("checkout", "-q", "main")
	inv := scanOK(t, r)
	if len(inv.Uses) != 0 || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

func TestUnreadableWorktreeDirectory(t *testing.T) {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("mode 000 does not stop root")
	}
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	if err := os.Chmod(b, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(b, 0o755) })
	inv := scanOK(t, r)
	if len(inv.Unreadable) != 1 || inv.Unreadable[0].Source != b || inv.Unreadable[0].Reason == "" {
		t.Errorf("got %+v", inv.Unreadable)
	}
}

func TestOrderWorktreesThenRefsSortedByPath(t *testing.T) {
	r := newRepo(t)
	commitRef(r, "refs/heads/f", map[string]string{"openspec/changes/beta/p.md": "x"})
	write(t, r.Dir, "openspec/changes/zed/p.md")
	write(t, r.Dir, "openspec/changes/alpha/p.md")
	inv := scanOK(t, r)
	var got []string
	for _, u := range inv.Uses {
		got = append(got, u.Slug)
	}
	if !slices.Equal(got, []string{"alpha", "zed", "beta"}) {
		t.Errorf("got %v", got)
	}
}

func TestChangesFromSubdirectory(t *testing.T) {
	r := newRepo(t)
	c := commitRef(r, "refs/heads/f", map[string]string{"openspec/changes/add-x/p.md": "x"})
	write(t, r.Dir, "sub/dir/keep")
	inv, err := Changes(gitx.Git{Dir: filepath.Join(r.Dir, "sub/dir"), Env: r.Env})
	if err != nil {
		t.Fatal(err)
	}
	want := Use{Slug: "add-x", Path: "openspec/changes/add-x", Source: "refs/heads/f", Commit: c}
	if !slices.Contains(inv.Uses, want) {
		t.Errorf("got %+v", inv)
	}
}

func TestOpenspecThatIsAFileHoldsNoUse(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	write(t, b, "openspec")
	c := addWorktree(t, r, "c")
	write(t, c, "openspec/changes")
	inv := scanOK(t, r)
	if len(inv.Uses) != 0 || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

func TestDatedArchiveFolderOnDisk(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	write(t, b, "openspec/changes/archive/2026-01-02-add-z/p.md")
	inv := scanOK(t, r)
	want := Use{Slug: "add-z", Archived: true, Path: "openspec/changes/archive/2026-01-02-add-z", Source: b}
	if !slices.Contains(inv.Uses, want) || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

func TestLinkedArchiveFolderOnly(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	write(t, b, "openspec/changes/add-x/p.md")
	if err := os.Symlink(t.TempDir(), filepath.Join(b, "openspec/changes/archive")); err != nil {
		t.Fatal(err)
	}
	inv := scanOK(t, r)
	if len(inv.Unreadable) != 1 || inv.Unreadable[0].Source != b ||
		!strings.Contains(inv.Unreadable[0].Reason, "openspec/changes/archive") {
		t.Errorf("got %+v", inv)
	}
}

func TestRefReadFailureIsUnreadable(t *testing.T) {
	r := newRepo(t)
	blob := r.Git("rev-parse", "HEAD:README.md")
	// update-ref refuses a non-commit, so write the ref file by hand.
	if err := os.WriteFile(filepath.Join(r.Dir, ".git/refs/heads/broken"), []byte(blob+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inv := scanOK(t, r)
	if len(inv.Unreadable) != 1 || inv.Unreadable[0].Source != "refs/heads/broken" ||
		!strings.Contains(inv.Unreadable[0].Reason, "ls-tree") {
		t.Errorf("got %+v", inv)
	}
}

// counted returns g with a fake git first on PATH, and a function that gives
// the arguments of every git run so far, one line each.
func counted(t *testing.T, g gitx.Git) (gitx.Git, func() []string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := realPath(t, t.TempDir())
	log := filepath.Join(bin, "log")
	script := "#!/bin/sh\necho \"$@\" >> '" + log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	g.Env = append(slices.Clone(g.Env), "PATH="+bin+":"+os.Getenv("PATH")) // the last PATH wins
	return g, func() []string {
		b, _ := os.ReadFile(log)
		return strings.Split(strings.TrimSpace(string(b)), "\n")
	}
}

func TestScanGitRuns(t *testing.T) {
	r := newRepo(t)
	c1 := r.Commit("one", map[string]string{"openspec/changes/a/proposal.md": "x", "docket/records/r.md": "x"})
	c2 := commitRef(r, "refs/heads/two", map[string]string{"openspec/changes/b/proposal.md": "x", "docket/issues/i.md": "x"})
	r.Git("branch", "again", c1)
	g, runs := counted(t, gitOf(r))
	ws, err := Scan(g)
	if err != nil {
		t.Fatal(err)
	}
	n := map[string]int{}
	for _, l := range runs() {
		n[strings.Fields(l)[0]]++
	}
	if len(runs()) != 4 || n["worktree"] != 1 || n["for-each-ref"] != 1 || n["ls-tree"] != 2 {
		t.Fatalf("runs %q", runs())
	}
	if len(ws.Trees) != 2 || ws.Trees[c2] == nil {
		t.Fatalf("trees %v", ws.Trees)
	}
	count := func(inv Inventory, ref string) (k int) {
		for _, u := range inv.Uses {
			if u.Source == ref {
				k++
			}
		}
		return k
	}
	if count(ws.Changes, "refs/heads/again") != 1 || count(ws.Changes, "refs/heads/two") != 2 ||
		count(ws.Records, "refs/heads/again") != 1 || count(ws.Issues, "refs/heads/two") != 1 || count(ws.Issues, "refs/heads/again") != 0 {
		t.Fatalf("%+v", ws)
	}
	for _, f := range []func(gitx.Git) (Inventory, error){Changes, Records, Issues} {
		if inv, err := f(gitOf(r)); err != nil || len(inv.Unreadable) != 0 {
			t.Fatalf("%+v %v", inv, err)
		}
	}
}

func TestScanUnreadableRef(t *testing.T) {
	r := newRepo(t)
	r.Commit("one", map[string]string{"openspec/changes/a/proposal.md": "x", "docket/records/r.md": "x"})
	blob := r.Git("hash-object", "-w", "README.md")
	r.Git("update-ref", "refs/remotes/origin/bad", blob)
	ws, err := Scan(gitOf(r))
	if err != nil {
		t.Fatal(err)
	}
	for name, inv := range map[string]Inventory{"changes": ws.Changes, "records": ws.Records, "issues": ws.Issues} {
		if len(inv.Unreadable) != 1 || inv.Unreadable[0].Source != "refs/remotes/origin/bad" {
			t.Errorf("%s: %+v", name, inv.Unreadable)
		}
	}
	if len(ws.Changes.Uses) == 0 || len(ws.Records.Uses) == 0 {
		t.Errorf("main not scanned: %+v", ws)
	}
	if _, ok := ws.Trees[blob]; ok {
		t.Errorf("trees hold the blob")
	}
}
