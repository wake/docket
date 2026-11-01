package status

import (
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/delta"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
)

// Metadata is what a change's metadata files record about where a change came from.
type Metadata struct {
	Kind    string `json:"kind"`
	Source  string `json:"source"`
	Base    string `json:"base"`
	Created string `json:"created"`
}

// Documents says which documents a change folder holds. Specs lists the
// capability paths of its delta specs, sorted.
type Documents struct {
	Proposal bool     `json:"proposal"`
	Design   bool     `json:"design"`
	Plan     bool     `json:"plan"`
	Specs    []string `json:"specs"`
}

// Activity is the newest commit that touched or named a change.
type Activity struct {
	Commit   string `json:"commit"`
	Time     string `json:"time"`
	IdleDays int    `json:"idle_days"`
}

// Change is one active change with its phase, tasks and activity.
type Change struct {
	Slug         string        `json:"slug"`
	Schema       string        `json:"schema"`
	Title        *string       `json:"title"`
	Sources      []scan.Source `json:"sources"`
	Metadata     *Metadata     `json:"metadata"`
	Documents    Documents     `json:"documents"`
	Phase        string        `json:"phase"`
	Tasks        []Task        `json:"tasks"`
	Progress     Progress      `json:"progress"`
	LastActivity *Activity     `json:"last_activity"`
	Uncommitted  *int          `json:"uncommitted"`
}

// Load reads every active change of inv from the first source of each, in
// slug order. A slug with any archived use is left out: archived means
// archived anywhere. Files that cannot be read are returned, and the change
// is listed from what could be read. now is the time idle days count to.
func Load(g gitx.Git, inv scan.Inventory, now time.Time) ([]Change, []scan.Unreadable, error) {
	keep := func(rel string) bool {
		switch rel {
		case change.DocketFile, change.OpenSpecFile, "proposal.md", "design.md", "plan.md", "tasks.md":
			return true
		}
		_, ok := delta.CapabilityPath(rel)
		return ok
	}
	changes := []Change{}
	unread := []scan.Unreadable{}
	for _, c := range inv.Changes {
		if _, archived := inv.Uses(c.Slug); len(archived) > 0 {
			continue
		}
		src := c.Sources[0]
		dir := "openspec/changes/" + c.Slug
		files, bad, err := scan.ReadFiles(g, src, dir, keep)
		if err != nil {
			return nil, nil, err
		}
		unread = append(unread, bad...)
		ch := Change{Slug: c.Slug, Sources: c.Sources, Documents: documentsOf(files)}
		// Invalid schema metadata counts as docket; dkt lint reports the file.
		if ch.Schema, err = change.ReadSchema(files); err != nil {
			ch.Schema = change.DocketSchema
		}
		base := ""
		// Unreadable metadata leaves the change without metadata, title, base,
		// commits and activity; dkt lint reports the file.
		if m, err := change.ReadMetadata(files); err == nil {
			ch.Metadata = &Metadata{Kind: m.Kind, Source: m.Source, Base: m.Base, Created: m.Created}
			if m.Title != "" {
				ch.Title = &m.Title
			}
			base = m.Base
		}
		commits, err := rangeCommits(g, base, src.Commit)
		if err != nil {
			return nil, nil, err
		}
		if ch.Schema != change.DocketSchema {
			b, ok := files["tasks.md"]
			ch.Phase, ch.Progress = SummarizeChecklist(b, ok)
			ch.Tasks = []Task{}
		} else {
			var plan *change.Plan
			var named []change.NamedTask
			if b, ok := files["plan.md"]; ok {
				p := change.ParsePlan(b)
				plan = &p
				// dkt status does not report unknown Task trailers; dkt sync does.
				named, _, _ = change.NamedTasks(p, c.Slug, commits)
			}
			ch.Phase, ch.Tasks, ch.Progress = Summarize(plan, named)
		}
		if ch.LastActivity, err = lastActivity(g, c.Slug, dir, base, src.Commit, commits, now); err != nil {
			return nil, nil, err
		}
		if src.Kind == scan.KindWorktree {
			wt := gitx.Git{Dir: src.Name, Env: g.Env}
			if paths, err := wt.UncommittedUnder(dir); err != nil {
				unread = append(unread, scan.Unreadable{Source: src, Path: dir, Reason: err.Error()})
			} else {
				n := len(paths)
				ch.Uncommitted = &n
			}
		}
		changes = append(changes, ch)
	}
	return changes, unread, nil
}

func documentsOf(files map[string][]byte) Documents {
	d := Documents{Specs: []string{}}
	_, d.Proposal = files["proposal.md"]
	_, d.Design = files["design.md"]
	_, d.Plan = files["plan.md"]
	for rel := range files {
		if capability, ok := delta.CapabilityPath(rel); ok {
			d.Specs = append(d.Specs, capability)
		}
	}
	sort.Strings(d.Specs)
	return d
}

// rangeCommits lists the commits in base..head when base names a commit, and
// none otherwise.
func rangeCommits(g gitx.Git, base, head string) ([]gitx.Commit, error) {
	if base == "" || strings.HasPrefix(base, "-") {
		return nil, nil
	}
	ok, err := g.RefExists(base + "^{commit}")
	if err != nil || !ok {
		return nil, err
	}
	return g.Commits(base, head)
}

// lastActivity picks the newest commit of the range that names the change in
// a Change trailer or touches its folder; on a tie the later one wins.
func lastActivity(g gitx.Git, slug, dir, base, head string, commits []gitx.Commit, now time.Time) (*Activity, error) {
	if len(commits) == 0 {
		return nil, nil
	}
	folder, err := g.Commits(base, head, dir)
	if err != nil {
		return nil, err
	}
	touched := map[string]bool{}
	for _, c := range folder {
		touched[c.SHA] = true
	}
	var best *gitx.Commit
	for i := range commits {
		c := &commits[i]
		if !slices.Contains(c.Change, slug) && !touched[c.SHA] {
			continue
		}
		if best == nil || !c.Time.Before(best.Time) {
			best = c
		}
	}
	if best == nil {
		return nil, nil
	}
	idle := 0
	if now.After(best.Time) {
		idle = int(now.Sub(best.Time) / (24 * time.Hour))
	}
	return &Activity{Commit: best.SHA, Time: best.Time.Format(time.RFC3339), IdleDays: idle}, nil
}
