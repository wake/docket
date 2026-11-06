package conflicts

import (
	"slices"
	"strings"
)

// Entry is one change's part in an overlap.
type Entry struct {
	Slug       string   `json:"slug"`
	Operations []string `json:"operations"`
	Sources    []string `json:"sources"`
}

// Overlap is a requirement that two or more changes touch.
type Overlap struct {
	Capability  string  `json:"capability"`
	Requirement string  `json:"requirement"`
	Changes     []Entry `json:"changes"`
}

var operationOrder = []string{"ADDED", "MODIFIED", "REMOVED", "RENAMED-FROM", "RENAMED-TO"}

func fold(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }

// Find returns the requirements that two or more distinct changes touch,
// sorted by capability and folded requirement name, entries by slug.
func Find(parsed []Parsed) []Overlap {
	type key struct{ capability, name string }
	type part struct {
		spelling string
		ops      map[string]bool
		sources  []string
	}
	groups := map[key]map[string]*part{}
	var keys []key
	for _, pd := range parsed {
		for _, t := range pd.Touches {
			k := key{t.Capability, fold(t.Requirement)}
			if groups[k] == nil {
				groups[k] = map[string]*part{}
				keys = append(keys, k)
			}
			e := groups[k][pd.Copy.Slug]
			if e == nil {
				e = &part{spelling: t.Requirement, ops: map[string]bool{}}
				groups[k][pd.Copy.Slug] = e
			}
			e.ops[t.Operation] = true
			for _, s := range pd.Copy.Sources {
				if !slices.Contains(e.sources, s) {
					e.sources = append(e.sources, s)
				}
			}
		}
	}
	slices.SortFunc(keys, func(a, b key) int {
		if c := strings.Compare(a.capability, b.capability); c != 0 {
			return c
		}
		return strings.Compare(a.name, b.name)
	})
	overlaps := []Overlap{}
	for _, k := range keys {
		g := groups[k]
		if len(g) < 2 {
			continue
		}
		slugs := make([]string, 0, len(g))
		for s := range g {
			slugs = append(slugs, s)
		}
		slices.Sort(slugs)
		o := Overlap{Capability: k.capability, Requirement: g[slugs[0]].spelling}
		for _, s := range slugs {
			e := Entry{Slug: s, Operations: []string{}, Sources: g[s].sources}
			for _, op := range operationOrder {
				if g[s].ops[op] {
					e.Operations = append(e.Operations, op)
				}
			}
			o.Changes = append(o.Changes, e)
		}
		overlaps = append(overlaps, o)
	}
	return overlaps
}

// Involving keeps the overlaps that name slug.
func Involving(overlaps []Overlap, slug string) []Overlap {
	out := []Overlap{}
	for _, o := range overlaps {
		if slices.ContainsFunc(o.Changes, func(e Entry) bool { return e.Slug == slug }) {
			out = append(out, o)
		}
	}
	return out
}
