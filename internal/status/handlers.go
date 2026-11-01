package status

import (
	"slices"
	"sort"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/scan"
)

// findHandlers finds the records and changes whose source names one of paths,
// on every source. Each list holds one handler per kind and slug, sorted by
// kind then slug. It reads nothing when paths is empty.
func findHandlers(g gitx.Git, inv scan.Inventory, paths map[string]bool) (map[string][]Handler, []scan.Unreadable, error) {
	handlers := map[string][]Handler{}
	for p := range paths {
		handlers[p] = []Handler{}
	}
	if len(paths) == 0 {
		return handlers, nil, nil
	}
	// One handler per kind and slug.
	add := func(kind, slug, source string, merged bool) {
		p := change.SourcePath(source)
		if !paths[p] || slices.ContainsFunc(handlers[p], func(h Handler) bool { return h.Kind == kind && h.Slug == slug }) {
			return
		}
		handlers[p] = append(handlers[p], Handler{Kind: kind, Slug: slug, Merged: merged})
	}
	// A record or change the tip holds is read from the tip, any other from its first source.
	readFrom := func(srcs []scan.Source) scan.Source {
		if t := tipSource(srcs); t != nil {
			return *t
		}
		return srcs[0]
	}
	var unread []scan.Unreadable
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
	for _, hs := range handlers {
		sort.Slice(hs, func(i, j int) bool {
			if hs[i].Kind != hs[j].Kind {
				return hs[i].Kind < hs[j].Kind
			}
			return hs[i].Slug < hs[j].Slug
		})
	}
	return handlers, unread, nil
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
