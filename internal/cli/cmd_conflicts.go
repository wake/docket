package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/wake/docket/internal/conflicts"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/scan"
)

type conflictsResult struct {
	Changes  []string            `json:"changes"`  // never null
	Overlaps []conflicts.Overlap `json:"overlaps"` // never null
}

func conflictsCommand() Command {
	return Command{
		Name:  "conflicts",
		Args:  -1,
		Usage: "conflicts [<slug>] [--json]",
		Run:   runConflicts,
	}
}

// runConflicts is the body of dkt conflicts.
func runConflicts(c *Context, names []string) (Outcome, error) {
	if len(names) > 1 {
		return Outcome{}, UsageError{Msg: "conflicts takes at most one slug"}
	}
	inv, err := scan.Changes(c.Git)
	if err != nil {
		return Outcome{}, err
	}
	copies, copyBad, err := conflicts.Copies(c.Git, inv)
	if err != nil {
		return Outcome{}, err
	}
	archived := map[string][]string{} // slug -> archive folders
	active := map[string]bool{}
	for _, u := range inv.Uses {
		if u.Archived {
			archived[u.Slug] = append(archived[u.Slug], u.Path)
		}
	}
	for _, u := range inv.Uses {
		if !u.Archived && archived[u.Slug] == nil {
			active[u.Slug] = true
		}
	}
	res := conflictsResult{Changes: []string{}, Overlaps: []conflicts.Overlap{}}
	for s := range active {
		res.Changes = append(res.Changes, s)
	}
	slices.Sort(res.Changes)

	out := Outcome{Result: &res}
	var parseBad []scan.Unreadable
	var slug string
	compared := true // false when the slug is not active and nothing was compared
	if len(names) == 1 {
		slug = names[0]
	}
	switch {
	case slug != "" && !active[slug]:
		compared = false
		// No OpenSpec is needed to say the slug is not there.
		if dirs := sortedUnique(archived[slug]); len(dirs) > 0 {
			out.Findings = append(out.Findings, Finding{Rule: "change-missing", Change: slug,
				Message: fmt.Sprintf("%s is not an active change: it is archived (%s).", slug, strings.Join(dirs, ", "))})
		} else if len(inv.Unreadable) == 0 {
			out.Findings = append(out.Findings, Finding{Rule: "change-missing", Change: slug,
				Message: fmt.Sprintf("No source holds a change named %s.", slug)})
		}
	case len(res.Changes) >= 2:
		r, err := openSpecRunner(c)
		if err != nil {
			return Outcome{}, err
		}
		show := func(dir, s string) ([]openspec.Delta, error) {
			scratch := *r
			scratch.Dir = dir
			return scratch.ShowChange(s)
		}
		var parsed []conflicts.Parsed
		if parsed, parseBad, err = conflicts.Parse(copies, show); err != nil {
			return Outcome{}, err
		}
		res.Overlaps = conflicts.Find(parsed)
		if slug != "" {
			res.Overlaps = conflicts.Involving(res.Overlaps, slug)
		}
	}

	for _, o := range res.Overlaps {
		out.Human = append(out.Human, fmt.Sprintf("%s / %q", o.Capability, o.Requirement))
		var parts []string
		for _, e := range o.Changes {
			ops := strings.Join(e.Operations, ", ")
			out.Human = append(out.Human, fmt.Sprintf("  %s (%s): %s", e.Slug, ops, strings.Join(e.Sources, ", ")))
			parts = append(parts, fmt.Sprintf("%s (%s)", e.Slug, ops))
		}
		out.Findings = append(out.Findings, Finding{Rule: "requirement-overlap",
			Message: fmt.Sprintf("%s / %q: %s", o.Capability, o.Requirement, strings.Join(parts, ", "))})
	}
	if len(res.Overlaps) == 0 && compared {
		noun := "changes"
		if len(res.Changes) == 1 {
			noun = "change"
		}
		out.Human = append(out.Human, fmt.Sprintf("Compared %d %s; no requirement is touched by two of them.", len(res.Changes), noun))
	}
	if out.Findings == nil {
		out.Findings = []Finding{}
	}
	for _, list := range [][]scan.Unreadable{inv.Unreadable, copyBad, parseBad} {
		for _, u := range list {
			out.Unreadable = append(out.Unreadable, Unreadable{Source: u.Source, Reason: u.Reason})
		}
	}
	return out, nil
}

func sortedUnique(s []string) []string {
	s = slices.Clone(s)
	slices.Sort(s)
	return slices.Compact(s)
}
