package domain

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/markdown"
)

// Capability is a main spec, a delta spec of an active change, or both.
type Capability struct {
	Path, Title string
	Main        bool
	Changes     []string
}

// Unreadable names a source that could not be read.
type Unreadable struct{ Source, Reason string }

// Loaded is the tree and the capabilities it classifies. Stopped says the
// tree file could not be used; Nodes and Capabilities are then empty and
// nothing else was read.
type Loaded struct {
	Found, Stopped bool
	Nodes          []*Node
	Capabilities   map[string]*Capability
	Problems       []Problem
	Unreadable     []Unreadable
}

const (
	specsDir   = "openspec/specs"
	changesDir = "openspec/changes"
)

// Load reads the tree file and the capabilities of the project in root.
func Load(root fs.FS) Loaded {
	l := Loaded{Capabilities: map[string]*Capability{}}
	data, found, err := readTree(root)
	l.Found = found
	if err != nil {
		l.Stopped = true
		l.Unreadable = []Unreadable{{File, err.Error()}}
		return l
	}
	if found {
		nodes, probs := Parse(data)
		if len(probs) > 0 {
			l.Stopped, l.Problems = true, probs
			return l
		}
		l.Nodes = nodes
	}

	unreadable := func(src string, err error) {
		l.Unreadable = append(l.Unreadable, Unreadable{src, err.Error()})
	}
	get := func(p string) *Capability {
		if l.Capabilities[p] == nil {
			l.Capabilities[p] = &Capability{Path: p}
		}
		return l.Capabilities[p]
	}

	walkSpecs(root, specsDir, unreadable, func(p, cp string) {
		c := get(cp)
		c.Main = true
		b, err := fs.ReadFile(root, p)
		if err != nil {
			unreadable(p, err)
			return
		}
		if lines := markdown.Lines(b); len(lines) > 0 {
			c.Title, _ = change.SpecTitle(lines[0], cp)
		}
	})

	entries, err := fs.ReadDir(root, changesDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		unreadable(changesDir, err)
	}
	names := map[string]map[string]string{} // slug -> name under specs:
	for _, e := range entries {
		slug := e.Name()
		if !e.IsDir() || slug == "archive" {
			continue
		}
		dir := path.Join(changesDir, slug)
		walkSpecs(root, dir+"/specs", unreadable, func(_, cp string) {
			c := get(cp)
			c.Changes = append(c.Changes, slug)
		})
		files, err := fs.ReadDir(root, dir)
		if err != nil {
			unreadable(dir, err)
			continue
		}
		// Found through the listing, so a linked proposal is skipped, never read through.
		if !slices.ContainsFunc(files, func(f fs.DirEntry) bool { return f.Name() == "proposal.md" && f.Type().IsRegular() }) {
			continue
		}
		prop := dir + "/proposal.md"
		b, err := fs.ReadFile(root, prop)
		if err != nil {
			unreadable(prop, err)
			continue
		}
		// A proposal whose front matter cannot be parsed gives no name; lint reports it.
		if fm, _, err := change.ReadFrontMatter(b); err == nil {
			names[slug] = fm.Specs
		}
	}
	for cp, c := range l.Capabilities {
		sort.Strings(c.Changes)
		for _, slug := range c.Changes {
			if c.Title != "" {
				break
			}
			c.Title = strings.TrimSpace(names[slug][cp])
		}
	}

	if len(l.Unreadable) == 0 {
		var visit func([]*Node)
		visit = func(ns []*Node) {
			for _, n := range ns {
				for _, e := range n.Capabilities {
					if l.Capabilities[e.Path] == nil {
						l.Problems = append(l.Problems, Problem{RuleUnknown, e.Line,
							fmt.Sprintf("%s lists %s, which has no main spec and no delta in an active change", File, e.Path)})
					}
				}
				visit(n.Children)
			}
		}
		visit(l.Nodes)
	}
	return l
}

// readTree finds docket/domains.md through the directory listing, so that a
// link is never read through.
func readTree(root fs.FS) ([]byte, bool, error) {
	if info, err := fs.Stat(root, "docket"); err != nil || !info.IsDir() {
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, false, err
		}
		return nil, false, nil
	}
	entries, err := fs.ReadDir(root, "docket")
	if err != nil {
		return nil, false, err
	}
	for _, e := range entries {
		if e.Name() != "domains.md" {
			continue
		}
		switch t := e.Type(); {
		case t&fs.ModeSymlink != 0:
			return nil, true, fmt.Errorf("%s is a symbolic link; Docket does not read through it", File)
		case !t.IsRegular():
			return nil, true, fmt.Errorf("%s is not a regular file", File)
		}
		b, err := fs.ReadFile(root, File)
		return b, true, err
	}
	return nil, false, nil
}

// walkSpecs calls fn for each regular spec.md under dir with its capability path.
func walkSpecs(root fs.FS, dir string, unreadable func(string, error), fn func(file, cp string)) {
	_ = fs.WalkDir(root, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				unreadable(p, err)
			}
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || d.Name() != "spec.md" {
			return nil
		}
		if cp := strings.Trim(strings.TrimSuffix(strings.TrimPrefix(p, dir), "spec.md"), "/"); cp != "" {
			fn(p, cp)
		}
		return nil
	})
}
