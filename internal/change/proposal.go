package change

import (
	"strings"

	"github.com/wake/docket/internal/markdown"
)

// Section is one heading and its text. Line is 1-based.
type Section struct {
	Title string
	Line  int
	Body  string
}

// Proposal is the structure of proposal.md.
type Proposal struct {
	Sections  []Section
	Contract  []Section
	ChangeLog *markdown.Table
}

// ParseProposal reads the level-2 sections of proposal.md, the level-3
// sections of its Implementation Contract and the Change Log table.
func ParseProposal(b []byte) Proposal {
	lines := markdown.Lines(b)
	mask := markdown.FenceMask(lines)
	var p Proposal
	for _, h := range markdown.Headings(lines, mask) {
		if h.Level != 2 {
			continue
		}
		p.Sections = append(p.Sections, sectionOf(lines, mask, h))
		from, to := markdown.Body(lines, mask, h)
		switch {
		case strings.EqualFold(h.Text, "Implementation Contract"):
			for _, sub := range markdown.Headings(lines[:to], mask[:to]) {
				if sub.Index >= from && sub.Level == 3 {
					p.Contract = append(p.Contract, sectionOf(lines, mask, sub))
				}
			}
		case strings.EqualFold(h.Text, "Change Log") && p.ChangeLog == nil:
			if t, ok := markdown.FindTable(lines, mask, from, to); ok {
				p.ChangeLog = &t
			}
		}
	}
	return p
}

func sectionOf(lines []string, mask []bool, h markdown.Heading) Section {
	from, to := markdown.Body(lines, mask, h)
	body := markdown.StripComments(strings.Join(lines[from:to], "\n"))
	return Section{Title: h.Text, Line: h.Index + 1, Body: strings.TrimSpace(body)}
}

// Section returns the first level-2 section with the title, ignoring case.
func (p Proposal) Section(title string) (Section, bool) {
	for _, s := range p.Sections {
		if strings.EqualFold(s.Title, title) {
			return s, true
		}
	}
	return Section{}, false
}
