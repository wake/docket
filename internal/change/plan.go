package change

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/wake/docket/internal/markdown"
)

// Task is one `### Task N: <name>` section of plan.md. Line is 1-based.
type Task struct {
	Number int
	Name   string
	Line   int
	Steps  []Step
}

// Plan is what ticking needs from plan.md: its tasks in order, and the numbers
// listed under `## Cancelled Tasks`.
type Plan struct {
	Tasks     []Task
	Cancelled []int
}

var (
	taskHeadingRe = regexp.MustCompile(`^Task ([1-9][0-9]*): (\S.*)$`)
	cancelledNum  = regexp.MustCompile(`^[-*+] Task ([1-9][0-9]*)\b`)
)

// ParsePlan reads the tasks (with their checkbox steps) and the cancelled task
// numbers of plan.md.
func ParsePlan(b []byte) Plan {
	lines := markdown.Lines(b)
	mask := markdown.FenceMask(lines)
	blanked := markdown.BlankComments(lines, mask)
	var p Plan
	for _, h := range markdown.Headings(lines, mask) {
		m := taskHeadingRe.FindStringSubmatch(h.Text)
		if h.Level == 3 && m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			t := Task{Number: n, Name: strings.TrimSpace(m[2]), Line: h.Index + 1}
			from, to := markdown.Body(lines, mask, h)
			for i := from; i < to; i++ {
				if sm := stepRe.FindStringSubmatch(blanked[i]); !mask[i] && sm != nil {
					t.Steps = append(t.Steps, Step{Line: i + 1, Done: sm[1] != " "})
				}
			}
			p.Tasks = append(p.Tasks, t)
		} else if h.Level == 2 && strings.EqualFold(h.Text, "Cancelled Tasks") {
			from, to := markdown.Body(lines, mask, h)
			for i := from; i < to; i++ {
				if m := cancelledNum.FindStringSubmatch(lines[i]); !mask[i] && m != nil {
					if n, err := strconv.Atoi(m[1]); err == nil {
						p.Cancelled = append(p.Cancelled, n)
					}
				}
			}
		}
	}
	return p
}
