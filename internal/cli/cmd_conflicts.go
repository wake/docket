package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/conflicts"
	"github.com/wake/docket/internal/scan"
)

// ConflictsResult is the result object of dkt conflicts. Changes holds the
// sorted slugs that were compared.
type ConflictsResult struct {
	Changes  []string            `json:"changes"`
	Overlaps []conflicts.Overlap `json:"overlaps"`
}

func conflictsCommand() Command {
	return Command{
		Name:  "conflicts",
		Usage: "conflicts [<slug>]",
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) > 1 {
				return Outcome{}, UsageError{Msg: "conflicts takes at most one slug"}
			}
			fail := func(rule, slug, msg string) (Outcome, error) {
				return Outcome{Findings: []Finding{{Rule: rule, Change: slug, Message: msg}}}, nil
			}
			slug := ""
			if len(args) == 1 {
				slug = args[0]
				if err := change.ValidateSlug(slug); err != nil {
					return fail("slug-invalid", slug, err.Error())
				}
			}
			_, inv, err := scanDefaultBranch(c)
			if err != nil {
				return Outcome{}, err
			}
			changes, unread, err := conflicts.Load(c.Git, inv)
			if err != nil {
				return Outcome{}, err
			}
			out := Outcome{Unreadable: append(scanUnreadable(inv.Unreadable), scanUnreadable(unread)...)}

			slugs := make([]string, 0, len(changes))
			for _, ch := range changes {
				slugs = append(slugs, ch.Slug)
			}
			sort.Strings(slugs)
			overlaps := conflicts.Find(changes)
			if slug != "" {
				found := false
				for _, s := range slugs {
					found = found || s == slug
				}
				if !found {
					msg := fmt.Sprintf("%q is not an active change", slug)
					_, archived := inv.Uses(slug)
					for _, a := range archived {
						msg += "; archived as " + a.Folder
					}
					out.Findings = append(out.Findings, Finding{Rule: "change-not-active", Change: slug, Message: msg})
					out.Result = ConflictsResult{Changes: slugs, Overlaps: []conflicts.Overlap{}}
					return out, nil
				}
				overlaps = conflicts.Involving(overlaps, slug)
			}

			for _, o := range overlaps {
				head := fmt.Sprintf("%s / %q", o.Capability, o.Requirement)
				out.Human = append(out.Human, head)
				var parts []string
				for _, e := range o.Changes {
					ops := strings.Join(e.Operations, ", ")
					parts = append(parts, fmt.Sprintf("%s (%s)", e.Slug, ops))
					srcs := make([]string, len(e.Sources))
					for i, s := range e.Sources {
						srcs[i] = s.Name + "@" + shortCommit(s.Commit)
					}
					out.Human = append(out.Human, fmt.Sprintf("  %s %s: %s", e.Slug, ops, strings.Join(srcs, ", ")))
				}
				out.Findings = append(out.Findings, Finding{Rule: "requirement-overlap",
					Message: head + ": " + strings.Join(parts, ", ")})
			}
			out.Result = ConflictsResult{Changes: slugs, Overlaps: overlaps}
			return out, nil
		},
	}
}

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

// scanUnreadable maps unreadable sources to the envelope form: a worktree is
// named "worktree:<path>", any other source by its refname, and a non-empty
// Path is appended as ":<path>".
func scanUnreadable(us []scan.Unreadable) []Unreadable {
	out := make([]Unreadable, 0, len(us))
	for _, u := range us {
		name := u.Source.Name
		if u.Source.Kind == scan.KindWorktree {
			name = "worktree:" + name
		}
		if u.Path != "" {
			name += ":" + u.Path
		}
		out = append(out, Unreadable{Source: name, Reason: u.Reason})
	}
	return out
}
