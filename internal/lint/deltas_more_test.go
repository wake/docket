package lint

import (
	"errors"
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
