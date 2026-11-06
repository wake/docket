package scan

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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
	commitRef(r, "refs/heads/f", map[string]string{recordPath: "x"})
	inv := recordsOK(t, r)
	want := Use{Slug: "2026-11-06-fix-x", Path: recordPath, Source: "refs/heads/f"}
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
		!strings.Contains(inv.Unreadable[0].Reason, "docket/records is a symbolic link") {
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
