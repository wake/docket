// Package issue holds the rules of file issues: the files
// docket/issues/<YYYY-MM-DD>-<name>.md.
package issue

import (
	"go.yaml.in/yaml/v3"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/record"
)

// Dir is the folder that holds file issues.
const Dir = "docket/issues"

// Sections are the level-two headings of an issue, in order.
var Sections = []string{"Current", "Expected", "Unchanged", "Found"}

// FrontMatter is the front matter of an issue file.
type FrontMatter struct {
	Title   string `yaml:"title"`
	Source  string `yaml:"source"`
	Dropped *bool  `yaml:"dropped,omitempty"`
	Idea    string `yaml:"idea,omitempty"`
}

// ParseFrontMatter reads the front matter. It fails when b is not a YAML
// mapping or a key does not decode to its type. An invalid title reads as "".
func ParseFrontMatter(b []byte) (FrontMatter, error) {
	var f FrontMatter
	if err := change.DecodeMapping(b, &f); err != nil {
		return FrontMatter{}, err
	}
	if !change.ValidTitle(f.Title) {
		f.Title = ""
	}
	return f, nil
}

// File is a parsed issue.
type File struct {
	FrontMatter    FrontMatter
	FrontMatterErr error
	Sections       []record.Section
}

// Parse reads an issue: Split, then the front matter.
func Parse(b []byte) File {
	var f File
	front, secs, _, err := record.Split(b)
	f.Sections = secs
	if err != nil {
		f.FrontMatterErr = err
	} else {
		f.FrontMatter, f.FrontMatterErr = ParseFrontMatter(front)
	}
	return f
}

// Render is the content of a new issue: the front matter, a blank line and
// the template.
func Render(title, source, template string) ([]byte, error) {
	front, err := yaml.Marshal(FrontMatter{Title: title, Source: source})
	if err != nil {
		return nil, err
	}
	return record.Wrap(front, template), nil
}

// Result is the result object of dkt issue.
type Result struct {
	Slug   string `json:"slug"`
	Path   string `json:"path"`
	Title  string `json:"title"`
	Source string `json:"source"`
	// Requested is the slug asked for; Slug differs when a suffix was taken.
	Requested string `json:"requested"`
	// Used lists where the requested slug was found; never null.
	Used []string `json:"used"`
}
