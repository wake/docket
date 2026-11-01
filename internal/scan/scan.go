package scan

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/record"
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

// Record is a record, reported once however many sources hold it.
type Record struct {
	Slug    string
	Sources []Source // Sources[0] is ranked as for changes
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
	Records       []Record
	Unreadable    []Unreadable
}

const (
	changesDir = "openspec/changes"
	archiveDir = changesDir + "/archive"
)

var archiveName = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-(.+)$`)

type found struct {
	active   map[string][]Source
	archived map[string][]Source            // keyed by archive folder
	dated    map[string]map[string][]Source // dated folder, then slug
	rank     map[Source]int
}

// datedDirs are the folders of dated files, each file named <date>-<name>.md.
// Their parent is docket/, which the batched tree walk relies on.
var datedDirs = []string{record.Dir}

func (f *found) add(src Source, rank int, active, archived []string, dated map[string][]string) {
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
	for dir, slugs := range dated {
		for _, s := range slugs {
			f.dated[dir][s] = append(f.dated[dir][s], src)
		}
	}
}

// Scan reads every worktree, every unmerged local and remote-tracking branch
// and the default branch tip. It contacts no remote. Only failures not tied to
// one source are returned; a source that cannot be read goes into Unreadable.
func Scan(g gitx.Git, db DefaultBranch) (Inventory, error) {
	inv := Inventory{DefaultBranch: db, Changes: []Change{}, Archived: []Archived{}, Records: []Record{}, Unreadable: []Unreadable{}}
	f := &found{active: map[string][]Source{}, archived: map[string][]Source{}, dated: map[string]map[string][]Source{}, rank: map[Source]int{}}
	for _, d := range datedDirs {
		f.dated[d] = map[string][]Source{}
	}

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
		r := readDisk(w.Path)
		err := r.err
		rank := rankContainedWorktree
		// Settle ancestry before noteRead, so a worktree unreadable as a whole
		// is listed once, not also for its records folder.
		if err == nil && !unbornHead(w.Head) {
			var contained bool
			if contained, err = g.IsAncestor(w.Head, db.Commit); err == nil && !contained {
				rank = rankUnmergedWorktree
			}
		}
		if !inv.noteRead(src, r.datedErr, err) {
			continue
		}
		f.add(src, rank, r.active, r.archived, r.dated)
	}

	unmerged, err := unmergedRefs(g, db)
	if err != nil {
		return inv, err
	}
	// One batch per level of the trees reads every commit: the unmerged refs
	// and the default tip.
	commits := []string{db.Commit}
	for _, u := range unmerged {
		if u.Err == nil {
			commits = append(commits, u.Ref.SHA)
		}
	}
	trees := readTrees(g, commits)
	for _, u := range unmerged {
		r := u.Ref
		kind, rank := KindLocalBranch, rankLocalBranch
		if strings.HasPrefix(r.Name, "refs/remotes/") {
			kind, rank = KindRemoteBranch, rankRemoteBranch
		}
		src := Source{Kind: kind, Name: r.Name, Commit: r.SHA}
		if u.Err != nil {
			inv.Unreadable = append(inv.Unreadable, Unreadable{Source: src, Reason: u.Err.Error()})
			continue
		}
		t := trees[r.SHA]
		if !inv.noteRead(src, t.datedErr, t.err) {
			continue
		}
		f.add(src, rank, t.active, t.archived, t.dated)
	}

	dbSrc := Source{Kind: KindDefaultBranch, Name: db.Ref, Commit: db.Commit}
	if t := trees[db.Commit]; inv.noteRead(dbSrc, t.datedErr, t.err) {
		f.add(dbSrc, rankDefaultBranch, t.active, t.archived, t.dated)
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
	for slug, srcs := range f.dated[record.Dir] {
		inv.Records = append(inv.Records, Record{Slug: slug, Sources: f.sortSources(srcs)})
	}
	sort.Slice(inv.Records, func(i, j int) bool { return inv.Records[i].Slug < inv.Records[j].Slug })
	return inv, nil
}

// noteRead files what reading a source failed on: err makes the whole source
// unreadable, datedErr only the dated folder it is keyed by. It reports whether the
// source's changes were read.
func (inv *Inventory) noteRead(src Source, datedErr map[string]error, err error) bool {
	if err != nil {
		inv.Unreadable = append(inv.Unreadable, Unreadable{Source: src, Reason: err.Error()})
		return false
	}
	for _, d := range datedDirs {
		if e := datedErr[d]; e != nil {
			inv.Unreadable = append(inv.Unreadable, Unreadable{Source: src, Path: d, Reason: e.Error()})
		}
	}
	return true
}

// unmergedRef is a ref to read, or, when Err is set, one whose ancestry
// could not be computed.
type unmergedRef struct {
	Ref gitx.Ref
	Err error
}

// unmergedRefs lists, in ref order, the local and remote-tracking refs other
// than the default branch's own that the default branch tip does not contain,
// and the ones whose ancestry cannot be computed. Two filtered listings
// settle most refs in two git processes. A ref that is in both or in neither
// (git leaves out a ref whose tip it cannot read) is asked on its own, and
// when either listing fails, or the history behind the unmerged tips cannot
// be walked, every ref is asked on its own.
func unmergedRefs(g gitx.Git, db DefaultBranch) (unmerged []unmergedRef, err error) {
	patterns := []string{"refs/heads", "refs/remotes"}
	all, err := g.Refs(patterns...)
	if err != nil {
		return nil, err
	}
	inMerged, inUnmerged := map[string]bool{}, map[string]bool{}
	merged, errM := g.RefsFiltered("--merged="+db.Commit, patterns...)
	no, errU := g.RefsFiltered("--no-merged="+db.Commit, patterns...)
	listed := errM == nil && errU == nil
	if listed {
		for _, r := range merged {
			inMerged[r.Name] = true
		}
		var tips []string
		for _, r := range no {
			inUnmerged[r.Name] = true
			if r.Name != db.Ref {
				tips = append(tips, r.SHA)
			}
		}
		// The listings do not notice damaged history behind a tip.
		if len(tips) > 0 && g.WalkUnmerged(tips, db.Commit) != nil {
			listed = false
		}
	}
	for _, r := range all {
		if r.Name == db.Ref {
			continue
		}
		if listed && inMerged[r.Name] != inUnmerged[r.Name] {
			if inUnmerged[r.Name] {
				unmerged = append(unmerged, unmergedRef{Ref: r})
			}
			continue
		}
		isMerged, err := g.IsAncestor(r.SHA, db.Commit)
		if err != nil || !isMerged {
			unmerged = append(unmerged, unmergedRef{Ref: r, Err: err})
		}
	}
	return unmerged, nil
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

// readDisk reads a worktree. datedErr holds a failure to list a dated
// folder alone; the changes are still returned.
func readDisk(root string) (r treeRead) {
	if _, err := os.Stat(root); err != nil {
		return treeRead{err: err}
	}
	var err error
	if r.active, err = diskDirs(filepath.Join(root, changesDir)); err != nil {
		return treeRead{err: err}
	}
	if r.archived, err = diskDirs(filepath.Join(root, archiveDir)); err != nil {
		return treeRead{err: err}
	}
	r.dated, r.datedErr = map[string][]string{}, map[string]error{}
	for _, dir := range datedDirs {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			r.datedErr[dir] = err
		}
		for _, e := range entries {
			if slug, ok := record.SlugOf(e.Name()); ok && e.Type().IsRegular() {
				r.dated[dir] = append(r.dated[dir], slug)
			}
		}
	}
	return r
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

// readTree reads a commit with one ls-tree. datedErr holds a failure to list
// a dated folder alone.
func readTree(g gitx.Git, rev string) treeRead {
	args := []string{changesDir + "/", archiveDir + "/"}
	for _, d := range datedDirs {
		args = append(args, d, d+"/")
	}
	entries, err := g.ListTree(rev, args...)
	if err != nil {
		// One unreadable folder may be a dated one alone: ask for each in turn.
		return readTreeEach(g, rev)
	}
	r := treeRead{dated: map[string][]string{}, datedErr: map[string]error{}}
	for _, e := range entries {
		dir, name := path.Split(e.Path)
		switch {
		case slices.Contains(datedDirs, e.Path):
			r.datedErr[e.Path] = fmt.Errorf("%s is not a folder at %s", e.Path, rev)
		case dir == changesDir+"/" && e.Type == "tree":
			r.active = append(r.active, name)
		case dir == archiveDir+"/" && e.Type == "tree":
			r.archived = append(r.archived, name)
		case slices.Contains(datedDirs, strings.TrimSuffix(dir, "/")) && e.Type == "blob" && (e.Mode == "100644" || e.Mode == "100755"):
			if slug, ok := record.SlugOf(name); ok {
				d := strings.TrimSuffix(dir, "/")
				r.dated[d] = append(r.dated[d], slug)
			}
		}
	}
	return r
}

// readTreeEach is readTree with one git call per folder, which tells a
// failure of a dated folder from a failure of the changes.
func readTreeEach(g gitx.Git, rev string) (r treeRead) {
	var err error
	if r.active, err = g.ListDirs(rev, changesDir); err != nil {
		return treeRead{err: err}
	}
	if r.archived, err = g.ListDirs(rev, archiveDir); err != nil {
		return treeRead{err: err}
	}
	r.dated, r.datedErr = map[string][]string{}, map[string]error{}
	for _, dir := range datedDirs {
		paths, err := g.ListFiles(rev, dir)
		if err != nil {
			r.datedErr[dir] = err
		}
		for _, p := range paths {
			name := strings.TrimPrefix(p, dir+"/")
			if slug, ok := record.SlugOf(name); ok && !strings.Contains(name, "/") {
				r.dated[dir] = append(r.dated[dir], slug)
			}
		}
	}
	return r
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

// RecordSources returns the sources holding the record, nil when none.
func (inv Inventory) RecordSources(slug string) []Source {
	for _, r := range inv.Records {
		if r.Slug == slug {
			return r.Sources
		}
	}
	return nil
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
		var kept []string
		for _, p := range paths {
			if keep(strings.TrimPrefix(p, dir+"/")) {
				kept = append(kept, p)
			}
		}
		// One batch reads the files; Show reads again what the batch did not
		// return (and everything when the batch fails), so a file git cannot
		// read gets Show's error.
		blobs, missed, err := g.ShowAll(src.Commit, kept)
		if err != nil {
			blobs, missed = nil, kept
		}
		for _, p := range kept {
			b, ok := blobs[p]
			if !ok {
				continue
			}
			files[strings.TrimPrefix(p, dir+"/")] = b
		}
		for _, p := range missed {
			b, err := g.Show(src.Commit, p)
			if err != nil {
				fail(p, err)
				continue
			}
			files[strings.TrimPrefix(p, dir+"/")] = b
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
