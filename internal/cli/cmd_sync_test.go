package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

type syncOut struct {
	Repo       *Repo
	Findings   []Finding
	Unreadable []Unreadable
	Result     *syncResult
}

const syncDir = "openspec/changes/add-x/"

// syncRepo is a repository with a docket change add-x and the given plan.md, committed.
func syncRepo(t *testing.T, plan string) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Write(syncDir+".openspec.yaml", "schema: docket\ncreated: 2026-01-01\n")
	r.Write(syncDir+"plan.md", plan)
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "plan")
	return r
}

func trail(r *testutil.Repo, values ...string) string {
	r.Git("commit", "-q", "--allow-empty", "-m", "work", "-m", "Change: "+strings.Join(values, "\nChange: "))
	return r.Git("rev-parse", "HEAD")
}

func doSync(t *testing.T, r *testutil.Repo, args ...string) (int, syncOut, string) {
	t.Helper()
	return doSyncIn(t, r.Dir, r.Env, args...)
}

func doSyncIn(t *testing.T, dir string, env []string, args ...string) (int, syncOut, string) {
	t.Helper()
	code, out, errs := runIn(t, dir, env, append([]string{"sync", "--json"}, args...)...)
	var m syncOut
	if out != "" {
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatalf("%q: %v", out, err)
		}
	}
	return code, m, out + errs
}

func readPlan(t *testing.T, r *testutil.Repo) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.Dir, syncDir, "plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func ruleList(fs []Finding) string {
	var s []string
	for _, f := range fs {
		s = append(s, f.Rule)
	}
	return strings.Join(s, ",")
}

const plan4 = `# Plan

### Task 1: One

- [ ] a
- [ ] b

### Task 2: Two

- [ ] c
- [x] d

### Task 3: Three

- [ ] e
- [ ] f
- [ ] g
- [ ] h

### Task 4: Four

- [ ] i

### Task 5: Five

- [ ] j

### Task 6: Six

- [x] k
`

func TestSyncTwoTasksOneCommit(t *testing.T) { // Two tasks in one commit
	r := syncRepo(t, plan4)
	sha := trail(r, "add-x/1,2")
	code, m, raw := doSync(t, r, "add-x")
	if code != 0 {
		t.Fatalf("%d %s", code, raw)
	}
	for _, i := range []int{0, 1} {
		if c := m.Result.Tasks[i].Commits; len(c) != 1 || c[0] != sha {
			t.Errorf("task %d commits %v", i+1, c)
		}
	}
}

func TestSyncLongerSlugIgnored(t *testing.T) { // Longer slug ignored
	r := syncRepo(t, plan4)
	trail(r, "add-x-2/1")
	code, m, raw := doSync(t, r, "add-x")
	if code != 0 || readPlan(t, r) != plan4 || m.Result == nil || len(m.Result.Tasks[0].Commits) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestSyncTrailerNotInLastParagraph(t *testing.T) { // Trailer outside the last paragraph
	r := syncRepo(t, plan4)
	r.Git("commit", "-q", "--allow-empty", "-m", "work", "-m", "Change: add-x/1", "-m", "more text")
	_, _, raw := doSync(t, r, "add-x")
	if readPlan(t, r) != plan4 {
		t.Fatal(raw)
	}
}

func TestSyncTicksTask(t *testing.T) { // Task with commits ticked; Task without commits left alone
	r := syncRepo(t, plan4)
	trail(r, "add-x/3")
	trail(r, "add-x/3")
	code, _, raw := doSync(t, r, "add-x")
	want := strings.NewReplacer("- [ ] e", "- [x] e", "- [ ] f", "- [x] f", "- [ ] g", "- [x] g", "- [ ] h", "- [x] h").Replace(plan4)
	if code != 0 || readPlan(t, r) != want {
		t.Fatalf("%d %s\n%s", code, raw, readPlan(t, r))
	}
}

func TestSyncCodeBlockCheckbox(t *testing.T) { // Checkbox in a code block
	p := "### Task 1: One\n\n- [ ] a\n\n```\n- [ ] example\n```\n"
	r := syncRepo(t, p)
	trail(r, "add-x/1")
	doSync(t, r, "add-x")
	if got := readPlan(t, r); got != "### Task 1: One\n\n- [x] a\n\n```\n- [ ] example\n```\n" {
		t.Fatal(got)
	}
}

func TestSyncCRLFNothingElseChanges(t *testing.T) { // Nothing else changes
	p := strings.ReplaceAll(plan4, "\n", "\r\n")
	r := syncRepo(t, p)
	trail(r, "add-x/1,3")
	code, _, raw := doSync(t, r, "add-x")
	if code != 0 {
		t.Fatal(raw)
	}
	got := readPlan(t, r)
	if len(got) != len(p) {
		t.Fatal("length changed")
	}
	n := 0
	for i := range p {
		if p[i] != got[i] {
			if p[i] != ' ' || got[i] != 'x' || p[i-1] != '[' {
				t.Fatalf("byte %d: %q -> %q", i, p[i], got[i])
			}
			n++
		}
	}
	if n != 6 {
		t.Fatalf("ticked %d", n)
	}
	if st := r.Git("status", "--porcelain"); !strings.HasSuffix(st, "M openspec/changes/add-x/plan.md") || strings.Contains(st, "\n") {
		t.Fatalf("status %q", st)
	}
}

func TestSyncBOMAndMode(t *testing.T) { // BOM kept, mode 0600 kept
	r := syncRepo(t, "\ufeff### Task 1: One\n\n- [ ] a\n")
	p := filepath.Join(r.Dir, syncDir, "plan.md")
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	trail(r, "add-x/1")
	doSync(t, r, "add-x")
	if got := readPlan(t, r); got != "\ufeff### Task 1: One\n\n- [x] a\n" {
		t.Fatalf("%q", got)
	}
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", fi.Mode())
	}
}

