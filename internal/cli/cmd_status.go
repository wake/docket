package cli

import (
	"flag"
	"fmt"
	"slices"
	"strconv"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/status"
)

// StatusResult is the result object of dkt status.
type StatusResult struct {
	Changes      []status.Change `json:"changes"`
	Records      []status.Record `json:"records"`
	RecordsTotal int             `json:"records_total"`
}

func statusCommand() Command {
	var merged bool
	var since, limit string
	var limitSet bool
	return Command{
		Name:  "status",
		Usage: "status [<slug>...] [--merged [--since <rev>] [--limit <n>]]",
		Define: func(fs *flag.FlagSet) {
			fs.BoolVar(&merged, "merged", false, "list merged records too")
			fs.StringVar(&since, "since", "", "with --merged, only records merged after `rev`")
			fs.Func("limit", "with --merged, list the newest `n` merged records", func(v string) error {
				limit, limitSet = v, true
				return nil
			})
		},
		Run: func(c *Context, args []string) (Outcome, error) {
			if (since != "" || limitSet) && !merged {
				return Outcome{}, UsageError{Msg: "--since and --limit need --merged"}
			}
			if (merged || since != "" || limitSet) && len(args) > 0 {
				return Outcome{}, UsageError{Msg: "--merged, --since and --limit take no slugs"}
			}
			n := 0
			if limitSet {
				var err error
				if n, err = strconv.Atoi(limit); err != nil || n < 1 {
					return Outcome{}, UsageError{Msg: "--limit needs a positive integer"}
				}
			}
			var out Outcome
			var slugs, recSlugs []string
			for _, slug := range args {
				if slices.Contains(slugs, slug) || slices.Contains(recSlugs, slug) {
					continue
				}
				if change.StartsWithDate(slug) {
					recSlugs = append(recSlugs, slug)
					if _, ok := record.SlugOf(slug + ".md"); !ok {
						out.Findings = append(out.Findings, Finding{Rule: "slug-invalid", Change: slug, Message: "not a record slug: a date, a hyphen and a name"})
					}
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
			if len(slugs) > 0 || len(recSlugs) > 0 {
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
			q := status.RecordQuery{Merged: merged, Since: since, Limit: n}
			if since != "" {
				if ok, err := c.Git.RefExists(since + "^{commit}"); err != nil {
					return Outcome{}, err
				} else if !ok {
					return Outcome{}, UsageError{Msg: fmt.Sprintf("--since %q does not name a commit", since)}
				}
			}
			for _, slug := range recSlugs {
				if _, ok := record.SlugOf(slug + ".md"); !ok {
					continue
				}
				if inv.RecordSources(slug) == nil {
					out.Findings = append(out.Findings, Finding{Rule: "record-missing", Change: slug, Message: fmt.Sprintf("no source holds record %q", slug)})
					continue
				}
				q.Slugs = append(q.Slugs, slug)
			}
			records, total, recUnread, err := status.LoadRecords(c.Git, inv, q)
			if err != nil {
				return Outcome{}, err
			}
			unread = append(unread, recUnread...)
			out.Unreadable = append(scanUnreadable(inv.Unreadable), scanUnreadable(unread)...)
			for _, ch := range changes {
				out.Human = append(out.Human, statusLine(ch))
				for _, s := range ch.Sources {
					out.Human = append(out.Human, "  "+s.Name+"@"+shortCommit(s.Commit))
				}
			}
			for _, r := range records {
				out.Human = append(out.Human, statusRecordLine(r))
				for _, s := range r.Sources {
					out.Human = append(out.Human, "  "+s.Name+"@"+shortCommit(s.Commit))
				}
			}
			if cut := total - len(records); cut > 0 {
				out.Human = append(out.Human, fmt.Sprintf("%d more merged records not listed; raise --limit", cut))
			}
			out.Result = StatusResult{Changes: changes, Records: records, RecordsTotal: total}
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

func statusRecordLine(r status.Record) string {
	line := r.Slug
	if r.Title != nil {
		line += fmt.Sprintf("  %q", *r.Title)
	}
	if r.Kind != nil {
		line += "  " + *r.Kind
	}
	if r.State == status.Open {
		line += "  open"
	} else {
		when, tag := "unknown", "untagged"
		if r.Merge != nil && len(r.Merge.Time) >= 10 {
			when = r.Merge.Time[:10]
		}
		if r.Tag != nil {
			tag = *r.Tag
		}
		line += "  merged " + when + "  " + tag
	}
	return fmt.Sprintf("%s  %d/%d tasks", line, r.Progress.TasksDone, r.Progress.TasksTotal)
}
