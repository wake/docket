package lint

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/wake/docket/internal/openspec"
)

type fakeOS struct {
	deltas    []openspec.Delta
	showErr   error
	specs     map[string][]openspec.Requirement
	specErr   map[string]error
	issues    []openspec.Issue
	valErr    error
	specCalls map[string]int
}

func (f *fakeOS) ShowChange(string) ([]openspec.Delta, error) { return f.deltas, f.showErr }
func (f *fakeOS) Validate(string) ([]openspec.Issue, error)   { return f.issues, f.valErr }
func (f *fakeOS) ShowSpec(c string) ([]openspec.Requirement, error) {
	if f.specCalls == nil {
		f.specCalls = map[string]int{}
	}
	f.specCalls[c]++
	return f.specs[c], f.specErr[c]
}

func req(name string, scen ...string) openspec.Requirement {
	r := openspec.Requirement{Name: name}
	for _, s := range scen {
		r.Scenarios = append(r.Scenarios, openspec.Scenario{Name: s})
	}
	return r
}

func delta(op, capability string, reqs ...openspec.Requirement) openspec.Delta {
	return openspec.Delta{Spec: capability, Operation: op, Requirements: reqs}
}

func renamed(from, to string) openspec.Delta {
	return openspec.Delta{Spec: "booking", Operation: "RENAMED", Rename: &openspec.Rename{From: from, To: to}}
}

// doc writes a spec file holding each requirement with its scenarios.
func doc(reqs ...openspec.Requirement) string {
	var b strings.Builder
	b.WriteString("# T Specification\n\n")
	for _, r := range reqs {
		b.WriteString("### Requirement: " + r.Name + "\nbody\n\n")
		for _, s := range r.Scenarios {
			b.WriteString("#### Scenario: " + s.Name + "\n- x\n\n")
		}
	}
	return b.String()
}

func dp(c string) string { return "openspec/changes/add-x/specs/" + c + "/spec.md" }
func mp(c string) string { return "openspec/specs/" + c + "/spec.md" }

func planWith(scen string) string {
	return strings.Replace(goodPlan, "**Scenarios:** none", scen, 1)
}

func lintDeltas(t *testing.T, f *fakeOS, files map[string]string) Report {
	t.Helper()
	m := map[string]string{
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\n",
		"openspec/changes/add-x/proposal.md":    goodProposal,
		"internal/a.go":                         "package a\n",
	}
	for k, v := range files {
		m[k] = v
	}
	in, _ := repo(t, m)
	in.OpenSpec = func() (OpenSpec, error) { return f, nil }
	in.Slugs = []string{"add-x"}
	return run(t, in)
}

