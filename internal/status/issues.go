package status

import (
	"slices"
	"sort"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/issue"
	"github.com/wake/docket/internal/record"
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

// LoadIssues lists the issues that are pending or in progress. An issue that
// is dropped, moved to an idea, or handled by work the default branch holds is
// not listed.
func LoadIssues(g gitx.Git, inv scan.Inventory) ([]Issue, []scan.Unreadable, error) {
	issues := []Issue{}
	groups, unread, err := readGroups(g, issue.Dir, len(inv.Issues),
		func(i int) scan.Source { return inv.Issues[i].Sources[0] },
		func(i int) []string { return []string{inv.Issues[i].Slug + ".md"} })
	if err != nil {
		return nil, nil, err
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
	if len(issues) == 0 {
		return issues, unread, nil
	}
	named := map[string]*Issue{}
	for i := range issues {
		named[issue.Dir+"/"+issues[i].Slug+".md"] = &issues[i]
	}
	// One handler per kind and slug.
	add := func(kind, slug, source string, merged bool) {
		is, ok := named[strings.TrimSpace(source)]
		if !ok || slices.ContainsFunc(is.Handlers, func(h Handler) bool { return h.Kind == kind && h.Slug == slug }) {
			return
		}
		is.Handlers = append(is.Handlers, Handler{Kind: kind, Slug: slug, Merged: merged})
	}
	// A record or change the tip holds is read from the tip, any other from its first source.
	readFrom := func(srcs []scan.Source) scan.Source {
		if t := tipSource(srcs); t != nil {
			return *t
		}
		return srcs[0]
	}
	recGroups, bad, err := readGroups(g, record.Dir, len(inv.Records),
		func(i int) scan.Source { return readFrom(inv.Records[i].Sources) },
		func(i int) []string { return []string{inv.Records[i].Slug + ".md"} })
	if err != nil {
		return nil, nil, err
	}
	unread = append(unread, bad...)
	for _, grp := range recGroups {
		for _, i := range grp.Idx {
			r := inv.Records[i]
			add("record", r.Slug, record.Parse(grp.Files[r.Slug+".md"]).FrontMatter.Source, tipSource(r.Sources) != nil)
		}
	}
	// Active and archived copies of a change, read once per source from openspec/changes.
	type copyOf struct {
		slug, rel string
		srcs      []scan.Source
	}
	var copies []copyOf
	tipHeld := map[string]bool{}
	for _, c := range inv.Changes {
		copies = append(copies, copyOf{c.Slug, c.Slug, c.Sources})
		tipHeld[c.Slug] = tipHeld[c.Slug] || tipSource(c.Sources) != nil
	}
	for _, a := range inv.Archived {
		copies = append(copies, copyOf{a.Slug, "archive/" + a.Folder, a.Sources})
		tipHeld[a.Slug] = tipHeld[a.Slug] || tipSource(a.Sources) != nil
	}
	chGroups, bad, err := readGroups(g, "openspec/changes", len(copies),
		func(i int) scan.Source { return readFrom(copies[i].srcs) },
		func(i int) []string {
			return []string{copies[i].rel + "/" + change.DocketFile, copies[i].rel + "/" + change.OpenSpecFile}
		})
	if err != nil {
		return nil, nil, err
	}
	unread = append(unread, bad...)
	for _, grp := range chGroups {
		for _, i := range grp.Idx {
			c := copies[i]
			// When the tip holds the slug only its copies count, as for records.
			if tipHeld[c.slug] && tipSource(c.srcs) == nil {
				continue
			}
			files := map[string][]byte{}
			for _, name := range []string{change.DocketFile, change.OpenSpecFile} {
				if b, ok := grp.Files[c.rel+"/"+name]; ok {
					files[name] = b
				}
			}
			if m, err := change.ReadMetadata(files); err == nil {
				add("change", c.slug, m.Source, tipHeld[c.slug])
			}
		}
	}
	out := issues[:0]
	for _, is := range issues {
		sort.Slice(is.Handlers, func(i, j int) bool {
			a, b := is.Handlers[i], is.Handlers[j]
			if a.Kind != b.Kind {
				return a.Kind < b.Kind
			}
			return a.Slug < b.Slug
		})
		is.State = Pending
		held := false
		for _, h := range is.Handlers {
			if !h.Merged {
				is.State = InProgress
			}
			held = held || h.Merged
		}
		if is.State == Pending && held {
			continue
		}
		out = append(out, is)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out, unread, nil
}

// tipSource is the default branch source among srcs, or nil.
func tipSource(srcs []scan.Source) *scan.Source {
	for i := range srcs {
		if srcs[i].Kind == scan.KindDefaultBranch {
			return &srcs[i]
		}
	}
	return nil
}
