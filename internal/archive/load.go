package archive

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/delta"
)

// loadedChange is a change folder read once: its metadata, its plan when
// asked for, and its delta specs by capability path.
type loadedChange struct {
	Slug      string
	Dir       string // repository-relative, slash-separated
	Meta      change.Metadata
	Plan      change.Plan
	PlanBytes []byte
	Deltas    map[string]delta.Delta
}

// baseKey is the key that holds the base in the file m was read from.
func baseKey(m change.Metadata) string {
	if m.File == change.OpenSpecFile {
		return "x-base"
	}
	return "base"
}

func (c *loadedChange) path(name string) string { return path.Join(c.Dir, name) }

// headOrNoCommits returns the commit HEAD names. When the repository has no
// commits it appends a no-commits finding for slug to finds and reports done.
func headOrNoCommits(e Env, slug string, finds *[]Finding) (head string, done bool, err error) {
	head, ok, err := e.Git.Head()
	if err != nil {
		return "", false, err
	}
	if !ok {
		*finds = append(*finds, Finding{Rule: "no-commits", Change: slug,
			Message: "the repository has no commits; commit first"})
		return "", true, nil
	}
	return head, false, nil
}

// loadAtHead returns the commit HEAD names and the change loaded as o asks.
// A nil change comes with the findings and unreadable sources to report,
// which include a no-commits finding in a repository without commits.
func loadAtHead(e Env, slug string, o loadOpts) (head string, c *loadedChange, finds []Finding, unread []Unreadable, err error) {
	head, done, err := headOrNoCommits(e, slug, &finds)
	if err != nil || done {
		return "", nil, finds, nil, err
	}
	c, finds, unread = loadChange(e, slug, o)
	return head, c, finds, unread, nil
}

// loadOpts says what loadChange reads beyond the metadata: Plan reads
// plan.md, BaseIsCommit requires the base to name a commit, Deltas reads the
// delta specs.
type loadOpts struct{ Plan, BaseIsCommit, Deltas bool }

// loadChange reads openspec/changes/<slug>/ as o asks. It returns nil
// together with the findings and unreadable sources when the change cannot
// be used.
func loadChange(e Env, slug string, o loadOpts) (*loadedChange, []Finding, []Unreadable) {
	var finds []Finding
	var unread []Unreadable
	dir := "openspec/changes/" + slug
	add := func(rule, file string, format string, args ...any) {
		f := Finding{Rule: rule, Change: slug, Message: fmt.Sprintf(format, args...)}
		if file != "" {
			f.File = path.Join(dir, file)
		}
		finds = append(finds, f)
	}
	missing := func() (*loadedChange, []Finding, []Unreadable) {
		add("change-missing", "", "change %q does not exist under openspec/changes", slug)
		return nil, finds, unread
	}
	if slug == "" || slug == "archive" || slug == "." || !fs.ValidPath(slug) || strings.Contains(slug, "/") {
		return missing()
	}
	root, err := os.OpenRoot(e.Root)
	if err != nil {
		return nil, nil, []Unreadable{{Source: "repository", Reason: err.Error()}}
	}
	defer root.Close()

	fi, err := root.Lstat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return missing()
	case err != nil:
		return nil, nil, []Unreadable{{Source: dir, Reason: err.Error()}}
	case !fi.IsDir():
		return missing()
	}
	rfs := root.FS()
	c := &loadedChange{Slug: slug, Dir: dir, Deltas: map[string]delta.Delta{}}

	files := map[string][]byte{}
	for _, name := range []string{change.DocketFile, change.OpenSpecFile} {
		b, rerr := fs.ReadFile(rfs, c.path(name))
		switch {
		case errors.Is(rerr, fs.ErrNotExist):
		case rerr != nil:
			unread = append(unread, Unreadable{Source: c.path(name), Reason: rerr.Error()})
		default:
			files[name] = b
		}
	}
	if len(unread) == 0 {
		m, merr := change.ReadMetadata(files)
		switch {
		case merr != nil:
			add("metadata-invalid", m.File, "%v", merr)
		case m.Base == "":
			add("metadata-invalid", m.File, "%s has no %s", m.File, baseKey(m))
		default:
			c.Meta = m
			if o.BaseIsCommit {
				ok := !strings.HasPrefix(m.Base, "-")
				if ok {
					ok, err = e.Git.RefExists(m.Base + "^{commit}")
					if err != nil {
						return nil, nil, []Unreadable{{Source: "git", Reason: err.Error()}}
					}
				}
				if !ok {
					add("base-missing", m.File, "the base %q in %s is not a commit in this repository", m.Base, m.File)
				}
			}
		}
	}

	if o.Plan {
		b, err := fs.ReadFile(rfs, c.path("plan.md"))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			add("plan-missing", "plan.md", "plan.md is missing")
		case err != nil:
			unread = append(unread, Unreadable{Source: c.path("plan.md"), Reason: err.Error()})
		default:
			c.PlanBytes = b
			c.Plan = change.ParsePlan(b)
		}
	}

	specs := c.path("specs")
	if o.Deltas {
		unread = append(unread, loadDeltas(rfs, specs, c)...)
	}

	if len(finds) > 0 || len(unread) > 0 {
		sort.SliceStable(unread, func(i, j int) bool { return unread[i].Source < unread[j].Source })
		return nil, finds, unread
	}
	return c, nil, nil
}

// loadDeltas reads every specs/**/spec.md below specs into c.Deltas.
// It returns the sources it could not read.
func loadDeltas(rfs fs.FS, specs string, c *loadedChange) (unread []Unreadable) {
	_ = fs.WalkDir(rfs, specs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				unread = append(unread, Unreadable{Source: p, Reason: err.Error()})
			}
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		capPath, ok := delta.CapabilityPath("specs/" + strings.TrimPrefix(p, specs+"/"))
		if !d.Type().IsRegular() || !ok {
			return nil
		}
		b, err := fs.ReadFile(rfs, p)
		if err != nil {
			unread = append(unread, Unreadable{Source: p, Reason: err.Error()})
			return nil
		}
		c.Deltas[capPath] = delta.Parse(b)
		return nil
	})
	return unread
}
