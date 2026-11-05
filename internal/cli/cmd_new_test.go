package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/testutil"
)

type newOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     struct {
		Slug, Requested, Path, Title, Kind, Source string
		Used                                       []struct {
			Path    string
			Sources []string
		}
	}
}

type newEnv struct {
	r        *testutil.Repo
	cmd, log string
}

// newProject is a committed repository set up with the fake OpenSpec and dkt init.
func newProject(t *testing.T) *newEnv {
	t.Helper()
	r := testutil.NewRepo(t)
	cmd, log := testutil.FakeOpenSpec(t, "1.7.0")
	if code, _, out, _ := doInitCmd(t, r, r.Dir, cmd, log, nil, "--tools", "claude"); code != 0 {
		t.Fatalf("init: %d %s", code, out)
	}
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "set up")
	if err := os.WriteFile(log, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return &newEnv{r, cmd, log}
}

var okArgs = []string{"add-x", "--kind", "feature", "--source", "x", "--title", "X"}

// doNew runs dkt new in dir; it returns the exit code, parsed JSON, raw output and the fake's call log.
func (e *newEnv) doNew(t *testing.T, dir string, extra []string, args ...string) (int, newOut, string, string) {
	t.Helper()
	return e.doNewCmd(t, e.cmd, dir, extra, args...)
}

func (e *newEnv) doNewCmd(t *testing.T, cmd, dir string, extra []string, args ...string) (int, newOut, string, string) {
	t.Helper()
	env := append(append([]string{}, e.r.Env...), "DOCKET_OPENSPEC="+cmd)
	env = append(env, extra...)
	code, out, errs := runIn(t, dir, env, append([]string{"new", "--json"}, args...)...)
	var m newOut
	if out != "" {
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatalf("%q: %v", out, err)
		}
	}
	calls, _ := os.ReadFile(e.log)
	return code, m, out + errs, string(calls)
}

func (e *newEnv) exists(rel string) bool {
	_, err := os.Lstat(filepath.Join(e.r.Dir, rel))
	return err == nil
}

