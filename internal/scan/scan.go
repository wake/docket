// Package scan finds the changes in every worktree and branch of a repository.
package scan

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/safepath"
)

// Use is one change folder found in one source.
type Use struct {
	Slug     string `json:"slug"`
	Archived bool   `json:"archived"`
	Path     string `json:"path"`   // the folder, relative to the repository root, slash-separated
	Source   string `json:"source"` // a worktree's path, or a ref's full name
	Commit   string `json:"-"`      // the commit of a ref use; empty for a worktree use
}

// Unreadable is a source the scan could not read.
type Unreadable struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

// Inventory is what a scan found.
type Inventory struct {
	Uses       []Use
	Unreadable []Unreadable
}

const (
	changesDir = "openspec/changes"
	archiveDir = changesDir + "/archive"
)

var archivedName = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-(.+)$`)

// useAt returns the use for a change folder path, if the path is one.
func useAt(p, source string) (Use, bool) {
	dir, name := path.Split(p)
	switch strings.TrimSuffix(dir, "/") {
	case changesDir:
		if name == "archive" {
			return Use{}, false
		}
		return Use{Slug: name, Path: p, Source: source}, true
	case archiveDir:
		if m := archivedName.FindStringSubmatch(name); m != nil {
			return Use{Slug: m[1], Archived: true, Path: p, Source: source}, true
		}
	}
	return Use{}, false
}

func sortedUses(paths []string, source string) []Use {
	var uses []Use
	for _, p := range paths {
		if u, ok := useAt(p, source); ok {
			uses = append(uses, u)
		}
	}
	slices.SortFunc(uses, func(a, b Use) int { return strings.Compare(a.Path, b.Path) })
	return uses
}

// Changes scans every worktree, local branch and remote-tracking branch.
func Changes(g gitx.Git) (Inventory, error) {
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
		paths, reason := worktreeDirs(w.Path)
		if reason != "" {
			inv.Unreadable = append(inv.Unreadable, Unreadable{Source: w.Path, Reason: reason})
			continue
		}
		inv.Uses = append(inv.Uses, sortedUses(paths, w.Path)...)
	}
	type result struct {
		paths []string
		err   error
	}
	byCommit := map[string]result{}
	for _, r := range refs {
		res, ok := byCommit[r.Commit]
		if !ok {
			res.paths, res.err = g.TreeDirs(r.Commit, changesDir+"/", archiveDir+"/")
			byCommit[r.Commit] = res
		}
		if res.err != nil {
			inv.Unreadable = append(inv.Unreadable, Unreadable{Source: r.Name, Reason: res.err.Error()})
			continue
		}
		uses := sortedUses(res.paths, r.Name)
		for i := range uses {
			uses[i].Commit = r.Commit
		}
		inv.Uses = append(inv.Uses, uses...)
	}
	return inv, nil
}

// worktreeMissing says why root cannot be read at all, or "" if it can.
func worktreeMissing(root string) string {
	if _, err := os.Lstat(root); errors.Is(err, fs.ErrNotExist) {
		return "directory missing; if the worktree was deleted, run git worktree prune"
	} else if err != nil {
		return err.Error()
	}
	return ""
}

// worktreeDirs lists the change folders on disk under root as repository
// paths; a non-empty reason means the worktree is unreadable.
func worktreeDirs(root string) (paths []string, reason string) {
	if reason := worktreeMissing(root); reason != "" {
		return nil, reason
	}
	if err := safepath.Check(root, archiveDir); err != nil {
		var se *safepath.SymlinkError
		if errors.As(err, &se) {
			return nil, se.Path + " is a symbolic link; Docket does not read through it"
		}
		if errors.Is(err, syscall.ENOTDIR) || strings.HasSuffix(err.Error(), "is not a directory") {
			return nil, "" // a file where a folder would be: no changes here
		}
		return nil, err.Error()
	}
	for _, dir := range []string{changesDir, archiveDir} {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			continue
		}
		if err != nil {
			return nil, err.Error()
		}
		for _, e := range entries {
			if e.IsDir() {
				paths = append(paths, dir+"/"+e.Name())
			}
		}
	}
	return paths, ""
}
