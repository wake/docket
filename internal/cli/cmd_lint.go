package cli

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/lint"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/scan"
)

// LintResult is the result object of dkt lint.
type LintResult struct {
	Changes []string `json:"changes"`
	Records []string `json:"records"`
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

			// One look at the branch: any failure is one unreadable entry.
			in := lint.Input{Repo: root.FS(), Templates: docket.Schema(), Slugs: args, RecordTemplate: docket.RecordTemplate()}
			base, err := branchBase(c)
			if err == nil {
				var all []string
				if all, err = addedFiles(c, base, append(append([]string{}, lint.RoutedDirs...), record.Dir)...); err == nil {
					for _, f := range all {
						if strings.HasPrefix(f, record.Dir+"/") {
							in.AddedRecords = append(in.AddedRecords, f)
						} else {
							in.Added = append(in.Added, f)
						}
					}
					if base != "" {
						in.Commits, err = c.Git.Commits(base, "HEAD")
					}
				}
			}
			in.AddedErr = err
			rep := lint.Run(in)

			out := Outcome{Result: LintResult{Changes: append([]string{}, rep.Changes...), Records: append([]string{}, rep.Records...)}}
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

// branchBase returns the merge base of HEAD and the default branch, or ""
// when HEAD has no commit. The error says why it could not be found.
func branchBase(c *Context) (string, error) {
	if c.Repo.Head == nil {
		return "", nil
	}
	db, err := scan.ResolveDefaultBranch(c.Git, c.DefaultBranch)
	if err != nil {
		if errors.Is(err, scan.ErrNoDefaultBranch) {
			return "", fmt.Errorf("%w; pass --default-branch to say which branch to compare with", err)
		}
		return "", err
	}
	base, ok, err := c.Git.MergeBase(*c.Repo.Head, db.Commit)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("HEAD and %s share no history; pass --default-branch to name another branch", db.Name)
	}
	return base, nil
}

// addedFiles returns the files the branch adds under the given folders:
// committed since base (when not empty), plus uncommitted ones.
func addedFiles(c *Context, base string, dirs ...string) ([]string, error) {
	set := map[string]bool{}
	if base != "" {
		files, err := c.Git.AddedFiles(base, dirs...)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			set[f] = true
		}
	}
	files, err := c.Git.UncommittedAdded(dirs...)
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