func (e *newEnv) worktree(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	e.r.Git("worktree", "add", "-q", "-b", name, dir)
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

func hasRule(m newOut, rule string) bool {
	for _, f := range m.Findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func TestNewCreated(t *testing.T) { // Change created, Free slug
	e := newProject(t)
	code, m, raw, calls := e.doNew(t, e.r.Dir, nil, "add-x", "--kind", "feature", "--source", "issue 12", "--title", "Booking waitlist")
	if code != 0 || len(m.Findings) != 0 {
		t.Fatalf("code=%d %s", code, raw)
	}
	res := m.Result
	if res.Slug != "add-x" || res.Requested != "add-x" || res.Path != "openspec/changes/add-x" || res.Title != "Booking waitlist" || res.Kind != "feature" || res.Source != "issue 12" {
		t.Errorf("%+v", res)
	}
	if !strings.Contains(raw, `"used":[]`) {
		t.Errorf("used not an empty list: %s", raw)
	}
	if got := readFile(t, e.r, "openspec/changes/add-x/.openspec.yaml"); !strings.Contains(got, "schema: docket") {
		t.Errorf("%q", got)
	}
	if got, want := readFile(t, e.r, "openspec/changes/add-x/proposal.md"), string(change.FrontMatter("Booking waitlist", "feature", "issue 12")); got != want {
		t.Errorf("proposal %q", got)
	}
	if !strings.Contains(calls, "new\tchange\tadd-x\t--schema\tdocket\n") {
		t.Errorf("calls %q", calls)
	}
}

func TestNewText(t *testing.T) {
	e := newProject(t)
	env := append(append([]string{}, e.r.Env...), "DOCKET_OPENSPEC="+e.cmd)
	code, out, _ := runIn(t, e.r.Dir, env, append([]string{"new"}, okArgs...)...)
	if code != 0 || !strings.Contains(out, "created openspec/changes/add-x") {
		t.Errorf("%d %q", code, out)
	}
}

func TestNewUsageErrors(t *testing.T) { // Missing source, Unknown kind, Title on two lines, Kind outside the list, Blank title, Source on two lines
	cases := map[string][]string{
		"missing source": {"add-x", "--kind", "feature", "--title", "X"},
		"missing kind":   {"add-x", "--source", "x", "--title", "X"},
		"missing title":  {"add-x", "--kind", "feature", "--source", "x"},
		"missing slug":   {"--kind", "feature", "--source", "x", "--title", "X"},
		"unknown kind":   {"add-x", "--kind", "fix", "--source", "x", "--title", "X"},
		"two-line title": {"add-x", "--kind", "feature", "--source", "x", "--title", "Line one\nline two"},
		"blank title":    {"add-x", "--kind", "feature", "--source", "x", "--title", "   "},
		"two-line src":   {"add-x", "--kind", "feature", "--title", "X", "--source", "a\nb"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			e := newProject(t)
			code, _, raw, calls := e.doNew(t, e.r.Dir, nil, args...)
			if code != 2 || calls != "" || e.exists("openspec/changes/add-x") {
				t.Fatalf("code=%d calls=%q %s", code, calls, raw)
			}
			if name == "unknown kind" {
				for _, k := range []string{"feature", "change-request", "baseline"} {
					if !strings.Contains(raw, k) {
						t.Errorf("no %s in %s", k, raw)
					}
				}
			}
		})
	}
}

func TestNewSlugRules(t *testing.T) {
	e := newProject(t)
	e.r.Write("openspec/changes/archive/2026-01-02-keep/proposal.md", "k")
	for slug, want := range map[string]string{"2026-10-24-add-x": "archiving adds the date", "archive": "archive"} { // Date prefix refused, Reserved name refused
		code, m, raw, calls := e.doNew(t, e.r.Dir, nil, slug, "--kind", "feature", "--source", "x", "--title", "X")
		if code != 1 || !hasRule(m, "slug-invalid") || !strings.Contains(raw, want) || calls != "" {
			t.Errorf("%s: code=%d calls=%q %s", slug, code, calls, raw)
		}
	}
	if !e.exists("openspec/changes/archive/2026-01-02-keep/proposal.md") || e.exists("openspec/changes/2026-10-24-add-x") {
		t.Error("changes folder altered")
	}
}

func TestNewFormLeftToOpenSpec(t *testing.T) { // Form left to OpenSpec
	e := newProject(t)
	code, m, raw, _ := e.doNew(t, e.r.Dir, nil, "Add-X", "--kind", "feature", "--source", "x", "--title", "X")
	if code != 1 || !hasRule(m, "openspec-failed") || !strings.Contains(raw, "must be lowercase") || e.exists("openspec/changes/Add-X") {
		t.Errorf("code=%d %s", code, raw)
	}
}

func TestNewSchema(t *testing.T) { // Schema not installed, Edited schema accepted
	e := newProject(t)
	p := filepath.Join(e.r.Dir, "openspec/schemas/docket/schema.yaml")
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("# edited\n")
	f.Close()
	if code, _, raw, _ := e.doNew(t, e.r.Dir, nil, okArgs...); code != 0 {
		t.Fatalf("edited: %d %s", code, raw)
	}
	os.Remove(p)
	code, m, raw, calls := e.doNew(t, e.r.Dir, nil, "add-y", "--kind", "feature", "--source", "x", "--title", "X")
	if code != 1 || !hasRule(m, "schema-missing") || !strings.Contains(raw, "dkt init") || e.exists("openspec/changes/add-y") || strings.Contains(calls, "add-y") {
		t.Errorf("code=%d %s", code, raw)
	}
}

func TestNewActiveInOtherWorktree(t *testing.T) { // Slug active in another worktree
	e := newProject(t)
	wt := e.worktree(t, "other")
	if err := os.MkdirAll(filepath.Join(wt, "openspec/changes/add-x"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, m, raw, _ := e.doNew(t, e.r.Dir, nil, okArgs...)
	if code != 0 || m.Result.Slug != "add-x-2" || m.Result.Requested != "add-x" || !e.exists("openspec/changes/add-x-2") {
		t.Fatalf("code=%d %s", code, raw)
	}
	u := m.Result.Used
	if len(u) != 1 || u[0].Path != "openspec/changes/add-x" || len(u[0].Sources) != 1 || u[0].Sources[0] != wt {
		t.Errorf("used %+v", u)
	}
}

func TestNewArchivedOnBranch(t *testing.T) { // Slug archived on a branch
	e := newProject(t)
	e.r.Git("checkout", "-q", "-b", "other")
	e.r.Commit("archive", map[string]string{"openspec/changes/archive/2026-01-02-add-x/proposal.md": "a"})
	e.r.Git("checkout", "-q", "main")
	code, m, raw, _ := e.doNew(t, e.r.Dir, nil, okArgs...)
	u := m.Result.Used
	if code != 0 || m.Result.Slug != "add-x-2" || len(u) != 1 || u[0].Path != "openspec/changes/archive/2026-01-02-add-x" || len(u[0].Sources) != 1 || u[0].Sources[0] != "refs/heads/other" {
		t.Fatalf("code=%d %s", code, raw)
	}
}

func TestNewSuffixAlsoUsed(t *testing.T) { // Suffix also used
	e := newProject(t)
	wt := e.worktree(t, "other")
	for _, s := range []string{"add-x", "add-x-2"} {
		os.MkdirAll(filepath.Join(wt, "openspec/changes", s), 0o755)
	}
	code, m, raw, _ := e.doNew(t, e.r.Dir, nil, okArgs...)
	if code != 0 || m.Result.Slug != "add-x-3" || !e.exists("openspec/changes/add-x-3") {
		t.Fatalf("code=%d %s", code, raw)
	}
}

func TestNewStrayFile(t *testing.T) { // a stray file in this worktree
	e := newProject(t)
	e.r.Write("openspec/changes/add-x", "stray")
	code, m, raw, _ := e.doNew(t, e.r.Dir, nil, okArgs...)
	root, _ := filepath.EvalSymlinks(e.r.Dir)
	u := m.Result.Used
	if code != 0 || m.Result.Slug != "add-x-2" || len(u) != 1 || u[0].Path != "openspec/changes/add-x" || len(u[0].Sources) != 1 || u[0].Sources[0] != root {
		t.Fatalf("code=%d %s", code, raw)
	}
}

func TestNewUnreadable(t *testing.T) { // Worktree directory missing
	e := newProject(t)
	wt := e.worktree(t, "gone")
	os.RemoveAll(wt)
	code, m, raw, calls := e.doNew(t, e.r.Dir, nil, okArgs...)
	if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != wt || strings.Contains(calls, "new") || e.exists("openspec/changes/add-x") {
		t.Fatalf("code=%d calls=%q %s", code, calls, raw)
	}
}

func TestNewOpenSpecFails(t *testing.T) { // OpenSpec fails
	e := newProject(t)
	code, m, raw, calls := e.doNew(t, e.r.Dir, []string{"FAKE_OPENSPEC_NEW_FAIL=1"}, okArgs...)
	if code != 1 || !hasRule(m, "openspec-failed") || !strings.Contains(raw, "new") || e.exists("openspec/changes/add-x") || !strings.Contains(calls, "new\tchange\tadd-x") {
		t.Fatalf("code=%d %s", code, raw)
	}
}

func TestNewChangesIsLink(t *testing.T) { // Changes folder is a link
	e := newProject(t)
	outside := t.TempDir()
	os.RemoveAll(filepath.Join(e.r.Dir, "openspec/changes"))
	if err := os.Symlink(outside, filepath.Join(e.r.Dir, "openspec/changes")); err != nil {
		t.Fatal(err)
	}
	code, m, raw, calls := e.doNew(t, e.r.Dir, nil, okArgs...)
	left, _ := os.ReadDir(outside)
	if code != 1 || !hasRule(m, "symlink-refused") || !strings.Contains(raw, "openspec/changes") || calls != "" || len(left) != 0 {
		t.Fatalf("code=%d calls=%q %s", code, calls, raw)
	}
}

func TestNewLinkAppearsWhileRunning(t *testing.T) {
	e := newProject(t)
	target := filepath.Join(t.TempDir(), "victim")
	code, m, raw, _ := e.doNew(t, e.r.Dir, []string{"FAKE_OPENSPEC_NEW_LINK=" + target}, okArgs...)
	if code != 1 || !hasRule(m, "symlink-refused") || !e.exists("openspec/changes/add-x/.openspec.yaml") {
		t.Fatalf("code=%d %s", code, raw)
	}
	if _, err := os.Stat(target); err == nil {
		t.Error("wrote through the link")
	}
}

func TestNewTwoRuns(t *testing.T) { // Two runs at once
	e := newProject(t)
	env := append(append([]string{}, e.r.Env...), "DOCKET_OPENSPEC="+e.cmd, "FAKE_OPENSPEC_NEW_SLEEP=0.3")
	var wg sync.WaitGroup
	outs := make([]bytes.Buffer, 2)
	codes := make([]int, 2)
	titles := []string{"A", "B"}
	for i := range outs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = Run(Options{Args: []string{"new", "--json", "add-x", "--kind", "feature", "--source", "x", "--title", titles[i]}, Dir: e.r.Dir, Env: env, Stdout: &outs[i], Stderr: &bytes.Buffer{}})
		}()
	}
	wg.Wait()
	slugs := map[string]string{}
	for i := range outs {
		var m newOut
		if err := json.Unmarshal(outs[i].Bytes(), &m); err != nil || codes[i] != 0 {
			t.Fatalf("run %d: %d %s", i, codes[i], outs[i].String())
		}
		slugs[m.Result.Slug] = titles[i]
	}
	if len(slugs) != 2 || slugs["add-x"] == "" || slugs["add-x-2"] == "" {
		t.Fatalf("%v", slugs)
	}
	for s, title := range slugs {
		if got := readFile(t, e.r, "openspec/changes/"+s+"/proposal.md"); got != string(change.FrontMatter(title, "feature", "x")) {
			t.Errorf("%s: %q", s, got)
		}
	}
}

func TestNewOutsideRepository(t *testing.T) { // New outside a repository
	e := newProject(t)
	code, m, _, calls := e.doNew(t, t.TempDir(), nil, okArgs...)
	if code != 4 || !hasRule(m, "no-repository") || calls != "" {
		t.Fatalf("code=%d calls=%q %+v", code, calls, m)
	}
}

func TestNewOpenSpecUnavailable(t *testing.T) {
	e := newProject(t)
	code, m, _, _ := e.doNewCmd(t, filepath.Join(t.TempDir(), "nosuch"), e.r.Dir, nil, okArgs...)
	if code != 4 || !hasRule(m, "openspec-missing") || e.exists("openspec/changes/add-x") {
		t.Errorf("missing: %d %+v", code, m)
	}
	old, log := testutil.FakeOpenSpec(t, "1.6.0")
	e.log = log
	code, m, _, calls := e.doNewCmd(t, old, e.r.Dir, nil, okArgs...)
	if code != 4 || !hasRule(m, "openspec-unsupported") || e.exists("openspec/changes/add-x") || strings.Contains(calls, "new") {
		t.Errorf("old: %d %+v", code, m)
	}
}

func TestNewIgnoresConfigSchema(t *testing.T) { // config.yaml naming another schema
	e := newProject(t)
	e.r.Write("openspec/config.yaml", "schema: spec-driven\n")
	code, _, raw, calls := e.doNew(t, e.r.Dir, nil, okArgs...)
	if code != 0 || !strings.Contains(calls, "new\tchange\tadd-x\t--schema\tdocket\n") {
		t.Fatalf("code=%d calls=%q %s", code, calls, raw)
	}
}

func TestNewTextUsed(t *testing.T) {
	e := newProject(t)
	e.r.Git("checkout", "-q", "-b", "other")
	e.r.Commit("archive", map[string]string{"openspec/changes/archive/2026-01-02-add-x/proposal.md": "a"})
	e.r.Git("checkout", "-q", "main")
	e.r.Git("branch", "b2", "other")
	e.r.Git("branch", "b3", "other")
	e.r.Git("branch", "b4", "other")
	env := append(append([]string{}, e.r.Env...), "DOCKET_OPENSPEC="+e.cmd)
	code, out, _ := runIn(t, e.r.Dir, env, append([]string{"new"}, okArgs...)...)
	want := "add-x is used at openspec/changes/archive/2026-01-02-add-x in refs/heads/b2, refs/heads/b3, refs/heads/b4 and 1 more\ncreated openspec/changes/add-x-2\n"
	if code != 0 || out != want {
		t.Errorf("%d %q", code, out)
	}
}
