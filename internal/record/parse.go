package record

import (
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/markdown"
)

// Section is one level-two section of a record. Line is the 1-based line of
// its heading; Text is everything under it up to the next level-one or
// level-two heading, with HTML comments blanked.
type Section struct {
	Name string
	Line int
	Text string
}

// Parsed is the structure of a record file.
type Parsed struct {
	Front    change.Front
	HasFront bool
	FrontErr error
	Sections []Section
	Outside  []int       // 1-based lines with text outside every section
	Tasks    change.Plan // read from under `## Tasks`, with the true line numbers
}

// Parse reads the front matter, the level-two sections and the tasks of a record.
func Parse(b []byte) Parsed {
	var p Parsed
	p.Front, p.HasFront, p.FrontErr = change.ReadFrontMatter(b)
	lines := markdown.Lines(b)
	// The front matter is blanked, so a `# comment` line in it is no heading.
	// shortcut: front matter that never closes leaves no body to read.
	if p.HasFront {
		end := len(lines)
		for i := 1; i < len(lines); i++ {
			if lines[i] == "---" {
				end = i + 1
				break
			}
		}
		lines = append(make([]string, end), lines[end:]...)
	}
	mask := markdown.FenceMask(lines)
	blank := markdown.BlankComments(lines, mask)
	covered := make([]bool, len(lines))
	var tasksFrom, tasksTo int
	for _, h := range markdown.Headings(blank, mask) {
		if h.Level != 2 {
			continue
		}
		from, to := markdown.Body(blank, mask, h)
		p.Sections = append(p.Sections, Section{Name: h.Text, Line: h.Index + 1, Text: strings.Join(blank[from:to], "\n")})
		for i := h.Index; i < to; i++ {
			covered[i] = true
		}
		if tasksTo == 0 && strings.EqualFold(h.Text, "Tasks") {
			tasksFrom, tasksTo = from, to
		}
	}
	for i, l := range blank {
		if !covered[i] && markdown.Trim(l) != "" {
			p.Outside = append(p.Outside, i+1)
		}
	}
	if tasksTo > 0 {
		only := make([]string, len(lines))
		copy(only[tasksFrom:tasksTo], lines[tasksFrom:tasksTo])
		p.Tasks = change.ParsePlan([]byte(strings.Join(only, "\n")))
	}
	return p
}

// BodySections are the sections every record holds, in order; they count toward BodyLimit.
var BodySections = []string{"What", "Why", "Verification", "Review"}

// BodySize is the sum of change.Size over the four named sections.
func (p Parsed) BodySize() int {
	n := 0
	for _, s := range p.Sections {
		for _, name := range BodySections {
			if strings.EqualFold(s.Name, name) {
				n += change.Size(s.Text)
			}
		}
	}
	return n
}
