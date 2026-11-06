package lint

import (
	"errors"
	"io/fs"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/markdown"
)

// checkFront judges the front matter of a docket change's proposal.
func checkFront(r *state, c *loaded, fm change.Front, has bool, err error) {
	switch {
	case err != nil:
		r.add(c, "metadata-invalid", proposalFile, 0, "front matter is not valid: %v", err)
		return
	case !has:
		r.add(c, "title-missing", proposalFile, 0, "proposal.md has no front matter with a title")
		r.add(c, "metadata-invalid", proposalFile, 0, "kind is missing: proposal.md has no front matter")
		r.add(c, "metadata-invalid", proposalFile, 0, "source is missing: proposal.md has no front matter")
		return
	}
	if f, ok := fm.Fields["title"]; !ok {
		r.add(c, "title-missing", proposalFile, 0, "title is missing from the front matter")
	} else if !change.ValidLine(f.Value) {
		r.add(c, "title-missing", proposalFile, f.Line, "title must be one line of text")
	} else if msg, tooLong := change.TitleTooLong(f.Value); tooLong {
		r.add(c, "title-too-long", proposalFile, f.Line, "%s", msg)
	}
	if f, ok := fm.Fields["kind"]; !ok {
		r.add(c, "metadata-invalid", proposalFile, 0, "kind is missing from the front matter")
	} else if !slices.Contains(change.Kinds, f.Value) {
		r.add(c, "metadata-invalid", proposalFile, f.Line, "kind must be one of %s", strings.Join(change.Kinds, ", "))
	}
	if f, ok := fm.Fields["source"]; !ok {
		r.add(c, "metadata-invalid", proposalFile, 0, "source is missing from the front matter")
	} else if !change.ValidLine(f.Value) {
		r.add(c, "metadata-invalid", proposalFile, f.Line, "source must be one line of text")
	}
}

// checkNewCapabilities wants a name under specs: for each capability the
// change has a delta file for and the repository has no main spec for.
func checkNewCapabilities(r *state, c *loaded, fm change.Front, readable bool) {
	var caps []string
	for name := range c.Files {
		if rest, ok := strings.CutPrefix(name, "specs/"); ok {
			if cp, ok := strings.CutSuffix(rest, "/spec.md"); ok && cp != "" {
				caps = append(caps, cp)
			}
		}
	}
	sort.Strings(caps)
	names, line := map[string]string(nil), 0
	if readable {
		names, line = fm.Specs, fm.SpecsLine
	}
	for _, cp := range caps {
		main := specsDir + "/" + cp + "/spec.md"
		if _, err := fs.Stat(r.in.Root, main); err == nil {
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			r.unreadable(main, err)
			continue
		}
		if strings.TrimSpace(names[cp]) == "" {
			r.add(c, "title-missing", proposalFile, line,
				"new capability %q has no name: add `%s: <name>` under specs: in the front matter", cp, cp)
		}
	}
}

var requiredSections = []string{"Why", "What Changes", "Non-Goals", "Capabilities", "Impact"}

// checkSections applies the proposal-section rule.
func checkSections(r *state, c *loaded) {
	if c.Skipped(proposalFile) {
		return
	}
	if !c.Has(proposalFile) {
		r.add(c, "proposal-section", proposalFile, 0, "proposal.md is missing")
		return
	}
	lines := markdown.Lines(c.Files[proposalFile])
	mask := markdown.FenceMask(lines)
	blank := markdown.BlankComments(lines, mask)
	heads := markdown.Headings(blank, mask)
	for _, name := range requiredSections {
		i := slices.IndexFunc(heads, func(h markdown.Heading) bool {
			return h.Level == 2 && strings.EqualFold(markdown.Trim(h.Text), name)
		})
		if i < 0 {
			r.add(c, "proposal-section", proposalFile, 0, "section %q is missing", name)
			continue
		}
		if name != "Non-Goals" {
			continue
		}
		from, to := markdown.Body(blank, mask, heads[i])
		if !slices.ContainsFunc(blank[from:to], func(l string) bool { return markdown.Trim(l) != "" }) {
			r.add(c, "proposal-section", proposalFile, heads[i].Index+1, "section %q is empty", name)
		}
	}
}