func TestScenarioUnclaimed(t *testing.T) {
	r1 := req("Join", "Join when full")
	d := doc(r1)
	f := &fakeOS{deltas: []openspec.Delta{delta("ADDED", "booking", r1)}}
	rep := lintDeltas(t, f, map[string]string{pl: goodPlan, dp("booking"): d})
	if !has(rep, "scenario-unclaimed", dp("booking"), lineOf(d, "Scenario: Join when full"), "booking#join-when-full") || len(rep.Findings) != 1 {
		t.Fatalf("%+v", rep.Findings)
	}
	rep = lintDeltas(t, f, map[string]string{pl: planWith("**Scenarios:** booking#join-when-full"), dp("booking"): d})
	if len(rep.Findings) != 0 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestScenarioRefUnknown(t *testing.T) {
	plan := planWith("**Scenarios:** booking#join-when-ful")
	f := &fakeOS{deltas: []openspec.Delta{delta("ADDED", "booking", req("Join"))}}
	rep := lintDeltas(t, f, map[string]string{pl: plan})
	if !has(rep, "scenario-ref", pl, lineOf(plan, "**Scenarios:**"), "booking#join-when-ful") {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestTaskWithoutScenariosLine(t *testing.T) {
	plan := strings.Replace(goodPlan, "**Scenarios:** none\n", "", 1)
	rep := lintDeltas(t, &fakeOS{}, map[string]string{pl: plan})
	if !has(rep, "scenario-ref", pl, lineOf(plan, "### Task 1"), "Task 1") || len(rep.Findings) != 1 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestMalformedItems(t *testing.T) {
	plan := planWith("**Scenarios:** booking, #x, booking#, booking#a#b")
	rep := lintDeltas(t, &fakeOS{}, map[string]string{pl: plan})
	if count(rep, "scenario-ref") != 4 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestMainScenarioUntouchedAccepted(t *testing.T) {
	m := req("Leave", "Leave anytime")
	f := &fakeOS{
		deltas: []openspec.Delta{delta("ADDED", "booking", req("Other"))},
		specs:  map[string][]openspec.Requirement{"booking": {m}},
	}
	rep := lintDeltas(t, f, map[string]string{pl: planWith("**Scenarios:** booking#leave-anytime"), mp("booking"): doc(m)})
	if len(rep.Findings) != 0 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestNoPlanStillDuplicateAndInvalid(t *testing.T) {
	r1, r2 := req("A", "Success"), req("B", "Success")
	d := doc(r1, r2)
	f := &fakeOS{
		deltas: []openspec.Delta{delta("ADDED", "booking", r1, r2)},
		issues: []openspec.Issue{{Level: "ERROR", Path: "booking/spec.md", Message: "bad", Line: 7}},
	}
	rep := lintDeltas(t, f, map[string]string{dp("booking"): d})
	if count(rep, "scenario-ref")+count(rep, "scenario-unclaimed") != 0 {
		t.Fatalf("%+v", rep.Findings)
	}
	line1 := lineOf(d, "Scenario: Success")
	line2 := lineOf(d, "Requirement: B") + 3
	if !has(rep, "scenario-duplicate", dp("booking"), line2, "booking#success") ||
		!has(rep, "scenario-duplicate", dp("booking"), line2, dp("booking")+":"+strconv.Itoa(line1)) {
		t.Fatalf("%+v", rep.Findings)
	}
	if !has(rep, "openspec-invalid", dp("booking"), 7, "booking/spec.md: bad") {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestRemovedFreesSlug(t *testing.T) {
	old := req("Old", "Success")
	nw := req("New", "Success")
	f := &fakeOS{
		deltas: []openspec.Delta{delta("REMOVED", "booking", req("Old")), delta("ADDED", "booking", nw)},
		specs:  map[string][]openspec.Requirement{"booking": {old}},
	}
	rep := lintDeltas(t, f, map[string]string{mp("booking"): doc(old), dp("booking"): doc(nw)})
	if count(rep, "scenario-duplicate") != 0 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestRenamedKeepsScenarios(t *testing.T) {
	join := req("Join", "Success")
	specs := map[string][]openspec.Requirement{"booking": {join}}
	files := map[string]string{mp("booking"): doc(join)}
	f := &fakeOS{deltas: []openspec.Delta{renamed("Join", "Joined")}, specs: specs}
	if rep := lintDeltas(t, f, files); count(rep, "scenario-duplicate") != 0 {
		t.Fatalf("self clash: %+v", rep.Findings)
	}
	other := req("Other", "Success")
	files[dp("booking")] = doc(other)
	f = &fakeOS{deltas: []openspec.Delta{renamed("Join", "Joined"), delta("ADDED", "booking", other)}, specs: specs}
	rep := lintDeltas(t, f, files)
	if !has(rep, "scenario-duplicate", dp("booking"), -1, mp("booking")) {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestRenamedAndModifiedReplaces(t *testing.T) {
	join := req("Join", "Old one")
	mod := req("Joined", "New one")
	other := req("Other", "Old one")
	f := &fakeOS{
		deltas: []openspec.Delta{renamed("Join", "Joined"), delta("MODIFIED", "booking", mod), delta("ADDED", "booking", other)},
		specs:  map[string][]openspec.Requirement{"booking": {join}},
	}
	rep := lintDeltas(t, f, map[string]string{mp("booking"): doc(join), dp("booking"): doc(mod, other)})
	if count(rep, "scenario-duplicate") != 0 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestMainClashReportedNamingBoth(t *testing.T) {
	a, b := req("A", "Success"), req("B", "Success")
	md := doc(a, b)
	f := &fakeOS{
		deltas: []openspec.Delta{delta("ADDED", "booking", req("C"))},
		specs:  map[string][]openspec.Requirement{"booking": {a, b}},
	}
	rep := lintDeltas(t, f, map[string]string{mp("booking"): md, dp("booking"): doc(req("C"))})
	var got *Finding
	for i := range rep.Findings {
		if rep.Findings[i].Rule == "scenario-duplicate" {
			got = &rep.Findings[i]
		}
	}
	if got == nil || got.Change != "add-x" || got.File != mp("booking") || got.Line != lineOf(md, "Requirement: B")+3 ||
		!strings.Contains(got.Message, mp("booking")+":"+strconv.Itoa(lineOf(md, "Requirement: A")+3)) {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestModifiedReplacesMain(t *testing.T) {
	m := req("Leave", "Leave anytime")
	f := &fakeOS{
		deltas: []openspec.Delta{delta("MODIFIED", "booking", m)},
		specs:  map[string][]openspec.Requirement{"booking": {m}},
	}
	rep := lintDeltas(t, f, map[string]string{pl: planWith("**Scenarios:** booking#leave-anytime"), mp("booking"): doc(m), dp("booking"): doc(m)})
	if len(rep.Findings) != 0 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestLineZeroWhenHeadingMissing(t *testing.T) {
	f := &fakeOS{deltas: []openspec.Delta{delta("ADDED", "booking", req("Join", "Gone"))}}
	rep := lintDeltas(t, f, map[string]string{pl: goodPlan, dp("booking"): "# T Specification\n"})
	if !has(rep, "scenario-unclaimed", dp("booking"), 0, "booking#gone") {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestShowSpecFailsForOneCapability(t *testing.T) {
	x, y := req("A", "Success"), req("B", "Success")
	c := req("C", "S")
	f := &fakeOS{
		deltas:  []openspec.Delta{delta("ADDED", "booking", c), delta("ADDED", "rooms", x, y)},
		specErr: map[string]error{"booking": errors.New("boom")},
	}
	plan := planWith("**Scenarios:** booking#zzz, rooms#success")
	rep := lintDeltas(t, f, map[string]string{
		pl: plan, mp("booking"): doc(), mp("rooms"): doc(), dp("booking"): doc(c), dp("rooms"): doc(x, y),
	})
	if count(rep, "openspec-failed") != 1 || !has(rep, "openspec-failed", "", 0, "boom") ||
		count(rep, "scenario-duplicate") != 1 || count(rep, "scenario-ref") != 0 || count(rep, "scenario-unclaimed") != 0 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestShowChangeFails(t *testing.T) {
	f := &fakeOS{showErr: errors.New("cannot show"), issues: []openspec.Issue{{Level: "error", Path: "x", Message: "m"}}}
	rep := lintDeltas(t, f, map[string]string{pl: goodPlan})
	if !has(rep, "openspec-failed", "", 0, "cannot show") || !has(rep, "openspec-invalid", "", 0, "x: m") || len(rep.Findings) != 2 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestValidateFailsAndWarnings(t *testing.T) {
	f := &fakeOS{valErr: errors.New("no validate")}
	if rep := lintDeltas(t, f, nil); !has(rep, "openspec-failed", "", 0, "no validate") {
		t.Fatalf("%+v", rep.Findings)
	}
	f = &fakeOS{issues: []openspec.Issue{{Level: "WARNING", Path: "booking/spec.md", Message: "w"}}}
	if rep := lintDeltas(t, f, nil); len(rep.Findings) != 0 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestShowSpecOncePerRunAndOnlyWhenMainExists(t *testing.T) {
	f := &fakeOS{deltas: []openspec.Delta{delta("ADDED", "booking", req("A")), delta("ADDED", "rooms", req("B"))}}
	m := map[string]string{mp("booking"): doc(), "internal/a.go": "package a\n"}
	for _, s := range []string{"add-x", "add-y"} {
		m["openspec/changes/"+s+"/.openspec.yaml"] = "schema: docket\n"
		m["openspec/changes/"+s+"/proposal.md"] = goodProposal
	}
	in, _ := repo(t, m)
	in.OpenSpec = func() (OpenSpec, error) { return f, nil }
	run(t, in)
	if f.specCalls["booking"] != 1 || f.specCalls["rooms"] != 0 {
		t.Fatalf("%v", f.specCalls)
	}
}
