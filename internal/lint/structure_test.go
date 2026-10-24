package lint

import "testing"

func lintDelta(d string) Report {
	return runOn(repoWith(map[string]string{
		"openspec/changes/add-x/proposal.md": validProposal, deltaAt("booking"): d,
	}))
}

func TestRequirementOutsideAnyDeltaSection(t *testing.T) {
	d := "## ADDED Requirements\n\n### Requirement: A\n\nx\n\n#### Scenario: S\n\n- a\n\n## Notes\n\n### Requirement: Stray\n\nx\n"
	r := lintDelta(d)
	for _, f := range r.Findings {
		if f.Rule == "unparseable" && f.Line == lineOf(d, "### Requirement: Stray") && f.File == deltaAt("booking") {
			return
		}
	}
	t.Fatalf("findings = %+v", r.Findings)
}

func TestHeaderInsideACodeBlock(t *testing.T) {
	d := "## ADDED Requirements\n\n### Requirement: A\n\nx\n\n```\n## REMOVED Requirements\n### Requirement: Fake\n```\n\n#### Scenario: S\n\n- a\n"
	if r := lintDelta(d); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestUnpairedRename(t *testing.T) {
	d := "## RENAMED Requirements\n\n- FROM: `### Requirement: Old`\n"
	r := lintDelta(d)
	if len(r.Findings) == 0 || r.Findings[0].Rule != "unparseable" || r.Findings[0].Line != lineOf(d, "- FROM") {
		t.Fatalf("findings = %+v", r.Findings)
	}
}
