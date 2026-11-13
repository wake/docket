// Package lint checks the structure of OpenSpec change folders in a
// repository. It reads only through the fs.FS it is given.
package lint

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/domain"
	"github.com/wake/docket/internal/openspec"
)

// OpenSpec is what lint asks of the OpenSpec CLI; *openspec.Runner satisfies it.
type OpenSpec interface {
	ShowChange(slug string) ([]openspec.Delta, error)
	ShowSpec(capability string) ([]openspec.Requirement, error)
	Validate(slug string) ([]openspec.Issue, error)
}

// Input says what to lint. Root is the repository, Templates the shipped
// schema (holding templates/*.md), Slugs the changes named, if any. OpenSpec
// is called once, before any rule, when a docket change is in scope.
type Input struct {
	Root      fs.FS
	Templates fs.FS
	Slugs     []string
	OpenSpec  func() (OpenSpec, error)
	Added     []string // files added since the default branch; the routing rule reads them
	// Records are the paths the branch adds under docket/records, nil when
	// skipped; only regular .md files directly in that folder are linted.
	Records []string
	// RecordTemplate is the shipped record template; its comments are the
	// placeholders a record must not keep.
	RecordTemplate []byte
	// Issues are the paths the branch adds under docket/issues, nil when
	// skipped; IssueTemplate is the shipped issue template.
	Issues        []string
	IssueTemplate []byte
	// RecordTrailers are the Record trailer values of the commits the branch
	// adds; each must name a regular file under docket/records.
	RecordTrailers []RecordTrailer
}

// Linted is a change that was linted and the schema it names.
type Linted struct{ Slug, Schema string }

// Finding is one problem. Change is empty and Line is 0 when they do not
// apply; File is repository-relative and slash-separated.
type Finding struct {
	Rule, Change, File string
	Line               int
	Message            string
}

// Unreadable is a source lint needed but could not read.
type Unreadable struct{ Source, Reason string }

// Report is the outcome of Run: the changes linted (sorted by slug), the
// slugs of the records linted (sorted), the number of main specs whose titles
// were checked, and the findings (sorted).
type Report struct {
	Changes    []Linted
	Records    []string
	Issues     []string
	Specs      int
	Findings   []Finding
	Unreadable []Unreadable
}

const (
	changesDir   = "openspec/changes"
	specsDir     = "openspec/specs"
	proposalFile = "proposal.md"
	docketSchema = "docket"
)

// loaded is one change folder, read once. Files holds every readable regular
// file by path relative to the folder.
type loaded struct {
	Slug   string
	Schema string
	Dir    string // repository-relative
	Files  map[string][]byte
	bad    map[string]bool // unreadable files
	failed bool            // the folder itself could not be read
}

func (c *loaded) Has(name string) bool     { _, ok := c.Files[name]; return ok }
func (c *loaded) Skipped(name string) bool { return c.bad[name] }
func (c *loaded) Path(name string) string  { return path.Join(c.Dir, name) }

type state struct {
	in  Input
	rep *Report

	os    OpenSpec // kept for the delta rules
	mains map[string]*mainSpec

	known     map[string]string
	knownDone bool

	listings map[string]*listing // by folder

	externals map[string]string // the external of each issue file read to compare, by path; "" when it has none
}

// listing is every regular .md file directly under a dated folder, linted or
// not, and whether the folder could not be read.
type listing struct {
	slugs      map[string]bool
	unreadable bool
}

func (r *state) add(c *loaded, rule, file string, line int, format string, args ...any) {
	f := Finding{Rule: rule, Change: c.Slug, Line: line, Message: fmt.Sprintf(format, args...)}
	if file != "" {
		f.File = c.Path(file)
	}
	r.rep.Findings = append(r.rep.Findings, f)
}

func (r *state) unreadable(source string, err error) {
	u := Unreadable{Source: source, Reason: err.Error()}
	for _, x := range r.rep.Unreadable {
		if x == u {
			return // already reported by another rule
		}
	}
	r.rep.Unreadable = append(r.rep.Unreadable, u)
}

// RecordTrailer is one Record trailer value and the commit that carries it.
type RecordTrailer struct{ Commit, Value string }

