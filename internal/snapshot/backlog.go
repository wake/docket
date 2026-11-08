package snapshot

import (
	"slices"
	"sort"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/idea"
	"github.com/wake/docket/internal/scan"
)

// BacklogInput is what Backlog reads. Ideas holds the idea files of the default
// branch tip by path; Issues are the issue uses (Path is the issue file) with
// their files in IssueFiles by source; Dates holds the idea dates by path;
// Works are the works of Works, with merged and locations set.
type BacklogInput struct {
	Order      []byte
	Ideas      map[string][]byte
	Issues     []scan.Use
	IssueFiles map[string]Files
	Dates      map[string]string
	Works      []WorkEntry
	Places     map[string]Place
}

type issueGroup struct {
	slug, path string
	uses       []scan.Use
}

func (g *issueGroup) date() string { return g.slug[:min(10, len(g.slug))] }

// Backlog reports the ideas, the issues and the order. It sets the position of
// each work whose source names an item of the order, and puts the source idea
// or issue of a work first in its documents.
func Backlog(in BacklogInput) (ideas []IdeaEntry, issues []IssueEntry, order []OrderItem, problems []idea.Problem) {
	// Issues are grouped by slug and read from the first location.
	var list []*issueGroup
	byPath := map[string]*issueGroup{}
	bySlug := map[string]*issueGroup{}
	for _, u := range in.Issues {
		g := bySlug[u.Slug]
		if g == nil {
			g = &issueGroup{slug: u.Slug, path: u.Path}
			bySlug[u.Slug], byPath[u.Path] = g, g
			list = append(list, g)
		}
		g.uses = append(g.uses, u)
	}
	sort.SliceStable(list, func(a, b int) bool { return list[a].slug < list[b].slug })
	first := func(g *issueGroup) []byte { return in.IssueFiles[g.uses[0].Source][g.uses[0].Path] }

	lines, _, problems, _ := idea.Check(in.Order, func(paths []string) (map[string][]byte, error) {
		got := map[string][]byte{}
		for _, p := range paths {
			if b, ok := in.Ideas[p]; ok {
				got[p] = b
			} else if g := byPath[p]; g != nil && first(g) != nil {
				got[p] = first(g)
			}
		}
		return got, nil
	})
	position := map[string]int{}
	for n, l := range lines {
		position[l.Path] = n + 1
		order = append(order, OrderItem{Position: n + 1, Path: l.Path})
	}

	// Who names each item, and which items the tip's own copy of a work names.
	named := map[string][]WorkRef{}
	onTip := map[string]bool{}
	for _, w := range in.Works {
		if w.Source == nil {
			continue
		}
		named[*w.Source] = append(named[*w.Source], WorkRef{Type: w.Type, Slug: w.Slug})
		for _, l := range w.Locations {
			onTip[*w.Source] = onTip[*w.Source] || l.Default
		}
	}

	paths := make([]string, 0, len(in.Ideas))
	parsed := map[string]idea.Idea{}
	for p, b := range in.Ideas {
		paths = append(paths, p)
		parsed[p], _ = idea.Parse(b) // an idea that does not parse is listed bare; Check reports it
	}
	sort.Strings(paths)
	for _, p := range paths {
		id := parsed[p]
		e := IdeaEntry{Path: p, Title: str(id.Title), Needs: id.Needs, Dropped: id.Dropped,
			Date: str(in.Dates[p]), TakenBy: named[p], WaitingOn: []string{}}
		for _, n := range id.Needs {
			if !parsed[n].Dropped && !onTip[n] {
				e.WaitingOn = append(e.WaitingOn, n)
			}
		}
		if n, ok := position[p]; ok {
			e.Position = &n
		}
		switch {
		case id.Dropped:
			e.State = "dropped"
		case len(named[p]) > 0:
			e.State = "taken"
		case e.Position != nil:
			e.State = "queued"
		default:
			e.State = "pool"
		}
		ideas = append(ideas, e)
	}

	for _, g := range list {
		var fm change.Front
		if f, ok, err := change.ReadFrontMatter(first(g)); err == nil && ok {
			fm = f
		}
		e := IssueEntry{Slug: g.slug, Path: g.path, Title: field(fm, "title"), Source: field(fm, "source"),
			Date: g.date(), State: "open", Handlers: []Handler{}, Locations: []Location{}}
		if n, ok := position[g.path]; ok {
			e.Position = &n
		}
		for _, u := range g.uses {
			p := in.Places[u.Source]
			e.Locations = append(e.Locations, Location{Source: u.Source, Kind: p.Kind, Branch: str(p.Branch),
				Default: p.Default, Commit: str(p.Commit), Time: str(p.Time), Path: u.Path, Archived: u.Archived})
		}
		merged := 0
		for _, w := range in.Works {
			if w.Source != nil && *w.Source == g.path {
				e.Handlers = append(e.Handlers, Handler{Type: w.Type, Slug: w.Slug, Merged: w.Merged})
				if w.Merged {
					merged++
				}
			}
		}
		if len(e.Handlers) > 0 {
			e.State = "in-progress"
			if merged == len(e.Handlers) {
				e.State = "done"
			}
		}
		issues = append(issues, e)
	}

	// The default ref is the first non-worktree source that is the default branch.
	var defRef string
	srcs := make([]string, 0, len(in.Places))
	for s := range in.Places {
		srcs = append(srcs, s)
	}
	slices.Sort(srcs)
	for _, s := range srcs {
		if p := in.Places[s]; p.Default && p.Kind != "worktree" {
			defRef = s
			break
		}
	}

	for i := range in.Works {
		w := &in.Works[i]
		if w.Source == nil {
			continue
		}
		if n, ok := position[*w.Source]; ok {
			w.Position = &n
		}
		var doc Document
		var date string
		if _, ok := in.Ideas[*w.Source]; ok {
			doc, date = Document{Kind: "idea", Path: *w.Source, Source: defRef}, in.Dates[*w.Source]
		} else if g := byPath[*w.Source]; g != nil {
			doc, date = Document{Kind: "issue", Path: g.path, Source: g.uses[0].Source}, g.date()
		} else {
			continue
		}
		w.Documents = append([]Document{doc}, w.Documents...)
		if date != "" {
			w.Proposed = str(date)
		}
	}
	return ideas, issues, order, problems
}
