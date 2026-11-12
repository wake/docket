package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/safefile"
	"github.com/wake/docket/internal/safepath"
)

type syncResult struct {
	Slug  string     `json:"slug"`
	Check bool       `json:"check"`
	Tasks []syncTask `json:"tasks"` // never null
}

type syncTask struct {
	Number  int      `json:"number"`
	Name    string   `json:"name"`
	Commits []string `json:"commits"` // full SHAs, oldest first; never null
	Steps   int      `json:"steps"`
	Done    int      `json:"done"`   // ticked steps after the run (as the file is, with --check)
	Ticked  int      `json:"ticked"` // steps this run ticked (or would, with --check)
}

// runSync is the body of dkt sync.
func runSync(c *Context, slug string, check bool) (Outcome, error) {
	if slug == "" || slug == "." || slug == ".." || strings.ContainsAny(slug, `/\`) {
		return Outcome{}, UsageError{Msg: fmt.Sprintf("%q is not a change name", slug)}
	}
	root := c.Repo.Root
	if change.HasDate(slug) {
		return syncRecord(c, slug, check)
	}
	rel := "openspec/changes/" + slug
	plan := rel + "/plan.md"
	if err := safepath.Check(root, plan); err != nil {
		return refuse(err)
	}
	if fi, err := os.Stat(filepath.Join(root, rel)); err != nil || !fi.IsDir() {
		return finding("change-missing", rel+" does not exist")
	}
	meta := rel + "/.openspec.yaml"
	schema := "docket"
	b, err := safefile.ReadNoFollow(filepath.Join(root, meta))
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return Outcome{Unreadable: []Unreadable{{Source: meta, Reason: err.Error()}}}, nil
	default:
		schema = change.SchemaOf(b)
	}
	if schema != "docket" {
		return Outcome{Findings: []Finding{{Rule: "other-schema", File: meta,
			Message: fmt.Sprintf("%s names the schema %s; dkt sync handles docket changes only", meta, schema)}}}, nil
	}
	planPath := filepath.Join(root, plan)
	src, err := safefile.ReadNoFollow(planPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return finding("plan-missing", plan+" does not exist")
	case err != nil:
		return Outcome{Unreadable: []Unreadable{{Source: plan, Reason: err.Error()}}}, nil
	}
	var commits []gitx.Trailers
	if c.Repo.Head != nil {
		if commits, err = c.Git.ChangeTrailers("HEAD"); err != nil {
			return Outcome{}, err
		}
	}
	return syncFile(syncFileArgs{Slug: slug, File: plan, Root: root, Src: src, Plan: change.ParsePlan(src),
		Key: "Change", Commits: commits, Finding: slug, Check: check})
}

// syncRecord is dkt sync for a record: its tasks and the Record trailer.
func syncRecord(c *Context, slug string, check bool) (Outcome, error) {
	rel := record.Path(slug)
	if err := safepath.Check(c.Repo.Root, rel); err != nil {
		return refuse(err)
	}
	file := filepath.Join(c.Repo.Root, rel)
	src, err := safefile.ReadNoFollow(file)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return finding("record-missing", rel+" does not exist")
	case err != nil:
		return Outcome{Unreadable: []Unreadable{{Source: rel, Reason: err.Error()}}}, nil
	}
	var commits []gitx.Trailers
	if c.Repo.Head != nil {
		if commits, err = c.Git.RecordTrailers("HEAD"); err != nil {
			return Outcome{}, err
		}
	}
	return syncFile(syncFileArgs{Slug: slug, File: rel, Root: c.Repo.Root, Src: src, Plan: record.Parse(src).Tasks,
		Key: "Record", Commits: commits, Check: check})
}

type syncFileArgs struct {
	Slug, File, Root string // File is the path relative to Root, the repository root
	Src              []byte
	Plan             change.Plan
	Key              string // the trailer key: Change or Record
	Commits          []gitx.Trailers
	Finding          string // the Change of each finding; empty for a record
	Check            bool
}

// syncFile ticks the tasks that the commits' trailers name and reports.
func syncFile(a syncFileArgs) (Outcome, error) {
	slug, plan, src, p, check := a.Slug, a.File, a.Src, a.Plan, a.Check
	res := syncResult{Slug: slug, Check: check, Tasks: []syncTask{}}
	for _, t := range p.Tasks {
		res.Tasks = append(res.Tasks, syncTask{Number: t.Number, Name: t.Name, Commits: []string{}, Steps: len(t.Steps)})
	}
	var out Outcome
	add := func(rule, msg string, line int) {
		out.Findings = append(out.Findings, Finding{Rule: rule, Message: msg, Change: a.Finding, File: plan, Line: line})
	}

	for _, cm := range a.Commits {
		short := cm.SHA[:min(7, len(cm.SHA))]
		for _, v := range cm.Values {
			nums, ok, mine := change.TaskRefs(v, slug)
			if !mine {
				continue
			}
			if !ok {
				add("trailer-malformed", fmt.Sprintf("commit %s has the trailer %s: %s, whose task list is not positive numbers separated by commas", short, a.Key, v), 0)
				continue
			}
			var seen []int
			for _, n := range nums {
				if slices.Contains(seen, n) {
					continue
				}
				seen = append(seen, n)
				found := false
				for i := range res.Tasks {
					if res.Tasks[i].Number != n {
						continue
					}
					found = true
					if !slices.Contains(res.Tasks[i].Commits, cm.SHA) {
						res.Tasks[i].Commits = append(res.Tasks[i].Commits, cm.SHA)
					}
				}
				if !found {
					add("trailer-unknown-task", fmt.Sprintf("commit %s has the trailer %s: %s, but %s has no Task %d", short, a.Key, v, path.Base(plan), n), 0)
				}
			}
		}
	}

	var due []int
	for i, t := range p.Tasks {
		var open []int
		for _, s := range t.Steps {
			if s.Done {
				res.Tasks[i].Done++
			} else {
				open = append(open, s.Line)
			}
		}
		st := &res.Tasks[i]
		switch {
		case len(st.Commits) > 0 && len(open) > 0:
			st.Ticked = len(open)
			due = append(due, open...)
			if check {
				add("task-unticked", fmt.Sprintf("Task %d has commits and %d unticked steps; run dkt sync %s", t.Number, len(open), slug), t.Line)
			} else {
				st.Done += len(open)
			}
		case len(open) > 0 && check:
			add("task-no-commit", fmt.Sprintf("Task %d has no commit carrying %s: %s/%d and %d unticked steps", t.Number, a.Key, slug, t.Number, len(open)), t.Line)
		}
	}
	if !check && len(due) > 0 {
		ticked, err := change.Tick(src, due)
		if err != nil {
			return Outcome{}, err
		}
		root, err := os.OpenRoot(a.Root) // a folder swapped for a link since the read cannot lead the write out
		if err != nil {
			return Outcome{}, err
		}
		defer root.Close()
		name := filepath.FromSlash(plan)
		fi, err := root.Stat(name)
		if err != nil {
			return Outcome{}, err
		}
		if err := safefile.InstallIn(root, name, ticked, fi.Mode().Perm(), true, plan); err != nil {
			return Outcome{}, err
		}
	}

	out.Result = res
	for _, t := range res.Tasks {
		line := fmt.Sprintf("Task %d: %d/%d steps, %d commits", t.Number, t.Done, t.Steps, len(t.Commits))
		switch {
		case t.Ticked > 0 && check:
			line += fmt.Sprintf(", %d to tick", t.Ticked)
		case t.Ticked > 0:
			line += fmt.Sprintf(", ticked %d", t.Ticked)
		}
		out.Human = append(out.Human, line)
	}
	return out, nil
}
