package lint

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/openspec"
)

func TestScenarioLineSkipsWrongLevelAndStopsAtNextRequirement(t *testing.T) {
	src := "## Requirement: Join\n\n### Requirement: Join\n\n#### Scenario: Success\n"
	if got := scenarioLine([]byte(src), "Join", "Success"); got != 5 {
		t.Errorf("level 2 heading: got %d, want 5", got)
	}
	src = "### Requirement: A\n\n### Requirement: B\n\n#### Scenario: Success\n"
	if got := scenarioLine([]byte(src), "A", "Success"); got != 0 {
		t.Errorf("A = %d, want 0", got)
	}
	if got := scenarioLine([]byte(src), "B", "Success"); got != 5 {
		t.Errorf("B = %d, want 5", got)
	}
}

func TestShowChangeFailureSkipsScenarioRules(t *testing.T) {
	f := &fakeOS{showErr: errors.New("cannot show")}
	rep := lintDeltas(t, f, map[string]string{pl: planWith("**Scenarios:** booking#join"), dp("booking"): doc(req("Join", "Join"))})
	if count(rep, "scenario-ref")+count(rep, "scenario-unclaimed") != 0 || count(rep, "openspec-failed") != 1 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestEachMalformedItemGivesScenarioRef(t *testing.T) {
	for _, item := range []string{"booking#a#b", "#join"} {
		rep := lintDeltas(t, &fakeOS{}, map[string]string{pl: planWith("**Scenarios:** " + item)})
		if count(rep, "scenario-ref") != 1 {
			t.Errorf("%s: %+v", item, rep.Findings)
		}
	}
}

func TestRemovedMainScenarioIsKnown(t *testing.T) {
	old := req("Old", "Success")
	f := &fakeOS{
		deltas: []openspec.Delta{delta("REMOVED", "booking", req("Old"))},
		specs:  map[string][]openspec.Requirement{"booking": {old}},
	}
	rep := lintDeltas(t, f, map[string]string{pl: planWith("**Scenarios:** booking#success"), mp("booking"): doc(old)})
	if count(rep, "scenario-ref") != 0 {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestModifiedScenarioUnclaimed(t *testing.T) {
	m := req("Leave", "Leave anytime")
	f := &fakeOS{
		deltas: []openspec.Delta{delta("MODIFIED", "booking", m)},
		specs:  map[string][]openspec.Requirement{"booking": {m}},
	}
	rep := lintDeltas(t, f, map[string]string{pl: goodPlan, mp("booking"): doc(m), dp("booking"): doc(m)})
	if !has(rep, "scenario-unclaimed", dp("booking"), -1, "booking#leave-anytime") {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestUnsafeCapabilityNotAskedOfOpenSpec(t *testing.T) {
	f := &fakeOS{}
	files := map[string]string{mp("-x"): doc()}
	for _, item := range []string{"../x#y", "-x#y"} {
		files[pl] = planWith("**Scenarios:** " + item)
		rep := lintDeltas(t, f, files)
		if count(rep, "scenario-ref") != 1 {
			t.Errorf("%s: %+v", item, rep.Findings)
		}
	}
	if len(f.specCalls) != 0 {
		t.Fatalf("ShowSpec called: %v", f.specCalls)
	}
}

func TestUnreadableMainSpecIsSkipped(t *testing.T) {
	skipRoot(t)
	f := &fakeOS{deltas: []openspec.Delta{delta("ADDED", "booking", req("A", "Success"), req("B", "Success"))}}
	dir := t.TempDir()
	for p, c := range map[string]string{
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\n",
		"openspec/changes/add-x/proposal.md":    goodProposal,
		"internal/a.go":                         "package a\n",
		pl:                                      planWith("**Scenarios:** booking#join"),
		dp("booking"):                           doc(req("A", "Success"), req("B", "Success")),
		mp("booking"):                           doc(),
	} {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	locked := filepath.Join(dir, "openspec/specs/booking")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	in, _ := open(t, dir)
	in.OpenSpec = func() (OpenSpec, error) { return f, nil }
	in.Slugs = []string{"add-x"}
	rep := run(t, in)
	found := false
	for _, u := range rep.Unreadable {
		found = found || u.Source == mp("booking")
	}
	if !found || count(rep, "scenario-ref") != 0 || count(rep, "scenario-duplicate") != 0 ||
		count(rep, "openspec-failed") != 0 || f.specCalls["booking"] != 0 {
		t.Fatalf("%+v %+v %v", rep.Findings, rep.Unreadable, f.specCalls)
	}
}
