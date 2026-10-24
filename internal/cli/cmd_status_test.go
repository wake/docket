package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wake/docket/internal/testutil"
)

var statusNow = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func runStatus(t *testing.T, dir string, asJSON bool, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	a := append([]string{"status"}, args...)
	if asJSON {
		a = append(a, "--json")
	}
	code := Run(Options{Args: a, Dir: dir, Env: testutil.Env(t), Stdout: &so, Stderr: &se, Now: statusNow})
	return code, so.String(), se.String()
}

const statusPlan = "### Task 1: One\n\n- [ ] **Step 1: a**\n\n### Task 2: Two\n\n- [ ] **Step 1: b**\n"

const statusMeta = "schema: docket\ncreated: 2026-10-01\nx-kind: fix\nx-source: issue 7\n"

// statusChange adds an untracked change of three files in a new worktree.
func statusChange(t *testing.T, r *testutil.Repo, slug string) *testutil.Repo {
	t.Helper()
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt-"+slug), "br-"+slug)
	dir := "openspec/changes/" + slug + "/"
	w.Write(dir+".openspec.yaml", statusMeta)
	w.Write(dir+"specs/booking/spec.md", deltaFor("ADDED", "Cancel booking"))
	w.Write(dir+"plan.md", statusPlan)
	return w
}

// realPath resolves symbolic links, as git reports worktree paths.
func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func statusChanges(t *testing.T, so string) []any {
	t.Helper()
	res, _ := decode(t, so)["result"].(map[string]any)
	if res == nil {
		t.Fatalf("result is null\n%s", so)
	}
	return res["changes"].([]any)
}

func statusSlugs(t *testing.T, so string) []string {
	t.Helper()
	var out []string
	for _, c := range statusChanges(t, so) {
		out = append(out, c.(map[string]any)["slug"].(string))
	}
	return out
}

func TestStatusLists(t *testing.T) {
	r := cleanRepo(t)
	statusChange(t, r, "add-x")
	statusChange(t, r, "add-y")
	code, so, se := runStatus(t, r.Dir, true)
	if code != 0 || decode(t, so)["ok"] != true {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got := statusSlugs(t, so); len(got) != 2 || got[0] != "add-x" || got[1] != "add-y" {
		t.Fatalf("slugs = %v", got)
	}
	x := statusChanges(t, so)[0].(map[string]any)
	for _, k := range []string{"sources", "phase", "tasks", "progress"} {
		if x[k] == nil {
			t.Errorf("%s missing", k)
		}
	}
	if x["metadata"].(map[string]any)["kind"] != "fix" {
		t.Errorf("metadata = %v", x["metadata"])
	}
	specs := x["documents"].(map[string]any)["specs"].([]any)
	if len(specs) != 1 || specs[0] != "booking" {
		t.Errorf("specs = %v", specs)
	}
	compact := strings.ReplaceAll(so, " ", "")
	for _, bad := range []string{`"tasks":null`, `"sources":null`, `"specs":null`, `"commits":null`} {
		if strings.Contains(compact, bad) {
			t.Errorf("found %s", bad)
		}
	}
}

func TestStatusUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads everything")
	}
	r := cleanRepo(t)
	w := statusChange(t, r, "add-x")
	if err := os.Chmod(filepath.Join(w.Dir, "openspec/changes/add-x/plan.md"), 0); err != nil {
		t.Fatal(err)
	}
	code, so, se := runStatus(t, r.Dir, true)
	if code != 3 || len(statusChanges(t, so)) != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	u, _ := decode(t, so)["unreadable"].([]any)
	want := "worktree:" + realPath(t, w.Dir) + ":openspec/changes/add-x/plan.md"
	if len(u) != 1 || u[0].(map[string]any)["source"] != want {
		t.Fatalf("unreadable = %v, want %s", u, want)
	}
}

