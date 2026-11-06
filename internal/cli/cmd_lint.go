package cli

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/issue"
	"github.com/wake/docket/internal/lint"
	"github.com/wake/docket/internal/record"
)

type lintResult struct {
	Changes []lintChange `json:"changes"` // never null
	Records []string     `json:"records"` // slugs; never null
	Issues  []string     `json:"issues"`  // slugs; never null
	Specs   int          `json:"specs"`
	// DefaultBranch is null when the routing rule was skipped.
	DefaultBranch *string `json:"default_branch"`
}

type lintChange struct {
	Slug   string `json:"slug"`
	Schema string `json:"schema"`
}

func lintCommand() Command {
	return Command{
		Name:  "lint",
		Args:  -1,
		Usage: "lint [<slug>...] [--json]",
		Run:   runLint,
	}
}

// runLint is the body of dkt lint.
func runLint(c *Context, names []string) (Outcome, error) {
	for _, name := range names {
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
			return Outcome{}, UsageError{Msg: fmt.Sprintf("%q is not a change name", name)}
		}
	}
	root, err := os.OpenRoot(c.Repo.Root)
	if err != nil {
		return Outcome{}, err
	}
	defer root.Close()
	templates, err := fs.Sub(docket.Shipped, "shipped/schema")
	if err != nil {
		return Outcome{}, err
	}
	var added, records, issues []string
	var recordTrailers []lint.RecordTrailer
	var branch *string
	skip := "No default branch found; the routing rule and the record and issue checks were skipped."
	db, ok, err := c.Git.DefaultBranch()
	if err != nil {
		return Outcome{}, err
	}
	if ok {
		skip = "No merge base with " + db.Name + "; the routing rule and the record and issue checks were skipped."
		if c.Repo.Head != nil {
			// AddedSince needs the repository root as the git directory.
			g := gitx.Git{Dir: c.Repo.Root, Env: c.Git.Env}
			base, found, err := g.MergeBase(*c.Repo.Head, db.Commit)
			if err != nil {
				return Outcome{}, err
			}
			if found {
				if added, err = g.AddedSince(base, "docs/superpowers/specs", "docs/superpowers/plans"); err != nil {
					return Outcome{}, err
				}
				if added == nil {
					added = []string{}
				}
				if len(names) == 0 {
					if records, err = g.AddedSince(base, record.Dir); err != nil {
						return Outcome{}, err
					}
					if issues, err = g.AddedSince(base, issue.Dir); err != nil {
						return Outcome{}, err
					}
					trs, err := g.RecordTrailers(base + "..HEAD")
					if err != nil {
						return Outcome{}, err
					}
					for _, t := range trs {
						for _, v := range t.Values {
							recordTrailers = append(recordTrailers, lint.RecordTrailer{Commit: t.SHA, Value: v})
						}
					}
				}
				branch, skip = &db.Name, ""
			}
		}
	}
	recordTemplate, err := docket.Shipped.ReadFile("shipped/record/template.md")
	if err != nil {
		return Outcome{}, err
	}
	issueTemplate, err := docket.Shipped.ReadFile("shipped/issue/template.md")
	if err != nil {
		return Outcome{}, err
	}
	rep, err := lint.Run(lint.Input{
		Root: root.FS(), Templates: templates, Slugs: names, Added: added,
		Records: records, RecordTemplate: recordTemplate, Issues: issues, IssueTemplate: issueTemplate, RecordTrailers: recordTrailers,
		OpenSpec: func() (lint.OpenSpec, error) {
			r, err := openSpecRunner(c)
			if err != nil {
				return nil, err // not r: a typed nil would pass for a value
			}
			return r, nil
		},
	})
	if err != nil {
		return Outcome{}, err
	}
	res := lintResult{Changes: []lintChange{}, Records: append([]string{}, rep.Records...), Issues: append([]string{}, rep.Issues...), Specs: rep.Specs, DefaultBranch: branch}
	for _, l := range rep.Changes {
		res.Changes = append(res.Changes, lintChange{Slug: l.Slug, Schema: l.Schema})
	}
	out := Outcome{Result: res,
		Human: []string{fmt.Sprintf("Linted %d changes, %d records, %d issues and %d main spec titles.", len(res.Changes), len(res.Records), len(res.Issues), res.Specs)}}
	if skip != "" {
		out.Human = append(out.Human, skip)
	}
	for _, f := range rep.Findings {
		out.Findings = append(out.Findings, Finding{Rule: f.Rule, Message: f.Message, Change: f.Change, File: f.File, Line: f.Line})
	}
	for _, u := range rep.Unreadable {
		out.Unreadable = append(out.Unreadable, Unreadable{Source: u.Source, Reason: u.Reason})
	}
	return out, nil
}
