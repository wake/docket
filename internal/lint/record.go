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

const recordTemplateName = "shipped/record/template.md"

// readRecords lists the regular .md files directly under the records folder,
// keeping their slugs on the state; a missing folder, or a file in its place,
// holds none.
func (r *state) readRecords() {
	r.recordSlugs = map[string]bool{}
	entries, err := fs.ReadDir(r.in.Root, record.Dir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
			r.unreadable(record.Dir, err)
			r.recordsUnreadable = true
		}
		return
	}
	for _, e := range entries {
		if slug, ok := strings.CutSuffix(e.Name(), ".md"); ok && e.Type().IsRegular() {
			r.recordSlugs[slug] = true
		}
	}
}

// recordScope returns the slugs of the records to lint, sorted: the ones in
// Input.Records and the ones named, reporting named ones that are not files.
func (r *state) recordScope(named []string) []string {
	if len(r.in.Records) == 0 && len(named) == 0 {
		return nil // no record is in scope, so the folder does not matter
	}
	r.readRecords()
	want := map[string]bool{}
	for _, p := range r.in.Records {
		if slug, ok := strings.CutSuffix(path.Base(p), ".md"); ok && path.Dir(p) == record.Dir && r.recordSlugs[slug] {
			want[slug] = true
		}
	}
	for _, n := range named {
		switch {
		case r.recordSlugs[n]:
			want[n] = true
		case !r.recordsUnreadable:
			r.rep.Findings = append(r.rep.Findings, Finding{Rule: "record-missing", File: record.Path(n),
				Message: fmt.Sprintf("record %q does not exist: %s is not a regular file", n, record.Path(n))})
		}
	}
	slugs := make([]string, 0, len(want))
	for s := range want {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)
	return slugs
}

// lintRecord applies the record rules to one record.
func lintRecord(r *state, slug string, known map[string]string) {
	file := slug + ".md"
	b, err := fs.ReadFile(r.in.Root, record.Path(slug))
	if err != nil {
		r.unreadable(record.Path(slug), err)
		return
	}
	// A record is not a change: its findings carry a file and no change.
	c := &loaded{Dir: record.Dir, Files: map[string][]byte{file: b}}
	p := record.Parse(b)
	checkRecordFront(r, c, file, p)
	checkRecordSections(r, c, file, p)
	if size := p.BodySize(); size > record.BodyLimit {
		r.add(c, "record-too-long", file, 0, "the four sections hold %d bytes; the limit is %d", size, record.BodyLimit)
	}
	if len(known) > 0 {
		checkPlaceholderFile(r, c, file, known)
	}
	checkTaskHeadings(r, c, file, p.Tasks)
}

func checkRecordFront(r *state, c *loaded, file string, p record.Parsed) {
	switch {
	case p.FrontErr != nil:
		r.add(c, "record-front-matter", file, 0, "front matter is not valid: %v", p.FrontErr)
		return
	case !p.HasFront:
		r.add(c, "record-front-matter", file, 0, "the record has no front matter with a title, kind and source")
		return
	}
	fm := p.Front
	if f, ok := fm.Fields["title"]; !ok {
		r.add(c, "record-front-matter", file, 0, "title is missing from the front matter")
	} else if !change.ValidLine(f.Value) {
		r.add(c, "record-front-matter", file, f.Line, "title must be one line of text")
	} else if msg, tooLong := change.TitleTooLong(f.Value); tooLong {
		r.add(c, "title-too-long", file, f.Line, "%s", msg)
	}
	if f, ok := fm.Fields["kind"]; !ok {
		r.add(c, "record-kind", file, 0, "kind is missing from the front matter; it must be one of %s", strings.Join(record.Kinds, ", "))
	} else if !slices.Contains(record.Kinds, f.Value) {
		r.add(c, "record-kind", file, f.Line, "kind must be one of %s", strings.Join(record.Kinds, ", "))
	}
	if f, ok := fm.Fields["source"]; !ok {
		r.add(c, "record-front-matter", file, 0, "source is missing from the front matter")
	} else if !change.ValidLine(f.Value) {
		r.add(c, "record-front-matter", file, f.Line, "source must be one line of text")
	}
}

func checkRecordSections(r *state, c *loaded, file string, p record.Parsed) {
	canon := append(slices.Clone(record.BodySections), "Tasks")
	present := map[string]bool{}
	last, ordered := -1, true
	for _, s := range p.Sections {
		i := slices.IndexFunc(canon, func(n string) bool { return strings.EqualFold(n, s.Name) })
		if i < 0 {
			r.add(c, "record-section", file, s.Line, "section %q is not part of a record; use %s", s.Name, strings.Join(canon, ", "))
			continue
		}
		present[canon[i]] = true
		if i <= last && ordered {
			ordered = false
			r.add(c, "record-section", file, s.Line,
				"section %q is out of order or repeated; the order is %s, each once", s.Name, strings.Join(canon, ", "))
		}
		last = max(last, i)
		if i < len(record.BodySections) && strings.TrimSpace(s.Text) == "" {
			r.add(c, "record-section", file, s.Line, "section %q is empty", canon[i])
		}
	}
	for _, name := range record.BodySections {
		if !present[name] {
			r.add(c, "record-section", file, 0, "section %q is missing", name)
		}
	}
	for _, line := range p.Outside {
		r.add(c, "record-section", file, line, "text outside the sections; put it under one of %s", strings.Join(canon, ", "))
	}
}
