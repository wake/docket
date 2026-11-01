// Package idea holds the rules of ideas and the order: the files
// docket/ideas/<name>.md and docket/order.md.
package idea

import (
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/markdown"
	"github.com/wake/docket/internal/record"
)

// Dir is the folder that holds ideas; OrderFile is the order.
const (
	Dir       = "docket/ideas"
	OrderFile = "docket/order.md"
)

// FrontMatter is the front matter of an idea file.
type FrontMatter struct {
	Title   string   `yaml:"title"`
	Dropped *bool    `yaml:"dropped,omitempty"`
	Needs   []string `yaml:"needs,omitempty"`
}

// Parse reads the front matter of an idea. It fails when the front matter is
// missing, is not a YAML mapping or a key does not decode to its type. An
// invalid title reads as "".
func Parse(b []byte) (FrontMatter, error) {
	front, _, _, err := record.Split(b)
	if err != nil {
		return FrontMatter{}, err
	}
	var f FrontMatter
	if err := change.DecodeMapping(front, &f); err != nil {
		return FrontMatter{}, err
	}
	if !change.ValidTitle(f.Title) {
		f.Title = ""
	}
	return f, nil
}

// IsIdeaPath reports whether p, a path from the repository root, is
// docket/ideas/<name>.md with a clean path and a name free of "/".
func IsIdeaPath(p string) bool {
	name, ok := strings.CutPrefix(p, Dir+"/")
	return ok && path.Clean(p) == p && !strings.Contains(name, "/") && strings.HasSuffix(name, ".md")
}

// Line is a line of the order that names a path; At is its 1-based line in
// the file.
type Line struct {
	Number int
	Path   string
	At     int
}

// Problem is a line of the order that breaks its format.
type Problem struct {
	At      int
	Message string
}

var orderLine = regexp.MustCompile("^([1-9][0-9]*)\\. `([^`]+)`$")

// numbered matches the start of a line that tries to be an order line.
var numbered = regexp.MustCompile("^([0-9]+)\\.")

// ParseOrder reads the order. Lines of the form are returned even after a
// problem.
func ParseOrder(b []byte) ([]Line, []Problem) {
	var lines []Line
	var problems []Problem
	headed := false
	prev := 0 // the number the last numbered line carried
	for i, raw := range markdown.Lines(b) {
		s := markdown.Trim(raw)
		if s == "" {
			continue
		}
		at := i + 1
		m := orderLine.FindStringSubmatch(s)
		if !headed {
			headed = true
			if s == "# Order" {
				continue
			}
			problems = append(problems, Problem{at, "the order must start with `# Order`"})
			if m == nil {
				continue
			}
		}
		if m == nil {
			problems = append(problems, Problem{at, "a line of the order must read `<n>. `docket/ideas/<name>.md``"})
			if d := numbered.FindStringSubmatch(s); d != nil {
				prev, _ = strconv.Atoi(d[1])
			}
			continue
		}
		n, _ := strconv.Atoi(m[1])
		if n != prev+1 {
			problems = append(problems, Problem{at, "the numbers of the order must run 1, 2, 3 without a gap"})
		}
		prev = n
		lines = append(lines, Line{n, m[2], at})
	}
	if !headed {
		problems = append(problems, Problem{1, "the order must start with `# Order`"})
	}
	return lines, problems
}