// Run lints the named changes and records, or every active change, the main
// specs' titles and the records in Input.Records when none is named. A name
// starting with a date is a record.
func Run(in Input) (Report, error) {
	rep := Report{}
	r := &state{in: in, rep: &rep}
	var changes []*loaded
	needOpenSpec := false
	var changeNames, recordNames []string
	for _, n := range in.Slugs {
		if change.HasDate(n) {
			recordNames = append(recordNames, n)
		} else {
			changeNames = append(changeNames, n)
		}
	}
	var changeSlugs []string
	if len(in.Slugs) == 0 || len(changeNames) > 0 {
		changeSlugs = r.scope(changeNames)
	}
	for _, slug := range changeSlugs {
		c := r.load(slug)
		changes = append(changes, c)
		if c.Schema == docketSchema && !c.failed {
			needOpenSpec = true
		}
	}
	if needOpenSpec {
		o, err := in.OpenSpec()
		if err != nil {
			return Report{}, err
		}
		r.os = o
	}
	for _, c := range changes {
		rep.Changes = append(rep.Changes, Linted{Slug: c.Slug, Schema: c.Schema})
		if !c.failed {
			lintChange(r, c)
		}
	}
	rep.Records = r.lintKind(recordKind, in.Records, recordNames, in.RecordTemplate)
	rep.Issues = r.lintKind(issueKind, in.Issues, recordNames, in.IssueTemplate)
	r.checkDatedMissing(recordNames)
	r.checkRecordTrailers()
	if len(in.Slugs) == 0 {
		fnd, us, n := SpecTitles(in.Root)
		rep.Findings = append(rep.Findings, fnd...)
		for _, u := range us {
			r.unreadable(u.Source, errors.New(u.Reason))
		}
		rep.Specs = n
		d := domain.Load(in.Root)
		for _, p := range d.Problems {
			rep.Findings = append(rep.Findings, Finding{Rule: p.Rule, File: domain.File, Line: p.Line, Message: p.Message})
		}
		// With no tree, lint's own rules report what they cannot read.
		for _, u := range d.Unreadable {
			if d.Found {
				r.unreadable(u.Source, errors.New(u.Reason))
			}
		}
	}
	for _, p := range in.Added {
		// Added only holds paths under docs/superpowers/plans or docs/superpowers/specs.
		dir := "docs/superpowers/specs"
		if strings.HasPrefix(p, "docs/superpowers/plans/") {
			dir = "docs/superpowers/plans"
		}
		rep.Findings = append(rep.Findings, Finding{Rule: "routing", File: p,
			Message: "written to " + dir + "; write it into the change folder instead (proposal.md, specs/, design.md or plan.md)"})
	}
	sort.Slice(rep.Changes, func(i, j int) bool { return rep.Changes[i].Slug < rep.Changes[j].Slug })
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
	return rep, nil
}

// scope returns the slugs to lint, sorted, reporting named ones that are not
// changes. A link is never a change.
func (r *state) scope(names []string) []string {
	entries, err := fs.ReadDir(r.in.Root, changesDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		r.unreadable(changesDir, err)
	}
	isChange := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() && e.Name() != "archive" {
			isChange[e.Name()] = true
		}
	}
	var slugs []string
	if len(names) == 0 {
		for s := range isChange {
			slugs = append(slugs, s)
		}
	} else {
		seen := map[string]bool{}
		for _, n := range names {
			if seen[n] {
				continue
			}
			seen[n] = true
			if isChange[n] {
				slugs = append(slugs, n)
			} else if err == nil || errors.Is(err, fs.ErrNotExist) {
				r.rep.Findings = append(r.rep.Findings, Finding{
					Rule:    "change-missing",
					Change:  n,
					Message: fmt.Sprintf("change %q does not exist under %s", n, changesDir),
				})
			}
		}
	}
	sort.Strings(slugs)
	return slugs
}

// load reads every regular file of a change and settles its schema.
func (r *state) load(slug string) *loaded {
	c := &loaded{Slug: slug, Dir: path.Join(changesDir, slug), Files: map[string][]byte{}, bad: map[string]bool{}}
	_ = fs.WalkDir(r.in.Root, c.Dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			r.unreadable(p, err)
			if p == c.Dir {
				c.failed = true
			}
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel := strings.TrimPrefix(p, c.Dir+"/")
		b, err := fs.ReadFile(r.in.Root, p)
		if err != nil {
			c.bad[rel] = true
			r.unreadable(p, err)
			return nil
		}
		c.Files[rel] = b
		return nil
	})
	c.Schema = change.SchemaOf(c.Files[".openspec.yaml"])
	return c
}

// lintChange applies the rules of one change's schema, in order.
func lintChange(r *state, c *loaded) {
	var fm change.Front
	var hasFront bool
	var fmErr error
	if b, ok := c.Files[proposalFile]; ok {
		fm, hasFront, fmErr = change.ReadFrontMatter(b)
	}
	docket := c.Schema == docketSchema
	if docket && c.Has(proposalFile) {
		checkFront(r, c, fm, hasFront, fmErr)
	}
	// A docket front matter error is the one finding; no names are read from it.
	if !c.Skipped(proposalFile) && !(docket && fmErr != nil) {
		checkNewCapabilities(r, c, fm, hasFront && fmErr == nil)
	}
	if !docket {
		return
	}
	checkSections(r, c)
	checkPlaceholders(r, c)
	names := make([]string, 0, len(c.Files))
	for n := range c.Files {
		if strings.HasSuffix(n, ".md") {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		checkFenceInComments(r, c, n)
	}
	if c.Has(planFile) {
		checkPlan(r, c)
	}
	checkDeltas(r, c)
}
