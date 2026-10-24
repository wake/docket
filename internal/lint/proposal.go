package lint

import (
	"regexp"
	"strings"
)

const proposalFile = "proposal.md"

var proposalSections = []string{"Why", "What Changes", "Non-Goals", "Builds On", "Capabilities", "Impact", "Implementation Contract", "Change Log"}
var contractSections = []string{"Observable Behavior", "Interfaces", "Failure Modes", "Acceptance Criteria"}
var mustHoldText = map[string]bool{"Non-Goals": true, "Builds On": true}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// checkProposal applies the proposal-section and change-log rules.
func checkProposal(r *run, c *loaded) {
	if c.Skipped(proposalFile) {
		return
	}
	if !c.Has(proposalFile) {
		r.add(c, "proposal-section", "", 0, "proposal.md is missing")
		return
	}
	p := c.Proposal
	for _, title := range proposalSections {
		s, ok := p.Section(title)
		if !ok {
			r.add(c, "proposal-section", proposalFile, 0, "section %q is missing", title)
			continue
		}
		if mustHoldText[title] && s.Body == "" {
			r.add(c, "proposal-section", proposalFile, s.Line, "section %q is empty", title)
		}
		switch title {
		case "Implementation Contract":
			end := 0
			for _, next := range p.Sections {
				if next.Line > s.Line && (end == 0 || next.Line < end) {
					end = next.Line
				}
			}
			for _, sub := range contractSections {
				found := false
				for _, cs := range p.Contract {
					if cs.Line < s.Line || (end != 0 && cs.Line >= end) {
						continue
					}
					if !strings.EqualFold(cs.Title, sub) {
						continue
					}
					found = true
					if cs.Body == "" {
						r.add(c, "proposal-section", proposalFile, cs.Line, "section %q is empty", sub)
					}
					break
				}
				if !found {
					r.add(c, "proposal-section", proposalFile, s.Line, "section %q is missing from Implementation Contract", sub)
				}
			}
		case "Change Log":
			if p.ChangeLog == nil {
				r.add(c, "change-log", proposalFile, s.Line, "Change Log has no table")
				continue
			}
			for _, row := range p.ChangeLog.Rows {
				if len(row.Cells) < 2 || !dateRe.MatchString(strings.TrimSpace(row.Cells[0])) {
					r.add(c, "change-log", proposalFile, row.Index+1, "Change Log row needs a date in the form YYYY-MM-DD")
				} else if strings.TrimSpace(row.Cells[1]) == "" {
					r.add(c, "change-log", proposalFile, row.Index+1, "Change Log row needs a Source")
				}
			}
		}
	}
}
