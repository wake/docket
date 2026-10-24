package scan

import (
	"errors"
	"fmt"
	"io"
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

// Content precedence, lowest first. Sources[0] of a Change supplies content.
// Kind alone does not decide it: a worktree ranks by whether the default
// branch tip contains its HEAD.
const (
	rankUnmergedWorktree = iota // worktree with commits the default branch lacks
	rankLocalBranch
	rankRemoteBranch
	rankContainedWorktree // worktree whose HEAD the default tip contains, or unborn
	rankDefaultBranch
)

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
	Path   string // empty when the whole source is unreadable, else the one file or folder that was not
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
	rank     map[Source]int
}

func (f *found) add(src Source, rank int, active, archived []string) {
	f.rank[src] = rank
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
	f := &found{active: map[string][]Source{}, archived: map[string][]Source{}, rank: map[Source]int{}}

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
			inv.Unreadable = append(inv.Unreadable, Unreadable{Source: src, Reason: "worktree is prunable: its directory is gone"})
			continue
		}
		active, archived, err := readDisk(w.Path)
		if err != nil {
			inv.Unreadable = append(inv.Unreadable, Unreadable{Source: src, Reason: err.Error()})
			continue
		}
		rank := rankContainedWorktree
		if !unbornHead(w.Head) {
			contained, err := g.IsAncestor(w.Head, db.Commit)
			if err != nil {
				inv.Unreadable = append(inv.Unreadable, Unreadable{Source: src, Reason: err.Error()})
				continue
			}
			if !contained {
				rank = rankUnmergedWorktree
			}
		}
		f.add(src, rank, active, archived)
	}

	refs, err := g.Refs("refs/heads", "refs/remotes")
	if err != nil {
		return inv, err
	}
	for _, r := range refs {
		if r.Name == db.Ref {
			continue
		}
		kind, rank := KindLocalBranch, rankLocalBranch
		if strings.HasPrefix(r.Name, "refs/remotes/") {
			kind, rank = KindRemoteBranch, rankRemoteBranch
		}
		src := Source{Kind: kind, Name: r.Name, Commit: r.SHA}
		merged, err := g.IsAncestor(r.SHA, db.Commit)
		if err != nil {
			inv.Unreadable = append(inv.Unreadable, Unreadable{Source: src, Reason: err.Error()})
			continue
		}
		if merged {
			continue
		}
		active, archived, err := readTree(g, r.SHA)
		if err != nil {
			inv.Unreadable = append(inv.Unreadable, Unreadable{Source: src, Reason: err.Error()})
			continue
		}
		f.add(src, rank, active, archived)
	}

	dbSrc := Source{Kind: KindDefaultBranch, Name: db.Ref, Commit: db.Commit}
	active, archived, err := readTree(g, db.Commit)
	if err != nil {
		inv.Unreadable = append(inv.Unreadable, Unreadable{Source: dbSrc, Reason: err.Error()})
	} else {
		f.add(dbSrc, rankDefaultBranch, active, archived)
	}

	for slug, srcs := range f.active {
		inv.Changes = append(inv.Changes, Change{Slug: slug, Sources: f.sortSources(srcs)})
	}
	sort.Slice(inv.Changes, func(i, j int) bool { return inv.Changes[i].Slug < inv.Changes[j].Slug })
	for folder, srcs := range f.archived {
		slug := archiveName.FindStringSubmatch(folder)[1]
		inv.Archived = append(inv.Archived, Archived{Slug: slug, Folder: folder, Sources: f.sortSources(srcs)})
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

// unbornHead reports a worktree whose HEAD has no commit yet.
func unbornHead(head string) bool {
	return strings.Trim(head, "0") == ""
}

func (f *found) sortSources(s []Source) []Source {
	sort.Slice(s, func(i, j int) bool {
		if f.rank[s[i]] != f.rank[s[j]] {
			return f.rank[s[i]] < f.rank[s[j]]
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

// ReadFile reads rel (relative to the repository root) from a source. rel must
// be a local path; a worktree file is read through os.Root, so neither ".." nor
// a symlink can lead outside the worktree.
func ReadFile(g gitx.Git, src Source, rel string) ([]byte, error) {
	if !filepath.IsLocal(rel) {
		return nil, fmt.Errorf("path %q is not a local path inside the repository", rel)
	}
	if src.Kind == KindWorktree {
		root, err := os.OpenRoot(src.Name)
		if err != nil {
			return nil, err
		}
		defer root.Close()
		f, err := root.Open(rel)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(f)
	}
	return g.Show(src.Commit, rel)
}

// ReadFiles reads every regular file below dir (repository-relative, slash
// separated) from a source for which keep, given the path relative to dir,
// returns true; the result is keyed by that relative path. Files keep rejects
// are neither read nor reported. A worktree is walked through os.Root, so
// uncommitted and untracked files are read and links are neither read nor
// followed; any other source is listed and read from its commit. A missing dir
// gives an empty result; a dir that exists but is not a folder (a file or a
// link) is returned as Unreadable. A kept file or a folder that cannot be read is
// returned as Unreadable with its Path, and the rest is still read.
func ReadFiles(g gitx.Git, src Source, dir string, keep func(rel string) bool) (map[string][]byte, []Unreadable, error) {
	if !filepath.IsLocal(dir) {
		return nil, nil, fmt.Errorf("path %q is not a local path inside the repository", dir)
	}
	dir = filepath.ToSlash(filepath.Clean(dir))
	files := map[string][]byte{}
	var unread []Unreadable
	fail := func(path string, err error) {
		unread = append(unread, Unreadable{Source: src, Path: path, Reason: err.Error()})
	}
	if src.Kind != KindWorktree {
		paths, err := g.ListFiles(src.Commit, dir)
		if err != nil {
			fail(dir, err)
			return files, unread, nil
		}
		for _, p := range paths {
			rel := strings.TrimPrefix(p, dir+"/")
			if !keep(rel) {
				continue
			}
			b, err := g.Show(src.Commit, p)
			if err != nil {
				fail(p, err)
				continue
			}
			files[rel] = b
		}
		return files, unread, nil
	}
	root, err := os.OpenRoot(src.Name)
	if err != nil {
		fail("", err)
		return files, unread, nil
	}
	defer root.Close()
	if fi, err := root.Lstat(dir); err != nil || !fi.IsDir() {
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			fail(dir, err)
		} else if err == nil {
			fail(dir, errors.New("not a folder"))
		}
		return files, unread, nil
	}
	fsys := root.FS()
	walkErr := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			fail(p, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // folders are walked into; links and the rest are skipped
		}
		rel := strings.TrimPrefix(p, dir+"/")
		if !keep(rel) {
			return nil
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			fail(p, err)
			return nil
		}
		files[rel] = b
		return nil
	})
	if walkErr != nil {
		fail(dir, walkErr)
	}
	return files, unread, nil
}
