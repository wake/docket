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

const specPath = "openspec/specs/booking/spec.md"

func req(name, then string) string {
	return "### Requirement: " + name + "\nThe system SHALL " + name + ".\n\n#### Scenario: ok\n- **WHEN** a thing\n- **THEN** " + then + "\n\n"
}

func mainSpec(reqs ...string) string {
	return "# booking\n\n## Purpose\nBooking.\n\n## Requirements\n\n" + strings.Join(reqs, "")
}

const deltaPath = "openspec/changes/add-y/specs/booking/spec.md"

// baseRepo commits mainSpec(original) as c1 with x-base c1 and a change
// add-y carrying the given delta; it returns the repo, env and c1.
func baseRepo(t *testing.T, deltaSpec string, original ...string) (*testutil.Repo, Env, string) {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	c1 := r.Commit("init", map[string]string{specPath: mainSpec(original...)})
	r.Commit("add the change", map[string]string{
		"openspec/changes/add-y/.openspec.yaml": "schema: docket\nx-base: " + c1 + "\n",
		deltaPath:                               deltaSpec,
	})
	e := Env{Git: gitx.Git{Dir: r.Dir, Env: r.Env}, Root: r.Dir}
	return r, e, c1
}

func modDelta(names ...string) string {
	s := "## MODIFIED Requirements\n\n"
	for _, n := range names {
		s += req(n, "new")
	}
	return s
}

func check(t *testing.T, e Env) ([]Drift, []Finding, []Unreadable, int) {
	t.Helper()
	c, f, u := loadChange(e, "add-y", loadOpts{BaseIsCommit: true})
	if c == nil {
		t.Fatalf("load: %+v %+v", f, u)
	}
	return checkBase(e, c)
}

func ruleList(fs []Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Rule)
	}
	return out
}

func TestRequirementChangedSinceBase(t *testing.T) {
	r, e, _ := baseRepo(t, modDelta("Cancel booking"), req("Cancel booking", "old"), req("Refund points", "old"))
	r.Commit("main moves", map[string]string{specPath: mainSpec(req("Cancel booking", "changed"), req("Refund points", "old"))})
	drifts, finds, unread, n := check(t, e)
	if len(unread) != 0 || n != 1 || len(drifts) != 1 {
		t.Fatalf("%+v %+v %d", drifts, unread, n)
	}
	d := drifts[0]
	if d.Capability != "booking" || d.Requirement != "Cancel booking" ||
		!strings.Contains(d.Base, "THEN** old") || !strings.Contains(d.Now, "THEN** changed") {
		t.Fatalf("%+v", d)
	}
	if len(finds) != 1 || finds[0].Rule != "base-drift" || finds[0].File != specPath || finds[0].Line != 8 ||
		!strings.Contains(finds[0].Message, "dkt rebase add-y") {
		t.Fatalf("%+v", finds)
	}
}

func TestOtherRequirementInTheSameCapability(t *testing.T) {
	r, e, _ := baseRepo(t, modDelta("Cancel booking"), req("Cancel booking", "old"), req("Refund points", "old"))
	r.Commit("main moves", map[string]string{specPath: mainSpec(req("Cancel booking", "old"), req("Refund points", "changed"))})
	drifts, finds, _, n := check(t, e)
	if len(drifts) != 0 || len(finds) != 0 || n != 1 {
		t.Fatalf("%+v %+v %d", drifts, finds, n)
	}
}

func TestAddedNameNowTaken(t *testing.T) {
	r, e, _ := baseRepo(t, "## ADDED Requirements\n\n"+req("Waitlist cap", "x"), req("Cancel booking", "old"))
	r.Commit("main moves", map[string]string{specPath: mainSpec(req("Cancel booking", "old"), req("Waitlist cap", "other"))})
	drifts, finds, _, _ := check(t, e)
	if !reflect.DeepEqual(ruleList(finds), []string{"name-taken"}) || len(drifts) != 1 ||
		drifts[0].Base != "" || !strings.Contains(drifts[0].Now, "other") || finds[0].Line == 0 {
		t.Fatalf("%+v %+v", drifts, finds)
	}
}

func TestBlocksDifferingOnlyInWhitespace(t *testing.T) {
	orig := req("Cancel booking", "old")
	r, e, _ := baseRepo(t, modDelta("Cancel booking"), orig)
	crlf := strings.ReplaceAll(mainSpec(orig), "\n", "\r\n") + "\r\n\r\n"
	r.Commit("whitespace", map[string]string{specPath: crlf})
	if drifts, finds, _, _ := check(t, e); len(drifts) != 0 || len(finds) != 0 {
		t.Fatalf("%+v %+v", drifts, finds)
	}
	r.Commit("real change", map[string]string{specPath: mainSpec(strings.Replace(orig, "THEN** old", "THEN** older", 1))})
	if drifts, _, _, _ := check(t, e); len(drifts) != 1 {
		t.Fatalf("%+v", drifts)
	}
}

func TestRemovedAndRenamedChecked(t *testing.T) {
	d := "## REMOVED Requirements\n\n### Requirement: Cancel booking\n\n## RENAMED Requirements\n\n- FROM: `### Requirement: Refund points`\n- TO: `### Requirement: Refund credits`\n"
	r, e, _ := baseRepo(t, d, req("Cancel booking", "old"), req("Refund points", "old"))
	r.Commit("main moves", map[string]string{specPath: mainSpec(req("Cancel booking", "x"), req("Refund points", "y"))})
	drifts, finds, _, n := check(t, e)
	if len(drifts) != 2 || n != 2 || !reflect.DeepEqual(ruleList(finds), []string{"base-drift", "base-drift"}) {
		t.Fatalf("%+v %+v %d", drifts, finds, n)
	}
}

func TestCapabilityMissingAtBase(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	c1 := r.Commit("init", map[string]string{"README.md": "x\n"})
	r.Commit("add the change", map[string]string{
		"openspec/changes/add-y/.openspec.yaml": "schema: docket\nx-base: " + c1 + "\n",
		deltaPath:                               "## ADDED Requirements\n\n" + req("Waitlist cap", "x"),
	})
	e := Env{Git: gitx.Git{Dir: r.Dir, Env: r.Env}, Root: r.Dir}
	if drifts, finds, unread, _ := check(t, e); len(drifts) != 0 || len(finds) != 0 || len(unread) != 0 {
		t.Fatalf("%+v %+v %+v", drifts, finds, unread)
	}
}

func TestBaseCheckReadsTheWorktreeNotHead(t *testing.T) {
	r, e, _ := baseRepo(t, modDelta("Cancel booking"), req("Cancel booking", "old"))
	if err := os.WriteFile(filepath.Join(r.Dir, specPath), []byte(mainSpec(req("Cancel booking", "edited"))), 0o644); err != nil {
		t.Fatal(err)
	}
	drifts, finds, _, n := check(t, e)
	if n != 1 || len(drifts) != 1 || !strings.Contains(drifts[0].Now, "THEN** edited") ||
		!reflect.DeepEqual(ruleList(finds), []string{"base-drift"}) {
		t.Fatalf("%+v %+v %d", drifts, finds, n)
	}
}
