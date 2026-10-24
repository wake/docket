// Package delta reads delta specs and main specs the way OpenSpec 1.14.3
// does, so that linting needs no OpenSpec at run time. Line numbers in the
// results are 1-based.
package delta

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/wake/docket/internal/markdown"
)

// Scenario is a scenario heading under a requirement.
type Scenario struct {
	Name string
	Line int
}

// Requirement is a requirement header with the scenarios directly under it.
type Requirement struct {
	Name      string
	Line      int
	Scenarios []Scenario
}

// Rename is a FROM/TO pair; Line is the line of the FROM entry.
type Rename struct {
	From, To string
	Line     int
}

// Problem is something OpenSpec would silently ignore or refuse.
type Problem struct {
	Line    int
	Message string
}

// Delta is the content of one delta spec, each list in document order.
// Removed entries carry no scenarios.
type Delta struct {
	Added, Modified, Removed []Requirement
	Renamed                  []Rename
	Problems                 []Problem
}

const sp = markdown.Space

var (
	requirementRe = regexp.MustCompile(`(?i)^###` + sp + `*Requirement:` + sp + `*(.+)` + sp + `*$`)
	sectionRe     = regexp.MustCompile(`^(##)` + sp + `+(.+)$`)
	topLevelRe    = regexp.MustCompile(`^##` + sp + `+`)
	specSectionRe = regexp.MustCompile(`(?i)^##` + sp + `+Requirements` + sp + `*$`)
	removedBullet = regexp.MustCompile("^" + sp + "*[-*+]" + sp + "*`?###" + sp + "*Requirement:" + sp + "*(.+?)`?" + sp + "*$")
	fromRe        = regexp.MustCompile("^" + sp + "*[-*+]?" + sp + "*FROM:" + sp + "*`?###" + sp + "*Requirement:" + sp + "*(.+?)`?" + sp + "*$")
	toRe          = regexp.MustCompile("^" + sp + "*[-*+]?" + sp + "*TO:" + sp + "*`?###" + sp + "*Requirement:" + sp + "*(.+?)`?" + sp + "*$")
	scenarioPre   = regexp.MustCompile(`(?i)^Scenario:` + sp + `*`)
	closingRun    = regexp.MustCompile(`[ \t]+#+[ \t]*$`)
	heading3Re    = regexp.MustCompile(`^###` + sp)
	titleRe       = regexp.MustCompile(`^#` + sp)
)

type doc struct {
	lines []string
	mask  []bool
	heads []markdown.Heading
}

func read(b []byte) doc {
	lines := markdown.Lines(b)
	mask := markdown.FenceMask(lines)
	return doc{lines, mask, markdown.Headings(lines, mask)}
}

func reqName(s string) string {
	return strings.TrimSpace(closingRun.ReplaceAllString(s, ""))
}

type section struct {
	title      string // trimmed
	index      int    // 0-based line of the "## " line
	start, end int    // body line range [start, end)
}

func (d doc) sections() []section {
	var out []section
	for i, l := range d.lines {
		if d.mask[i] {
			continue
		}
		if m := sectionRe.FindStringSubmatch(l); m != nil {
			out = append(out, section{title: strings.TrimSpace(m[2]), index: i, start: i + 1})
		}
	}
	for k := range out {
		if k+1 < len(out) {
			out[k].end = out[k+1].index
		} else {
			out[k].end = len(d.lines)
		}
	}
	return out
}

// requirements reads the requirement blocks in lines [from, to).
func (d doc) requirements(from, to int) []Requirement {
	var out []Requirement
	for i := from; i < to; i++ {
		if d.mask[i] {
			continue
		}
		m := requirementRe.FindStringSubmatch(d.lines[i])
		if m == nil {
			continue
		}
		end := i + 1
		for end < to {
			if !d.mask[end] && (requirementRe.MatchString(d.lines[end]) || topLevelRe.MatchString(d.lines[end])) {
				break
			}
			end++
		}
		out = append(out, Requirement{Name: reqName(m[1]), Line: i + 1, Scenarios: d.scenarios(i, end)})
		i = end - 1
	}
	return out
}

// scenarios returns the headings directly under the requirement at line
// index req, within the block ending at end.
func (d doc) scenarios(req, end int) []Scenario {
	var out []Scenario
	stack := []int{3}
	for _, h := range d.heads {
		if h.Index <= req || h.Index >= end {
			continue
		}
		if h.Level <= 3 {
			break
		}
		for len(stack) > 1 && stack[len(stack)-1] >= h.Level {
			stack = stack[:len(stack)-1]
		}
		direct := len(stack) == 1
		stack = append(stack, h.Level)
		if !direct {
			continue
		}
		from, to := markdown.Body(d.lines, d.mask, h)
		if to > end {
			to = end
		}
		if strings.TrimSpace(strings.Join(d.lines[from:to], "\n")) == "" {
			continue
		}
		name := strings.TrimSpace(scenarioPre.ReplaceAllString(h.Text, ""))
		out = append(out, Scenario{Name: name, Line: h.Index + 1})
	}
	return out
}

