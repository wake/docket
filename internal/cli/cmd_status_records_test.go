package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

// mergeRecord adds a record on a branch and merges it into main with --no-ff,
// the merge commit dated date (RFC 3339).
func mergeRecord(t *testing.T, r *testutil.Repo, slug, date string) string {
	t.Helper()
	r.Git("checkout", "-q", "-b", "b-"+slug)
	r.Commit("add "+slug, map[string]string{"docket/records/" + slug + ".md": statusRecordText(slug)})
	r.Git("checkout", "-q", "main")
	saved := r.Env
	r.Env = append(append([]string{}, saved...), "GIT_COMMITTER_DATE="+date)
	defer func() { r.Env = saved }()
	r.Git("merge", "-q", "--no-ff", "-m", "merge "+slug, "b-"+slug)
	return r.Git("rev-parse", "HEAD")
}

// removeObject deletes a loose object from the repository.
func removeObject(t *testing.T, r *testutil.Repo, sha string) {
	t.Helper()
	obj := filepath.Join(r.Dir, ".git", "objects", sha[:2], sha[2:])
	if err := os.Chmod(obj, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(obj); err != nil {
		t.Fatal(err)
	}
}

func TestMergedRecordsWithMergeAndTag(t *testing.T) {
	r := cleanRepo(t)
	merge := mergeRecord(t, r, "2026-10-30-fix-a", "2026-10-30T12:00:00Z")
	r.Git("tag", "v0.9.1")
	code, so, se := runStatus(t, r.Dir, true, "--merged")
	recs := statusRecords(t, so)
	if code != 0 || len(recs) != 1 || recs[0]["state"] != "merged" || recs[0]["tag"] != "v0.9.1" {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	m := recs[0]["merge"].(map[string]any)
	if m["commit"] != merge || m["time"] != "2026-10-30T12:00:00Z" {
		t.Fatalf("merge = %v", m)
	}
	_, hs, _ := runStatus(t, r.Dir, false, "--merged")
	if !strings.Contains(hs, "  merged 2026-10-30  v0.9.1  0/0 tasks\n") {
		t.Fatalf("human:\n%s", hs)
	}
}

func TestMergedSince(t *testing.T) {
	r := cleanRepo(t)
	mergeRecord(t, r, "2026-10-30-fix-a", "2026-10-30T12:00:00Z")
	r.Git("tag", "v0.9.1")
	mergeRecord(t, r, "2026-10-31-fix-b", "2026-10-31T12:00:00Z")
	code, so, se := runStatus(t, r.Dir, true, "--merged", "--since", "v0.9.1")
	res := decode(t, so)["result"].(map[string]any)
	if got := recordSlugs(t, so); code != 0 || strings.Join(got, " ") != "2026-10-31-fix-b" || res["records_total"] != float64(1) {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func mergeFive(t *testing.T, r *testutil.Repo) {
	t.Helper()
	// Merge order, slug order and time order all differ.
	for _, m := range [][2]string{{"1", "03"}, {"2", "01"}, {"3", "05"}, {"4", "02"}, {"5", "04"}} {
		mergeRecord(t, r, "2026-10-30-fix-"+m[0], "2026-10-"+m[1]+"T12:00:00Z")
	}
}

func TestLimitReportsTotal(t *testing.T) {
	r := cleanRepo(t)
	mergeFive(t, r)
	code, so, se := runStatus(t, r.Dir, true, "--merged", "--limit", "2")
	res := decode(t, so)["result"].(map[string]any)
	if got := recordSlugs(t, so); code != 0 || strings.Join(got, " ") != "2026-10-30-fix-3 2026-10-30-fix-5" || res["records_total"] != float64(5) {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	_, hs, _ := runStatus(t, r.Dir, false, "--merged", "--limit", "2")
	if !strings.Contains(hs, "\n3 more merged records not listed; raise --limit\n") {
		t.Fatalf("human:\n%s", hs)
	}
	_, so, _ = runStatus(t, r.Dir, true, "--merged")
	if got := recordSlugs(t, so); strings.Join(got, " ") != "2026-10-30-fix-3 2026-10-30-fix-5 2026-10-30-fix-1 2026-10-30-fix-4 2026-10-30-fix-2" {
		t.Fatalf("order = %v", got)
	}
}

func TestRecordsTotalOpenAndMerged(t *testing.T) {
	r := cleanRepo(t)
	for i, d := range []string{"01", "02", "03"} {
		mergeRecord(t, r, "2026-10-30-fix-"+string(rune('a'+i)), "2026-10-"+d+"T12:00:00Z")
	}
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "wt-b")
	w.Write("docket/records/2026-11-01-fix-x.md", statusRecordText("X"))
	code, so, se := runStatus(t, r.Dir, true, "--merged", "--limit", "1")
	res := decode(t, so)["result"].(map[string]any)
	if got := recordSlugs(t, so); code != 0 || strings.Join(got, " ") != "2026-11-01-fix-x 2026-10-30-fix-c" || res["records_total"] != float64(4) {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestMergedSameCommitBySlug(t *testing.T) {
	r := cleanRepo(t)
	r.Git("checkout", "-q", "-b", "both")
	r.Commit("add both", map[string]string{
		"docket/records/2026-10-30-fix-b.md": statusRecordText("B"),
		"docket/records/2026-10-30-fix-a.md": statusRecordText("A"),
	})
	r.Git("checkout", "-q", "main")
	r.Git("merge", "-q", "--ff-only", "both")
	_, so, _ := runStatus(t, r.Dir, true, "--merged")
	if got := recordSlugs(t, so); strings.Join(got, " ") != "2026-10-30-fix-a 2026-10-30-fix-b" {
		t.Fatalf("order = %v", got)
	}
	_, so, _ = runStatus(t, r.Dir, true, "--merged", "--limit", "1")
	if got := recordSlugs(t, so); strings.Join(got, " ") != "2026-10-30-fix-a" {
		t.Fatalf("limit = %v", got)
	}
}

func TestFilterFlagsNeedMerged(t *testing.T) {
	r := cleanRepo(t)
	mergeRecord(t, r, "2026-10-30-fix-a", "2026-10-30T12:00:00Z")
	for _, args := range [][]string{
		{"--limit", "2"}, {"--since", "main"}, {"--merged", "2026-10-30-fix-a"},
		{"--merged", "--limit", "0"}, {"--merged", "--limit", "x"}, {"--merged", "--since", "nosuch"},
		{"--limit", "2", "2026-10-30-fix-a"},
	} {
		if code, so, se := runStatus(t, r.Dir, true, args...); code != 2 {
			t.Errorf("%v: exit %d\n%s\n%s", args, code, so, se)
		}
	}
}

func TestNamedRecordListed(t *testing.T) {
	r := cleanRepo(t)
	merge := mergeRecord(t, r, "2026-10-30-fix-a", "2026-10-30T12:00:00Z")
	r.Git("tag", "v0.9.1")
	code, so, se := runStatus(t, r.Dir, true, "2026-10-30-fix-a")
	recs := statusRecords(t, so)
	if code != 0 || len(recs) != 1 || recs[0]["state"] != "merged" || recs[0]["tag"] != "v0.9.1" ||
		recs[0]["merge"].(map[string]any)["commit"] != merge {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func lostTag(t *testing.T, r *testutil.Repo) {
	t.Helper()
	r.Commit("only for the tag", map[string]string{"lost.txt": "lost\n"})
	r.Git("tag", "lost")
	tree := r.Git("rev-parse", "lost^{tree}")
	r.Git("reset", "-q", "--hard", "HEAD~1")
	removeObject(t, r, tree)
}

func TestUnreadableTag(t *testing.T) {
	r := cleanRepo(t)
	mergeRecord(t, r, "2026-10-30-fix-a", "2026-10-30T12:00:00Z")
	lostTag(t, r)
	code, so, se := runStatus(t, r.Dir, true, "--merged")
	if code != 3 || !strings.Contains(so, "refs/tags/lost") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if code, so, se := runStatus(t, r.Dir, true); code != 0 {
		t.Fatalf("without --merged: exit %d\n%s\n%s", code, so, se)
	}
}

func TestTagListingFails(t *testing.T) {
	r := cleanRepo(t)
	mergeRecord(t, r, "2026-10-30-fix-a", "2026-10-30T12:00:00Z")
	sha := r.Commit("only for the tag", map[string]string{"lost.txt": "lost\n"})
	r.Git("tag", "lost")
	r.Git("reset", "-q", "--hard", "HEAD~1")
	removeObject(t, r, sha)
	code, so, se := runStatus(t, r.Dir, true, "--merged")
	recs := statusRecords(t, so)
	unread := decode(t, so)["unreadable"].([]any)
	if code != 3 || len(recs) != 1 || recs[0]["tag"] != nil || len(unread) != 1 || !strings.Contains(so, `"refs/tags"`) {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

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
