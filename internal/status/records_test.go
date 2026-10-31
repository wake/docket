package status

import (
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
)

func TestTwoRecordsOnOneBranch(t *testing.T) {
	r, _ := baseRepo(t)
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "both")
	plan := "---\ntitle: T\n---\n\n## Tasks\n\n### Task 1: One\n\n- [ ] **Step 1: a**\n"
	a, b := "2026-11-01-fix-a", "2026-11-01-fix-b"
	w.Commit("add", map[string]string{"docket/records/" + a + ".md": plan, "docket/records/" + b + ".md": plan})
	ca := w.Commit("work a\n\nRecord: "+a+"\nTask: 1", map[string]string{"a.txt": "a"})
	cb := w.Commit("work b\n\nRecord: "+b+"\nTask: 1", map[string]string{"b.txt": "b"})
	g := gitx.Git{Dir: r.Dir, Env: r.Env}
	db, err := scan.ResolveDefaultBranch(g, "")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := scan.Scan(g, db)
	if err != nil {
		t.Fatal(err)
	}
	recs, total, _, err := LoadRecords(g, inv, RecordQuery{})
	if err != nil || total != 2 || len(recs) != 2 {
		t.Fatalf("recs = %v, total %d, err %v", recs, total, err)
	}
	for i, sha := range []string{ca, cb} {
		got := recs[i].Tasks[0].Commits
		if len(got) != 1 || got[0] != sha {
			t.Errorf("%s commits = %v, want [%s]", recs[i].Slug, got, sha)
		}
	}
}
