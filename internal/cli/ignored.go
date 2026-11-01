package cli

import (
	"fmt"
	"path/filepath"

	"github.com/wake/docket/internal/setup"
)

// ignoredFindings reports the project files in files that git ignores. A git
// failure lists each path asked about as unreadable.
func ignoredFindings(c *Context, files []setup.File) ([]Finding, []Unreadable) {
	var paths []string
	for _, f := range files {
		if !filepath.IsAbs(f.Path) && f.Status != setup.Removed {
			paths = append(paths, f.Path)
		}
	}
	ign, err := c.Git.Ignored(paths)
	if err != nil {
		var un []Unreadable
		for _, p := range paths {
			un = append(un, Unreadable{Source: p, Reason: err.Error()})
		}
		return nil, un
	}
	var fnds []Finding
	for _, i := range ign {
		fnds = append(fnds, Finding{Rule: "install-ignored", File: i.Path, Message: fmt.Sprintf("ignored by %s:%d: %s", i.Source, i.Line, i.Pattern)})
	}
	return fnds, nil
}
