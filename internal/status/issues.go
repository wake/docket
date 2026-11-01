package status

import (
	"sort"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/issue"
	"github.com/wake/docket/internal/scan"
)

// InProgress is the state of an issue that work has started on; the other
// listed state is Pending.
const InProgress = "in-progress"

// Handler is a record or change whose source names an issue.
type Handler struct {
	Kind   string `json:"kind"`
	Slug   string `json:"slug"`
	Merged bool   `json:"merged"`
}

// Issue is a file issue that is waiting for work or being worked on.
type Issue struct {
	Slug     string        `json:"slug"`
	Title    *string       `json:"title"`
	Source   *string       `json:"source"`
	State    string        `json:"state"`
	Handlers []Handler     `json:"handlers"`
	Sources  []scan.Source `json:"sources"`
}

// Backlog is what waits for work: the issues, the order and the ideas.
type Backlog struct {
	Issues []Issue
	Order  []OrderLine
	Next   *int
	Ideas  []Idea
}

// LoadBacklog lists the issues that are pending or in progress. An issue that
// is dropped, moved to an idea, or handled by work the default branch holds is
// not listed.
func LoadBacklog(g gitx.Git, inv scan.Inventory) (Backlog, []scan.Unreadable, error) {
	issues := []Issue{}
	groups, unread, err := readGroups(g, issue.Dir, len(inv.Issues),
		func(i int) scan.Source { return inv.Issues[i].Sources[0] },
		func(i int) []string { return []string{inv.Issues[i].Slug + ".md"} })
	if err != nil {
		return Backlog{}, nil, err
	}
	for _, grp := range groups {
		for _, i := range grp.Idx {
			is := inv.Issues[i]
			f := issue.Parse(grp.Files[is.Slug+".md"])
			if (f.FrontMatter.Dropped != nil && *f.FrontMatter.Dropped) || f.FrontMatter.Idea != "" {
				continue
			}
			issues = append(issues, Issue{Slug: is.Slug, Title: orNil(f.FrontMatter.Title), Source: orNil(f.FrontMatter.Source),
				Handlers: []Handler{}, Sources: is.Sources})
		}
	}
	paths := map[string]bool{}
	for _, is := range issues {
		paths[issue.Dir+"/"+is.Slug+".md"] = true
	}
	tipIdeas, lines, bad, err := readTip(g, inv.DefaultBranch.Source())
	if err != nil {
		return Backlog{}, nil, err
	}
	unread = append(unread, bad...)
	for p := range tipIdeas {
		paths[p] = true
	}
	handlers, bad, err := findHandlers(g, inv, paths)
	if err != nil {
		return Backlog{}, nil, err
	}
	unread = append(unread, bad...)
	out := issues[:0]
	for _, is := range issues {
		is.Handlers = handlers[issue.Dir+"/"+is.Slug+".md"]
		if is.State = workState(false, is.Handlers); is.State == Done {
			continue
		}
		out = append(out, is)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	order, next, ideas := listIdeas(tipIdeas, lines, handlers)
	return Backlog{Issues: out, Order: order, Next: next, Ideas: ideas}, unread, nil
}
