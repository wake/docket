package change

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/wake/docket/internal/markdown"
)

// Task is one `### Task N: <name>` section of plan.md. Line fields are
// 1-based; ScenariosLine is 0 when the task has no Scenarios line.
type Task struct {
	Number        int
	Name          string
	Line          int
	Scenarios     []string
	ScenariosLine int
	Compensates   []Ref
}

// Ref is one Compensates line; OK is false when it is not exactly
// `**Compensates:** Task N`.
type Ref struct {
	Number int
	Line   int
	OK     bool
}

// BadHeading is a heading that looks like a task heading but is not one.
type BadHeading struct {
	Text  string
	Level int
	Line  int
}

// Cancelled is one bullet under `## Cancelled Tasks`.
type Cancelled struct {
	Number       int
	Name, Reason string
	Line         int
	OK           bool
}

// Plan is the structure of plan.md.
type Plan struct {
	Tasks       []Task
	BadHeadings []BadHeading
	Cancelled   []Cancelled
	ReuseLine   int
	Reuse       *markdown.Table
}

var (
	taskHeadingRe = regexp.MustCompile(`^Task ([1-9][0-9]*): (\S.*)$`)
	taskLikeRe    = regexp.MustCompile(`^Task\s+\d`)
	compensatesRe = regexp.MustCompile(`^\*\*Compensates:\*\* Task ([1-9][0-9]*)$`)
	cancelledRe   = regexp.MustCompile(`^- Task ([1-9][0-9]*): (\S.*?) - (\S.*)$`)
	cancelledNum  = regexp.MustCompile(`^- Task ([1-9][0-9]*)\b`)
)

const scenariosPrefix = "**Scenarios:**"

// ParsePlan reads tasks, malformed task headings, cancelled tasks and the
// Reuse table of plan.md.
func ParsePlan(b []byte) Plan {
	lines := markdown.Lines(b)
	mask := markdown.FenceMask(lines)
	var p Plan
	for _, h := range markdown.Headings(lines, mask) {
		m := taskHeadingRe.FindStringSubmatch(h.Text)
		if h.Level == 3 && m != nil {
			n, _ := strconv.Atoi(m[1])
			_, to := markdown.Body(lines, mask, h)
			p.Tasks = append(p.Tasks, readTask(lines, mask, h.Index, to, n, m[2]))
			continue
		}
		if taskLikeRe.MatchString(h.Text) {
			p.BadHeadings = append(p.BadHeadings, BadHeading{Text: h.Text, Level: h.Level, Line: h.Index + 1})
			continue
		}
		if h.Level != 2 {
			continue
		}
		from, to := markdown.Body(lines, mask, h)
		switch {
		case strings.EqualFold(h.Text, "Cancelled Tasks"):
			for i := from; i < to; i++ {
				if !mask[i] && strings.HasPrefix(lines[i], "- ") {
					p.Cancelled = append(p.Cancelled, readCancelled(lines[i], i+1))
				}
			}
		case strings.EqualFold(h.Text, "Reuse") && p.ReuseLine == 0:
			p.ReuseLine = h.Index + 1
			if t, ok := markdown.FindTable(lines, mask, from, to); ok {
				p.Reuse = &t
			}
		}
	}
	return p
}

func readTask(lines []string, mask []bool, head, to, number int, name string) Task {
	t := Task{Number: number, Name: strings.TrimSpace(name), Line: head + 1}
	for i := head + 1; i < to; i++ {
		if mask[i] {
			continue
		}
		line := strings.TrimSpace(lines[i])
		switch {
		case strings.HasPrefix(line, scenariosPrefix) && t.ScenariosLine == 0:
			t.ScenariosLine = i + 1
			t.Scenarios = splitScenarios(strings.TrimPrefix(line, scenariosPrefix))
		case strings.HasPrefix(line, "**Compensates:**"):
			r := Ref{Line: i + 1}
			if m := compensatesRe.FindStringSubmatch(line); m != nil {
				r.Number, _ = strconv.Atoi(m[1])
				r.OK = true
			}
			t.Compensates = append(t.Compensates, r)
		}
	}
	return t
}

func splitScenarios(s string) []string {
	s = strings.TrimSpace(s)
	if strings.EqualFold(s, "none") {
		return nil
	}
	var out []string
	for _, item := range strings.Split(s, ",") {
		item = strings.Trim(strings.TrimSpace(item), "`")
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func readCancelled(line string, number int) Cancelled {
	c := Cancelled{Line: number}
	if m := cancelledRe.FindStringSubmatch(strings.TrimRight(line, " \t")); m != nil {
		c.Number, _ = strconv.Atoi(m[1])
		c.Name, c.Reason, c.OK = m[2], m[3], true
	} else if m := cancelledNum.FindStringSubmatch(line); m != nil {
		c.Number, _ = strconv.Atoi(m[1])
	}
	return c
}
