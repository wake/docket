package archive

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/testutil"
)

const planFile = "openspec/changes/add-x/plan.md"

func planText(eol string) string {
	s := "# Plan\n\n### Task 1: One\n\n- [ ] **Step 1: a**\n- [ ] **Step 2: b**\n\n" +
		"### Task 2: Two\n\n- [ ] **Step 1: a**\n\n" +
		"### Task 3: Three\n\n- [ ] **Step 1: a**\n- [ ] **Step 2: b**\n- [ ] **Step 3: c**\n- [ ] **Step 4: d**\n\n" +
		"### Task 4: Four\n\n- [ ] **Step 1: a**\n\n" +
		"## Cancelled Tasks\n\n- Task 5: Five - not needed\n"
	return strings.ReplaceAll(s, "\n", eol)
}

// syncRepo is a repository whose change add-x is committed on top of a base
// commit that x-base names.
func syncRepo(t *testing.T, plan string) (*testutil.Repo, Env) {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	base := r.Commit("init", map[string]string{"README.md": "x\n"})
	r.Commit("add the change", map[string]string{
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\nx-base: " + base + "\n",
		"openspec/changes/add-x/plan.md":        plan,
	})
	return r, Env{Git: gitx.Git{Dir: r.Dir, Env: r.Env}, Root: r.Dir}
}

var workN int

func work(r *testutil.Repo, change string, tasks ...string) {
	workN++
	msg := "work\n\nChange: " + change
	for _, n := range tasks {
		msg += "\nTask: " + n
	}
	r.Commit(msg, map[string]string{"work-" + string(rune('a'+workN)) + ".txt": "x\n"})
}

func readPlan(t *testing.T, r *testutil.Repo) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.Dir, planFile))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func rules(rep SyncReport) []string {
	var out []string
	for _, f := range rep.Findings {
		out = append(out, f.Rule)
	}
	return out
}

func TestTaskWithCommitsTicked(t *testing.T) {
	r, e := syncRepo(t, planText("\n"))
	work(r, "add-x", "3")
	work(r, "add-x", "3")
	rep, err := Sync(e, "add-x", false)
	if err != nil || len(rep.Findings) != 0 {
		t.Fatalf("%v %+v", err, rep)
	}
	if !reflect.DeepEqual(rep.Ticked, []TaskTicks{{3, 4}}) {
		t.Fatalf("ticked %+v", rep.Ticked)
	}
	want := strings.Replace(planText("\n"), "- [ ] **Step 1: a**\n- [ ] **Step 2: b**\n- [ ] **Step 3: c**\n- [ ] **Step 4: d**", "- [x] **Step 1: a**\n- [x] **Step 2: b**\n- [x] **Step 3: c**\n- [x] **Step 4: d**", 1)
	if got := readPlan(t, r); got != want {
		t.Fatalf("plan:\n%s", got)
	}
}

func TestTaskWithoutCommitsLeftAlone(t *testing.T) {
	r, e := syncRepo(t, planText("\n"))
	work(r, "add-x", "3")
	if _, err := Sync(e, "add-x", false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readPlan(t, r), "### Task 4: Four\n\n- [ ] **Step 1: a**\n") {
		t.Fatal("Task 4 changed")
	}
}

func TestTwoTasksInOneCommit(t *testing.T) {
	r, e := syncRepo(t, planText("\n"))
	work(r, "add-x", "1", "2")
	rep, _ := Sync(e, "add-x", false)
	if !reflect.DeepEqual(rep.Ticked, []TaskTicks{{1, 2}, {2, 1}}) {
		t.Fatalf("ticked %+v", rep.Ticked)
	}
}

func TestNothingElseChanges(t *testing.T) {
	plan := "\xef\xbb\xbf" + planText("\r\n")
	r, e := syncRepo(t, plan)
	work(r, "add-x", "1")
	if _, err := Sync(e, "add-x", false); err != nil {
		t.Fatal(err)
	}
	got := readPlan(t, r)
	if len(got) != len(plan) {
		t.Fatalf("length changed")
	}
	diff := 0
	for i := range got {
		if got[i] != plan[i] {
			diff++
			if plan[i] != ' ' || got[i] != 'x' {
				t.Fatalf("byte %d: %q -> %q", i, plan[i], got[i])
			}
		}
	}
	if diff != 2 {
		t.Fatalf("%d bytes differ", diff)
	}
	if st := r.Git("status", "--porcelain"); !strings.HasSuffix(st, planFile) || strings.Count(st, "\n") != 0 {
		t.Fatalf("status %q", st)
	}
}

