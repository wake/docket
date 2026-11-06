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
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/safepath"
)

// Records scans every worktree, local branch and remote-tracking branch for
// record files; each use is one docket/records/<slug>.md file and its Slug has
// the .md removed.
func Records(g gitx.Git) (Inventory, error) {
	var inv Inventory
	wts, err := g.Worktrees()
	if err != nil {
		return inv, err
	}
	refs, err := g.Refs()
	if err != nil {
		return inv, err
	}
	for _, w := range wts {
		if w.Bare {
			continue
		}
		paths, reason := worktreeRecords(w.Path)
		if reason != "" {
			inv.Unreadable = append(inv.Unreadable, Unreadable{Source: w.Path, Reason: reason})
			continue
		}
		inv.Uses = append(inv.Uses, recordUses(paths, w.Path)...)
	}
	type result struct {
		paths []string
		err   error
	}
	byCommit := map[string]result{}
	for _, r := range refs {
		res, ok := byCommit[r.Commit]
		if !ok {
			res.paths, res.err = treeRecords(g, r.Commit)
			byCommit[r.Commit] = res
		}
		if res.err != nil {
			inv.Unreadable = append(inv.Unreadable, Unreadable{Source: r.Name, Reason: res.err.Error()})
			continue
		}
		inv.Uses = append(inv.Uses, recordUses(res.paths, r.Name)...)
	}
	return inv, nil
}

// treeRecords lists the files directly in the records folder of commit's tree.
func treeRecords(g gitx.Git, commit string) ([]string, error) {
	files, err := g.TreeFiles(commit, record.Dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, f := range files {
		if path.Dir(f.Path) == record.Dir {
			paths = append(paths, f.Path)
		}
	}
	return paths, nil
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

// worktreeRecords lists the files on disk in root's records folder as
// repository paths; a non-empty reason means the worktree is unreadable.
func worktreeRecords(root string) (paths []string, reason string) {
	if reason := worktreeMissing(root); reason != "" {
		return nil, reason
	}
	if err := safepath.Check(root, record.Dir); err != nil {
		var se *safepath.SymlinkError
		if errors.As(err, &se) {
			return nil, se.Path + " is a symbolic link; Docket does not read through it"
		}
		if errors.Is(err, syscall.ENOTDIR) || strings.HasSuffix(err.Error(), "is not a directory") {
			return nil, "" // a file where a folder would be: no records here
		}
		return nil, err.Error()
	}
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(record.Dir)))
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return nil, ""
	}
	if err != nil {
		return nil, err.Error()
	}
	for _, e := range entries {
		if !e.IsDir() {
			paths = append(paths, record.Dir+"/"+e.Name())
		}
	}
	return paths, ""
}
