package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

const syncPlan = "# Plan\n\n### Task 1: One\n\n- [ ] **Step 1: a**\n- [ ] **Step 2: b**\n- [ ] **Step 3: c**\n- [ ] **Step 4: d**\n"

func runSync(t *testing.T, r *testutil.Repo, env []string, args ...string) (int, string, string) {
	t.Helper()
	if env == nil {
		env = r.Env
	}
	var so, se bytes.Buffer
	code := Run(Options{Args: append([]string{"sync"}, args...), Dir: r.Dir, Env: env, Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

func syncFixture(t *testing.T) *testutil.Repo {
	t.Helper()
	r := cleanRepo(t)
	base := r.Git("rev-parse", "HEAD")
	r.Commit("add the change", map[string]string{
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\nx-base: " + base + "\n",
		"openspec/changes/add-x/plan.md":        syncPlan,
	})
	r.Commit("work\n\nChange: add-x\nTask: 1", map[string]string{"a.txt": "a\n"})
	return r
}

func TestSyncCommand(t *testing.T) {
	r := syncFixture(t)
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC=/nonexistent/openspec")
	code, so, se := runSync(t, r, env, "add-x", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	tk := res["ticked"].([]any)
	if res["change"] != "add-x" || res["check"] != false || len(tk) != 1 ||
		tk[0].(map[string]any)["task"] != float64(1) || tk[0].(map[string]any)["steps"] != float64(4) {
		t.Fatalf("result %v", res)
	}
	b, _ := os.ReadFile(filepath.Join(r.Dir, "openspec/changes/add-x/plan.md"))
	if strings.Contains(string(b), "[ ]") {
		t.Fatalf("not ticked:\n%s", b)
	}
	code, so, _ = runSync(t, r, nil, "add-x", "--json")
	if code != 0 || decode(t, so)["result"].(map[string]any)["ticked"] == nil {
		t.Fatalf("second run: %d %s", code, so)
	}
}

func TestSyncCheckExitsOne(t *testing.T) {
	r := syncFixture(t)
	code, so, se := runSync(t, r, nil, "add-x", "--check", "--json")
	if code != 1 || !hasRule(t, so, "plan-unsynced") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	b, _ := os.ReadFile(filepath.Join(r.Dir, "openspec/changes/add-x/plan.md"))
	if string(b) != syncPlan {
		t.Fatal("plan written")
	}
}

func TestSyncNeedsOneSlug(t *testing.T) {
	r := syncFixture(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{}, "sync needs a slug"},
		{[]string{"add-x", "add-y"}, "sync takes one slug"},
	} {
		code, so, se := runSync(t, r, nil, tc.args...)
		if code != 2 || !strings.Contains(se, tc.want) {
			t.Fatalf("%v: exit %d, want 2 with %q\n%s\n%s", tc.args, code, tc.want, so, se)
		}
	}
}

func TestSyncSymlinkedPlanRefused(t *testing.T) {
	r := syncFixture(t)
	p := filepath.Join(r.Dir, "openspec/changes/add-x/plan.md")
	if err := os.Rename(p, filepath.Join(r.Dir, "real-plan.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../../real-plan.md", p); err != nil {
		t.Fatal(err)
	}
	code, so, se := runSync(t, r, nil, "add-x", "--json")
	if code != 1 || !hasRule(t, so, "symlink-refused") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	b, _ := os.ReadFile(filepath.Join(r.Dir, "real-plan.md"))
	if string(b) != syncPlan {
		t.Fatal("link target written")
	}
}

const recSlug = "2026-11-01-fix-x"
const recFile = "docket/records/" + recSlug + ".md"
const recBody = "# Fix x\n\n## Tasks\n\n### Task 1: A\n\n- [ ] **Step 1: a**\n\n### Task 2: B\n\n- [ ] **Step 1: b**\n- [ ] **Step 2: c**\n"

// recordRepo has the record committed on main, then a branch fix-x.
func recordRepo(t *testing.T) *testutil.Repo {
	t.Helper()
	r := cleanRepo(t)
	r.Commit("add record", map[string]string{recFile: recBody})
	r.Git("checkout", "-q", "-b", "fix-x")
	return r
}

func recordText(t *testing.T, r *testutil.Repo) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.Dir, recFile))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRecordTasksTicked(t *testing.T) {
	r := recordRepo(t)
	r.Commit("work\n\nRecord: "+recSlug+"\nTask: 2", map[string]string{"a.txt": "a\n"})
	code, so, se := runSync(t, r, nil, recSlug, "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	if res["record"] != recSlug || res["change"] != nil {
		t.Fatalf("result %v", res)
	}
	want := strings.Replace(strings.Replace(recBody, "- [ ] **Step 1: b", "- [x] **Step 1: b", 1), "- [ ] **Step 2: c", "- [x] **Step 2: c", 1)
	if got := recordText(t, r); got != want {
		t.Fatalf("record:\n%s", got)
	}
	if st := r.Git("status", "--porcelain"); !strings.Contains(st, recFile) || strings.Count(st, "\n") != 0 {
		t.Fatalf("status %q", st)
	}
}

func TestRecordSyncIgnoresMergedCommits(t *testing.T) {
	r := cleanRepo(t)
	r.Commit("add record", map[string]string{recFile: recBody})
	r.Commit("old\n\nRecord: "+recSlug+"\nTask: 1", map[string]string{"m.txt": "m\n"})
	r.Git("checkout", "-q", "-b", "fix-x")
	if code, so, se := runSync(t, r, nil, recSlug); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if recordText(t, r) != recBody {
		t.Fatal("record changed")
	}
}

func TestRecordSyncCheck(t *testing.T) {
	r := recordRepo(t)
	r.Commit("work\n\nRecord: "+recSlug+"\nTask: 2", map[string]string{"a.txt": "a\n"})
	code, so, _ := runSync(t, r, nil, recSlug, "--check", "--json")
	if code != 1 || !hasRule(t, so, "plan-unsynced") || !strings.Contains(so, "Task 2 has commits and 2 unticked steps") {
		t.Fatalf("exit %d\n%s", code, so)
	}
	if recordText(t, r) != recBody {
		t.Fatal("record written")
	}
}

func TestRecordSyncUnknownTask(t *testing.T) {
	r := recordRepo(t)
	sha := r.Commit("bad\n\nRecord: "+recSlug+"\nTask: 9", map[string]string{"a.txt": "a\n"})
	r.Commit("work\n\nRecord: "+recSlug+"\nTask: 2", map[string]string{"b.txt": "b\n"})
	code, so, _ := runSync(t, r, nil, recSlug, "--json")
	if code != 1 || !hasRule(t, so, "task-unknown") || !strings.Contains(so, sha[:7]) {
		t.Fatalf("exit %d\n%s", code, so)
	}
	if !strings.Contains(recordText(t, r), "- [x] **Step 2: c") {
		t.Fatal("Task 2 not ticked")
	}
}

func TestRecordSyncNeedsDefaultBranch(t *testing.T) {
	r := recordRepo(t)
	r.Git("branch", "-m", "main", "trunk")
	if code, so, se := runSync(t, r, nil, recSlug); code != 4 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if recordText(t, r) != recBody {
		t.Fatal("record changed")
	}
}

func TestRecordSyncThroughLink(t *testing.T) {
	r := cleanRepo(t)
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, recSlug+".md"), []byte(recBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(r.Dir, "docket"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(out, filepath.Join(r.Dir, "docket/records")); err != nil {
		t.Fatal(err)
	}
	r.Git("checkout", "-q", "-b", "fix-x")
	r.Commit("work\n\nRecord: "+recSlug+"\nTask: 2", map[string]string{"a.txt": "a\n"})
	code, so, _ := runSync(t, r, nil, recSlug, "--json")
	if code != 1 || !hasRule(t, so, "symlink-refused") || !strings.Contains(so, "docket/records") {
		t.Fatalf("exit %d\n%s", code, so)
	}
	if b, _ := os.ReadFile(filepath.Join(out, recSlug+".md")); string(b) != recBody {
		t.Fatal("outside file written")
	}
}

func TestRecordSyncMissing(t *testing.T) {
	r := cleanRepo(t)
	code, so, _ := runSync(t, r, nil, recSlug, "--json")
	if code != 1 || !hasRule(t, so, "record-missing") {
		t.Fatalf("exit %d\n%s", code, so)
	}
}

func TestRecordSyncBadSlug(t *testing.T) {
	r := cleanRepo(t)
	code, so, _ := runSync(t, r, nil, "2026-11-01-", "--json")
	if code != 1 || !hasRule(t, so, "slug-invalid") {
		t.Fatalf("exit %d\n%s", code, so)
	}
}

func TestRecordSyncBadSlugWithoutDefaultBranch(t *testing.T) {
	r := cleanRepo(t)
	r.Git("branch", "-m", "main", "trunk")
	code, so, _ := runSync(t, r, nil, "2026-11-01-", "--json")
	if code != 1 || !hasRule(t, so, "slug-invalid") {
		t.Fatalf("exit %d\n%s", code, so)
	}
}
