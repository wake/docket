package change

import (
	"testing"
)

const proposalFixture = "## Why\n\n<!-- template comment -->\n\n## Non-Goals\n\nnone\n\n## Implementation Contract\n\n### Scope\n\nA\n\n### Data\n\nB\n\n## Change Log\n\n| Date | Change |\n|---|---|\n| 2026-01-01 | first |\n"

func TestParseProposalSections(t *testing.T) {
	p := ParseProposal([]byte(proposalFixture))
	want := []string{"Why", "Non-Goals", "Implementation Contract", "Change Log"}
	if len(p.Sections) != len(want) {
		t.Fatalf("sections = %+v", p.Sections)
	}
	for i, s := range p.Sections {
		if s.Title != want[i] {
			t.Fatalf("title %d = %q", i, s.Title)
		}
	}
	if s, ok := p.Section("why"); !ok || s.Body != "" || s.Line != 1 {
		t.Fatalf("why = %+v ok=%v", s, ok)
	}
	if s, _ := p.Section("NON-GOALS"); s.Body != "none" || s.Line != 5 {
		t.Fatalf("non-goals = %+v", s)
	}
	if _, ok := p.Section("missing"); ok {
		t.Fatal("found missing section")
	}
	if len(p.Contract) != 2 || p.Contract[0].Title != "Scope" || p.Contract[0].Body != "A" || p.Contract[1].Body != "B" {
		t.Fatalf("contract = %+v", p.Contract)
	}
	if p.ChangeLog == nil || len(p.ChangeLog.Rows) != 1 || p.ChangeLog.Rows[0].Cells[1] != "first" {
		t.Fatalf("change log = %+v", p.ChangeLog)
	}
}

func TestProposalCRLF(t *testing.T) {
	b := []byte("\xef\xbb\xbf## Why\r\n\r\nbecause\r\n\r\n## Non-Goals\r\nnone\r\n")
	p := ParseProposal(b)
	if len(p.Sections) != 2 || p.Sections[0].Body != "because" || p.Sections[1].Line != 5 {
		t.Fatalf("sections = %+v", p.Sections)
	}
}

func TestFencedHeadingsIgnored(t *testing.T) {
	p := ParseProposal([]byte("## Why\n\n```\n## Non-Goals\n```\n"))
	if len(p.Sections) != 1 {
		t.Fatalf("sections = %+v", p.Sections)
	}
	if _, ok := p.Section("Non-Goals"); ok {
		t.Fatal("fenced heading is a section")
	}
	pl := ParsePlan([]byte("### Task 1: a\n\n```\n### Task 9: x\n```\n"))
	if len(pl.Tasks) != 1 || len(pl.BadHeadings) != 0 {
		t.Fatalf("plan = %+v", pl)
	}
}
