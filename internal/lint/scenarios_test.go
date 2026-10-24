package lint

import (
	"strconv"
	"strings"
	"testing"
)

const bookingDelta = "## ADDED Requirements\n\n### Requirement: Join\n\nThe system SHALL join.\n\n#### Scenario: Join when full\n\n- **WHEN** x\n- **THEN** y\n"

func planWith(scen string) string {
	return planPrefix + "### Task 1: One\n\n**Scenarios:** " + scen + "\n"
}

func scenRepo(plan string, files map[string]string) Report {
	m := planRepo(plan)
	for p, s := range files {
		m[p] = repoWith(map[string]string{p: s})[p]
	}
	return runOn(m)
}

func deltaAt(capPath string) string { return "openspec/changes/add-x/specs/" + capPath + "/spec.md" }

func TestScenarioNobodyClaims(t *testing.T) {
	r := scenRepo(planWith("none"), map[string]string{deltaAt("booking"): bookingDelta})
	f := wantOne(t, r, "scenario-unclaimed", lineOf(bookingDelta, "#### Scenario"))
	if !strings.Contains(f.Message, "booking#join-when-full") || f.File != deltaAt("booking") {
		t.Fatalf("finding = %+v", f)
	}
}

func TestClaimedScenarioIsClean(t *testing.T) {
	r := scenRepo(planWith("`booking#join-when-full`"), map[string]string{deltaAt("booking"): bookingDelta})
	if len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestReferenceToAnUnknownScenario(t *testing.T) {
	plan := planWith("booking#join-when-ful")
	r := scenRepo(plan, map[string]string{deltaAt("booking"): bookingDelta})
	var got *Finding
	for i, f := range r.Findings {
		if f.Rule == "scenario-ref" {
			got = &r.Findings[i]
		}
	}
	if got == nil || got.Line != lineOf(plan, "**Scenarios:**") || !strings.Contains(got.Message, "booking#join-when-ful") {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestMalformedReference(t *testing.T) {
	for _, item := range []string{"booking", "#x", "booking#"} {
		r := scenRepo(planWith(item), nil)
		wantOne(t, r, "scenario-ref", lineOf(planWith(item), "**Scenarios:**"))
	}
}

func TestEmptyScenariosValueIsMalformed(t *testing.T) {
	plan := planWith("")
	wantOne(t, scenRepo(plan, nil), "scenario-ref", lineOf(plan, "**Scenarios:**"))
}

func TestEmptySlugDuplicates(t *testing.T) {
	d := "## ADDED Requirements\n\n### Requirement: A\n\nx\n\n#### Scenario: !!!\n\n- a\n\n#### Scenario: ???\n\n- b\n"
	got := dupFindings(scenRepo(planWith("none"), map[string]string{deltaAt("booking"): d}))
	if len(got) != 1 || got[0].Line != lineOf(d, "???") {
		t.Fatalf("dups = %+v", got)
	}
}

func TestDuplicateModifiedBeforeAdded(t *testing.T) {
	d := "## MODIFIED Requirements\n\n### Requirement: Old\n\nx\n\n#### Scenario: Success\n\n- a\n\n## ADDED Requirements\n\n### Requirement: New\n\nx\n\n#### Scenario: Success\n\n- b\n"
	got := dupFindings(scenRepo(planWith("none"), map[string]string{deltaAt("booking"): d}))
	if len(got) != 1 || got[0].Line != lineOf(d, "Success\n\n- b") {
		t.Fatalf("dups = %+v", got)
	}
	if !strings.HasSuffix(got[0].Message, ":"+strconv.Itoa(lineOf(d, "Success\n\n- a"))) {
		t.Fatalf("message = %q", got[0].Message)
	}
}

func TestScaffoldingTask(t *testing.T) {
	if r := scenRepo(planWith("none"), nil); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestMissingScenariosLine(t *testing.T) {
	plan := planPrefix + "### Task 1: One\n\nbody\n"
	wantOne(t, scenRepo(plan, nil), "scenario-ref", lineOf(plan, "### Task 1"))
}

func TestReferenceToMainSpec(t *testing.T) {
	r := scenRepo(planWith("booking#join-when-full"), map[string]string{
		"openspec/specs/booking/spec.md": "## Requirements\n\n### Requirement: Join\n\nx\n\n#### Scenario: Join when full\n\n- **WHEN** a\n",
	})
	if len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestNonASCIISlugs(t *testing.T) {
	d := "## ADDED Requirements\n\n### Requirement: R\n\nx\n\n#### Scenario: 需求 R1#3\n\n- a\n\n#### Scenario: Join (when full)\n\n- b\n"
	r := scenRepo(planWith("booking#需求-r1-3, booking#join-when-full"), map[string]string{deltaAt("booking"): d})
	if len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func dupFindings(r Report) []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if f.Rule == "scenario-duplicate" {
			out = append(out, f)
		}
	}
	return out
}

func TestTwoScenariosNamedSuccess(t *testing.T) {
	d := "## ADDED Requirements\n\n### Requirement: A\n\nx\n\n#### Scenario: Success\n\n- a\n\n### Requirement: B\n\nx\n\n#### Scenario: Success\n\n- b\n"
	got := dupFindings(scenRepo(planWith("none"), map[string]string{deltaAt("booking"): d}))
	if len(got) != 1 || got[0].Line != lineOf(d, "Success\n\n- b") {
		t.Fatalf("dups = %+v", got)
	}
	if !strings.Contains(got[0].Message, "booking#success") {
		t.Fatalf("message = %q", got[0].Message)
	}
}

func TestDuplicateAgainstMainSpecAndModified(t *testing.T) {
	main := "## Requirements\n\n### Requirement: Old\n\nx\n\n#### Scenario: Success\n\n- a\n\n### Requirement: Keep\n\nx\n\n#### Scenario: Kept\n\n- a\n"
	// ADDED collides with main; MODIFIED replaces Old without colliding.
	d := "## ADDED Requirements\n\n### Requirement: New\n\nx\n\n#### Scenario: Kept\n\n- a\n\n## MODIFIED Requirements\n\n### Requirement: Old\n\nx\n\n#### Scenario: Success\n\n- a\n"
	got := dupFindings(scenRepo(planWith("none"), map[string]string{
		"openspec/specs/booking/spec.md": main, deltaAt("booking"): d,
	}))
	if len(got) != 1 || got[0].Line != lineOf(d, "Kept") || !strings.Contains(got[0].Message, "openspec/specs/booking/spec.md") {
		t.Fatalf("dups = %+v", got)
	}
}

func TestPunctuatedDuplicates(t *testing.T) {
	d := "## ADDED Requirements\n\n### Requirement: A\n\nx\n\n#### Scenario: Join (when full)\n\n- a\n\n#### Scenario: Join when full\n\n- a\n\n#### Scenario: 需求 R1#3\n\n- a\n\n#### Scenario: 需求 r1 3\n\n- a\n"
	got := dupFindings(scenRepo(planWith("none"), map[string]string{deltaAt("booking"): d}))
	if len(got) != 2 {
		t.Fatalf("dups = %+v", got)
	}
}

func TestRenameThenModifyIsNotADuplicate(t *testing.T) {
	main := "## Requirements\n\n### Requirement: Old\n\nx\n\n#### Scenario: Join when full\n\n- a\n"
	d := "## RENAMED Requirements\n\n- FROM: `### Requirement: Old`\n- TO: `### Requirement: New`\n\n## MODIFIED Requirements\n\n### Requirement: New\n\nx\n\n#### Scenario: Join when full\n\n- a\n"
	got := dupFindings(scenRepo(planWith("none"), map[string]string{
		"openspec/specs/booking/spec.md": main, deltaAt("booking"): d,
	}))
	if len(got) != 0 {
		t.Fatalf("dups = %+v", got)
	}
}

func TestPureRenameKeepsScenarios(t *testing.T) {
	main := "## Requirements\n\n### Requirement: Old\n\nx\n\n#### Scenario: Success\n\n- a\n"
	d := "## RENAMED Requirements\n\n- FROM: `### Requirement: Old`\n- TO: `### Requirement: New`\n\n## ADDED Requirements\n\n### Requirement: Other\n\nx\n\n#### Scenario: Success\n\n- a\n"
	got := dupFindings(scenRepo(planWith("none"), map[string]string{
		"openspec/specs/booking/spec.md": main, deltaAt("booking"): d,
	}))
	if len(got) != 1 || got[0].Line != lineOf(d, "Success") || !strings.Contains(got[0].Message, "openspec/specs/booking/spec.md") {
		t.Fatalf("dups = %+v", got)
	}
}

func TestDuplicatesWithoutPlan(t *testing.T) {
	d := "## ADDED Requirements\n\n### Requirement: A\n\nx\n\n#### Scenario: S\n\n- a\n\n#### Scenario: S\n\n- a\n"
	r := runOn(repoWith(map[string]string{
		"openspec/changes/add-x/proposal.md": validProposal, deltaAt("booking"): d,
	}))
	if len(dupFindings(r)) != 1 || len(r.Findings) != 1 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}
