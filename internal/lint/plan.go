package lint

import (
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/wake/docket/internal/change"
)

const planFile = "plan.md"

var reuseHeader = []string{"Need", "Existing (path:symbol)", "Decision", "Reason", "Evidence"}

// checkPlan applies the plan rules to a change that has a plan.md.
func checkPlan(r *run, c *loaded) {
	p := c.Plan
	checkTaskHeadings(r, c, planFile, p)
	checkReuse(r, c)
	cancelled := map[int]bool{}
	for _, x := range p.Cancelled {
		if !x.OK {
			r.add(c, "cancelled-task", planFile, x.Line,
				"a cancelled task must read `- Task <N>: <name> - <reason>`, with a reason")
		}
		if x.Number > 0 {
			cancelled[x.Number] = true
		}
	}
	tasks := map[int]bool{}
	for _, t := range p.Tasks {
		tasks[t.Number] = true
	}
	for _, t := range p.Tasks {
		for _, ref := range t.Compensates {
			switch {
			case !ref.OK:
				r.add(c, "compensates", planFile, ref.Line,
					"Task %d: a Compensates line must read `**Compensates:** Task <N>`", t.Number)
			case cancelled[ref.Number]:
				r.add(c, "compensates", planFile, ref.Line,
					"Task %d compensates Task %d, which is cancelled", t.Number, ref.Number)
			case !tasks[ref.Number]:
				r.add(c, "compensates", planFile, ref.Line,
					"Task %d compensates Task %d, which is not a task in this plan", t.Number, ref.Number)
			}
		}
	}
}

// checkTaskHeadings reports the headings of p that are not task headings and
// the task numbers it uses more than once, in file.
func checkTaskHeadings(r *run, c *loaded, file string, p change.Plan) {
	for _, h := range p.BadHeadings {
		r.add(c, "task-heading", file, h.Line,
			"%q at heading level %d is not a task heading; use `### Task <N>: <name>`", h.Text, h.Level)
	}
	checkTaskNumbers(r, c, file, p)
}

// checkTaskNumbers reports each task number used more than once across the
// tasks and the cancelled bullets, on the later line.
func checkTaskNumbers(r *run, c *loaded, file string, p change.Plan) {
	lines := map[int][]int{}
	for _, t := range p.Tasks {
		lines[t.Number] = append(lines[t.Number], t.Line)
	}
	for _, x := range p.Cancelled {
		if x.Number > 0 {
			lines[x.Number] = append(lines[x.Number], x.Line)
		}
	}
	var nums []int
	for n, ls := range lines {
		if len(ls) > 1 {
			nums = append(nums, n)
		}
	}
	sort.Ints(nums)
	for _, n := range nums {
		ls := lines[n]
		sort.Ints(ls)
		parts := make([]string, len(ls))
		for i, l := range ls {
			parts[i] = strconv.Itoa(l)
		}
		r.add(c, "task-heading", file, ls[len(ls)-1],
			"task number %d is used more than once (lines %s)", n, strings.Join(parts, ", "))
	}
}

func checkReuse(r *run, c *loaded) {
	p := c.Plan
	if p.ReuseLine == 0 {
		r.add(c, "reuse-table", planFile, 0, "plan.md has no `## Reuse` section")
		return
	}
	if p.Reuse == nil {
		r.add(c, "reuse-table", planFile, p.ReuseLine, "the Reuse section has no table")
		return
	}
	t := p.Reuse
	if !sameHeader(t.Header) {
		r.add(c, "reuse-table", planFile, t.Index+1,
			"the Reuse table header must be: %s", strings.Join(reuseHeader, " | "))
		return
	}
	if len(t.Rows) == 0 {
		r.add(c, "reuse-table", planFile, t.Index+1, "the Reuse table has no rows")
		return
	}
	for _, row := range t.Rows {
		rl := row.Index + 1
		need := strings.TrimSpace(row.Cells[0])
		existing := strings.TrimSpace(strings.ReplaceAll(row.Cells[1], "`", ""))
		decision := strings.TrimSpace(strings.ReplaceAll(row.Cells[2], "`", ""))
		switch decision {
		case "reuse", "extend":
			if !strings.HasPrefix(existing, "external:") {
				checkExistingPath(r, c, rl, need, existing)
			}
		case "new":
			if strings.TrimSpace(row.Cells[3]) == "" {
				r.add(c, "reuse-table", planFile, rl, "%q: a `new` row needs a Reason", need)
			}
		default:
			r.add(c, "reuse-table", planFile, rl,
				"%q: Decision %q must be reuse, extend or new", need, decision)
		}
		if strings.TrimSpace(row.Cells[4]) == "" {
			r.add(c, "reuse-table", planFile, rl, "%q: Evidence is empty", need)
		}
	}
}

func checkExistingPath(r *run, c *loaded, line int, need, existing string) {
	before, _, _ := strings.Cut(existing, ":")
	p := strings.TrimSuffix(strings.TrimSpace(before), "/")
	if p == "" || !fs.ValidPath(p) {
		r.add(c, "reuse-table", planFile, line,
			"%q: Existing must start with a repository path or `external:`", need)
		return
	}
	if _, err := fs.Stat(r.in.Repo, p); err != nil {
		r.add(c, "reuse-table", planFile, line, "%q: path %q does not exist in the repository", need, p)
	}
}

func sameHeader(h []string) bool {
	if len(h) != len(reuseHeader) {
		return false
	}
	for i := range h {
		if !strings.EqualFold(strings.TrimSpace(h[i]), reuseHeader[i]) {
			return false
		}
	}
	return true
}
