package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/conflicts"
	"github.com/wake/docket/internal/testutil"
)

type conflictsOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     *struct {
		Changes  []string
		Overlaps []conflicts.Overlap
	}
}

type cfEnv struct {
	r              *testutil.Repo
	cmd, log, answ string
}

func newCfEnv(t *testing.T) *cfEnv {
	t.Helper()
	cmd, log := testutil.FakeOpenSpec(t, "1.14.0")
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"README.md": "x"})
	return &cfEnv{r, cmd, log, t.TempDir()}
}

// answer sets what the fake OpenSpec says change slug touches: one delta per {capability, operation, requirement}.
func (e *cfEnv) answer(t *testing.T, slug string, deltas ...[3]string) {
	t.Helper()
	var ds []string
	for _, d := range deltas {
		ds = append(ds, fmt.Sprintf(`{"spec":%q,"operation":%q,"requirements":[{"name":%q,"scenarios":[]}]}`, d[0], d[1], d[2]))
	}
	testutil.OpenSpecAnswer(t, e.answ, 0, `{"deltas":[`+strings.Join(ds, ",")+`]}`,
		"show", slug, "--type", "change", "--json", "--deltas-only")
}

func spec(slug, text string) map[string]string {
	return map[string]string{"openspec/changes/" + slug + "/specs/booking/spec.md": text}
}

func (e *cfEnv) worktree(t *testing.T, branch string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "wt")
	e.r.Git("worktree", "add", "-q", "-b", branch, p)
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

// branch creates the local branch name at a commit holding files, leaving the repository on main.
func (e *cfEnv) branch(name string, files map[string]string) {
	e.r.Git("checkout", "-q", "-b", "tmp-cf", "main")
	c := e.r.Commit("c "+name, files)
	e.r.Git("update-ref", "refs/heads/"+name, c)
	e.r.Git("checkout", "-q", "main")
	e.r.Git("branch", "-q", "-D", "tmp-cf")
}

func put(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, c := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func (e *cfEnv) env(cmd string) []string {
	return append(append([]string{}, e.r.Env...), "DOCKET_OPENSPEC="+cmd, "FAKE_OPENSPEC_JSON="+e.answ)
}

func (e *cfEnv) run(t *testing.T, cmd string, args ...string) (int, conflictsOut, string, string) {
	t.Helper()
	code, out, errs := runIn(t, e.r.Dir, e.env(cmd), append([]string{"conflicts", "--json"}, args...)...)
	var m conflictsOut
	if out != "" {
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatalf("%q: %v", out, err)
		}
	}
	calls, _ := os.ReadFile(e.log)
	return code, m, out + errs, string(calls)
}

func (e *cfEnv) do(t *testing.T, args ...string) (int, conflictsOut, string, string) {
	t.Helper()
	return e.run(t, e.cmd, args...)
}

var modifyCancel = [3]string{"booking", "MODIFIED", "Cancel booking"}

