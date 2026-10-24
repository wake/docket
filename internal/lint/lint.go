// Package lint checks the structure of OpenSpec change folders in a
// worktree. It reads only through the fs.FS it is given.
package lint

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/wake/docket/internal/change"
)

// Finding is one problem found in a change. Change and File are empty and
// Line is 0 when they do not apply; File is relative to the repository root.
type Finding struct {
	Rule    string
	Change  string
	File    string
	Line    int
	Message string
}

// Unreadable is a source lint needed but could not read.
type Unreadable struct {
	Source string
	Reason string
}

// Report is the outcome of Run. Changes are the linted slugs, sorted;
// Findings are sorted by change, file, line and rule.
type Report struct {
	Changes    []string
	Findings   []Finding
	Unreadable []Unreadable
}

// Input says what to lint. Repo is the worktree root and Templates is the
// shipped schema (docket.Schema()). Added and AddedErr feed the routing
// rule: the files the branch adds, or why they are unknown.
type Input struct {
	Repo      fs.FS
	Templates fs.FS
	Slugs     []string
	Added     []string
	AddedErr  error
}

const changesDir = "openspec/changes"

// loaded is one change folder read once and handed to every rule. Files
// holds every readable regular file by path relative to the change folder.
type loaded struct {
	Slug  string
	Dir   string // repository-relative folder, slash-separated
	Files map[string][]byte
	bad   map[string]bool // unreadable files, relative paths

	Proposal change.Proposal // zero value unless proposal.md was read
	Plan     change.Plan     // zero value unless plan.md was read
}

// Has reports whether the change has a readable file.
func (c *loaded) Has(name string) bool { _, ok := c.Files[name]; return ok }

// Skipped reports whether a file exists but could not be read.
func (c *loaded) Skipped(name string) bool { return c.bad[name] }

// Path returns the repository-relative path of a file of the change.
func (c *loaded) Path(name string) string { return path.Join(c.Dir, name) }

// run collects the output of the rules.
type run struct {
	in  Input
	rep *Report

	known     map[string]string // placeholder map, read on first use
	knownDone bool
}

func (r *run) add(c *loaded, rule, file string, line int, format string, args ...any) {
	f := Finding{Rule: rule, Change: c.Slug, Line: line, Message: fmt.Sprintf(format, args...)}
	if file != "" {
		f.File = c.Path(file)
	}
	r.rep.Findings = append(r.rep.Findings, f)
}

func (r *run) unreadable(source string, err error) {
	r.rep.Unreadable = append(r.rep.Unreadable, Unreadable{Source: source, Reason: err.Error()})
}

// Run lints the named changes, or every active change when none is named.
// Each rule is a function of (run, loaded) called once per change; later
// rules are added to the list in the loop below.
func Run(in Input) Report {
	rep := Report{}
	r := &run{in: in, rep: &rep}
	for _, slug := range r.scope() {
		c, ok := r.load(slug)
		if !ok {
			continue
		}
		rep.Changes = append(rep.Changes, slug)
		checkProposal(r, c)
		checkPlaceholders(r, c)
		if c.Has(planFile) {
			checkPlan(r, c)
		}
	}
	sort.Strings(rep.Changes)
	sort.SliceStable(rep.Findings, func(i, j int) bool {
		a, b := rep.Findings[i], rep.Findings[j]
		switch {
		case a.Change != b.Change:
			return a.Change < b.Change
		case a.File != b.File:
			return a.File < b.File
		case a.Line != b.Line:
			return a.Line < b.Line
		case a.Rule != b.Rule:
			return a.Rule < b.Rule
		}
		return a.Message < b.Message
	})
	return rep
}

// scope returns the slugs to lint, reporting named ones that do not exist.
func (r *run) scope() []string {
	if len(r.in.Slugs) == 0 {
		entries, err := fs.ReadDir(r.in.Repo, changesDir)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				r.unreadable(changesDir, err)
			}
			return nil
		}
		var slugs []string
		for _, e := range entries {
			if e.IsDir() && e.Name() != "archive" {
				slugs = append(slugs, e.Name())
			}
		}
		sort.Strings(slugs)
		return slugs
	}
	var slugs []string
	seen := map[string]bool{}
	for _, slug := range r.in.Slugs {
		if seen[slug] {
			continue
		}
		seen[slug] = true
		if slug == "archive" || slug == "." || !fs.ValidPath(slug) || strings.Contains(slug, "/") {
			r.missing(slug)
			continue
		}
		info, err := fs.Stat(r.in.Repo, path.Join(changesDir, slug))
		switch {
		case errors.Is(err, fs.ErrNotExist) || (err == nil && !info.IsDir()):
			r.missing(slug)
		case err != nil:
			r.unreadable(path.Join(changesDir, slug), err)
		default:
			slugs = append(slugs, slug)
		}
	}
	return slugs
}

func (r *run) missing(slug string) {
	r.rep.Findings = append(r.rep.Findings, Finding{
		Rule:    "change-missing",
		Message: fmt.Sprintf("change %q does not exist under %s", slug, changesDir),
	})
}

// load reads every regular file of a change. It returns false, after
// listing the folder as unreadable, when the folder cannot be read.
func (r *run) load(slug string) (*loaded, bool) {
	c := &loaded{Slug: slug, Dir: path.Join(changesDir, slug), Files: map[string][]byte{}, bad: map[string]bool{}}
	var dirErr error
	_ = fs.WalkDir(r.in.Repo, c.Dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == c.Dir {
				dirErr = err
				return err
			}
			r.unreadable(p, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel := strings.TrimPrefix(p, c.Dir+"/")
		b, err := fs.ReadFile(r.in.Repo, p)
		if err != nil {
			c.bad[rel] = true
			r.unreadable(p, err)
			return nil
		}
		c.Files[rel] = b
		return nil
	})
	if dirErr != nil {
		r.unreadable(c.Dir, dirErr)
		return nil, false
	}
	if b, ok := c.Files[proposalFile]; ok {
		c.Proposal = change.ParseProposal(b)
	}
	if b, ok := c.Files[planFile]; ok {
		c.Plan = change.ParsePlan(b)
	}
	return c, true
}
