package setup

import (
	"slices"
	"strconv"
	"strings"
	"unicode"

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

// TemplateVersion reads the positive integer in the front matter's template field, 0 when there is none.
func TemplateVersion(b []byte) int {
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

// Section is one level-three section of a workflow file; Text is normalized so that
// only edits a reader can see make texts differ.
type Section struct {
	Heading, Text string
}

// Sections returns the level-three sections in file order, the first of a repeated
// heading only. A section runs to the next heading of level three or above; its text
// has no fenced-code-masked heading, no HTML comment, no trailing whitespace on a line
// and no blank line at either end, and its lines are joined with LF.
func Sections(b []byte) []Section {
	lines := markdown.Lines(b)
	mask := markdown.FenceMask(lines)
	blanked := markdown.BlankComments(lines, mask)
	out := []Section{}
	seen := map[string]bool{}
	for _, h := range markdown.Headings(blanked, mask) {
		if h.Level != 3 || seen[h.Text] {
			continue
		}
		seen[h.Text] = true
		from, to := markdown.Body(blanked, mask, h)
		body := make([]string, 0, to-from)
		for _, l := range blanked[from:to] {
			body = append(body, strings.TrimRightFunc(l, unicode.IsSpace))
		}
		for len(body) > 0 && body[0] == "" {
			body = body[1:]
		}
		for len(body) > 0 && body[len(body)-1] == "" {
			body = body[:len(body)-1]
		}
		out = append(out, Section{h.Text, strings.Join(body, "\n")})
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
	st := WorkflowState{Version: TemplateVersion(got), Shipped: shipped}
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
