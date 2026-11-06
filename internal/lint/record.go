package lint

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"
	"syscall"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/record"
)

// dated describes a kind of dated file: where it lives and which rules apply.
type dated struct {
	dir          string
	sections     []string // the body sections, in order
	after        []string // sections allowed after them
	kinds        []string // the allowed kinds; nil when the files have none
	taskHeadings bool     // whether `### Task` headings under `## Tasks` are checked
	prefix       string   // names the rules (<prefix>-section) and the file in messages
	template     string
	limit        int
}

var recordKind = dated{
	dir:          record.Dir,
	sections:     record.BodySections,
	after:        []string{"Tasks"},
	kinds:        record.Kinds,
	taskHeadings: true,
	prefix:       "record",
	template:     "shipped/record/template.md",
	limit:        record.BodyLimit,
}

// list lists the regular .md files directly under dir, keeping their slugs; a
// missing folder, or a file in its place, holds none.
func (r *state) list(dir string) *listing {
	if l := r.listings[dir]; l != nil {
		return l // read already; the listing is the same for the whole run
	}
	l := &listing{slugs: map[string]bool{}}
	if r.listings == nil {
		r.listings = map[string]*listing{}
	}
	r.listings[dir] = l
	entries, err := fs.ReadDir(r.in.Root, dir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
			r.unreadable(dir, err)
			l.unreadable = true
		}
		return l
	}
	for _, e := range entries {
		if slug, ok := strings.CutSuffix(e.Name(), ".md"); ok && e.Type().IsRegular() {
			l.slugs[slug] = true
		}
	}
	return l
}

// checkRecordTrailers reports each trailer that is not a record's slug alone
// or with a task list, or whose record is not a regular file. An unreadable
// folder is already listed as unreadable, so it reports nothing here.
func (r *state) checkRecordTrailers() {
	if len(r.in.RecordTrailers) == 0 {
		return
	}
	l := r.list(record.Dir)
	if l.unreadable {
		return
	}
	for _, t := range r.in.RecordTrailers {
		slug, _, _ := strings.Cut(t.Value, "/")
		if _, ok, _ := change.TaskRefs(t.Value, slug); ok && l.slugs[slug] {
			continue
		}
		r.rep.Findings = append(r.rep.Findings, Finding{Rule: "record-trailer",
			Message: fmt.Sprintf("commit %s has the trailer Record: %s, which is not the slug of a record under %s, alone or followed by / and task numbers separated by commas",
				t.Commit[:min(7, len(t.Commit))], t.Value, record.Dir)})
	}
}

// datedScope returns the slugs of the files of kind d to lint, sorted: the
// ones in files and the ones named, reporting named ones that are not files.
func (r *state) datedScope(d dated, files, named []string) []string {
	if len(files) == 0 && len(named) == 0 {
		return nil // nothing is in scope, so the folder does not matter
	}
	l := r.list(d.dir)
	want := map[string]bool{}
	for _, p := range files {
		if slug, ok := strings.CutSuffix(path.Base(p), ".md"); ok && path.Dir(p) == d.dir && l.slugs[slug] {
			want[slug] = true
		}
	}
	for _, n := range named {
		switch {
		case l.slugs[n]:
			want[n] = true
		case !l.unreadable:
			file := d.dir + "/" + n + ".md"
			r.rep.Findings = append(r.rep.Findings, Finding{Rule: "record-missing", File: file,
				Message: fmt.Sprintf("record %q does not exist: %s is not a regular file", n, file)})
		}
	}
	slugs := make([]string, 0, len(want))
	for s := range want {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)
	return slugs
}

// lintDated applies the rules of kind d to one file.
func lintDated(r *state, d dated, slug string, known map[string]string) {
	file := slug + ".md"
	b, err := fs.ReadFile(r.in.Root, d.dir+"/"+file)
	if err != nil {
		r.unreadable(d.dir+"/"+file, err)
		return
	}
	// A dated file is not a change: its findings carry a file and no change.
	c := &loaded{Dir: d.dir, Files: map[string][]byte{file: b}}
	p := record.Parse(b)
	checkDatedFront(r, d, c, file, p)
	checkDatedSections(r, d, c, file, p)
	if size := p.Size(d.sections); size > d.limit {
		r.add(c, d.prefix+"-too-long", file, 0, "the four sections hold %d bytes; the limit is %d", size, d.limit)
	}
	if len(known) > 0 {
		checkPlaceholderFile(r, c, file, known)
	}
	if d.taskHeadings {
		checkTaskHeadings(r, c, file, p.Tasks)
	}
}

func checkDatedFront(r *state, d dated, c *loaded, file string, p record.Parsed) {
	front := d.prefix + "-front-matter"
	fields := "title and source"
	if d.kinds != nil {
		fields = "title, kind and source"
	}
	switch {
	case p.FrontErr != nil:
		r.add(c, front, file, 0, "front matter is not valid: %v", p.FrontErr)
		return
	case !p.HasFront:
		r.add(c, front, file, 0, "the %s has no front matter with a %s", d.prefix, fields)
		return
	}
	fm := p.Front
	if f, ok := fm.Fields["title"]; !ok {
		r.add(c, front, file, 0, "title is missing from the front matter")
	} else if !change.ValidLine(f.Value) {
		r.add(c, front, file, f.Line, "title must be one line of text")
	} else if msg, tooLong := change.TitleTooLong(f.Value); tooLong {
		r.add(c, "title-too-long", file, f.Line, "%s", msg)
	}
	if d.kinds != nil {
		if f, ok := fm.Fields["kind"]; !ok {
			r.add(c, d.prefix+"-kind", file, 0, "kind is missing from the front matter; it must be one of %s", strings.Join(d.kinds, ", "))
		} else if !slices.Contains(d.kinds, f.Value) {
			r.add(c, d.prefix+"-kind", file, f.Line, "kind must be one of %s", strings.Join(d.kinds, ", "))
		}
	}
	if f, ok := fm.Fields["source"]; !ok {
		r.add(c, front, file, 0, "source is missing from the front matter")
	} else if !change.ValidLine(f.Value) {
		r.add(c, front, file, f.Line, "source must be one line of text")
	}
}

func checkDatedSections(r *state, d dated, c *loaded, file string, p record.Parsed) {
	rule := d.prefix + "-section"
	canon := append(slices.Clone(d.sections), d.after...)
	present := map[string]bool{}
	last, ordered := -1, true
	for _, s := range p.Sections {
		i := slices.IndexFunc(canon, func(n string) bool { return strings.EqualFold(n, s.Name) })
		if i < 0 {
			r.add(c, rule, file, s.Line, "section %q is not part of a %s; use %s", s.Name, d.prefix, strings.Join(canon, ", "))
			continue
		}
		present[canon[i]] = true
		if i <= last && ordered {
			ordered = false
			r.add(c, rule, file, s.Line,
				"section %q is out of order or repeated; the order is %s, each once", s.Name, strings.Join(canon, ", "))
		}
		last = max(last, i)
		if i < len(d.sections) && strings.TrimSpace(s.Text) == "" {
			r.add(c, rule, file, s.Line, "section %q is empty", canon[i])
		}
	}
	for _, name := range d.sections {
		if !present[name] {
			r.add(c, rule, file, 0, "section %q is missing", name)
		}
	}
	for _, line := range p.Outside {
		r.add(c, rule, file, line, "text outside the sections; put it under one of %s", strings.Join(canon, ", "))
	}
}
