package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/testutil"
)

const (
	issA    = "2026-11-01-a"
	issAF   = "docket/issues/" + issA + ".md"
	issText = "---\ntitle: A\nsource: x\n---\n\n## Current\n\nx\n"
)

func recordNaming(source string) string {
	return "---\ntitle: R\nkind: fix\nsource: " + source + "\n---\n\n## What\n\nx\n"
}

func loadIssues(t *testing.T, r *testutil.Repo) ([]Issue, []scan.Unreadable) {
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
	issues, unread, err := LoadIssues(g, inv)
	if err != nil {
		t.Fatal(err)
	}
	return issues, unread
}

// only checks that exactly one issue is listed, in the state and with the handlers.
func only(t *testing.T, issues []Issue, state string, handlers ...Handler) Issue {
	t.Helper()
	if len(issues) != 1 || issues[0].State != state || len(issues[0].Handlers) != len(handlers) {
		t.Fatalf("issues = %+v, want one %s with %v", issues, state, handlers)
	}
	for i, h := range handlers {
		if issues[0].Handlers[i] != h {
			t.Fatalf("handlers = %+v, want %v", issues[0].Handlers, handlers)
		}
	}
	return issues[0]
}

func TestPendingIssue(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("issue", map[string]string{issAF: issText})
	is, _ := loadIssues(t, r)
	got := only(t, is, Pending)
	if got.Title == nil || *got.Title != "A" || got.Source == nil || *got.Source != "x" || got.Handlers == nil {
		t.Fatalf("issue = %+v", got)
	}
}

func TestHandlerOnABranch(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("issue", map[string]string{issAF: issText})
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "fix")
	w.Commit("record", map[string]string{"docket/records/2026-11-02-fix-a.md": recordNaming(issAF)})
	is, _ := loadIssues(t, r)
	only(t, is, InProgress, Handler{Kind: "record", Slug: "2026-11-02-fix-a"})
}

func TestHandlerMerged(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("issue and record", map[string]string{issAF: issText, "docket/records/2026-11-02-fix-a.md": recordNaming(issAF)})
	is, _ := loadIssues(t, r)
	if len(is) != 0 {
		t.Fatalf("handled issue listed: %+v", is)
	}
}

func TestMergedChangeNotYetArchived(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("issue and change", map[string]string{issAF: issText,
		"openspec/changes/add-x/docket.yaml": "title: X\nsource: " + issAF + "\n"})
	is, _ := loadIssues(t, r)
	if len(is) != 0 {
		t.Fatalf("handled issue listed: %+v", is)
	}
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "other")
	w.Commit("second", map[string]string{"openspec/changes/add-y/docket.yaml": "title: Y\nsource: " + issAF + "\n"})
	is, _ = loadIssues(t, r)
	only(t, is, InProgress, Handler{Kind: "change", Slug: "add-x", Merged: true}, Handler{Kind: "change", Slug: "add-y"})
}

func TestMergedRecordEditedElsewhere(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("issue and record", map[string]string{issAF: issText, "docket/records/2026-11-02-fix-a.md": recordNaming("other")})
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "edit")
	w.Commit("edit", map[string]string{"docket/records/2026-11-02-fix-a.md": recordNaming(issAF)})
	is, _ := loadIssues(t, r)
	only(t, is, Pending)
}

func TestHandlerSourceMatching(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("issue", map[string]string{issAF: issText,
		"docket/records/2026-11-02-fix-a.md": recordNaming("see " + issAF + " for details"),
		"docket/records/2026-11-02-fix-b.md": recordNaming("\"  " + issAF + "  \""),
	})
	if is, _ := loadIssues(t, r); len(is) != 0 {
		t.Fatalf("padded source should hand the issue to fix-b: %+v", is)
	}
	r2, _ := baseRepo(t)
	r2.Commit("issue", map[string]string{issAF: issText, "docket/records/2026-11-02-fix-a.md": recordNaming("see " + issAF + " for details")})
	is, _ := loadIssues(t, r2)
	only(t, is, Pending)
}

func TestDroppedIssueHidden(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("issue", map[string]string{issAF: "---\ntitle: A\nsource: x\ndropped: true\n---\n\nx\n"})
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "fix")
	w.Commit("record", map[string]string{"docket/records/2026-11-02-fix-a.md": recordNaming(issAF)})
	if is, _ := loadIssues(t, r); len(is) != 0 {
		t.Fatalf("dropped issue listed: %+v", is)
	}
}

