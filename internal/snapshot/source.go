package snapshot

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/safefile"
	"github.com/wake/docket/internal/safepath"
	"github.com/wake/docket/internal/scan"
)

// Files are file contents by path relative to the repository root.
type Files map[string][]byte

// Wanted is a folder or a file to read in one source. Commit is empty for a
// worktree, whose Source is its absolute path.
type Wanted struct {
	Source, Commit string
	Paths          []string
}

const linkReason = " is a symbolic link; Docket does not read through it"

// Read reads every file under each Wanted path, keyed by Source. A ref costs
// one ls-tree and one cat-file per distinct commit, however many sources and
// paths share it. A link under a path lists "<source>:<path>" as unreadable and
// leaves that path out; a source that cannot be read at all is listed by its
// name and has no entry. The error is for git failing as a whole.
func Read(g gitx.Git, want []Wanted) (map[string]Files, []scan.Unreadable, error) {
	out := map[string]Files{}
	var bad []scan.Unreadable
	byCommit := map[string][]Wanted{}
	var commits []string
	for _, w := range want {
		if w.Commit == "" {
			files, un, reason := readWorktree(w)
			if reason != "" {
				bad = append(bad, scan.Unreadable{Source: w.Source, Reason: reason})
			} else {
				out[w.Source] = files
				bad = append(bad, un...)
			}
			continue
		}
		if _, ok := byCommit[w.Commit]; !ok {
			commits = append(commits, w.Commit)
		}
		byCommit[w.Commit] = append(byCommit[w.Commit], w)
	}
	for _, commit := range commits {
		ws := byCommit[commit]
		var dirs []string
		for _, w := range ws {
			for _, p := range w.Paths {
				if !slices.Contains(dirs, p) {
					dirs = append(dirs, p)
				}
			}
		}
		if len(dirs) == 0 { // no path means no files, not the whole tree
			for _, w := range ws {
				out[w.Source] = Files{}
			}
			continue
		}
		tree, err := g.TreeFiles(commit, dirs...)
		var xe *gitx.ExitError
		if errors.As(err, &xe) { // such as a commit that is not one
			for _, w := range ws {
				bad = append(bad, scan.Unreadable{Source: w.Source, Reason: err.Error()})
			}
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		var paths []string
		for _, f := range tree {
			if !f.Link {
				paths = append(paths, f.Path)
			}
		}
		blobs, err := g.ReadBlobs(commit, paths)
		if err != nil {
			return nil, nil, err
		}
		for _, w := range ws {
			files := Files{}
			for _, p := range w.Paths {
				got, reason := Files{}, ""
				for _, f := range tree {
					if f.Path != p && !strings.HasPrefix(f.Path, p+"/") {
						continue
					}
					b, ok := blobs[f.Path]
					if f.Link {
						reason = f.Path + linkReason
					} else if !ok {
						reason = f.Path + " could not be read"
					} else {
						got[f.Path] = b
					}
					if reason != "" {
						break
					}
				}
				if reason != "" {
					bad = append(bad, scan.Unreadable{Source: w.Source + ":" + p, Reason: reason})
					continue
				}
				for k, v := range got {
					files[k] = v
				}
			}
			out[w.Source] = files
		}
	}
	return out, bad, nil
}

// readWorktree reads w's paths from disk. A non-empty reason means the whole
// source is unreadable; a link under one path is returned in the list and
// leaves only that path out. A path that is not there gives no files.
func readWorktree(w Wanted) (files Files, bad []scan.Unreadable, reason string) {
	if _, err := os.Lstat(w.Source); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, w.Source + " is gone since the scan"
		}
		return nil, nil, err.Error()
	}
	files = Files{}
	for _, rel := range w.Paths {
		if err := safepath.Check(w.Source, path.Dir(rel)); err != nil {
			var se *safepath.SymlinkError
			if errors.As(err, &se) {
				return nil, nil, se.Path + linkReason
			}
			return nil, nil, err.Error()
		}
		start := filepath.Join(w.Source, filepath.FromSlash(rel))
		if _, err := os.Lstat(start); errors.Is(err, fs.ErrNotExist) {
			continue // the change or record is not in this worktree
		}
		got := Files{}
		err := filepath.WalkDir(start, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			r, _ := filepath.Rel(w.Source, p)
			r = filepath.ToSlash(r)
			if d.Type()&fs.ModeSymlink != 0 {
				return errors.New(r + linkReason)
			}
			if d.IsDir() {
				return nil
			}
			b, err := safefile.ReadNoFollow(p)
			got[r] = b
			return err
		})
		if err != nil {
			bad = append(bad, scan.Unreadable{Source: w.Source + ":" + rel, Reason: err.Error()})
			continue
		}
		for k, v := range got {
			files[k] = v
		}
	}
	return files, bad, ""
}
