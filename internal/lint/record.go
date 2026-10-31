package lint

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/wake/docket/internal/record"
)

// lintRecords checks the named records, or the records the branch adds when
// none is named. A record that is gone is skipped unless it was named.
func (r *run) lintRecords(named []string) {
	var slugs []string
	if len(r.in.Slugs) == 0 {
		r.checkTrailers()
		seen := map[string]bool{}
		for _, p := range r.in.AddedRecords {
			name, ok := strings.CutPrefix(p, record.Dir+"/")
			if !ok || strings.Contains(name, "/") {
				continue
			}
			if slug, ok := record.SlugOf(name); ok && !seen[slug] {
				seen[slug] = true
				slugs = append(slugs, slug)
			}
		}
		sort.Strings(slugs)
	} else {
		seen := map[string]bool{}
		for _, slug := range named {
			if !seen[slug] {
				seen[slug] = true
				slugs = append(slugs, slug)
			}
		}
	}
	if len(slugs) == 0 {
		return
	}
	entries, err := fs.ReadDir(r.in.Repo, record.Dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		r.unreadable(record.Dir, err)
		return
	}
	regular := map[string]bool{}
	for _, e := range entries {
		regular[e.Name()] = e.Type().IsRegular()
	}
	for _, slug := range slugs {
		name := slug + ".md"
		if _, ok := record.SlugOf(name); !ok || strings.Contains(slug, "/") || !regular[name] {
			if len(r.in.Slugs) > 0 {
				r.rep.Findings = append(r.rep.Findings, Finding{
					Rule:    "record-missing",
					Message: fmt.Sprintf("record %q does not exist under %s", slug, record.Dir),
				})
			}
			continue
		}
		p := record.Dir + "/" + name
		b, err := fs.ReadFile(r.in.Repo, p)
		if err != nil {
			r.unreadable(p, err)
			continue
		}
		r.rep.Records = append(r.rep.Records, slug)
		c := &loaded{Dir: record.Dir, Files: map[string][]byte{name: b}}
		checkRecord(r, c, name, record.Parse(b))
	}
}

// sectionOrder is the order of the level-two sections of a record.
var sectionOrder = append(append([]string{}, record.Sections...), record.TasksSection)

// checkRecord applies the record rules to the record in file name of c.
func checkRecord(r *run, c *loaded, name string, f record.File) {
	checkRecordFrontMatter(r, c, name, f)
	checkRecordSections(r, c, name, f)
	checkTaskHeadings(r, c, name, f.Plan)
	checkRecordTasks(r, c, name, f)
	checkPlaceholderFile(r, c, name, c.Files[name])
}

func checkRecordFrontMatter(r *run, c *loaded, name string, f record.File) {
	if f.FrontMatterErr != nil {
		r.add(c, "record-front-matter", name, 1, "front matter: %v", f.FrontMatterErr)
		return
	}
	d := f.FrontMatter
	if d.Title == "" {
		r.add(c, "record-front-matter", name, 1, "front matter needs a `title` on one line")
	}
	if strings.TrimSpace(d.Source) == "" {
		r.add(c, "record-front-matter", name, 1, "front matter needs a non-empty `source`")
	}
	for _, k := range record.Kinds {
		if d.Kind == k {
			return
		}
	}
	r.add(c, "record-kind", name, 1, "kind %q is not a record kind; use one of %s", d.Kind, strings.Join(record.Kinds, ", "))
}

func checkRecordSections(r *run, c *loaded, name string, f record.File) {
	first := map[string]record.Section{}
	last := -1
	for _, s := range f.Sections {
		idx := indexOf(sectionOrder, s.Name)
		switch {
		case idx < 0:
			r.add(c, "record-section", name, s.Line, "`## %s` is not a record section; the sections are %s", s.Name, headingList())
			continue
		case first[s.Name].Line != 0:
			r.add(c, "record-section", name, s.Line, "`## %s` appears more than once (first on line %d)", s.Name, first[s.Name].Line)
			continue
		case idx < last:
			r.add(c, "record-section", name, s.Line, "`## %s` is out of order; the sections are %s", s.Name, headingList())
		}
		first[s.Name] = s
		last = max(last, idx)
	}
	for _, n := range record.Sections {
		s, ok := first[n]
		if !ok {
			r.add(c, "record-section", name, 0, "the record has no `## %s` section", n)
			continue
		}
		if strings.TrimSpace(s.Text) == "" {
			r.add(c, "record-section", name, s.Line, "`## %s` is empty", n)
		}
		if w := record.Words(s.Text); w > record.WordLimit {
			r.add(c, "record-too-long", name, s.Line, "`## %s` has %s words; the limit is %d",
				n, strconv.FormatFloat(w, 'f', -1, 64), record.WordLimit)
		}
	}
}

// checkRecordTasks reports task headings that are not under `## Tasks`.
func checkRecordTasks(r *run, c *loaded, name string, f record.File) {
	tasksLine := 0
	for _, s := range f.Sections {
		if s.Name == record.TasksSection {
			tasksLine = s.Line
			break
		}
	}
	for _, t := range f.Plan.Tasks {
		if tasksLine == 0 || t.Line < tasksLine {
			r.add(c, "task-heading", name, t.Line, "Task %d: task headings belong under `## %s`", t.Number, record.TasksSection)
		}
	}
}

func headingList() string {
	parts := make([]string, len(sectionOrder))
	for i, n := range sectionOrder {
		parts[i] = "`## " + n + "`"
	}
	return strings.Join(parts, ", ") + " (Tasks optional), in that order"
}

func indexOf(list []string, s string) int {
	for i, x := range list {
		if x == s {
			return i
		}
	}
	return -1
}

// checkTrailers reports each distinct Record trailer value of a branch commit
// that names no regular file under docket/records. fs.Stat rejects a value
// with ".." in a path element or a leading slash; SlugOf rejects the rest.
func (r *run) checkTrailers() {
	type key struct{ sha, v string }
	seen := map[key]bool{}
	for _, c := range r.in.Commits {
		for _, v := range c.Record {
			if seen[key{c.SHA, v}] {
				continue
			}
			seen[key{c.SHA, v}] = true
			if _, ok := record.SlugOf(v + ".md"); ok && !strings.Contains(v, "/") {
				if fi, err := fs.Stat(r.in.Repo, record.Dir+"/"+v+".md"); err == nil && fi.Mode().IsRegular() {
					continue
				}
			}
			short := c.SHA
			if len(short) > 7 {
				short = short[:7]
			}
			r.rep.Findings = append(r.rep.Findings, Finding{
				Rule:    "record-trailer",
				Message: fmt.Sprintf("commit %s has trailer Record: %s, which names no record under %s", short, v, record.Dir),
			})
		}
	}
}
