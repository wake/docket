package lint

import (
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"github.com/wake/docket/internal/markdown"
)

var commentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
var angleRe = regexp.MustCompile(`<[A-Za-z][^<>\n]*>`)

// placeholders maps each placeholder string of the shipped templates to the
// first template (by name) that holds it. Angle placeholders inside a
// comment belong to the comment.
func (r *run) placeholders() map[string]string {
	out := map[string]string{}
	if r.in.Templates == nil {
		return out
	}
	names, err := fs.Glob(r.in.Templates, "templates/*.md")
	if err != nil {
		return out
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
			if _, ok := out[m]; !ok {
				out[m] = name
			}
		}
		for _, m := range angleRe.FindAllString(commentRe.ReplaceAllString(text, ""), -1) {
			if _, ok := out[m]; !ok {
				out[m] = name
			}
		}
	}
	return out
}

// checkPlaceholders reports template text left in any file of the change.
func checkPlaceholders(r *run, c *loaded) {
	known := r.placeholders()
	if len(known) == 0 {
		return
	}
	names := make([]string, 0, len(c.Files))
	for n := range c.Files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
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
		// Matches are located in the blanked text and compared in the
		// original, so a code span inside a comment does not hide it.
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
		}
		for _, m := range angleRe.FindAllIndex(blank, -1) {
			report(m[0], m[1])
		}
	}
}

// blankRange replaces bytes with spaces, keeping newlines and offsets.
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
