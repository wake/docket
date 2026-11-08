package snapshot

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/scan"
	"github.com/wake/docket/internal/specdoc"
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
				front, err = p.Front, p.FrontErr
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
				if !has {
					front = change.Front{}
				}
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
		fill(&w, k, read, files, in)
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

// fill adds the tasks, documents, deltas, stages, merge, release and
// last_commit of a work.
func fill(w *WorkEntry, k Work, read scan.Use, files Files, in WorksInput) {
	isRecord := k.Key == "record"
	var plan change.Plan
	if isRecord {
		if b, ok := files[read.Path]; ok {
			plan = record.Parse(b).Tasks
		}
		w.Documents = append(w.Documents, Document{Kind: "record", Path: read.Path, Source: read.Source})
	} else {
		if b, ok := files[path.Join(read.Path, "plan.md")]; ok && w.Schema == "docket" {
			plan = change.ParsePlan(b)
		}
		prefix := read.Path + "/"
		var names []string
		for f := range files {
			if strings.HasPrefix(f, prefix) {
				names = append(names, f)
			}
		}
		slices.Sort(names)
		for _, f := range names {
			rel := strings.TrimPrefix(f, prefix)
			kind := "other"
			switch {
			case rel == "proposal.md" || rel == "design.md" || rel == "plan.md":
				kind = strings.TrimSuffix(rel, ".md")
			case isDeltaPath(rel):
				kind = "spec-delta"
				d := specdoc.ParseDelta(files[f])
				w.Deltas = append(w.Deltas, DeltaEntry{Capability: path.Dir(strings.TrimPrefix(rel, "specs/")), Path: f,
					Added: d.Added, Modified: d.Modified, Removed: d.Removed, Renamed: d.Renamed})
			}
			w.Documents = append(w.Documents, Document{Kind: kind, Path: f, Source: read.Source})
		}
		slices.SortStableFunc(w.Deltas, func(a, b DeltaEntry) int { return strings.Compare(a.Capability, b.Capability) })
	}

	var commits []string
	dev := false
	g := in.Graph
	if g != nil {
		for _, l := range w.Locations {
			if l.Commit != nil {
				commits = append(commits, *l.Commit)
			}
		}
		reach := g.reachable(commits)
		for _, s := range g.named()[Work{strings.ToLower(k.Key), k.Slug}] {
			if named, tasks := Names(g.bySHA[s], k); reach[s] && named && (isRecord || len(tasks) > 0) {
				dev = true
				break
			}
		}
		if l := g.Last(commits, k); l != nil {
			w.LastCommit = &LastCommit{Commit: l.Commit, Time: l.Time}
		}
		if m := g.Merge(in.Tip, k); m != nil {
			w.Merge = (*Merge)(m)
			if t := g.Release(in.Tags, m.Commit); t != nil {
				w.Release = &Release{Tag: t.Name, Time: t.Time}
			}
		}
	}
	if w.Tasks != nil {
		var byTask map[int][]string
		if g != nil && in.Places[read.Source].Commit != "" {
			byTask = g.TaskCommits(in.Places[read.Source].Commit, k)
		}
		for _, t := range plan.Tasks {
			ticked := 0
			for _, s := range t.Steps {
				if s.Done {
					ticked++
				}
			}
			cs := byTask[t.Number]
			if cs == nil {
				cs = []string{}
			}
			state := "pending"
			if len(cs) > 0 || len(t.Steps) > 0 && ticked == len(t.Steps) {
				state = "done"
				w.Progress.Done++
			}
			w.Tasks = append(w.Tasks, Task{Number: t.Number, Name: t.Name, Steps: len(t.Steps), Ticked: ticked, Commits: cs, State: state})
		}
		w.Progress.Total = len(w.Tasks)
	}
	for _, l := range w.Locations {
		w.Merged = w.Merged || l.Default
	}

	has := func(name string) bool { _, ok := files[path.Join(read.Path, name)]; return ok }
	add := func(ok bool, s string) {
		if ok {
			w.Stages = append(w.Stages, s)
		}
	}
	if isRecord {
		add(true, "record")
	} else {
		design := len(w.Deltas) > 0 || has("design.md")
		add(has("proposal.md"), "proposal")
		add(design, "design")
		add(len(w.Tasks) > 0, "plan")
	}
	add(dev, "dev")
	add(w.Merged, "merged")
	add(w.Archived != nil, "archived")
	// no stage reached leaves phase empty
	if n := len(w.Stages); n > 0 {
		w.Phase = w.Stages[n-1]
	}
}

func isDeltaPath(rel string) bool {
	return strings.HasPrefix(rel, "specs/") && strings.HasSuffix(rel, "/spec.md") && len(rel) > len("specs//spec.md")
}
