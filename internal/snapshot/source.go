package snapshot

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

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
// one ls-tree per path and one cat-file per distinct commit, however many
// sources share it. A source that cannot be read is listed and has no entry;
// the error is for git failing as a whole.
func Read(g gitx.Git, want []Wanted) (map[string]Files, []scan.Unreadable, error) {
	out := map[string]Files{}
	var bad []scan.Unreadable
	byCommit := map[string][]Wanted{}
	var commits []string
	for _, w := range want {
		if w.Commit == "" {
			files, reason := readWorktree(w)
			if reason != "" {
				bad = append(bad, scan.Unreadable{Source: w.Source, Reason: reason})
			} else {
				out[w.Source] = files
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
		listed := map[string][]gitx.TreeFile{} // by wanted path
		var failed string
		for _, w := range ws {
			for _, p := range w.Paths {
				if _, ok := listed[p]; ok || failed != "" {
					continue
				}
				tf, err := g.TreeFiles(commit, p)
				var xe *gitx.ExitError
				if errors.As(err, &xe) {
					failed = err.Error() // such as a commit that is not one
					continue
				}
				if err != nil {
					return nil, nil, err
				}
				listed[p] = tf
			}
		}
		if failed != "" {
			for _, w := range ws {
				bad = append(bad, scan.Unreadable{Source: w.Source, Reason: failed})
			}
			continue
		}
		var paths []string
		for _, tf := range listed {
			for _, f := range tf {
				if !f.Link {
					paths = append(paths, f.Path)
				}
			}
		}
		slices.Sort(paths)
		blobs, err := g.ReadBlobs(commit, slices.Compact(paths))
		if err != nil {
			return nil, nil, err
		}
		for _, w := range ws {
			files, reason := Files{}, ""
			for _, p := range w.Paths {
				for _, f := range listed[p] {
					b, ok := blobs[f.Path]
					switch {
					case f.Link:
						reason = f.Path + linkReason
					case !ok:
						reason = f.Path + " could not be read"
					default:
						files[f.Path] = b
					}
				}
			}
			if reason != "" {
				bad = append(bad, scan.Unreadable{Source: w.Source, Reason: reason})
			} else {
				out[w.Source] = files
			}
		}
	}
	return out, bad, nil
}

// readWorktree reads w's paths from disk. A non-empty reason means the
// source is unreadable. A path that is not there gives no files.
func readWorktree(w Wanted) (Files, string) {
	if _, err := os.Lstat(w.Source); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, w.Source + " is gone since the scan"
		}
		return nil, err.Error()
	}
	files := Files{}
	for _, rel := range w.Paths {
		if err := safepath.Check(w.Source, rel); err != nil {
			var se *safepath.SymlinkError
			if errors.As(err, &se) {
				return nil, se.Path + linkReason
			}
			return nil, err.Error()
		}
		start := filepath.Join(w.Source, filepath.FromSlash(rel))
		if _, err := os.Lstat(start); errors.Is(err, fs.ErrNotExist) {
			continue // the change or record is not in this worktree
		}
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
			files[r] = b
			return err
		})
		if err != nil {
			return nil, err.Error()
		}
	}
	return files, ""
}
