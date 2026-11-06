package setup

import (
	"slices"
	"strconv"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/markdown"
)

const workflowPath = "docket/workflow.md"

// WorkflowState is what docket/workflow.md says, for dkt init and dkt inspect.
// Version is the file's template version, 0 when it has none. Missing lists the
// template's level-3 headings the file lacks, and is nil unless Version == Shipped.
type WorkflowState struct {
	Version int
	Shipped int
	Missing []string
}

// templateVersion reads the positive integer in the front matter's template field, 0 when there is none.
func templateVersion(b []byte) int {
	fm, ok, err := change.ReadFrontMatter(b)
	if !ok || err != nil {
		return 0
	}
	n, err := strconv.Atoi(fm.Fields["template"].Value)
	if err != nil || n < 1 {
		return 0
	}
	return n
}

func headingTexts(b []byte, level int) []string {
	lines := markdown.Lines(b)
	var out []string
	mask := markdown.FenceMask(lines)
	for _, h := range markdown.Headings(markdown.BlankComments(lines, mask), mask) {
		if level == 0 || h.Level == level {
			out = append(out, h.Text)
		}
	}
	return out
}

// decideWorkflow settles the docket/workflow.md target: written when missing,
// never replaced, so force makes no difference.
func decideWorkflow(d Decision, got, want []byte, shipped int) Decision {
	if !d.existed {
		d.Status, d.Action = StatusMissing, ActionCreate
		return d
	}
	st := WorkflowState{Version: templateVersion(got), Shipped: shipped}
	if st.Version == shipped {
		have := headingTexts(got, 0)
		for _, h := range headingTexts(want, 3) {
			if !slices.Contains(have, h) {
				st.Missing = append(st.Missing, h)
			}
		}
	}
	d.Workflow, d.Action = &st, ActionKeep
	switch {
	case st.Version > 0 && st.Version < shipped:
		d.Status = StatusOutdated
	case st.Version != shipped || len(st.Missing) > 0:
		d.Status = StatusDiffers
	default:
		d.Status = StatusSame
	}
	return d
}