// Parse reads one delta spec.
func Parse(b []byte) Delta {
	d := read(b)
	secs := d.sections()
	var out Delta
	var problems []Problem
	add := func(line int, format string, args ...any) {
		problems = append(problems, Problem{Line: line, Message: fmt.Sprintf(format, args...)})
	}
	comments := d.commentBlanked()
	found := false
	firstSection := len(d.lines)
	if len(secs) > 0 {
		firstSection = secs[0].index
	}
	for i := 0; i < firstSection; i++ {
		if strings.TrimSpace(comments[i]) != "" && !titleRe.MatchString(comments[i]) {
			add(i+1, "text above the first \"## \" section is not read")
			break
		}
	}
	for _, s := range secs {
		switch strings.ToLower(s.title) {
		case "added requirements", "modified requirements":
			found = true
			reqs := d.requirements(s.start, s.end)
			if strings.ToLower(s.title) == "added requirements" {
				out.Added = append(out.Added, reqs...)
			} else {
				out.Modified = append(out.Modified, reqs...)
			}
			limit := s.end
			for i := s.start; i < s.end; i++ {
				if !d.mask[i] && requirementRe.MatchString(d.lines[i]) {
					limit = i
					break
				}
			}
			for i := s.start; i < limit; i++ {
				if strings.TrimSpace(comments[i]) != "" && !heading3Re.MatchString(comments[i]) {
					add(i+1, "text before the first requirement in \"## %s\" is not read", s.title)
					break
				}
			}
		case "removed requirements":
			found = true
			out.Removed = append(out.Removed, d.removed(s)...)
		case "renamed requirements":
			found = true
			pairs, bad := d.renamed(s)
			out.Renamed = append(out.Renamed, pairs...)
			problems = append(problems, bad...)
		case "purpose":
		default:
			add(s.index+1, "section \"## %s\" is not read", s.title)
		}
	}
	// Requirement headers outside the four delta sections.
	cur := ""
	inSection := false
	for i, l := range d.lines {
		if d.mask[i] {
			continue
		}
		if m := sectionRe.FindStringSubmatch(l); m != nil {
			cur, inSection = strings.ToLower(strings.TrimSpace(m[2])), true
			continue
		}
		if inSection {
			switch cur {
			case "added requirements", "modified requirements", "removed requirements", "renamed requirements":
				continue
			}
		}
		if requirementRe.MatchString(l) {
			add(i+1, "requirement header outside any delta section is not read")
		}
	}
	if !found {
		add(1, "no ADDED, MODIFIED, REMOVED or RENAMED Requirements section")
	}
	sort.SliceStable(problems, func(i, j int) bool { return problems[i].Line < problems[j].Line })
	out.Problems = problems
	return out
}

func (d doc) removed(s section) []Requirement {
	var out []Requirement
	for i := s.start; i < s.end; i++ {
		if d.mask[i] {
			continue
		}
		if m := requirementRe.FindStringSubmatch(d.lines[i]); m != nil {
			out = append(out, Requirement{Name: reqName(m[1]), Line: i + 1})
		} else if m := removedBullet.FindStringSubmatch(d.lines[i]); m != nil {
			out = append(out, Requirement{Name: reqName(m[1]), Line: i + 1})
		}
	}
	return out
}

func (d doc) renamed(s section) ([]Rename, []Problem) {
	var pairs []Rename
	var bad []Problem
	var pending *Rename
	drop := func(side string, name string, line int) {
		bad = append(bad, Problem{Line: line, Message: fmt.Sprintf("%s entry for %q has no matching %s", side, name, map[string]string{"FROM": "TO", "TO": "FROM"}[side])})
	}
	for i := s.start; i < s.end; i++ {
		if d.mask[i] {
			continue
		}
		if m := fromRe.FindStringSubmatch(d.lines[i]); m != nil {
			if pending != nil {
				drop("FROM", pending.From, pending.Line)
			}
			pending = &Rename{From: reqName(m[1]), Line: i + 1}
		} else if m := toRe.FindStringSubmatch(d.lines[i]); m != nil {
			to := reqName(m[1])
			if pending == nil {
				drop("TO", to, i+1)
				continue
			}
			pending.To = to
			pairs = append(pairs, *pending)
			pending = nil
		}
	}
	if pending != nil {
		drop("FROM", pending.From, pending.Line)
	}
	return pairs, bad
}

// commentBlanked returns the lines with HTML comments outside fences
// replaced by spaces; an unclosed comment runs to the end. Fenced lines keep
// their text.
func (d doc) commentBlanked() []string {
	var b strings.Builder
	for i, l := range d.lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if !d.mask[i] {
			b.WriteString(l)
		}
	}
	s := b.String()
	var o strings.Builder
	for {
		i := strings.Index(s, "<!--")
		if i < 0 {
			o.WriteString(s)
			break
		}
		o.WriteString(s[:i])
		rest := s[i:]
		n := len(rest)
		body := rest[4:]
		j, k := strings.Index(body, "-->"), strings.Index(body, "--!>")
		switch {
		case j >= 0 && (k < 0 || j < k):
			n = 4 + j + 3
		case k >= 0:
			n = 4 + k + 4
		}
		for _, r := range rest[:n] {
			if r == '\n' {
				o.WriteByte('\n')
			} else {
				o.WriteByte(' ')
			}
		}
		s = rest[n:]
	}
	out := strings.Split(o.String(), "\n")
	for i := range out {
		if d.mask[i] {
			out[i] = d.lines[i]
		}
	}
	return out
}

// ParseSpec reads the requirements in the first "## Requirements" section of
// a main spec.
func ParseSpec(b []byte) []Requirement {
	d := read(b)
	for i, l := range d.lines {
		if d.mask[i] || !specSectionRe.MatchString(l) {
			continue
		}
		end := len(d.lines)
		for j := i + 1; j < len(d.lines); j++ {
			if !d.mask[j] && topLevelRe.MatchString(d.lines[j]) {
				end = j
				break
			}
		}
		return d.requirements(i+1, end)
	}
	return nil
}
