package cli

import (
	"fmt"
	"io/fs"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/setup"
)

// shippedFS is the shipped/ tree that dkt init writes from. Tests may replace
// it and must restore it, as setup.Earlier says.
var shippedFS = func() fs.FS {
	sub, err := fs.Sub(docket.Shipped, "shipped")
	if err != nil { // only a bad path name fails, and it is constant
		panic(err)
	}
	return sub
}

// problemOutcome turns the paths that failed a check into findings (a link)
// and unreadable sources (anything else).
func problemOutcome(ps []setup.Problem) ([]Finding, []Unreadable) {
	var fnds []Finding
	var un []Unreadable
	for _, p := range ps {
		if p.Kind == setup.ProblemSymlink {
			fnds = append(fnds, Finding{Rule: "symlink-refused", File: p.Path, Message: fmt.Sprintf("%s is a symbolic link; dkt will not read or write through it", p.Path)})
			continue
		}
		un = append(un, Unreadable{Source: p.Path, Reason: p.Err.Error()})
	}
	return fnds, un
}

// ignoredFindings reports the paths git ignores and does not track. A git
// failure lists each path asked about as unreadable.
func ignoredFindings(c *Context, paths []string) ([]Finding, []Unreadable) {
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
