// Package issue holds what Docket knows about an issue: where it lives and its
// sections.
package issue

import (
	"fmt"
	"slices"
	"strings"

	"github.com/wake/docket/internal/record"
)

// DroppedSection is the section that says why an issue was dropped.
const DroppedSection = "Dropped"

// Steps lists the writes to another tool that an issue's unsynced field may name.
var Steps = []string{"open", "taken", "pr", "merged", "dropped"}

// UnsyncedForm is how the unsynced field is written.
const UnsyncedForm = "unsynced: [<step>, <step>]"

// ParseUnsynced reads the value of unsynced: the steps in the order written,
// none for [].
func ParseUnsynced(v string) ([]string, error) {
	if !strings.HasPrefix(v, "[") || !strings.HasSuffix(v, "]") {
		return nil, fmt.Errorf("unsynced must be written as %s", UnsyncedForm)
	}
	steps := []string{}
	if inner := strings.TrimSpace(v[1 : len(v)-1]); inner != "" {
		for _, s := range strings.Split(inner, ",") {
			s = strings.TrimSpace(s)
			if !slices.Contains(Steps, s) {
				return nil, fmt.Errorf("%q is not a step; the steps are %s", s, strings.Join(Steps, ", "))
			}
			if slices.Contains(steps, s) {
				return nil, fmt.Errorf("step %q is given twice", s)
			}
			steps = append(steps, s)
		}
	}
	return steps, nil
}

// Dir is the folder holding the issues, relative to the repository root.
const Dir = "docket/issues"

// Sections lists the sections of an issue, in order.
var Sections = []string{"Current", "Expected", "Unchanged", "Evidence"}

// BodyLimit is the most the sections of an issue may hold, in bytes (see change.Size).
const BodyLimit = record.BodyLimit

// Path is the issue file of slug, relative to the repository root.
func Path(slug string) string { return Dir + "/" + slug + ".md" }