func TestMovedIssueHidden(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("issue", map[string]string{issAF: "---\ntitle: A\nsource: x\nidea: docket/ideas/a.md\n---\n\nx\n"})
	if is, _ := loadIssues(t, r); len(is) != 0 {
		t.Fatalf("moved issue listed: %+v", is)
	}
}

func TestArchivedLegacyChangeHandles(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("issue and archive", map[string]string{issAF: issText,
		"openspec/changes/archive/2026-11-02-add-x/.openspec.yaml": "schema: docket\ncreated: 2026-11-01\nx-source: " + issAF + "\n"})
	if is, _ := loadIssues(t, r); len(is) != 0 {
		t.Fatalf("handled issue listed: %+v", is)
	}
}

func TestIssueOnlyInWorktree(t *testing.T) {
	r, _ := baseRepo(t)
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "b")
	w.Write(issAF, issText)
	is, _ := loadIssues(t, r)
	got := only(t, is, Pending)
	if len(got.Sources) != 1 || got.Sources[0].Kind != scan.KindWorktree || !strings.HasSuffix(got.Sources[0].Name, "/wt") {
		t.Fatalf("sources = %+v", got.Sources)
	}
}

func TestIssueWithoutFrontMatterAndUnreadable(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("issues", map[string]string{issAF: "no front matter\n"})
	is, _ := loadIssues(t, r)
	got := only(t, is, Pending)
	if got.Title != nil || got.Source != nil {
		t.Fatalf("issue = %+v", got)
	}
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "b")
	bad := "docket/issues/2026-11-02-b.md"
	w.Write(bad, issText)
	if err := os.Chmod(filepath.Join(w.Dir, bad), 0); err != nil {
		t.Fatal(err)
	}
	is, unread := loadIssues(t, r)
	if len(is) != 2 || len(unread) != 1 || !strings.HasSuffix(unread[0].Path, "2026-11-02-b.md") {
		t.Fatalf("issues = %+v, unread = %+v", is, unread)
	}
}

func TestChangeCopiesElsewhereIgnoredWhenTipHolds(t *testing.T) {
	other := "title: X\nsource: other\n"
	named := "title: X\nsource: " + issAF + "\n"
	for name, c := range map[string][2]string{
		"tip archived, branch active": {"openspec/changes/archive/2026-11-01-add-x/docket.yaml", "openspec/changes/add-x/docket.yaml"},
		"tip active, branch archived": {"openspec/changes/add-x/docket.yaml", "openspec/changes/archive/2026-11-01-add-x/docket.yaml"},
	} {
		r, _ := baseRepo(t)
		r.Commit("issue and tip copy", map[string]string{issAF: issText, c[0]: other})
		w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "elsewhere")
		w.Commit("other copy", map[string]string{c[1]: named})
		is, _ := loadIssues(t, r)
		if len(is) != 1 || is[0].State != Pending || len(is[0].Handlers) != 0 {
			t.Errorf("%s: issues = %+v", name, is)
		}
	}
}

func TestChangeActiveAndArchivedIsOneHandler(t *testing.T) {
	meta := "title: X\nsource: " + issAF + "\n"
	r, _ := baseRepo(t)
	r.Commit("issue and archived", map[string]string{issAF: issText, "openspec/changes/archive/2026-11-01-add-x/docket.yaml": meta})
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "again")
	w.Commit("active", map[string]string{"openspec/changes/add-x/docket.yaml": meta})
	if is, _ := loadIssues(t, r); len(is) != 0 {
		t.Fatalf("handled issue listed: %+v", is)
	}
	r2, _ := baseRepo(t)
	r2.Commit("issue", map[string]string{issAF: issText})
	w2 := r2.AddWorktree(filepath.Join(t.TempDir(), "wt"), "both")
	w2.Commit("both", map[string]string{"openspec/changes/add-x/docket.yaml": meta, "openspec/changes/archive/2026-11-01-add-x/docket.yaml": meta})
	is, _ := loadIssues(t, r2)
	only(t, is, InProgress, Handler{Kind: "change", Slug: "add-x"})
}

func TestHandlerWithANote(t *testing.T) {
	r, _ := baseRepo(t)
	r.Commit("issue", map[string]string{issAF: issText,
		"docket/records/2026-11-02-fix-a.md": recordNaming("\"" + issAF + ", found while fixing y\""),
	})
	if is, _ := loadIssues(t, r); len(is) != 0 {
		t.Fatalf("source with a note should hand the issue over: %+v", is)
	}
}
