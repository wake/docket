package archive

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/lint"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/testutil"
)

const showCancel = `{"deltas":[{"spec":"booking","operation":"MODIFIED","requirement":{"name":"Cancel booking","scenarios":[{"name":"ok"}]}}]}`

// fakeRunner stands in for OpenSpec: it records calls, answers validate and
// show from fields, and archive moves the folder like OpenSpec does.
type fakeRunner struct {
	t          *testing.T
	root       string
	calls      [][]string
	validate   string
	validateEr error
	show       string
	archivedAs map[string]string
	hooks      map[string]func()
}

func (f *fakeRunner) CheckVersion() (string, error) {
	f.calls = append(f.calls, []string{"--version"})
	return "1.14.3", nil
}

func (f *fakeRunner) Run(args ...string) (string, error) {
	f.calls = append(f.calls, args)
	switch args[0] {
	case "validate":
		return f.validate, f.validateEr
	case "show":
		return f.show, nil
	}
	slug := args[1]
	as := "2026-11-02-" + slug
	if v, ok := f.archivedAs[slug]; ok {
		as = v
	}
	dst := filepath.Join(f.root, "openspec/changes/archive", "2026-11-02-"+slug)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(f.root, "openspec/changes", slug), dst); err != nil {
		f.t.Fatal(err)
	}
	if h := f.hooks[slug]; h != nil {
		h()
	}
	return fmt.Sprintf(`{"change":%q,"archive":{"archivedAs":%q}}`, slug, as), nil
}

func (f *fakeRunner) called(name string) bool {
	for _, c := range f.calls {
		if c[0] == name {
			return true
		}
	}
	return false
}

const archPlan = "# Plan\n\n### Task 1: One\n\n- [x] **Step 1: a**\n"

// archRepo is a repository on main whose changes (one per slug, each with a
// delta MODIFYing Cancel booking and a ticked plan) are committed with Task
// trailers on top of x-base.
func archRepo(t *testing.T, slugs ...string) (*testutil.Repo, Env, *fakeRunner) {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	c1 := r.Commit("init", map[string]string{specPath: mainSpec(req("Cancel booking", "old"), req("Refund points", "old"))})
	for _, s := range slugs {
		r.Commit("work on "+s+"\n\nChange: "+s+"\nTask: 1", map[string]string{
			"openspec/changes/" + s + "/.openspec.yaml":        "schema: docket\nx-kind: feature\nx-source: a | b\nx-base: " + c1 + "\n",
			"openspec/changes/" + s + "/plan.md":               archPlan,
			"openspec/changes/" + s + "/specs/booking/spec.md": modDelta("Cancel booking"),
		})
	}
	g := gitx.Git{Dir: r.Dir, Env: r.Env}
	def, err := scan.ResolveDefaultBranch(g, "main")
	if err != nil {
		t.Fatal(err)
	}
	return r, Env{Git: g, Root: r.Dir, Default: def}, &fakeRunner{t: t, root: r.Dir, validate: "\nChange is valid\n", show: showCancel}
}

func (f *fakeRunner) input(e Env, lintFn func(string) lint.Report, slugs ...string) Input {
	if lintFn == nil {
		lintFn = func(string) lint.Report { return lint.Report{} }
	}
	return Input{Env: e, OpenSpec: f, Lint: lintFn, Version: "0.1.0", Origin: "", Slugs: slugs}
}

func status(r *testutil.Repo) string { return r.Git("status", "--porcelain") }

