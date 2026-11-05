package lint

import (
	"errors"
	"strings"
	"testing"
)

func TestProposalOpenSpecCannotRead(t *testing.T) {
	proposal := strings.Replace(goodProposal, "## Why\n\nw\n\n", "", 1)
	plan := planWith("**Scenarios:** booking#join")
	f := &fakeOS{showErr: errors.New("no Why")}
	rep := lintDeltas(t, f, map[string]string{"openspec/changes/add-x/proposal.md": proposal, pl: plan, dp("booking"): doc(req("Join", "Join"))})
	if !has(rep, "proposal-section", "openspec/changes/add-x/proposal.md", -1, "Why") || count(rep, "openspec-failed") != 1 ||
		count(rep, "scenario-unclaimed") != 0 || count(rep, "scenario-ref") != 0 {
		t.Fatalf("%+v", rep.Findings)
	}
}