func TestStatusNoActiveChanges(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runStatus(t, r.Dir, true)
	if code != 0 || len(statusChanges(t, so)) != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestStatusText(t *testing.T) {
	r := cleanRepo(t)
	w := statusChange(t, r, "add-x")
	code, so, se := runStatus(t, r.Dir, false)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	head := strings.TrimSpace(w.Git("rev-parse", "HEAD"))[:7]
	for _, want := range []string{
		"add-x  ready  0/2 tasks  no commits  3 uncommitted\n",
		"  " + realPath(t, w.Dir) + "@" + head + "\n",
	} {
		if !strings.Contains(so, want) {
			t.Errorf("missing %q in:\n%s", want, so)
		}
	}
	if !strings.HasSuffix(so, "dkt status: ok\n") {
		t.Errorf("output:\n%s", so)
	}
}

func TestStatusUnsyncedWritesNothing(t *testing.T) {
	r := cleanRepo(t)
	w := statusChange(t, r, "add-x")
	w.Write("openspec/changes/add-x/.openspec.yaml", statusMeta+"x-base: "+r.Git("rev-parse", "HEAD")+"\n")
	w.Git("add", "-A")
	w.Git("commit", "-q", "-m", "work", "--trailer", "Change: add-x", "--trailer", "Task: 1")
	before, wbefore, refsBefore := snapshot(t, r.Dir), snapshot(t, w.Dir), r.Git("for-each-ref")
	code, so, se := runStatus(t, r.Dir, true)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	task := statusChanges(t, so)[0].(map[string]any)["tasks"].([]any)[0].(map[string]any)
	if task["unsynced"] != true {
		t.Errorf("task = %v", task)
	}
	if code, so, _ = runStatus(t, r.Dir, false); code != 0 {
		t.Errorf("text exit %d\n%s", code, so)
	}
	sameSnapshot(t, before, snapshot(t, r.Dir))
	sameSnapshot(t, wbefore, snapshot(t, w.Dir))
	if got := r.Git("for-each-ref"); got != refsBefore {
		t.Errorf("refs changed:\n%s\n%s", refsBefore, got)
	}
}

func TestStatusFocus(t *testing.T) {
	r := cleanRepo(t)
	statusChange(t, r, "add-x")
	statusChange(t, r, "add-y")
	for _, args := range [][]string{{"add-y"}, {"add-y", "add-y"}} {
		code, so, se := runStatus(t, r.Dir, true, args...)
		if got := statusSlugs(t, so); code != 0 || len(got) != 1 || got[0] != "add-y" {
			t.Fatalf("%v: exit %d slugs %v\n%s", args, code, got, se)
		}
	}
}

func TestStatusSlugInUnreadableSource(t *testing.T) {
	r := newRepo(t)
	statusChange(t, r, "add-y")
	gone := r.AddWorktree(filepath.Join(t.TempDir(), "wt-gone"), "gone")
	if err := os.RemoveAll(gone.Dir); err != nil {
		t.Fatal(err)
	}
	code, so, se := runStatus(t, r.Dir, true, "add-q")
	if code != 3 || hasRule(t, so, "change-not-active") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	u, _ := decode(t, so)["unreadable"].([]any)
	if len(u) != 1 || !strings.HasSuffix(u[0].(map[string]any)["source"].(string), "wt-gone") {
		t.Fatalf("unreadable = %v", u)
	}
}

func TestStatusSlugNotActive(t *testing.T) {
	r := newRepo(t)
	statusChange(t, r, "add-y")
	code, so, se := runStatus(t, r.Dir, true, "add-q")
	if code != 1 || !hasRule(t, so, "change-not-active") || !strings.Contains(so, "add-q") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	code, so, se = runStatus(t, r.Dir, true, "add-x")
	if code != 1 || !strings.Contains(so, "archived as 2025-01-02-add-x") {
		t.Fatalf("archived: exit %d\n%s\n%s", code, so, se)
	}
	code, so, se = runStatus(t, r.Dir, true, "add-q", "add-y")
	if got := statusSlugs(t, so); code != 1 || len(got) != 1 || got[0] != "add-y" {
		t.Fatalf("exit %d slugs %v\n%s\n%s", code, got, so, se)
	}
}

func TestStatusIdleDays(t *testing.T) {
	r := cleanRepo(t)
	w := statusChange(t, r, "add-x")
	w.Write("openspec/changes/add-x/.openspec.yaml", statusMeta+"x-base: "+r.Git("rev-parse", "HEAD")+"\n")
	w.Git("add", "-A")
	when := statusNow.AddDate(0, 0, -5).Format(time.RFC3339)
	env := w.Env
	w.Env = append(append([]string{}, env...), "GIT_COMMITTER_DATE="+when)
	w.Git("commit", "-q", "-m", "work", "--trailer", "Change: add-x", "--trailer", "Task: 1")
	w.Env = env
	code, so, se := runStatus(t, r.Dir, true)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	act, _ := statusChanges(t, so)[0].(map[string]any)["last_activity"].(map[string]any)
	if act == nil || act["idle_days"] != float64(5) {
		t.Fatalf("last_activity = %v", act)
	}
	code, so, se = runStatus(t, r.Dir, false)
	if code != 0 || !strings.Contains(so, "add-x  implementing  0/2 tasks  idle 5d  0 uncommitted\n") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestStatusSlugActiveAndArchived(t *testing.T) {
	r := newRepo(t)
	statusChange(t, r, "add-x")
	code, so, se := runStatus(t, r.Dir, true, "add-x")
	if code != 1 || !hasRule(t, so, "change-not-active") || len(statusChanges(t, so)) != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestStatusSlugInvalid(t *testing.T) {
	r := cleanRepo(t)
	statusChange(t, r, "add-y")
	code, so, se := runStatus(t, r.Dir, true, "Bad_Slug", "add-y")
	if code != 1 || !hasRule(t, so, "slug-invalid") || len(statusChanges(t, so)) != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestStatusUnreadableWorktree(t *testing.T) {
	r := cleanRepo(t)
	statusChange(t, r, "add-x")
	gone := r.AddWorktree(filepath.Join(t.TempDir(), "wt-gone"), "gone")
	if err := os.RemoveAll(gone.Dir); err != nil {
		t.Fatal(err)
	}
	code, so, se := runStatus(t, r.Dir, true)
	if code != 3 || len(statusChanges(t, so)) != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	u, _ := decode(t, so)["unreadable"].([]any)
	if len(u) != 1 || !strings.HasSuffix(u[0].(map[string]any)["source"].(string), "wt-gone") {
		t.Fatalf("unreadable = %v", u)
	}
}
