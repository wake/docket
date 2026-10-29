package cli

import (
	"fmt"
	"slices"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/status"
)

// StatusResult is the result object of dkt status.
type StatusResult struct {
	Changes []status.Change `json:"changes"`
}

func statusCommand() Command {
	return Command{
		Name:  "status",
		Usage: "status [<slug>...]",
		Run: func(c *Context, args []string) (Outcome, error) {
			var out Outcome
			var slugs []string
			for _, slug := range args {
				if slices.Contains(slugs, slug) {
					continue
				}
				slugs = append(slugs, slug)
				if err := change.ValidateSlug(slug); err != nil {
					out.Findings = append(out.Findings, Finding{Rule: "slug-invalid", Change: slug, Message: err.Error()})
				}
			}
			_, inv, err := scanDefaultBranch(c)
			if err != nil {
				return Outcome{}, err
			}
			if len(slugs) > 0 {
				keep := map[string]bool{}
				for _, slug := range slugs {
					if change.ValidateSlug(slug) != nil {
						continue
					}
					active, archived := inv.Uses(slug)
					if active == nil || len(archived) > 0 {
						if mayReportNotActive(inv, slug) {
							out.Findings = append(out.Findings, notActiveFinding(inv, slug))
						}
						continue
					}
					keep[slug] = true
				}
				var only []scan.Change
				for _, ch := range inv.Changes {
					if keep[ch.Slug] {
						only = append(only, ch)
					}
				}
				inv.Changes = only
			}
			changes, unread, err := status.Load(c.Git, inv, c.Now)
			if err != nil {
				return Outcome{}, err
			}
			out.Unreadable = append(scanUnreadable(inv.Unreadable), scanUnreadable(unread)...)
			for _, ch := range changes {
				out.Human = append(out.Human, statusLine(ch))
				for _, s := range ch.Sources {
					out.Human = append(out.Human, "  "+s.Name+"@"+shortCommit(s.Commit))
				}
			}
			out.Result = StatusResult{Changes: changes}
			return out, nil
		},
	}
}

func statusLine(ch status.Change) string {
	idle := "no commits"
	if ch.LastActivity != nil {
		idle = fmt.Sprintf("idle %dd", ch.LastActivity.IdleDays)
	}
	head := ch.Slug
	if ch.Title != nil {
		head = fmt.Sprintf("%s  %q", ch.Slug, *ch.Title)
	}
	line := fmt.Sprintf("%s  %s  %d/%d tasks  %s", head, ch.Phase, ch.Progress.TasksDone, ch.Progress.TasksTotal, idle)
	if ch.Uncommitted != nil {
		line += fmt.Sprintf("  %d uncommitted", *ch.Uncommitted)
	}
	return line
}
