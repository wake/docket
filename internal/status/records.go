package status

import (
	"slices"
	"sort"
	"strings"

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
// whatever their state and the other fields are unused. Limit 0 means none.
type RecordQuery struct {
	Slugs  []string
	Merged bool
	Since  string
	Limit  int
}

// LoadRecords lists the open records of inv, or the records named in q.Slugs
// that some source holds. total is the number of records listed.
func LoadRecords(g gitx.Git, inv scan.Inventory, q RecordQuery) (records []Record, total int, unread []scan.Unreadable, err error) {
	records = []Record{}
	unread = []scan.Unreadable{}
	var picked []scan.Record
	for _, r := range inv.Records {
		if slices.Contains(q.Slugs, r.Slug) || (len(q.Slugs) == 0 && stateOf(r) == Open) {
			picked = append(picked, r)
		}
	}
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
	tip := inv.DefaultBranch.Commit
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
			records = append(records, rec)
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		if (records[i].State == Open) != (records[j].State == Open) {
			return records[i].State == Open
		}
		return records[i].Slug < records[j].Slug
	})
	return records, len(records), unread, nil
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