func TestUncommittedChanges(t *testing.T) {
	r, e, f := archRepo(t, "add-x")
	r.Write("stray.txt", "x\n")
	before := status(r)
	rep, err := Archive(f.input(e, nil, "add-x"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ruleList(rep.Findings), []string{"uncommitted-changes"}) || !strings.Contains(rep.Findings[0].Message, "stray.txt") ||
		rep.Stopped == nil || *rep.Stopped != (Stop{"", "preconditions"}) || len(rep.Archived) != 0 {
		t.Fatalf("%+v", rep)
	}
	if len(f.calls) != 1 || f.calls[0][0] != "--version" || status(r) != before {
		t.Fatalf("%v", f.calls)
	}
}

func TestBranchBehindTheDefaultBranch(t *testing.T) {
	r, e, f := archRepo(t, "add-x")
	r.Git("checkout", "-q", "-b", "feature", "HEAD~1")
	r.Git("checkout", "-q", "main")
	r.Commit("main moves", map[string]string{"other.txt": "x\n"})
	r.Git("checkout", "-q", "feature")
	e.Default, _ = scan.ResolveDefaultBranch(e.Git, "main")
	rep, _ := Archive(f.input(e, nil, "add-x"))
	if !reflect.DeepEqual(ruleList(rep.Findings), []string{"behind-default-branch"}) || !strings.Contains(rep.Findings[0].Message, "main") ||
		rep.Stopped == nil || rep.Stopped.Step != "preconditions" {
		t.Fatalf("%+v", rep)
	}
}

func TestChangeNotMergedYet(t *testing.T) {
	r, e, f := archRepo(t)
	r.Git("checkout", "-q", "-b", "feature")
	c1 := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	r.Commit("work\n\nChange: add-x\nTask: 1", map[string]string{
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\nx-base: " + c1 + "\n",
		"openspec/changes/add-x/plan.md":        archPlan,
	})
	rep, _ := Archive(f.input(e, nil, "add-x"))
	if !reflect.DeepEqual(ruleList(rep.Findings), []string{"change-not-merged"}) || rep.Findings[0].Change != "add-x" ||
		rep.Stopped == nil || *rep.Stopped != (Stop{"", "preconditions"}) || f.called("archive") {
		t.Fatalf("%+v", rep)
	}
}

func TestSecondOfTwoFails(t *testing.T) {
	_, e, f := archRepo(t, "add-x", "add-y")
	lf := func(s string) lint.Report {
		if s == "add-y" {
			return lint.Report{Findings: []lint.Finding{{Rule: "scenario-missing", Change: s, Message: "m"}}}
		}
		return lint.Report{}
	}
	rep, err := Archive(f.input(e, lf, "add-x", "add-y"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Archived) != 1 || rep.Archived[0].Change != "add-x" || rep.Archived[0].Folder != "openspec/changes/archive/2026-11-02-add-x" ||
		rep.Archived[0].Evidence != "openspec/changes/archive/2026-11-02-add-x/evidence.md" ||
		rep.Stopped == nil || *rep.Stopped != (Stop{"add-y", "lint"}) ||
		!reflect.DeepEqual(ruleList(rep.Findings), []string{"scenario-missing"}) {
		t.Fatalf("%+v", rep)
	}
	if _, err := os.Stat(filepath.Join(e.Root, "openspec/changes/add-y/plan.md")); err != nil {
		t.Fatal("add-y was touched")
	}
}

func TestUntickedTask(t *testing.T) {
	r, e, f := archRepo(t, "add-x")
	plan := "# Plan\n\n### Task 1: One\n\n- [x] **Step 1: a**\n\n### Task 5: Five\n\n- [ ] **Step 1: z**\n\n## Review Focus\n\n- [ ] look here\n"
	r.Commit("plan\n\nChange: add-x\nTask: 1", map[string]string{"openspec/changes/add-x/plan.md": plan})
	rep, _ := Archive(f.input(e, nil, "add-x"))
	if !reflect.DeepEqual(ruleList(rep.Findings), []string{"plan-unticked"}) || !strings.Contains(rep.Findings[0].Message, "Task 5") ||
		rep.Stopped == nil || *rep.Stopped != (Stop{"add-x", "plan"}) || f.called("validate") {
		t.Fatalf("%+v", rep)
	}
}

func TestSquashedPullRequest(t *testing.T) {
	r, e, f := archRepo(t)
	c1 := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	r.Commit("squashed (#1)", map[string]string{
		"openspec/changes/add-x/.openspec.yaml":        "schema: docket\nx-base: " + c1 + "\n",
		"openspec/changes/add-x/plan.md":               archPlan,
		"openspec/changes/add-x/specs/booking/spec.md": modDelta("Cancel booking"),
	})
	e.Default, _ = scan.ResolveDefaultBranch(e.Git, "main")
	rep, _ := Archive(f.input(e, nil, "add-x"))
	if !reflect.DeepEqual(ruleList(rep.Findings), []string{"no-task-commits"}) || !strings.Contains(rep.Findings[0].Message, "squash") ||
		rep.Stopped == nil || *rep.Stopped != (Stop{"add-x", "plan"}) {
		t.Fatalf("%+v", rep)
	}
}

func TestArchiveStopsOnDrift(t *testing.T) {
	r, e, f := archRepo(t, "add-x")
	r.Commit("main moves", map[string]string{specPath: mainSpec(req("Cancel booking", "changed"), req("Refund points", "old"))})
	rep, _ := Archive(f.input(e, nil, "add-x"))
	if !reflect.DeepEqual(ruleList(rep.Findings), []string{"base-drift"}) || len(rep.Drifts) != 1 ||
		rep.Stopped == nil || *rep.Stopped != (Stop{"add-x", "base"}) || f.called("validate") {
		t.Fatalf("%+v", rep)
	}
}

func TestArchivePassesOtherRequirement(t *testing.T) {
	r, e, f := archRepo(t, "add-x")
	r.Commit("main moves", map[string]string{specPath: mainSpec(req("Cancel booking", "old"), req("Refund points", "changed"))})
	rep, _ := Archive(f.input(e, nil, "add-x"))
	if rep.Stopped != nil || len(rep.Archived) != 1 || len(rep.Findings) != 0 {
		t.Fatalf("%+v", rep)
	}
}

func TestArchiveStopsOnTakenName(t *testing.T) {
	r, e, f := archRepo(t)
	c1 := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	r.Commit("work\n\nChange: add-x\nTask: 1", map[string]string{
		"openspec/changes/add-x/.openspec.yaml":        "schema: docket\nx-base: " + c1 + "\n",
		"openspec/changes/add-x/plan.md":               archPlan,
		"openspec/changes/add-x/specs/booking/spec.md": "## ADDED Requirements\n\n" + req("Waitlist cap", "x"),
	})
	r.Commit("main moves", map[string]string{specPath: mainSpec(req("Cancel booking", "old"), req("Refund points", "old"), req("Waitlist cap", "other"))})
	e.Default, _ = scan.ResolveDefaultBranch(e.Git, "main")
	rep, _ := Archive(f.input(e, nil, "add-x"))
	if !reflect.DeepEqual(ruleList(rep.Findings), []string{"name-taken"}) || rep.Stopped == nil || *rep.Stopped != (Stop{"add-x", "base"}) {
		t.Fatalf("%+v", rep)
	}
}

func TestArchiveSymlinkedSpecRefused(t *testing.T) {
	r, e, f := archRepo(t, "add-x")
	r.Write("elsewhere.md", mainSpec(req("Cancel booking", "old"), req("Refund points", "old")))
	r.Git("rm", "-q", "-f", specPath)
	if err := os.MkdirAll(filepath.Dir(filepath.Join(r.Dir, specPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../../elsewhere.md", filepath.Join(r.Dir, specPath)); err != nil {
		t.Fatal(err)
	}
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "link")
	e.Default, _ = scan.ResolveDefaultBranch(e.Git, "main")
	rep, _ := Archive(f.input(e, nil, "add-x"))
	if !reflect.DeepEqual(ruleList(rep.Findings), []string{"symlink-refused"}) || rep.Stopped == nil || *rep.Stopped != (Stop{"add-x", "archive"}) || f.called("archive") {
		t.Fatalf("%+v", rep)
	}
}

func TestTwoChangesInOneRunTouchOneRequirement(t *testing.T) {
	_, e, f := archRepo(t, "add-x", "add-y")
	f.hooks = map[string]func(){"add-x": func() {
		if err := os.WriteFile(filepath.Join(e.Root, specPath), []byte(mainSpec(req("Cancel booking", "merged"), req("Refund points", "old"))), 0o644); err != nil {
			t.Fatal(err)
		}
	}}
	rep, _ := Archive(f.input(e, nil, "add-x", "add-y"))
	if len(rep.Archived) != 1 || rep.Archived[0].Change != "add-x" || rep.Stopped == nil || *rep.Stopped != (Stop{"add-y", "base"}) ||
		!reflect.DeepEqual(ruleList(rep.Findings), []string{"base-drift"}) {
		t.Fatalf("%+v", rep)
	}
}

func TestArchiveReadingsDisagree(t *testing.T) {
	_, e, f := archRepo(t, "add-x")
	f.show = `{"deltas":[]}`
	rep, _ := Archive(f.input(e, nil, "add-x"))
	if !reflect.DeepEqual(ruleList(rep.Findings), []string{"readings-disagree"}) || rep.Readings == nil ||
		rep.Stopped == nil || *rep.Stopped != (Stop{"add-x", "cross-check"}) || f.called("archive") {
		t.Fatalf("%+v", rep)
	}
}

func TestOpenSpecValidateFails(t *testing.T) {
	_, e, f := archRepo(t, "add-x")
	f.validate = "OUT-TEXT"
	f.validateEr = &openspec.FailedError{Command: []string{"validate"}, Code: 1, Stderr: "ERR-TEXT"}
	rep, err := Archive(f.input(e, nil, "add-x"))
	if err != nil || !reflect.DeepEqual(ruleList(rep.Findings), []string{"openspec-failed"}) ||
		!strings.Contains(rep.Findings[0].Message, "OUT-TEXT") || !strings.Contains(rep.Findings[0].Message, "ERR-TEXT") ||
		rep.Stopped == nil || *rep.Stopped != (Stop{"add-x", "validate"}) {
		t.Fatalf("%v %+v", err, rep)
	}
}

func TestOpenSpecUnavailableIsAnError(t *testing.T) {
	_, e, f := archRepo(t, "add-x")
	f.validateEr = errors.New("boom")
	if _, err := Archive(f.input(e, nil, "add-x")); err == nil {
		t.Fatal("want error")
	}
}

func TestUnexpectedArchivedAs(t *testing.T) {
	_, e, f := archRepo(t, "add-x")
	f.archivedAs = map[string]string{"add-x": "2026-11-02-other"}
	rep, _ := Archive(f.input(e, nil, "add-x"))
	if !reflect.DeepEqual(ruleList(rep.Findings), []string{"evidence-failed"}) || rep.Stopped == nil || rep.Stopped.Step != "archive" {
		t.Fatalf("%+v", rep)
	}
}

func TestNothingCommitted(t *testing.T) {
	r, e, f := archRepo(t, "add-x")
	head := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	rep, err := Archive(f.input(e, nil, "add-x"))
	if err != nil || rep.Stopped != nil || len(rep.Archived) != 1 {
		t.Fatalf("%v %+v", err, rep)
	}
	if strings.TrimSpace(r.Git("rev-parse", "HEAD")) != head {
		t.Fatal("HEAD moved")
	}
	if st := status(r); !strings.Contains(st, "openspec/changes") {
		t.Fatalf("status %q", st)
	}
	b, _ := json.Marshal(rep.Archived)
	if !strings.Contains(string(b), `"change":"add-x"`) || !strings.Contains(string(b), `"evidence"`) {
		t.Fatalf("%s", b)
	}
}