func TestSyncLinkedPlanRefused(t *testing.T) { // Linked plan refused
	for _, args := range [][]string{{"add-x"}, {"add-x", "--check"}} {
		r := syncRepo(t, plan4)
		target := filepath.Join(t.TempDir(), "real.md")
		if err := os.WriteFile(target, []byte(plan4), 0o644); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(r.Dir, syncDir, "plan.md")
		os.Remove(p)
		if err := os.Symlink(target, p); err != nil {
			t.Fatal(err)
		}
		trail(r, "add-x/1")
		code, m, raw := doSync(t, r, args...)
		b, _ := os.ReadFile(target)
		if code != 1 || ruleList(m.Findings) != "symlink-refused" || string(b) != plan4 || m.Result != nil {
			t.Fatalf("%v: %d %s", args, code, raw)
		}
	}
}

func TestSyncUnknownTask(t *testing.T) { // Trailer for an unknown task; cancelled number
	p := plan4 + "\n## Cancelled Tasks\n\n### Task 7: Gone\n\n- [ ] z\n"
	r := syncRepo(t, p)
	sha := trail(r, "add-x/9,1")
	trail(r, "add-x/7")
	code, m, raw := doSync(t, r, "add-x")
	if code != 1 || ruleList(m.Findings) != "trailer-unknown-task,trailer-unknown-task" {
		t.Fatalf("%d %s", code, raw)
	}
	f := m.Findings[0]
	if f.Message != "commit "+sha[:7]+" has the trailer Change: add-x/9,1, but plan.md has no Task 9" || f.File != syncDir+"plan.md" || f.Change != "add-x" {
		t.Fatalf("%+v", f)
	}
	if !strings.Contains(m.Findings[1].Message, "no Task 7") || !strings.Contains(readPlan(t, r), "- [x] a") {
		t.Fatal(raw)
	}
}

