package cli

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/lint"
)

type lintResult struct {
	Changes []lintChange `json:"changes"` // never null
	Specs   int          `json:"specs"`
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
	rep, err := lint.Run(lint.Input{
		Root: root.FS(), Templates: templates, Slugs: names,
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
	res := lintResult{Changes: []lintChange{}, Specs: rep.Specs}
	for _, l := range rep.Changes {
		res.Changes = append(res.Changes, lintChange{Slug: l.Slug, Schema: l.Schema})
	}
	out := Outcome{Result: res,
		Human: []string{fmt.Sprintf("Linted %d changes and %d main spec titles.", len(res.Changes), res.Specs)}}
	for _, f := range rep.Findings {
		out.Findings = append(out.Findings, Finding{Rule: f.Rule, Message: f.Message, Change: f.Change, File: f.File, Line: f.Line})
	}
	for _, u := range rep.Unreadable {
		out.Unreadable = append(out.Unreadable, Unreadable{Source: u.Source, Reason: u.Reason})
	}
	return out, nil
}