func cfHasRule(m conflictsOut, rule string) bool {
	for _, f := range m.Findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func TestConflictsOtherWorktree(t *testing.T) { // Overlap with another worktree
	e := newCfEnv(t)
	wt := e.worktree(t, "other")
	put(t, e.r.Dir, spec("add-x", "x"))
	put(t, wt, spec("add-y", "y"))
	e.answer(t, "add-x", modifyCancel)
	e.answer(t, "add-y", modifyCancel)
	code, m, raw, _ := e.do(t)
	if code != 1 || m.Result == nil || len(m.Result.Overlaps) != 1 {
		t.Fatalf("%d %s", code, raw)
	}
	o := m.Result.Overlaps[0]
	if o.Capability != "booking" || o.Requirement != "Cancel booking" || len(o.Changes) != 2 || o.Changes[1].Slug != "add-y" || !slices.Equal(o.Changes[1].Sources, []string{wt}) {
		t.Errorf("%+v", o)
	}
	if len(m.Findings) != 1 || m.Findings[0].Rule != "requirement-overlap" ||
		m.Findings[0].Message != `booking / "Cancel booking": add-x (MODIFIED), add-y (MODIFIED)` {
		t.Errorf("%+v", m.Findings)
	}
}

func TestConflictsOneChange(t *testing.T) { // One active change, whatever else is around
	missing := filepath.Join(t.TempDir(), "nope")
	setups := map[string]func(e *cfEnv, t *testing.T) int{
		"only add-x": func(e *cfEnv, t *testing.T) int { put(t, e.r.Dir, spec("add-x", "x")); return 0 },
		"two copies": func(e *cfEnv, t *testing.T) int {
			put(t, e.r.Dir, spec("add-x", "x"))
			put(t, e.worktree(t, "other"), spec("add-x", "different"))
			e.branch("br", spec("add-x", "third"))
			return 0
		},
		"archived sibling": func(e *cfEnv, t *testing.T) int {
			put(t, e.r.Dir, spec("add-x", "x"))
			put(t, e.r.Dir, spec("archive/2026-11-01-add-q", "q"))
			return 0
		},
		"no change, removed worktree": func(e *cfEnv, t *testing.T) int {
			os.RemoveAll(e.worktree(t, "gone"))
			return 3
		},
	}
	for name, setup := range setups {
		t.Run(name, func(t *testing.T) {
			e := newCfEnv(t)
			want := setup(e, t)
			code, m, raw, _ := e.run(t, missing)
			if code != want || m.Result == nil || len(m.Result.Overlaps) != 0 || cfHasRule(m, "openspec-missing") {
				t.Fatalf("%d %s", code, raw)
			}
			if !strings.Contains(raw, `"overlaps":[]`) {
				t.Errorf("overlaps not an empty list: %s", raw)
			}
		})
	}
}

func TestConflictsChangesList(t *testing.T) {
	e := newCfEnv(t)
	put(t, e.worktree(t, "other"), spec("add-y", "y"))
	e.branch("br", spec("add-y", "y2"))
	put(t, e.r.Dir, spec("add-x", "x"))
	e.r.Commit("arch", spec("archive/2026-11-01-add-q", "q"))
	e.branch("old", spec("add-q", "q"))
	e.answer(t, "add-x", [3]string{"booking", "ADDED", "A"})
	e.answer(t, "add-y", [3]string{"booking", "ADDED", "B"})
	for _, args := range [][]string{nil, {"add-x"}} {
		code, m, raw, _ := e.do(t, args...)
		if code != 0 || m.Result == nil || !slices.Equal(m.Result.Changes, []string{"add-x", "add-y"}) {
			t.Fatalf("%v: %d %s", args, code, raw)
		}
	}
}

func TestConflictsTwoSlugs(t *testing.T) {
	e := newCfEnv(t)
	code, _, raw, calls := e.do(t, "add-x", "add-y")
	if code != 2 || calls != "" {
		t.Fatalf("%d %q %s", code, calls, raw)
	}
}

func TestConflictsFocus(t *testing.T) {
	e := newCfEnv(t)
	put(t, e.r.Dir, spec("add-x", "x"))
	put(t, e.r.Dir, spec("add-y", "y"))
	put(t, e.r.Dir, spec("add-z", "z"))
	e.answer(t, "add-x", modifyCancel)
	e.answer(t, "add-y", modifyCancel)
	e.answer(t, "add-z", [3]string{"booking", "ADDED", "Other"})
	if code, m, raw, _ := e.do(t, "add-z"); code != 0 || len(m.Result.Overlaps) != 0 {
		t.Fatalf("add-z: %d %s", code, raw)
	}
	if code, m, raw, _ := e.do(t, "add-x"); code != 1 || len(m.Result.Overlaps) != 1 {
		t.Fatalf("add-x: %d %s", code, raw)
	}
}

func TestConflictsSlugArchived(t *testing.T) { // Slug not active
	e := newCfEnv(t)
	// The same folder comes from the worktree, refs/heads/a and a remote-tracking ref, with another folder between.
	e.r.Commit("arch", spec("archive/2026-11-01-add-q", "q"))
	e.branch("a", spec("archive/2026-11-01-add-q", "q2"))
	e.branch("b", spec("archive/2026-11-02-add-q", "q"))
	e.r.Git("update-ref", "refs/remotes/origin/c", e.r.Git("rev-parse", "a"))
	code, m, raw, calls := e.do(t, "add-q")
	if code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != "change-missing" || m.Findings[0].Change != "add-q" ||
		strings.Count(m.Findings[0].Message, "openspec/changes/archive/2026-11-01-add-q") != 1 ||
		!strings.Contains(m.Findings[0].Message, "openspec/changes/archive/2026-11-02-add-q") || calls != "" ||
		!strings.Contains(raw, `"changes":[]`) {
		t.Fatalf("%d %q %s", code, calls, raw)
	}
	_, out, _ := runIn(t, e.r.Dir, e.env(e.cmd), "conflicts", "add-q")
	if strings.Contains(out, "Compared") {
		t.Errorf("summary for an inactive slug: %q", out)
	}
}

func TestConflictsSlugUnreadable(t *testing.T) { // Slug possibly in an unreadable worktree
	e := newCfEnv(t)
	wt := e.worktree(t, "gone")
	os.RemoveAll(wt)
	code, m, raw, _ := e.do(t, "add-q")
	if code != 3 || cfHasRule(m, "change-missing") || len(m.Unreadable) != 1 || m.Unreadable[0].Source != wt {
		t.Fatalf("%d %s", code, raw)
	}
	_, out, _ := runIn(t, e.r.Dir, e.env(e.cmd), "conflicts", "add-q")
	if strings.Contains(out, "Compared") {
		t.Errorf("summary for an inactive slug: %q", out)
	}
}

func TestConflictsSlugNowhere(t *testing.T) {
	e := newCfEnv(t)
	code, m, raw, _ := e.do(t, "add-q")
	if code != 1 || !cfHasRule(m, "change-missing") || !strings.Contains(raw, `"changes":[]`) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestConflictsNone(t *testing.T) {
	e := newCfEnv(t)
	put(t, e.r.Dir, spec("add-x", "x"))
	put(t, e.r.Dir, spec("add-y", "y"))
	e.answer(t, "add-x", [3]string{"booking", "ADDED", "A"})
	e.answer(t, "add-y", [3]string{"booking", "ADDED", "B"})
	code, _, raw, _ := e.do(t)
	if code != 0 || !strings.Contains(raw, `"findings":[]`) {
		t.Fatalf("%d %s", code, raw)
	}
	_, out, _ := runIn(t, e.r.Dir, e.env(e.cmd), "conflicts")
	if !strings.Contains(out, "Compared 2 changes; no requirement is touched by two of them.") {
		t.Errorf("%q", out)
	}
}

func TestConflictsText(t *testing.T) {
	e := newCfEnv(t)
	put(t, e.r.Dir, spec("add-x", "x"))
	put(t, e.r.Dir, spec("add-y", "y"))
	e.answer(t, "add-x", modifyCancel)
	e.answer(t, "add-y", modifyCancel)
	code, out, _ := runIn(t, e.r.Dir, e.env(e.cmd), "conflicts")
	if code != 1 || !strings.Contains(out, "booking / \"Cancel booking\"\n  add-x (MODIFIED): ") || !strings.Contains(out, "\n  add-y (MODIFIED): ") {
		t.Fatalf("%d %q", code, out)
	}
}

func TestConflictsUnreadableWorktree(t *testing.T) {
	e := newCfEnv(t)
	put(t, e.r.Dir, spec("add-x", "x"))
	put(t, e.r.Dir, spec("add-y", "y"))
	e.answer(t, "add-x", modifyCancel)
	e.answer(t, "add-y", modifyCancel)
	os.RemoveAll(e.worktree(t, "gone"))
	code, m, raw, _ := e.do(t)
	if code != 3 || len(m.Result.Overlaps) != 1 || len(m.Unreadable) != 1 {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestConflictsOpenSpecMissing(t *testing.T) {
	e := newCfEnv(t)
	put(t, e.r.Dir, spec("add-x", "x"))
	put(t, e.r.Dir, spec("add-y", "y"))
	code, m, raw, _ := e.run(t, filepath.Join(t.TempDir(), "nope"))
	if code != 4 || !cfHasRule(m, "openspec-missing") {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestConflictsOutsideRepository(t *testing.T) { // Conflicts outside a repository
	cmd, log := testutil.FakeOpenSpec(t, "1.14.0")
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+cmd)
	code, out, _ := runIn(t, t.TempDir(), env, "conflicts", "--json")
	var m conflictsOut
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("%q: %v", out, err)
	}
	calls, _ := os.ReadFile(log)
	if code != 4 || ruleList(m.Findings) != "no-repository" || len(calls) != 0 {
		t.Fatalf("%d %s %q", code, out, calls)
	}
}

func TestConflictsSummaryCount(t *testing.T) {
	e := newCfEnv(t)
	put(t, e.r.Dir, spec("add-x", "x"))
	if _, out, _ := runIn(t, e.r.Dir, e.env(e.cmd), "conflicts"); !strings.Contains(out, "Compared 1 change; no requirement") {
		t.Errorf("%q", out)
	}
	put(t, e.r.Dir, spec("add-y", "y"))
	e.answer(t, "add-x", [3]string{"booking", "ADDED", "A"})
	e.answer(t, "add-y", [3]string{"booking", "ADDED", "B"})
	if _, out, _ := runIn(t, e.r.Dir, e.env(e.cmd), "conflicts"); !strings.Contains(out, "Compared 2 changes; no requirement") {
		t.Errorf("%q", out)
	}
}
