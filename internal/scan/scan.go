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
	"github.com/wake/docket/internal/issue"
	"github.com/wake/docket/internal/record"
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

// Workspace is what one Scan found: the three inventories, and the lists they
// were made from.
type Workspace struct {
	Changes, Records, Issues Inventory
	Worktrees                []gitx.Worktree
	Refs                     []gitx.Ref
	Trees                    map[string][]gitx.TreeFile // by commit; a commit that could not be listed is not here
}

// Scan scans every worktree, local branch and remote-tracking branch for
// changes, records and issues. It lists the worktrees and the refs once and
// each distinct ref commit once, under the three folders and extra.
func Scan(g gitx.Git, extra ...string) (Workspace, error) {
	ws := Workspace{Trees: map[string][]gitx.TreeFile{}}
	var err error
	if ws.Worktrees, err = g.Worktrees(); err != nil {
		return ws, err
	}
	if ws.Refs, err = g.Refs(); err != nil {
		return ws, err
	}
	for _, w := range ws.Worktrees {
		if w.Bare {
			continue
		}
		for _, one := range []struct {
			inv  *Inventory
			read func() ([]string, string)
			uses func([]string, string) []Use
		}{
			{&ws.Changes, func() ([]string, string) { return worktreeDirs(w.Path) }, sortedUses},
			{&ws.Records, func() ([]string, string) { return worktreeRecords(w.Path, record.Dir) }, recordUses},
			{&ws.Issues, func() ([]string, string) { return worktreeRecords(w.Path, issue.Dir) }, recordUses},
		} {
			paths, reason := one.read()
			if reason != "" {
				one.inv.Unreadable = append(one.inv.Unreadable, Unreadable{Source: w.Path, Reason: reason})
				continue
			}
			one.inv.Uses = append(one.inv.Uses, one.uses(paths, w.Path)...)
		}
	}
	roots := append([]string{changesDir, record.Dir, issue.Dir}, extra...)
	errs := map[string]error{}
	for _, r := range ws.Refs {
		if _, ok := ws.Trees[r.Commit]; ok || errs[r.Commit] != nil {
			continue
		}
		if files, err := g.TreeFiles(r.Commit, roots...); err != nil {
			errs[r.Commit] = err
		} else {
			ws.Trees[r.Commit] = files
		}
	}
	for _, r := range ws.Refs {
		if err := errs[r.Commit]; err != nil {
			for _, inv := range []*Inventory{&ws.Changes, &ws.Records, &ws.Issues} {
				inv.Unreadable = append(inv.Unreadable, Unreadable{Source: r.Name, Reason: err.Error()})
			}
			continue
		}
		files := ws.Trees[r.Commit]
		for _, one := range []struct {
			inv   *Inventory
			paths []string
			uses  func([]string, string) []Use
		}{
			{&ws.Changes, treeChangeDirs(files), sortedUses},
			{&ws.Records, treeRecords(files, record.Dir), recordUses},
			{&ws.Issues, treeRecords(files, issue.Dir), recordUses},
		} {
			uses := one.uses(one.paths, r.Name)
			for i := range uses {
				uses[i].Commit = r.Commit
			}
			one.inv.Uses = append(one.inv.Uses, uses...)
		}
	}
	return ws, nil
}

// Changes scans for change folders: see Scan.
func Changes(g gitx.Git) (Inventory, error) {
	ws, err := Scan(g)
	return ws.Changes, err
}

// treeChangeDirs returns the direct subfolders of the changes and archive
// folders that files has a file under, which is what ls-tree -d gave.
func treeChangeDirs(files []gitx.TreeFile) []string {
	var dirs []string
	seen := map[string]bool{}
	for _, f := range files {
		rest, ok := strings.CutPrefix(f.Path, changesDir+"/")
		if !ok {
			continue
		}
		parts := strings.Split(rest, "/")
		dir := changesDir + "/" + parts[0]
		if parts[0] == "archive" && len(parts) > 2 {
			dir = archiveDir + "/" + parts[1]
		}
		if len(parts) > 1 && !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	return dirs
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
