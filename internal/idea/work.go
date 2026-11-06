package idea

import (
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/safefile"
	"github.com/wake/docket/internal/safepath"
	"github.com/wake/docket/internal/scan"
)

const linkReason = " is a symbolic link; Docket does not read through it"

// FindWork reads the front matter of every change's proposal.md and every
// record in all sources and keys the result by their exact source value. Work
// is on the tip only when a copy read from commit tip names the idea.
func FindWork(g gitx.Git, tip string) (map[string]Work, []scan.Unreadable, error) {
	changes, err := scan.Changes(g)
	if err != nil {
		return nil, nil, err
	}
	records, err := scan.Records(g)
	if err != nil {
		return nil, nil, err
	}
	refs, err := g.Refs()
	if err != nil {
		return nil, nil, err
	}
	commits := map[string]string{}
	for _, r := range refs {
		commits[r.Name] = r.Commit
	}
	bad := append(append([]scan.Unreadable{}, changes.Unreadable...), records.Unreadable...)
	work := map[string]Work{}
	note := func(source, p string, b []byte, onTip bool) {
		fm, ok, err := change.ReadFrontMatter(b)
		if err != nil {
			bad = append(bad, scan.Unreadable{Source: source + ":" + p, Reason: err.Error()})
			return
		}
		if f, found := fm.Fields["source"]; ok && found {
			w := work[f.Value]
			w.Any, w.OnTip = true, w.OnTip || onTip
			work[f.Value] = w
		}
	}
	type file struct{ source, path string }
	byCommit := map[string][]file{}
	visit := func(u scan.Use, p string) {
		if filepath.IsAbs(u.Source) {
			b, reason := readWorktree(u.Source, p)
			if reason != "" {
				bad = append(bad, scan.Unreadable{Source: u.Source + ":" + p, Reason: reason})
			} else if b != nil {
				note(u.Source, p, b, false)
			}
			return
		}
		c := u.Commit
		if c == "" {
			c = commits[u.Source]
		}
		if c != "" { // a ref deleted since the scan names nothing
			byCommit[c] = append(byCommit[c], file{u.Source, p})
		}
	}
	for _, u := range changes.Uses {
		visit(u, u.Path+"/proposal.md")
	}
	for _, u := range records.Uses {
		visit(u, u.Path)
	}
	for c, files := range byCommit {
		paths := make([]string, len(files))
		for i, f := range files {
			paths[i] = f.path
		}
		slices.Sort(paths)
		paths = slices.Compact(paths)
		blobs, err := g.ReadBlobs(c, paths)
		if err != nil {
			return nil, nil, err
		}
		for _, f := range files {
			if b, ok := blobs[f.path]; ok {
				note(f.source, f.path, b, c == tip)
			}
		}
	}
	slices.SortFunc(bad, func(a, b scan.Unreadable) int { return strings.Compare(a.Source, b.Source) })
	return work, bad, nil
}

// readWorktree returns nil bytes and no reason when the file is missing.
func readWorktree(root, rel string) ([]byte, string) {
	if err := safepath.Check(root, rel); err != nil {
		var se *safepath.SymlinkError
		switch {
		case errors.As(err, &se):
			return nil, se.Path + linkReason
		case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
			return nil, ""
		}
		return nil, err.Error()
	}
	b, err := safefile.ReadNoFollow(filepath.Join(root, filepath.FromSlash(rel)))
	switch {
	case err == nil:
		return b, ""
	case errors.Is(err, fs.ErrNotExist):
		return nil, ""
	}
	return nil, err.Error()
}
