package snapshot

import (
	"errors"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/scan"
)

// Place is what a location adds to a use. Branch and Commit are empty for null.
type Place struct {
	Kind, Branch, Commit, Time string
	Default                    bool
}

// WorksInput is what Works reads. Places and Files are keyed by source; Dates
// holds the dates of ideas and issues by path, for proposed. Graph, Tip and
// Tags are for the fields of later tasks.
type WorksInput struct {
	Changes, Records []scan.Use
	Places           map[string]Place
	Files            map[string]Files
	Graph            *Graph
	Tip              string
	Tags             []gitx.Tag
	Dates            map[string]string
}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// Works reports each change and record once, read from the copy read_from
// names, sorted by created (null last), type and slug. Position, stages, the
// tasks' content, documents, deltas, merge, release and last_commit are left
// to later tasks; their arrays are empty, not nil.
func Works(in WorksInput) ([]WorkEntry, []scan.Unreadable) {
	var order []Work
	uses := map[Work][]scan.Use{}
	add := func(typ string, us []scan.Use) {
		for _, u := range us {
			k := Work{typ, u.Slug}
			if _, ok := uses[k]; !ok {
				order = append(order, k)
			}
			uses[k] = append(uses[k], u)
		}
	}
	add("change", in.Changes)
	add("record", in.Records)

	works := []WorkEntry{}
	bad := []scan.Unreadable{}
	for _, k := range order {
		us := uses[k]
		read := us[0]
		for _, u := range us {
			if u.Archived {
				read = u
				break
			}
		}
		w := WorkEntry{Type: k.Key, Slug: k.Slug, Schema: "docket", ReadFrom: read.Source,
			Stages: []string{}, Locations: []Location{}, Documents: []Document{}, Deltas: []DeltaEntry{}}
		for _, u := range us {
			p := in.Places[u.Source]
			w.Locations = append(w.Locations, Location{Source: u.Source, Kind: p.Kind, Branch: str(p.Branch),
				Default: p.Default, Commit: str(p.Commit), Time: str(p.Time), Path: u.Path, Archived: u.Archived})
		}
		files := in.Files[read.Source]
		var front change.Front
		var file string
		var err error
		if k.Key == "record" {
			file = read.Path
			if b, ok := files[file]; ok {
				p := record.Parse(b)
				front, err = p.Front, frontErr(p.HasFront, p.FrontErr)
			}
			if dateRe.MatchString(k.Slug) {
				w.Created = str(k.Slug[:10])
			}
			w.Tasks = []Task{}
		} else {
			file = path.Join(read.Path, "proposal.md")
			if b, ok := files[file]; ok {
				var has bool
				front, has, err = change.ReadFrontMatter(b)
				err = frontErr(has, err)
			}
			if read.Archived {
				w.Archived = str(read.Path)
			}
			if b, ok := files[path.Join(read.Path, ".openspec.yaml")]; ok {
				w.Schema = change.SchemaOf(b)
				w.Created = createdOf(b)
			}
			if w.Schema == "docket" {
				w.Tasks = []Task{}
			}
		}
		if err != nil {
			bad = append(bad, scan.Unreadable{Source: read.Source + ":" + file, Reason: err.Error()})
		} else {
			w.Title, w.Kind, w.Source = field(front, "title"), field(front, "kind"), field(front, "source")
		}
		w.Proposed = w.Created
		if w.Source != nil {
			if d, ok := in.Dates[*w.Source]; ok {
				w.Proposed = str(d)
			}
		}
		works = append(works, w)
	}
	slices.SortStableFunc(works, func(a, b WorkEntry) int {
		switch {
		case a.Created == nil && b.Created != nil:
			return 1
		case a.Created != nil && b.Created == nil:
			return -1
		case a.Created != nil && *a.Created != *b.Created:
			return strings.Compare(*a.Created, *b.Created)
		}
		if c := strings.Compare(a.Type, b.Type); c != 0 {
			return c
		}
		return strings.Compare(a.Slug, b.Slug)
	})
	return works, bad
}

// frontErr is an error when the front matter is broken or absent.
func frontErr(has bool, err error) error {
	if err == nil && !has {
		return errors.New("no front matter")
	}
	return err
}

// str is nil for the empty string.
func str(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func field(f change.Front, name string) *string {
	if v, ok := f.Fields[name]; ok {
		return &v.Value
	}
	return nil
}

// createdOf reads the created date of a change's .openspec.yaml.
func createdOf(b []byte) *string {
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSuffix(line, "\r"), "created:"); ok {
			v = strings.Trim(strings.TrimSpace(v), `"'`)
			if dateRe.MatchString(v) {
				return str(v[:10])
			}
			return nil
		}
	}
	return nil
}
