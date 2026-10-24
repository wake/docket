package lint

import (
	"strings"
	"testing"
)

func lineOf(doc, needle string) int {
	i := strings.Index(doc, needle)
	return strings.Count(doc[:i], "\n") + 1
}

func lintProposal(t *testing.T, proposal string) Report {
	t.Helper()
	return runOn(repoWith(map[string]string{"openspec/changes/add-x/proposal.md": proposal}))
}

func TestProposalMissing(t *testing.T) {
	r := runOn(repoWith(map[string]string{"openspec/changes/add-x/notes.txt": "x"}))
	f := wantOne(t, r, "proposal-section", 0)
	if f.File != "" || f.Change != "add-x" {
		t.Fatalf("finding = %+v", f)
	}
}

func TestNonGoalsMissing(t *testing.T) {
	doc := strings.Replace(validProposal, "## Non-Goals\n\n- Not this.\n\n", "", 1)
	f := wantOne(t, lintProposal(t, doc), "proposal-section", 0)
	if !strings.Contains(f.Message, "Non-Goals") || f.File != "openspec/changes/add-x/proposal.md" {
		t.Fatalf("finding = %+v", f)
	}
}

func TestContractSubsectionMissing(t *testing.T) {
	doc := strings.Replace(validProposal, "### Interfaces\n\nA command.\n\n", "", 1)
	f := wantOne(t, lintProposal(t, doc), "proposal-section", lineOf(validProposal, "## Implementation Contract"))
	if !strings.Contains(f.Message, "Interfaces") {
		t.Fatalf("finding = %+v", f)
	}
}

func TestEmptyFailureModes(t *testing.T) {
	comment := templateLine(t, "<!-- What happens when things go wrong")
	doc := strings.Replace(validProposal, "It fails loudly.", comment, 1)
	r := lintProposal(t, doc)
	if len(r.Findings) != 2 {
		t.Fatalf("findings = %+v, want proposal-section and placeholder", r.Findings)
	}
	var sec, ph *Finding
	for i := range r.Findings {
		switch r.Findings[i].Rule {
		case "proposal-section":
			sec = &r.Findings[i]
		case "placeholder":
			ph = &r.Findings[i]
		}
	}
	if sec == nil || ph == nil {
		t.Fatalf("findings = %+v", r.Findings)
	}
	if sec.Line != lineOf(doc, "### Failure Modes") || !strings.Contains(sec.Message, "Failure Modes") {
		t.Fatalf("section finding = %+v", sec)
	}
	if ph.Line != lineOf(doc, comment) {
		t.Fatalf("placeholder finding = %+v", ph)
	}
}

func TestNonGoalsEmpty(t *testing.T) {
	for name, body := range map[string]string{"blank": "\n", "comment": "\n<!-- nothing yet -->\n"} {
		doc := strings.Replace(validProposal, "\n- Not this.\n", body, 1)
		f := wantOne(t, lintProposal(t, doc), "proposal-section", lineOf(doc, "## Non-Goals"))
		if !strings.Contains(f.Message, "Non-Goals") {
			t.Fatalf("%s: finding = %+v", name, f)
		}
	}
}

func TestBuildsOnLeftEmpty(t *testing.T) {
	doc := strings.Replace(validProposal, "- internal/change: ParseProposal", "", 1)
	f := wantOne(t, lintProposal(t, doc), "proposal-section", lineOf(validProposal, "## Builds On"))
	if !strings.Contains(f.Message, "Builds On") {
		t.Fatalf("finding = %+v", f)
	}
}

func TestChangeLogRowWithoutSource(t *testing.T) {
	doc := strings.Replace(validProposal, "| 2026-10-24 | review |", "| 2026-10-24 |  |", 1)
	wantOne(t, lintProposal(t, doc), "change-log", lineOf(validProposal, "| 2026-10-24"))
}

func TestChangeLogRowWithBadDate(t *testing.T) {
	doc := strings.Replace(validProposal, "| 2026-10-24 |", "| Oct 24 |", 1)
	wantOne(t, lintProposal(t, doc), "change-log", lineOf(validProposal, "| 2026-10-24"))
}

func TestHeaderOnlyChangeLog(t *testing.T) {
	doc := strings.Replace(validProposal, "| 2026-10-24 | review | first | none |\n", "", 1)
	if r := lintProposal(t, doc); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestChangeLogWithoutTable(t *testing.T) {
	i := strings.Index(validProposal, "| Date")
	doc := validProposal[:i] + "Nothing yet.\n"
	wantOne(t, lintProposal(t, doc), "change-log", lineOf(validProposal, "## Change Log"))
}
