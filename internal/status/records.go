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
	groups := map[scan.Source][]scan.Record{}
	var order []scan.Source
	for _, r := range picked {
		src := r.Sources[0]
		if _, ok := groups[src]; !ok {
			order = append(order, src)
		}
		groups[src] = append(groups[src], r)
	}
	for _, src := range order {
		want := map[string]bool{}
		for _, r := range groups[src] {
			want[r.Slug+".md"] = true
		}
		files, bad, err := scan.ReadFiles(g, src, record.Dir, func(rel string) bool { return want[rel] })
		if err != nil {
			return nil, 0, nil, err
		}
		unread = append(unread, bad...)
		var commits []gitx.Commit
		if src.Commit != tip && strings.Trim(src.Commit, "0") != "" {
			if commits, err = g.Commits(tip, src.Commit); err != nil {
				return nil, 0, nil, err
			}
		}
		for _, r := range groups[src] {
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
