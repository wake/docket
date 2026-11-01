package cli

import (
	"errors"
	"path/filepath"

	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/safepath"
	"github.com/wake/docket/internal/scan"
)

// datedFile is a kind of file named with a date, a hyphen and a name, one per
// file under Dir: a record or an issue. Sources returns where the scan found
// slug, nil when nowhere.
type datedFile struct {
	Command, Dir string
	Sources      func(inv scan.Inventory, slug string) []scan.Source
}

// createdOutcome is the outcome of a dated file written: res with the line
// "created path", preceded by the used-slug line when a suffix was taken.
func createdOutcome(res any, slug, written, path string, used []string) Outcome {
	human := []string{"created " + path}
	if written != slug {
		human = append([]string{usedLine(slug, used, written)}, human...)
	}
	return Outcome{Result: res, Human: human}
}

// writeDated creates the file for name under d.Dir, in the first free slug
// from today's date and name. done is true when out (a finding, a refusal or
// unreadable sources) is the command's whole outcome. used lists where the
// wanted slug was taken, never nil.
func writeDated(c *Context, d datedFile, name string, content []byte) (written, path string, used []string, out Outcome, done bool, err error) {
	fail := func(rule, msg string) (Outcome, error) {
		return Outcome{Findings: []Finding{{Rule: rule, Message: msg}}}, nil
	}
	if err := record.ValidateName(d.Command, name); err != nil {
		out, _ = fail("slug-invalid", err.Error())
		return "", "", nil, out, true, nil
	}
	slug := record.Slug(c.Now, name)
	rel := d.Dir + "/" + slug + ".md"
	if err := safepath.Check(c.Repo.Root, rel); err != nil {
		out, err = refuse(fail, err)
		return "", "", nil, out, err == nil, err
	}
	_, inv, err := scanDefaultBranch(c)
	if err != nil {
		return "", "", nil, Outcome{}, false, err
	}
	if unreadable := scanUnreadable(inv.Unreadable); len(unreadable) > 0 {
		return "", "", nil, Outcome{Unreadable: unreadable}, true, nil
	}
	dir := filepath.Join(c.Repo.Root, filepath.FromSlash(d.Dir))
	written, err = record.Create(dir, slug, func(cand string) bool { return d.Sources(inv, cand) != nil }, content)
	var le *record.LinkError
	switch {
	case errors.As(err, &le):
		out, err = refuse(fail, &safepath.SymlinkError{Path: d.Dir + "/" + le.Name})
		return "", "", nil, out, err == nil, err
	case errors.Is(err, record.ErrTooLong):
		out, _ = fail("slug-invalid", err.Error())
		return "", "", nil, out, true, nil
	case err != nil:
		return "", "", nil, Outcome{}, false, err
	}
	used = []string{}
	if srcs := d.Sources(inv, slug); srcs != nil {
		used = []string{"in " + sourceNames(srcs)}
	} else if written != slug {
		used = []string{rel + " exists in this worktree"}
	}
	return written, d.Dir + "/" + written + ".md", used, Outcome{}, false, nil
}
