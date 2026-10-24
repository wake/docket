package lint

import (
	"strings"
	"testing"
)

func TestFirstContractSectionIsTheOneChecked(t *testing.T) {
	i := strings.Index(validProposal, "### Observable Behavior")
	j := strings.Index(validProposal, "## Change Log")
	contract := validProposal[i:j]
	// The first Implementation Contract loses its subsections; a duplicate
	// placed before the Change Log keeps all four.
	doc := validProposal[:i] + "## Implementation Contract\n\n" + contract + validProposal[j:]
	r := lintProposal(t, doc)
	line := lineOf(doc, "## Implementation Contract")
	if len(r.Findings) != 4 {
		t.Fatalf("findings = %+v, want four", r.Findings)
	}
	for _, f := range r.Findings {
		if f.Rule != "proposal-section" || f.Line != line {
			t.Fatalf("finding = %+v, want line %d", f, line)
		}
	}
}
