// Package conflicts finds requirements that several active changes touch.
package conflicts

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/safepath"
	"github.com/wake/docket/internal/scan"
)

// Copy is the delta specs of one active change as found in one or more sources.
type Copy struct {
	Slug    string
	Folder  string            // the change folder, such as openspec/changes/<slug>
	Sources []string          // in the scan's order
	Files   map[string][]byte // spec.md files by path under specs/, such as booking/spec.md
}

// Copies reads the delta specs of every active change from every source and
// merges identical copies. A slug that is archived in any source is left out.
// A copy that cannot be read is reported and gives no Copy.
func Copies(g gitx.Git, inv scan.Inventory) ([]Copy, []scan.Unreadable, error) {
	archived := map[string]bool{}
	for _, u := range inv.Uses {
		if u.Archived {
			archived[u.Slug] = true
		}
	}
	var copies []Copy
	var unreadable []scan.Unreadable
	index := map[string]int{} // slug + copy identity -> position in copies
	for _, u := range inv.Uses {
		if u.Archived || archived[u.Slug] {
			continue
		}
		var files map[string][]byte
		var reason string
		var err error
		if u.Commit == "" {
			files, reason = worktreeSpecs(u)
		} else if files, reason, err = refSpecs(g, u); err != nil {
			return nil, nil, err
		}
		if reason != "" {
			unreadable = append(unreadable, scan.Unreadable{Source: u.Source + ":" + u.Path, Reason: reason})
			continue
		}
		key := u.Slug + "\x00" + identity(files)
		if i, ok := index[key]; ok {
			copies[i].Sources = append(copies[i].Sources, u.Source)
			continue
		}
		index[key] = len(copies)
		copies = append(copies, Copy{Slug: u.Slug, Folder: u.Path, Sources: []string{u.Source}, Files: files})
	}
	return copies, unreadable, nil
}

// identity is the sorted (path, content) pairs of files as one string.
func identity(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for p := range files {
		names = append(names, p)
	}
	slices.Sort(names)
	var b strings.Builder
	for _, p := range names {
		b.WriteString(p + "\x00" + string(files[p]) + "\x00\x00")
	}
	return b.String()
}

const linkReason = " is a symbolic link; Docket does not read through it"

// worktreeSpecs reads the spec.md files under u's specs folder on disk. A
// non-empty reason means the copy is unreadable.
func worktreeSpecs(u scan.Use) (files map[string][]byte, reason string) {
	files = map[string][]byte{}
	if _, err := os.Lstat(filepath.Join(u.Source, filepath.FromSlash(u.Path))); errors.Is(err, fs.ErrNotExist) {
		return nil, u.Path + " is gone since the scan"
	} else if err != nil {
		return nil, err.Error()
	}
	specs := u.Path + "/specs"
	if err := safepath.Check(u.Source, specs); err != nil {
		var se *safepath.SymlinkError
		if errors.As(err, &se) {
			return nil, se.Path + linkReason
		}
		return nil, err.Error()
	}
	root, err := os.OpenRoot(u.Source)
	if err != nil {
		return nil, err.Error()
	}
	defer root.Close()
	sub, err := root.OpenRoot(specs)
	if errors.Is(err, fs.ErrNotExist) {
		return files, ""
	}
	if err != nil {
		return nil, err.Error()
	}
	defer sub.Close()
	fsys := sub.FS()
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return errors.New(specs + "/" + p + linkReason)
		}
		if d.IsDir() || d.Name() != "spec.md" {
			return nil
		}
		b, err := fs.ReadFile(fsys, p)
		files[p] = b
		return err
	})
	if err != nil {
		return nil, err.Error()
	}
	return files, ""
}

// refSpecs reads the spec.md files under u's specs folder in u's commit. A
// non-empty reason means the copy is unreadable; err is a git failure.
func refSpecs(g gitx.Git, u scan.Use) (files map[string][]byte, reason string, err error) {
	specs := u.Path + "/specs"
	tree, err := g.TreeFiles(u.Commit, specs)
	if err != nil {
		return nil, err.Error(), nil
	}
	var want []string
	for _, f := range tree {
		if f.Link {
			return nil, f.Path + linkReason, nil
		}
		if path.Base(f.Path) == "spec.md" {
			want = append(want, f.Path)
		}
	}
	blobs, err := g.ReadBlobs(u.Commit, want)
	if err != nil {
		return nil, err.Error(), nil
	}
	files = map[string][]byte{}
	for _, p := range want {
		b, ok := blobs[p]
		if !ok {
			return nil, p + " could not be read", nil
		}
		files[strings.TrimPrefix(p, specs+"/")] = b
	}
	return files, "", nil
}
