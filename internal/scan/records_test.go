package scan

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/testutil"
)

const recordPath = "docket/records/2026-11-06-fix-x.md"

func recordsOK(t *testing.T, r *testutil.Repo) Inventory {
	t.Helper()
	inv, err := Records(gitOf(r))
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

// Scenario: Record only in another worktree
func TestRecordsOtherWorktree(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	write(t, b, recordPath)
	inv := recordsOK(t, r)
	want := Use{Slug: "2026-11-06-fix-x", Path: recordPath, Source: b}
	if !slices.Contains(inv.Uses, want) || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

// Scenario: Record only on a branch
func TestRecordsOnBranch(t *testing.T) {
	r := newRepo(t)
	c := commitRef(r, "refs/heads/f", map[string]string{recordPath: "x"})
	inv := recordsOK(t, r)
	want := Use{Slug: "2026-11-06-fix-x", Path: recordPath, Source: "refs/heads/f", Commit: c}
	if !slices.Contains(inv.Uses, want) || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

// Scenario: Project without records
func TestRecordsNoFolder(t *testing.T) {
	r := newRepo(t)
	addWorktree(t, r, "b")
	inv := recordsOK(t, r)
	if len(inv.Uses) != 0 || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

func TestRecordsDocketIsFile(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	write(t, b, "docket")
	inv := recordsOK(t, r)
	if len(inv.Uses) != 0 || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

// Scenario: Linked records folder in another worktree
func TestRecordsLinkedFolder(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	outside := t.TempDir()
	write(t, outside, "2026-11-06-leak.md")
	if err := os.MkdirAll(filepath.Join(b, "docket"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(b, "docket/records")); err != nil {
		t.Fatal(err)
	}
	inv := recordsOK(t, r)
	if len(inv.Uses) != 0 || len(inv.Unreadable) != 1 || inv.Unreadable[0].Source != b ||
		!strings.Contains(inv.Unreadable[0].Reason, "docket/records is a symbolic link; Docket does not read through it") {
		t.Errorf("got %+v", inv)
	}
}

func TestRecordsBareEntry(t *testing.T) {
	r := newRepo(t)
	commitRef(r, "refs/heads/f", map[string]string{recordPath: "x"})
	bare := filepath.Join(t.TempDir(), "bare.git")
	r.Git("clone", "-q", "--bare", r.Dir, bare)
	write(t, bare, "docket/records/2026-11-06-on-disk.md")
	inv, err := Records(gitx.Git{Dir: bare, Env: r.Env})
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Unreadable) != 0 || len(inv.Uses) != 1 || inv.Uses[0].Source != "refs/heads/f" {
		t.Errorf("got %+v", inv)
	}
}

func TestRecordsUnreadableBranch(t *testing.T) {
	r := newRepo(t)
	commit := commitRef(r, "refs/heads/f", map[string]string{recordPath: "x"})
	tree := r.Git("rev-parse", commit+"^{tree}")
	if err := os.Remove(filepath.Join(r.Dir, ".git/objects", tree[:2], tree[2:])); err != nil {
		t.Fatal(err)
	}
	inv := recordsOK(t, r)
	if len(inv.Uses) != 0 || len(inv.Unreadable) != 1 || inv.Unreadable[0].Source != "refs/heads/f" {
		t.Errorf("got %+v", inv)
	}
}

func TestRecordsMissingWorktree(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	if err := os.RemoveAll(b); err != nil {
		t.Fatal(err)
	}
	inv := recordsOK(t, r)
	if len(inv.Unreadable) != 1 || inv.Unreadable[0].Source != b ||
		!strings.Contains(inv.Unreadable[0].Reason, "git worktree prune") {
		t.Errorf("got %+v", inv.Unreadable)
	}
}

func TestRecordsOnlyMarkdownFiles(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	write(t, b, "docket/records/notes.txt")
	write(t, b, "docket/records/x.md/inner.md")
	inv := recordsOK(t, r)
	if len(inv.Uses) != 0 || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

func TestRecordsNestedFileOnBranch(t *testing.T) {
	r := newRepo(t)
	commitRef(r, "refs/heads/f", map[string]string{"docket/records/sub/2026-11-06-x.md": "x"})
	inv := recordsOK(t, r)
	if len(inv.Uses) != 0 || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

const issuePath = "docket/issues/2026-11-06-lint-timeout.md"

func issuesOK(t *testing.T, r *testutil.Repo) Inventory {
	t.Helper()
	inv, err := Issues(gitOf(r))
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

// Scenario: Issue only on a branch
func TestIssuesOnBranch(t *testing.T) {
	r := newRepo(t)
	c := commitRef(r, "refs/remotes/origin/fix-y", map[string]string{issuePath: "x"})
	inv := issuesOK(t, r)
	want := Use{Slug: "2026-11-06-lint-timeout", Path: issuePath, Source: "refs/remotes/origin/fix-y", Commit: c}
	if !slices.Contains(inv.Uses, want) || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

func TestIssuesInWorktree(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	write(t, b, issuePath)
	inv := issuesOK(t, r)
	want := Use{Slug: "2026-11-06-lint-timeout", Path: issuePath, Source: b}
	if !slices.Contains(inv.Uses, want) || len(inv.Unreadable) != 0 {
		t.Errorf("got %+v", inv)
	}
}

func TestIssuesLinkedFolder(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	outside := t.TempDir()
	write(t, outside, "2026-11-06-leak.md")
	if err := os.MkdirAll(filepath.Join(b, "docket"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(b, "docket/issues")); err != nil {
		t.Fatal(err)
	}
	inv := issuesOK(t, r)
	if len(inv.Uses) != 0 || len(inv.Unreadable) != 1 || inv.Unreadable[0].Source != b ||
		!strings.Contains(inv.Unreadable[0].Reason, "docket/issues is a symbolic link; Docket does not read through it") {
		t.Errorf("got %+v", inv)
	}
}

// Scenario: Records are not issues
func TestIssuesAndRecordsApart(t *testing.T) {
	r := newRepo(t)
	b := addWorktree(t, r, "b")
	write(t, b, recordPath)
	commitRef(r, "refs/heads/f", map[string]string{recordPath: "x"})
	if inv := issuesOK(t, r); len(inv.Uses) != 0 || len(inv.Unreadable) != 0 {
		t.Errorf("issues from records: %+v", inv)
	}
	r2 := newRepo(t)
	b2 := addWorktree(t, r2, "b")
	write(t, b2, issuePath)
	commitRef(r2, "refs/heads/f", map[string]string{issuePath: "x"})
	if inv := recordsOK(t, r2); len(inv.Uses) != 0 || len(inv.Unreadable) != 0 {
		t.Errorf("records from issues: %+v", inv)
	}
}
