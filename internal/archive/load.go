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

// loadOpts says what loadChange reads beyond the metadata: Plan reads
// plan.md, BaseIsCommit requires x-base to name a commit, Deltas reads the
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

	b, err := fs.ReadFile(rfs, c.path(".openspec.yaml"))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		add("metadata-invalid", ".openspec.yaml", ".openspec.yaml is missing")
	case err != nil:
		unread = append(unread, Unreadable{Source: c.path(".openspec.yaml"), Reason: err.Error()})
	default:
		m, perr := change.ParseMetadata(b)
		switch {
		case perr != nil:
			add("metadata-invalid", ".openspec.yaml", ".openspec.yaml is not valid YAML: %v", perr)
		case m.Base == "":
			add("metadata-invalid", ".openspec.yaml", ".openspec.yaml has no x-base")
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
					add("base-missing", ".openspec.yaml", "x-base %q is not a commit in this repository", m.Base)
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
		if !d.Type().IsRegular() || path.Base(p) != "spec.md" {
			return nil
		}
		b, err := fs.ReadFile(rfs, p)
		if err != nil {
			unread = append(unread, Unreadable{Source: p, Reason: err.Error()})
			return nil
		}
		capPath := strings.TrimSuffix(strings.TrimPrefix(p, specs+"/"), "/spec.md")
		c.Deltas[capPath] = delta.Parse(b)
		return nil
	})
	return unread
}