func TestTrailerForAnUnknownTask(t *testing.T) {
	r, e := syncRepo(t, planText("\n"))
	work(r, "add-x", "1")
	work(r, "add-x", "9")
	work(r, "add-x", "03")
	work(r, "add-x", "3a")
	work(r, "add-x", "5")
	rep, err := Sync(e, "add-x", false)
	if err != nil {
		t.Fatal(err)
	}
	if got := rules(rep); !reflect.DeepEqual(got, []string{"task-unknown", "task-unknown", "task-unknown", "task-unknown"}) {
		t.Fatalf("%+v", rep.Findings)
	}
	if !strings.Contains(rep.Findings[0].Message, "9") || rep.Findings[0].File != planFile {
		t.Fatalf("%+v", rep.Findings[0])
	}
	if !reflect.DeepEqual(rep.Ticked, []TaskTicks{{1, 2}}) {
		t.Fatalf("ticked %+v", rep.Ticked)
	}
}

func TestCheckFindsUntickedWork(t *testing.T) {
	r, e := syncRepo(t, planText("\n"))
	work(r, "add-x", "2")
	rep, err := Sync(e, "add-x", true)
	if err != nil {
		t.Fatal(err)
	}
	if readPlan(t, r) != planText("\n") {
		t.Fatal("plan written")
	}
	if len(rep.Findings) != 1 || rep.Findings[0].Rule != "plan-unsynced" || rep.Findings[0].Line != 8 ||
		rep.Findings[0].File != planFile || !strings.Contains(rep.Findings[0].Message, "dkt sync add-x") {
		t.Fatalf("%+v", rep.Findings)
	}
	if !reflect.DeepEqual(rep.Ticked, []TaskTicks{{2, 1}}) {
		t.Fatalf("ticked %+v", rep.Ticked)
	}
}

func TestCheckOnASyncedPlan(t *testing.T) {
	r, e := syncRepo(t, planText("\n"))
	work(r, "add-x", "2")
	if _, err := Sync(e, "add-x", false); err != nil {
		t.Fatal(err)
	}
	rep, _ := Sync(e, "add-x", true)
	if len(rep.Findings) != 0 || len(rep.Ticked) != 0 {
		t.Fatalf("%+v", rep)
	}
}

func TestSyncOtherChangeIgnored(t *testing.T) {
	r, e := syncRepo(t, planText("\n"))
	work(r, "add-y", "1")
	rep, _ := Sync(e, "add-x", false)
	if len(rep.Ticked) != 0 || len(rep.Findings) != 0 || readPlan(t, r) != planText("\n") {
		t.Fatalf("%+v", rep)
	}
}

func TestSyncLoadFindings(t *testing.T) {
	r, e := syncRepo(t, planText("\n"))
	rep, _ := Sync(e, "add-z", false)
	if got := rules(rep); !reflect.DeepEqual(got, []string{"change-missing"}) {
		t.Fatalf("%v", got)
	}
	cases := []struct{ meta, rule string }{
		{"schema: docket\n", "metadata-invalid"},
		{"schema: docket\nx-base: " + strings.Repeat("0", 40) + "\n", "base-missing"},
	}
	for _, c := range cases {
		r.Commit("meta", map[string]string{"openspec/changes/add-x/.openspec.yaml": c.meta})
		rep, _ := Sync(e, "add-x", false)
		if got := rules(rep); !reflect.DeepEqual(got, []string{c.rule}) {
			t.Fatalf("%s: %v", c.rule, got)
		}
	}
	r.Commit("meta", map[string]string{"openspec/changes/add-x/.openspec.yaml": "schema: docket\nx-base: " + r.Git("rev-parse", "HEAD~1") + "\n"})
	if err := os.Remove(filepath.Join(r.Dir, planFile)); err != nil {
		t.Fatal(err)
	}
	rep, _ = Sync(e, "add-x", false)
	if got := rules(rep); !reflect.DeepEqual(got, []string{"plan-missing"}) {
		t.Fatalf("%v", got)
	}
}

func TestSyncNoCommits(t *testing.T) {
	r := testutil.NewRepo(t)
	rep, err := Sync(Env{Git: gitx.Git{Dir: r.Dir, Env: r.Env}, Root: r.Dir}, "add-x", false)
	if err != nil || !reflect.DeepEqual(rules(rep), []string{"no-commits"}) {
		t.Fatalf("%v %+v", err, rep)
	}
}
