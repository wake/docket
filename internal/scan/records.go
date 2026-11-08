package scan

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/safepath"
)

// Records scans for record files: see Scan.
func Records(g gitx.Git) (Inventory, error) {
	ws, err := Scan(g)
	return ws.Records, err
}

// Issues scans for issue files: see Scan.
func Issues(g gitx.Git) (Inventory, error) {
	ws, err := Scan(g)
	return ws.Issues, err
}

// treeRecords returns the files directly in dir from a commit's listing.
func treeRecords(files []gitx.TreeFile, dir string) []string {
	var paths []string
	for _, f := range files {
		if path.Dir(f.Path) == dir {
			paths = append(paths, f.Path)
		}
	}
	return paths
}

func recordUses(paths []string, source string) []Use {
	var uses []Use
	for _, p := range paths {
		if slug, ok := strings.CutSuffix(filepath.Base(p), ".md"); ok && slug != "" {
			uses = append(uses, Use{Slug: slug, Path: p, Source: source})
		}
	}
	slices.SortFunc(uses, func(a, b Use) int { return strings.Compare(a.Path, b.Path) })
	return uses
}

// worktreeRecords lists the files on disk in root's dir folder as
// repository paths; a non-empty reason means the worktree is unreadable.
func worktreeRecords(root, dir string) (paths []string, reason string) {
	if reason := worktreeMissing(root); reason != "" {
		return nil, reason
	}
	if err := safepath.Check(root, dir); err != nil {
		var se *safepath.SymlinkError
		if errors.As(err, &se) {
			return nil, se.Path + " is a symbolic link; Docket does not read through it"
		}
		if errors.Is(err, syscall.ENOTDIR) || strings.HasSuffix(err.Error(), "is not a directory") {
			return nil, "" // a file where a folder would be: no records here
		}
		return nil, err.Error()
	}
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return nil, ""
	}
	if err != nil {
		return nil, err.Error()
	}
	for _, e := range entries {
		if !e.IsDir() {
			paths = append(paths, dir+"/"+e.Name())
		}
	}
	return paths, ""
}
