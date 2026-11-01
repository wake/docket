package status

import (
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/scan"
)

// States of a record.
const (
	Open   = "open"
	Merged = "merged"
)

// Merge is the commit that brought a record to the default branch.
type Merge struct {
	Commit string `json:"commit"`
	Time   string `json:"time"`
}

// Record is one record with its tasks.
type Record struct {
	Slug     string        `json:"slug"`
	Title    *string       `json:"title"`
	Kind     *string       `json:"kind"`
	Source   *string       `json:"source"`
	Sources  []scan.Source `json:"sources"`
	State    string        `json:"state"`
	Merge    *Merge        `json:"merge"`
	Tag      *string       `json:"tag"`
	Tasks    []Task        `json:"tasks"`
	Progress Progress      `json:"progress"`
}

// RecordQuery picks the records to list. With Slugs, those records are listed
// whatever their state and the other fields are unused. Otherwise the open
// records are listed, and with Merged the merged ones too: Since keeps those
// merged after a revision, Limit (0 means no limit) keeps the newest few.
type RecordQuery struct {
	Slugs  []string
	Merged bool
	Since  string
	Limit  int
}

// LoadRecords lists the records q picks. total is the number of records the
// query matched before Limit cut the merged ones.
func LoadRecords(g gitx.Git, inv scan.Inventory, q RecordQuery) (records []Record, total int, unread []scan.Unreadable, err error) {
	records = []Record{}
	unread = []scan.Unreadable{}
	var picked, merged []scan.Record
	for _, r := range inv.Records {
		switch {
		case len(q.Slugs) > 0:
			if slices.Contains(q.Slugs, r.Slug) {
				picked = append(picked, r)
			}
		case stateOf(r) == Open:
			picked = append(picked, r)
		case q.Merged:
			merged = append(merged, r)
		}
	}
	tip := inv.DefaultBranch.Commit
	merges := map[string]gitx.Addition{}
	for _, r := range picked {
		if stateOf(r) == Merged {
			merged = append(merged, r)
		}
	}
	picked = slices.DeleteFunc(picked, func(r scan.Record) bool { return stateOf(r) == Merged })
	if len(merged) > 0 {
		adds, err := g.Additions(tip, q.Since, record.Dir)
		if err != nil {
			return nil, 0, nil, err
		}
		for _, a := range adds {
			if slug, ok := record.SlugOf(strings.TrimPrefix(a.Path, record.Dir+"/")); ok && a.Path == record.Dir+"/"+slug+".md" {
				if _, seen := merges[slug]; !seen {
					merges[slug] = a
				}
			}
		}
		if q.Since != "" {
			merged = slices.DeleteFunc(merged, func(r scan.Record) bool { _, ok := merges[r.Slug]; return !ok })
		}
		sort.SliceStable(merged, func(i, j int) bool { return mergedBefore(merged[i].Slug, merged[j].Slug, merges) })
	}
	total = len(picked) + len(merged)
	if q.Limit > 0 && len(merged) > q.Limit {
		merged = merged[:q.Limit]
	}
	picked = append(picked, merged...)
	// One read per first source; commits are read once per source commit.
	groups, bad, err := readGroups(g, record.Dir, len(picked),
		func(i int) scan.Source { return picked[i].Sources[0] },
		func(i int) string { return picked[i].Slug + ".md" })
	if err != nil {
		return nil, 0, nil, err
	}
	unread = append(unread, bad...)
	for _, grp := range groups {
		src, files := grp.Src, grp.Files
		var commits []gitx.Commit
		if src.Commit != tip && strings.Trim(src.Commit, "0") != "" {
			if commits, err = g.Commits(tip, src.Commit); err != nil {
				return nil, 0, nil, err
			}
		}
		for _, i := range grp.Idx {
			r := picked[i]
			f := record.Parse(files[r.Slug+".md"])
			rec := Record{Slug: r.Slug, Sources: r.Sources, State: stateOf(r),
				Title: orNil(f.FrontMatter.Title), Kind: orNil(f.FrontMatter.Kind), Source: orNil(f.FrontMatter.Source)}
			named, _, _ := change.MatchTasks(f.Plan, record.CommitsFor(r.Slug, commits))
			_, rec.Tasks, rec.Progress = Summarize(&f.Plan, named)
			if a, ok := merges[r.Slug]; ok && rec.State == Merged {
				rec.Merge = &Merge{Commit: a.SHA, Time: a.Time.Format(time.RFC3339)}
			}
			records = append(records, rec)
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		if (records[i].State == Open) != (records[j].State == Open) {
			return records[i].State == Open
		}
		if records[i].State == Open {
			return records[i].Slug < records[j].Slug
		}
		return mergedBefore(records[i].Slug, records[j].Slug, merges)
	})
	if len(merged) > 0 {
		tags, err := g.Tags()
		var found map[string]string
		if err != nil {
			unread = append(unread, scan.Unreadable{Source: scan.Source{Kind: scan.KindTag, Name: "refs/tags"}, Reason: err.Error()})
		} else {
			var bad []scan.Unreadable
			found, bad = scan.RecordTags(g, tags)
			unread = append(unread, bad...)
		}
		for i := range records {
			if t, ok := found[records[i].Slug]; ok && records[i].State == Merged {
				records[i].Tag = &t
			}
		}
	}
	return records, total, unread, nil
}

// mergedBefore orders merged records: newest merge first, ties by slug, those
// without a merge last by slug.
func mergedBefore(a, b string, merges map[string]gitx.Addition) bool {
	ma, oka := merges[a]
	mb, okb := merges[b]
	switch {
	case oka != okb:
		return oka
	case oka && !ma.Time.Equal(mb.Time):
		return ma.Time.After(mb.Time)
	}
	return a < b
}

func stateOf(r scan.Record) string {
	for _, s := range r.Sources {
		if s.Kind == scan.KindDefaultBranch {
			return Merged
		}
	}
	return Open
}

func orNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// sourceGroup is the items that are read from one source, by index, with the
// files read for them.
type sourceGroup struct {
	Src   scan.Source
	Idx   []int
	Files map[string][]byte
}

// readGroups reads dir once per distinct source of the n items, keeping the
// file nameOf names for each. Groups come in order of first appearance.
func readGroups(g gitx.Git, dir string, n int, srcOf func(int) scan.Source, nameOf func(int) string) ([]sourceGroup, []scan.Unreadable, error) {
	var groups []sourceGroup
	at := map[scan.Source]int{}
	want := map[scan.Source]map[string]bool{}
	for i := 0; i < n; i++ {
		src := srcOf(i)
		k, ok := at[src]
		if !ok {
			k = len(groups)
			at[src] = k
			groups = append(groups, sourceGroup{Src: src})
			want[src] = map[string]bool{}
		}
		groups[k].Idx = append(groups[k].Idx, i)
		want[src][nameOf(i)] = true
	}
	var unread []scan.Unreadable
	for k := range groups {
		src := groups[k].Src
		files, bad, err := scan.ReadFiles(g, src, dir, func(rel string) bool { return want[src][rel] })
		if err != nil {
			return nil, nil, err
		}
		unread = append(unread, bad...)
		groups[k].Files = files
	}
	return groups, unread, nil
}
