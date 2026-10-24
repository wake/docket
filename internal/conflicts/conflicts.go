// Package conflicts finds requirements that more than one active change
// touches.
package conflicts

import (
	"slices"
	"sort"

	"github.com/wake/docket/internal/delta"
	"github.com/wake/docket/internal/scan"
)

// Operations a change performs on a requirement, in report order.
const (
	Added       = "ADDED"
	Modified    = "MODIFIED"
	Removed     = "REMOVED"
	RenamedFrom = "RENAMED-FROM"
	RenamedTo   = "RENAMED-TO"
)

var operationOrder = []string{Added, Modified, Removed, RenamedFrom, RenamedTo}

// Change is one active change with its parsed delta specs, keyed by
// capability path.
type Change struct {
	Slug    string
	Sources []scan.Source
	Deltas  map[string]delta.Delta
}

// Entry is what one change does to an overlapping requirement.
type Entry struct {
	Slug       string        `json:"slug"`
	Operations []string      `json:"operations"`
	Sources    []scan.Source `json:"sources"`
}

// Overlap is a requirement that two or more changes touch.
type Overlap struct {
	Capability  string  `json:"capability"`
	Requirement string  `json:"requirement"`
	Changes     []Entry `json:"changes"`
}

type key struct{ capability, requirement string }

// Find returns every (capability, requirement) touched by two or more
// distinct slugs, sorted by capability, requirement and slug. The result is
// never nil.
func Find(changes []Change) []Overlap {
	touched := map[key]map[string]map[string]bool{}
	sources := map[string][]scan.Source{}
	for _, c := range changes {
		if _, seen := sources[c.Slug]; !seen {
			sources[c.Slug] = append([]scan.Source{}, c.Sources...)
		}
		for capability, d := range c.Deltas {
			touch := func(name, op string) {
				k := key{capability, name}
				if touched[k] == nil {
					touched[k] = map[string]map[string]bool{}
				}
				if touched[k][c.Slug] == nil {
					touched[k][c.Slug] = map[string]bool{}
				}
				touched[k][c.Slug][op] = true
			}
			for _, r := range d.Added {
				touch(r.Name, Added)
			}
			for _, r := range d.Modified {
				touch(r.Name, Modified)
			}
			for _, r := range d.Removed {
				touch(r.Name, Removed)
			}
			for _, r := range d.Renamed {
				touch(r.From, RenamedFrom)
				touch(r.To, RenamedTo)
			}
		}
	}

	out := []Overlap{}
	for k, bySlug := range touched {
		if len(bySlug) < 2 {
			continue
		}
		o := Overlap{Capability: k.capability, Requirement: k.requirement, Changes: []Entry{}}
		for slug, set := range bySlug {
			e := Entry{Slug: slug, Operations: []string{}, Sources: slices.Clone(sources[slug])}
			for _, op := range operationOrder {
				if set[op] {
					e.Operations = append(e.Operations, op)
				}
			}
			o.Changes = append(o.Changes, e)
		}
		sort.Slice(o.Changes, func(i, j int) bool { return o.Changes[i].Slug < o.Changes[j].Slug })
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Capability != out[j].Capability {
			return out[i].Capability < out[j].Capability
		}
		return out[i].Requirement < out[j].Requirement
	})
	return out
}

// Involving returns the overlaps that have an entry for slug, in order. The
// result is never nil.
func Involving(overlaps []Overlap, slug string) []Overlap {
	out := []Overlap{}
	for _, o := range overlaps {
		for _, e := range o.Changes {
			if e.Slug == slug {
				out = append(out, o)
				break
			}
		}
	}
	return out
}
