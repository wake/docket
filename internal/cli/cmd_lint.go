package cli

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/lint"
	"github.com/wake/docket/internal/scan"
)

// LintResult is the result object of dkt lint.
type LintResult struct {
	Changes []string `json:"changes"`
}

func lintCommand() Command {
	return Command{
		Name:  "lint",
		Usage: "lint [<slug>...]",
		Run: func(c *Context, args []string) (Outcome, error) {
			root, err := os.OpenRoot(c.Repo.Root)
			if err != nil {
				return Outcome{}, err
			}
			defer root.Close()

			added, addedErr := addedFiles(c)
			rep := lint.Run(lint.Input{
				Repo:      root.FS(),
				Templates: docket.Schema(),
				Slugs:     args,
				Added:     added,
				AddedErr:  addedErr,
			})

			out := Outcome{Result: LintResult{Changes: append([]string{}, rep.Changes...)}}
			for _, f := range rep.Findings {
				out.Findings = append(out.Findings, Finding{Rule: f.Rule, Message: f.Message, Change: f.Change, File: f.File, Line: f.Line})
			}
			for _, u := range rep.Unreadable {
				out.Unreadable = append(out.Unreadable, Unreadable{Source: u.Source, Reason: u.Reason})
			}
			return out, nil
		},
	}
}

// addedFiles returns the files the branch adds under the superpowers folders:
// committed since the merge base with the default branch, plus uncommitted
// ones. The error says why they could not be known.
func addedFiles(c *Context) ([]string, error) {
	set := map[string]bool{}
	if c.Repo.Head != nil {
		db, err := scan.ResolveDefaultBranch(c.Git, c.DefaultBranch)
		if err != nil {
			if errors.Is(err, scan.ErrNoDefaultBranch) {
				return nil, fmt.Errorf("%w; pass --default-branch to say which branch to compare with", err)
			}
			return nil, err
		}
		base, ok, err := c.Git.MergeBase(*c.Repo.Head, db.Commit)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("HEAD and %s share no history; pass --default-branch to name another branch", db.Name)
		}
		files, err := c.Git.AddedFiles(base, lint.RoutedDirs...)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			set[f] = true
		}
	}
	files, err := c.Git.UncommittedAdded(lint.RoutedDirs...)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		set[f] = true
	}
	out := make([]string, 0, len(set))
	for f := range set {
		out = append(out, f)
	}
	sort.Strings(out)
	return out, nil
}
