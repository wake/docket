package cli

import (
	"fmt"

	snap "github.com/wake/docket/internal/snapshot"
)

func snapshotCommand() Command {
	return Command{Name: "snapshot", Args: 0, Usage: "snapshot [--json]", Run: runSnapshot}
}

// runSnapshot is the body of dkt snapshot. It reads the repository and writes nothing.
func runSnapshot(c *Context, _ []string) (Outcome, error) {
	res, problems, unreadable, err := snap.Collect(c.Git)
	if err != nil {
		return Outcome{}, err
	}
	out := Outcome{Result: res, Findings: []Finding{}}
	for _, p := range problems {
		out.Findings = append(out.Findings, Finding{Rule: p.Rule, File: p.File, Line: p.Line, Message: p.Message})
	}
	for _, u := range unreadable {
		out.Unreadable = append(out.Unreadable, Unreadable{Source: u.Source, Reason: u.Reason})
	}
	for _, w := range res.Works {
		title := ""
		if w.Title != nil {
			title = *w.Title
		}
		out.Human = append(out.Human, fmt.Sprintf("%s %s  %s  %s", w.Type, w.Slug, w.Phase, title))
	}
	out.Human = append(out.Human, fmt.Sprintf("%d ideas, %d issues, %d specs", len(res.Ideas), len(res.Issues), len(res.Specs)))
	return out, nil
}