func TestSyncMalformed(t *testing.T) { // Malformed task list
	r := syncRepo(t, plan4)
	sha := trail(r, "add-x/two")
	code, m, raw := doSync(t, r, "add-x")
	if code != 1 || ruleList(m.Findings) != "trailer-malformed" ||
		m.Findings[0].Message != "commit "+sha[:7]+" has the trailer Change: add-x/two, whose task list is not positive numbers separated by commas" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestSyncSlugAlone(t *testing.T) { // value naming no task
	r := syncRepo(t, plan4)
	trail(r, "add-x")
	code, _, raw := doSync(t, r, "add-x")
	if code != 0 || readPlan(t, r) != plan4 {
		t.Fatal(raw)
	}
}

func TestSyncCheck(t *testing.T) { // Check finds unticked work / task without commits / ticked by hand / synced
	r := syncRepo(t, plan4)
	trail(r, "add-x/2")
	code, m, raw := doSync(t, r, "add-x", "--check")
	if code != 1 || readPlan(t, r) != plan4 {
		t.Fatalf("%d %s", code, raw)
	}
	var got []string
	for _, f := range m.Findings {
		got = append(got, f.Rule+":"+f.Message)
		if f.File != syncDir+"plan.md" || f.Change != "add-x" || f.Line == 0 {
			t.Errorf("%+v", f)
		}
	}
	joined := strings.Join(got, "\n")
	for _, w := range []string{
		"task-unticked:Task 2 has commits and 1 unticked steps; run dkt sync add-x",
		"task-no-commit:Task 5 has no commit carrying Change: add-x/5 and 1 unticked steps",
		"task-no-commit:Task 1 has no commit",
	} {
		if !strings.Contains(joined, w) {
			t.Errorf("missing %q in %s", w, joined)
		}
	}
	if strings.Contains(joined, "Task 6") {
		t.Error("Task 6 reported")
	}
	if m.Result == nil || !m.Result.Check || m.Result.Tasks[1].Ticked != 1 || m.Result.Tasks[1].Done != 1 {
		t.Fatalf("%+v", m.Result)
	}
	// Once synced and every task has commits, --check is clean.
	r2 := syncRepo(t, "### Task 1: One\n\n- [ ] a\n")
	trail(r2, "add-x/1")
	if code, _, raw := doSync(t, r2, "add-x"); code != 0 {
		t.Fatal(raw)
	}
	if code, _, raw := doSync(t, r2, "add-x", "--check"); code != 0 {
		t.Fatal(raw)
	}
}

func TestSyncCheckAllTicked(t *testing.T) { // commits, all steps ticked: no finding
	r := syncRepo(t, "### Task 1: One\n\n- [x] a\n")
	trail(r, "add-x/1")
	if code, m, raw := doSync(t, r, "add-x", "--check"); code != 0 || len(m.Findings) != 0 {
		t.Fatal(raw)
	}
}

func TestSyncUnbornHead(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write(syncDir+".openspec.yaml", "schema: docket\n")
	r.Write(syncDir+"plan.md", "### Task 1: One\n\n- [ ] a\n")
	code, m, raw := doSync(t, r, "add-x", "--check")
	if code != 1 || ruleList(m.Findings) != "task-no-commit" {
		t.Fatalf("%d %s", code, raw)
	}
	code, _, raw = doSync(t, r, "add-x")
	if code != 0 || readPlan(t, r) != "### Task 1: One\n\n- [ ] a\n" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestSyncChangeMissing(t *testing.T) { // No such change
	r := syncRepo(t, plan4)
	code, m, raw := doSync(t, r, "add-y")
	if code != 1 || ruleList(m.Findings) != "change-missing" || m.Findings[0].Message != "openspec/changes/add-y does not exist" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestSyncPlanMissing(t *testing.T) {
	r := syncRepo(t, plan4)
	os.Remove(filepath.Join(r.Dir, syncDir, "plan.md"))
	code, m, raw := doSync(t, r, "add-x")
	if code != 1 || ruleList(m.Findings) != "plan-missing" || m.Findings[0].Message != syncDir+"plan.md does not exist" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestSyncOtherSchema(t *testing.T) { // Sync refused for another schema
	r := syncRepo(t, plan4)
	r.Write(syncDir+".openspec.yaml", "schema: spec-driven\n")
	trail(r, "add-x/1")
	for _, args := range [][]string{{"add-x"}, {"add-x", "--check"}} {
		code, m, raw := doSync(t, r, args...)
		if code != 1 || ruleList(m.Findings) != "other-schema" || readPlan(t, r) != plan4 ||
			m.Findings[0].Message != syncDir+".openspec.yaml names the schema spec-driven; dkt sync handles docket changes only" {
			t.Fatalf("%d %s", code, raw)
		}
	}
}

func TestSyncUnreadable(t *testing.T) { // unreadable plan.md and .openspec.yaml
	if os.Geteuid() == 0 {
		t.Skip("root reads mode 000 files")
	}
	for _, name := range []string{"plan.md", ".openspec.yaml"} {
		r := syncRepo(t, plan4)
		trail(r, "add-x/1")
		p := filepath.Join(r.Dir, syncDir, name)
		if err := os.Chmod(p, 0); err != nil {
			t.Fatal(err)
		}
		code, m, raw := doSync(t, r, "add-x")
		os.Chmod(p, 0o644)
		if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != syncDir+name || readPlan(t, r) != plan4 {
			t.Fatalf("%s: %d %s", name, code, raw)
		}
	}
}

func TestSyncResult(t *testing.T) { // Result after ticking
	r := syncRepo(t, "### Task 1: One\n\n- [x] a\n- [ ] b\n\n### Task 2: Two\n\n- [ ] c\n")
	sha := trail(r, "add-x/1")
	code, m, raw := doSync(t, r, "add-x")
	got := m.Result.Tasks[0]
	if code != 0 || got.Number != 1 || got.Name != "One" || len(got.Commits) != 1 || got.Commits[0] != sha || got.Steps != 2 || got.Done != 2 || got.Ticked != 1 {
		t.Fatalf("%d %s", code, raw)
	}
	if !strings.Contains(raw, `"commits":[]`) {
		t.Fatal(raw)
	}
}

func TestSyncText(t *testing.T) {
	r := syncRepo(t, "### Task 1: One\n\n- [x] a\n- [ ] b\n\n### Task 2: Two\n\n- [ ] c\n")
	trail(r, "add-x/1")
	code, out, errs := runIn(t, r.Dir, r.Env, "sync", "add-x", "--check")
	if code != 1 || !strings.Contains(out, "Task 1: 1/2 steps, 1 commits, 1 to tick\n") || !strings.Contains(out, "Task 2: 0/1 steps, 0 commits\n") {
		t.Fatalf("%d %q %q", code, out, errs)
	}
	_, out, _ = runIn(t, r.Dir, r.Env, "sync", "add-x")
	if !strings.Contains(out, "Task 1: 2/2 steps, 1 commits, ticked 1\n") {
		t.Fatalf("%q", out)
	}
}

func TestSyncOutsideRepository(t *testing.T) { // Sync outside a repository
	code, m, raw := doSyncIn(t, t.TempDir(), testutil.Env(t), "add-x")
	if code != 4 || ruleList(m.Findings) != "no-repository" || m.Repo != nil {
		t.Fatalf("%d %s", code, raw)
	}
}
