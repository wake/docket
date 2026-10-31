package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func statusRecordText(title string) string {
	return "---\ntitle: " + title + "\nkind: fix\nsource: issue 1\n---\n\n## What\n\nx\n"
}

func statusRecords(t *testing.T, so string) []map[string]any {
	t.Helper()
	res, _ := decode(t, so)["result"].(map[string]any)
	if res == nil {
		t.Fatalf("result is null\n%s", so)
	}
	var out []map[string]any
	for _, r := range res["records"].([]any) {
		out = append(out, r.(map[string]any))
	}
	return out
}

func recordSlugs(t *testing.T, so string) []string {
	t.Helper()
	out := []string{}
	for _, r := range statusRecords(t, so) {
		out = append(out, r["slug"].(string))
	}
	return out
}

func TestOpenRecordsListed(t *testing.T) {
	r := cleanRepo(t)
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "wt-b")
	w.Write("docket/records/2026-11-01-fix-x.md", statusRecordText("Fix X"))
	y := r.AddWorktree(filepath.Join(t.TempDir(), "wt-y"), "fix-y")
	y.Commit("add y", map[string]string{"docket/records/2026-11-01-fix-y.md": statusRecordText("Fix Y")})
	r.Git("worktree", "remove", "--force", y.Dir)
	code, so, se := runStatus(t, r.Dir, true)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got := recordSlugs(t, so); strings.Join(got, " ") != "2026-11-01-fix-x 2026-11-01-fix-y" {
		t.Fatalf("slugs = %v", got)
	}
	recs := statusRecords(t, so)
	for _, rec := range recs {
		if rec["state"] != "open" || rec["kind"] != "fix" || rec["source"] != "issue 1" {
			t.Fatalf("record = %v", rec)
		}
	}
	if recs[0]["title"] != "Fix X" || recs[0]["sources"].([]any)[0].(map[string]any)["kind"] != "worktree" {
		t.Fatalf("x = %v", recs[0])
	}
	if recs[1]["sources"].([]any)[0].(map[string]any)["kind"] != "local-branch" {
		t.Fatalf("y = %v", recs[1])
	}
	_, hs, _ := runStatus(t, r.Dir, false)
	if !strings.Contains(hs, "2026-11-01-fix-x  \"Fix X\"  fix  open  0/0 tasks\n") {
		t.Fatalf("human:\n%s", hs)
	}
}

func TestMergedRecordsHidden(t *testing.T) {
	r := cleanRepo(t)
	r.Commit("add a", map[string]string{"docket/records/2026-10-31-fix-a.md": statusRecordText("A")})
	code, so, se := runStatus(t, r.Dir, true)
	res := decode(t, so)["result"].(map[string]any)
	if code != 0 || len(statusRecords(t, so)) != 0 || res["records_total"] != float64(0) {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestMergedRecordEditedOnBranchHidden(t *testing.T) {
	r := cleanRepo(t)
	r.Commit("add a", map[string]string{"docket/records/2026-10-31-fix-a.md": statusRecordText("A")})
	e := r.AddWorktree(filepath.Join(t.TempDir(), "wt-e"), "edit")
	e.Commit("edit a", map[string]string{"docket/records/2026-10-31-fix-a.md": statusRecordText("A2")})
	r.Git("worktree", "remove", "--force", e.Dir)
	code, so, se := runStatus(t, r.Dir, true)
	if code != 0 || len(statusRecords(t, so)) != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestRecordFrontMatterUnreadable(t *testing.T) {
	r := cleanRepo(t)
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "wt-b")
	w.Write("docket/records/2026-11-01-fix-x.md", "## What\n\nx\n")
	code, so, se := runStatus(t, r.Dir, true)
	recs := statusRecords(t, so)
	if code != 0 || len(recs) != 1 || recs[0]["title"] != nil || recs[0]["kind"] != nil || recs[0]["source"] != nil {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestUnreadableRecordFile(t *testing.T) {
	r := cleanRepo(t)
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "wt-b")
	w.Write("docket/records/2026-11-01-fix-x.md", statusRecordText("X"))
	p := filepath.Join(w.Dir, "docket/records/2026-11-01-fix-x.md")
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	code, so, se := runStatus(t, r.Dir, true)
	recs := statusRecords(t, so)
	if code != 3 || len(recs) != 1 || recs[0]["title"] != nil || !strings.Contains(so, "fix-x.md") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestRecordTasksFromTrailers(t *testing.T) {
	r := cleanRepo(t)
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt-x"), "fix-x")
	slug := "2026-11-01-fix-x"
	sha := w.Commit("work\n\nRecord: "+slug+"\nTask: 1", map[string]string{
		"docket/records/" + slug + ".md": statusRecordText("X") + "\n## Tasks\n\n### Task 1: One\n\n- [ ] **Step 1: a**\n\n### Task 2: Two\n\n- [ ] **Step 1: b**\n",
	})
	r.Git("worktree", "remove", "--force", w.Dir)
	code, so, se := runStatus(t, r.Dir, true)
	recs := statusRecords(t, so)
	if code != 0 || len(recs) != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	tasks := recs[0]["tasks"].([]any)
	t1, t2 := tasks[0].(map[string]any), tasks[1].(map[string]any)
	if t1["state"] != "started" || t1["unsynced"] != true || t1["commits"].([]any)[0] != sha || t2["state"] != "pending" {
		t.Fatalf("tasks = %v", tasks)
	}
	if p := recs[0]["progress"].(map[string]any); p["tasks_done"] != float64(0) || p["tasks_total"] != float64(2) || p["steps_ticked"] != float64(0) || p["steps_total"] != float64(2) {
		t.Fatalf("progress = %v", p)
	}
}

func TestNamedRecordMissing(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runStatus(t, r.Dir, true, "2026-11-01-fix-q")
	if code != 1 || !hasRule(t, so, "record-missing") || !strings.Contains(so, `"change":"2026-11-01-fix-q"`) {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestRecordInFocus(t *testing.T) {
	r := cleanRepo(t)
	statusChange(t, r, "add-x")
	statusChange(t, r, "add-y")
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "wt-b")
	w.Write("docket/records/2026-11-01-fix-x.md", statusRecordText("X"))
	code, so, se := runStatus(t, r.Dir, true, "add-y", "2026-11-01-fix-x")
	if code != 0 || strings.Join(statusSlugs(t, so), " ") != "add-y" || strings.Join(recordSlugs(t, so), " ") != "2026-11-01-fix-x" {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestNamedRecordBadSlug(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runStatus(t, r.Dir, true, "2026-11-01-")
	if code != 1 || !hasRule(t, so, "slug-invalid") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}