// SpecTitles reports main specs without a title, the sources it could not
// read and the number of main specs it found.
func SpecTitles(root fs.FS) ([]Finding, []Unreadable, int) {
	var findings []Finding
	var bad []Unreadable
	specs := 0
	_ = fs.WalkDir(root, specsDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				bad = append(bad, Unreadable{Source: p, Reason: err.Error()})
			}
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || d.Name() != "spec.md" {
			return nil
		}
		specs++
		b, err := fs.ReadFile(root, p)
		if err != nil {
			bad = append(bad, Unreadable{Source: p, Reason: err.Error()})
			return nil
		}
		first := ""
		if lines := markdown.Lines(b); len(lines) > 0 {
			first = lines[0]
		}
		cp := strings.Trim(strings.TrimSuffix(strings.TrimPrefix(p, specsDir), "spec.md"), "/")
		if _, ok := change.SpecTitle(first, cp); !ok {
			findings = append(findings, Finding{Rule: "title-missing", File: p, Line: 1,
				Message: "main spec has no title: its first line must be `# <name> Specification`, the name differing from " + cp})
		}
		return nil
	})
	return findings, bad, specs
}

var commentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
var angleRe = regexp.MustCompile(`<[A-Za-z][^<>\n]*>`)

// placeholders maps each placeholder text of the shipped templates to the
// first template that holds it, read once per run.
func (r *state) placeholders() map[string]string {
	if r.knownDone {
		return r.known
	}
	r.knownDone = true
	r.known = map[string]string{}
	const source = "schema templates/*.md"
	if r.in.Templates == nil {
		r.unreadable(source, errors.New("no templates given"))
		return r.known
	}
	names, err := fs.Glob(r.in.Templates, "templates/*.md")
	if err != nil {
		r.unreadable(source, err)
		return r.known
	}
	if len(names) == 0 {
		r.unreadable(source, errors.New("no template found"))
		return r.known
	}
	sort.Strings(names)
	for _, name := range names {
		b, err := fs.ReadFile(r.in.Templates, name)
		if err != nil {
			r.unreadable("schema "+name, err)
			continue
		}
		text := strings.Join(markdown.Lines(b), "\n")
		for _, m := range commentRe.FindAllString(text, -1) {
			if _, ok := r.known[m]; !ok {
				r.known[m] = name
			}
		}
		for _, m := range angleRe.FindAllString(commentRe.ReplaceAllString(text, ""), -1) {
			if _, ok := r.known[m]; !ok {
				r.known[m] = name
			}
		}
	}
	return r.known
}

// checkPlaceholders reports template text left in any Markdown file of the change.
func checkPlaceholders(r *state, c *loaded) {
	known := r.placeholders()
	if len(known) == 0 {
		return
	}
	names := make([]string, 0, len(c.Files))
	for n := range c.Files {
		if strings.HasSuffix(n, ".md") {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		checkPlaceholderFile(r, c, name, known)
	}
}

func checkPlaceholderFile(r *state, c *loaded, name string, known map[string]string) {
	lines := markdown.Lines(c.Files[name])
	mask := markdown.FenceMask(lines)
	orig := strings.Join(lines, "\n")
	blank := []byte(orig)
	off := 0
	for i, line := range lines {
		if mask[i] {
			blankRange(blank, off, off+len(line))
		} else {
			for _, sp := range markdown.CodeSpans(line) {
				blankRange(blank, off+sp[0], off+sp[1])
			}
		}
		off += len(line) + 1
	}
	// Matches are located in the blanked text and compared in the original,
	// so a code span inside a comment does not hide it.
	report := func(start, end int) {
		ph := orig[start:end]
		tmpl, ok := known[ph]
		if !ok {
			return
		}
		line := strings.Count(orig[:start], "\n") + 1
		r.add(c, "placeholder", name, line, "template text left: %s (from %s)", quote(ph), tmpl)
		blankRange(blank, start, end)
	}
	for _, m := range commentRe.FindAllIndex(blank, -1) {
		report(m[0], m[1])
		blankRange(blank, m[0], m[1]) // tags inside any comment belong to the comment
	}
	for _, m := range angleRe.FindAllIndex(blank, -1) {
		report(m[0], m[1])
	}
}

func blankRange(b []byte, from, to int) {
	for i := from; i < to && i < len(b); i++ {
		if b[i] != '\n' {
			b[i] = ' '
		}
	}
}

func quote(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 60 {
		s = string(r[:57]) + "..."
	}
	return "\"" + s + "\""
}
