package scan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/wake/docket/internal/gitx"
)

// Kind says what a Source is.
type Kind string

const (
	KindWorktree      Kind = "worktree"
	KindLocalBranch   Kind = "local-branch"
	KindRemoteBranch  Kind = "remote-branch"
	KindDefaultBranch Kind = "default-branch"
)

var kindRank = map[Kind]int{KindWorktree: 0, KindLocalBranch: 1, KindRemoteBranch: 2, KindDefaultBranch: 3}

// Source is one place a change was found.
type Source struct {
	Kind   Kind   `json:"kind"`
	Name   string `json:"name"` // worktree path or full refname
	Commit string `json:"commit"`
}

// Change is an active change, reported once however many sources hold it.
type Change struct {
	Slug    string
	Sources []Source // Sources[0] is where content is read from
}

// Archived is an archived use of a slug.
type Archived struct {
	Slug, Folder string
	Sources      []Source
}

// Unreadable is a source that could not be read.
type Unreadable struct {
	Source Source
	Reason string
}

// Inventory is the result of a scan.
type Inventory struct {
	DefaultBranch DefaultBranch
	Changes       []Change
	Archived      []Archived
	Unreadable    []Unreadable
}

const (
	changesDir = "openspec/changes"
	archiveDir = changesDir + "/archive"
)

var archiveName = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-(.+)$`)

type found struct {
	active   map[string][]Source
	archived map[string][]Source // keyed by archive folder
}

func (f *found) add(src Source, active, archived []string) {
	for _, s := range active {
		if s != "archive" {
			f.active[s] = append(f.active[s], src)
		}
	}
	for _, a := range archived {
		if archiveName.MatchString(a) {
			f.archived[a] = append(f.archived[a], src)
		}
	}
}

// Scan reads every worktree, every unmerged local and remote-tracking branch
// and the default branch tip. It contacts no remote. Only failures not tied to
// one source are returned; a source that cannot be read goes into Unreadable.
func Scan(g gitx.Git, db DefaultBranch) (Inventory, error) {
	inv := Inventory{DefaultBranch: db, Changes: []Change{}, Archived: []Archived{}, Unreadable: []Unreadable{}}
	f := &found{active: map[string][]Source{}, archived: map[string][]Source{}}

	wts, err := g.Worktrees()
	if err != nil {
		return inv, err
	}
	for _, w := range wts {
		if w.Bare {
			continue
		}
		src := Source{Kind: KindWorktree, Name: w.Path, Commit: w.Head}
		if w.Prunable {
			inv.Unreadable = append(inv.Unreadable, Unreadable{src, "worktree is prunable: its directory is gone"})
			continue
		}
		active, archived, err := readDisk(w.Path)
		if err != nil {
			inv.Unreadable = append(inv.Unreadable, Unreadable{src, err.Error()})
			continue
		}
		f.add(src, active, archived)
	}

	refs, err := g.Refs("refs/heads", "refs/remotes")
	if err != nil {
		return inv, err
	}
	for _, r := range refs {
		if r.Name == db.Ref {
			continue
		}
		kind := KindLocalBranch
		if strings.HasPrefix(r.Name, "refs/remotes/") {
			kind = KindRemoteBranch
		}
		src := Source{Kind: kind, Name: r.Name, Commit: r.SHA}
		merged, err := g.IsAncestor(r.SHA, db.Commit)
		if err != nil {
			inv.Unreadable = append(inv.Unreadable, Unreadable{src, err.Error()})
			continue
		}
		if merged {
			continue
		}
		active, archived, err := readTree(g, r.SHA)
		if err != nil {
			inv.Unreadable = append(inv.Unreadable, Unreadable{src, err.Error()})
			continue
		}
		f.add(src, active, archived)
	}

	dbSrc := Source{Kind: KindDefaultBranch, Name: db.Ref, Commit: db.Commit}
	active, archived, err := readTree(g, db.Commit)
	if err != nil {
		inv.Unreadable = append(inv.Unreadable, Unreadable{dbSrc, err.Error()})
	} else {
		f.add(dbSrc, active, archived)
	}

	for slug, srcs := range f.active {
		inv.Changes = append(inv.Changes, Change{Slug: slug, Sources: sortSources(srcs)})
	}
	sort.Slice(inv.Changes, func(i, j int) bool { return inv.Changes[i].Slug < inv.Changes[j].Slug })
	for folder, srcs := range f.archived {
		slug := archiveName.FindStringSubmatch(folder)[1]
		inv.Archived = append(inv.Archived, Archived{Slug: slug, Folder: folder, Sources: sortSources(srcs)})
	}
	sort.Slice(inv.Archived, func(i, j int) bool {
		a, b := inv.Archived[i], inv.Archived[j]
		if a.Slug != b.Slug {
			return a.Slug < b.Slug
		}
		return a.Folder < b.Folder
	})
	return inv, nil
}

func sortSources(s []Source) []Source {
	sort.Slice(s, func(i, j int) bool {
		if kindRank[s[i].Kind] != kindRank[s[j].Kind] {
			return kindRank[s[i].Kind] < kindRank[s[j].Kind]
		}
		return s[i].Name < s[j].Name
	})
	return s
}

func readDisk(root string) (active, archived []string, err error) {
	if _, err := os.Stat(root); err != nil {
		return nil, nil, err
	}
	if active, err = diskDirs(filepath.Join(root, changesDir)); err != nil {
		return nil, nil, err
	}
	if archived, err = diskDirs(filepath.Join(root, archiveDir)); err != nil {
		return nil, nil, err
	}
	return active, archived, nil
}

// diskDirs lists subdirectory names; a missing directory is empty.
func diskDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func readTree(g gitx.Git, rev string) (active, archived []string, err error) {
	if active, err = g.ListDirs(rev, changesDir); err != nil {
		return nil, nil, err
	}
	if archived, err = g.ListDirs(rev, archiveDir); err != nil {
		return nil, nil, err
	}
	return active, archived, nil
}

// Uses reports how a slug is used: the active change, if any, and every
// archived use.
func (inv Inventory) Uses(slug string) (active *Change, archived []Archived) {
	for i := range inv.Changes {
		if inv.Changes[i].Slug == slug {
			active = &inv.Changes[i]
		}
	}
	for _, a := range inv.Archived {
		if a.Slug == slug {
			archived = append(archived, a)
		}
	}
	return active, archived
}

// ReadFile reads rel (relative to the repository root) from a source.
func ReadFile(g gitx.Git, src Source, rel string) ([]byte, error) {
	if src.Kind == KindWorktree {
		return os.ReadFile(filepath.Join(src.Name, rel))
	}
	return g.Show(src.Commit, rel)
}
