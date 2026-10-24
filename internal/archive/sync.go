package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"syscall"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/safepath"
)

var taskNumberRe = regexp.MustCompile(`^[1-9][0-9]*$`)

// taskWork is what the trailers say about one task of the plan.
type taskWork struct {
	Task    change.Task
	Commits []gitx.Commit // commits that name the task, oldest first
	Lines   []int         // unticked step lines, to tick
}

// syncResult is the outcome of computeSync.
type syncResult struct {
	Tasks      []taskWork // tasks named by a commit, in plan order
	Findings   []Finding
	AnyTrailer bool // some commit for the change carries a Task trailer
}

// Ticks lists the tasks that have steps to tick.
func (r syncResult) Ticks() []TaskTicks {
	var out []TaskTicks
	for _, w := range r.Tasks {
		if len(w.Lines) > 0 {
			out = append(out, TaskTicks{Task: w.Task.Number, Steps: len(w.Lines)})
		}
	}
	return out
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

// computeSync applies the Task trailers of the commits that carry
// Change: <c.Slug> to c's plan.
func computeSync(c *loadedChange, commits []gitx.Commit) syncResult {
	var res syncResult
	byNumber := map[int]*taskWork{}
	for _, t := range c.Plan.Tasks {
		if _, dup := byNumber[t.Number]; dup {
			continue
		}
		w := &taskWork{Task: t}
		for _, s := range t.Steps {
			if !s.Done {
				w.Lines = append(w.Lines, s.Line)
			}
		}
		byNumber[t.Number] = w
	}
	named := map[int]bool{}
	for _, cm := range commits {
		if !slices.Contains(cm.Change, c.Slug) {
			continue
		}
		seen := map[string]bool{}
		for _, v := range cm.Task {
			res.AnyTrailer = true
			if seen[v] {
				continue
			}
			seen[v] = true
			n := 0
			if taskNumberRe.MatchString(v) {
				_, _ = fmt.Sscanf(v, "%d", &n)
			}
			w := byNumber[n]
			if w == nil {
				res.Findings = append(res.Findings, Finding{Rule: "task-unknown", Change: c.Slug, File: c.path("plan.md"),
					Message: fmt.Sprintf("commit %s has trailer Task: %s, which is not a task of plan.md", shortSHA(cm.SHA), v)})
				continue
			}
			w.Commits = append(w.Commits, cm)
			named[n] = true
		}
	}
	for _, t := range c.Plan.Tasks {
		if named[t.Number] {
			res.Tasks = append(res.Tasks, *byNumber[t.Number])
			named[t.Number] = false
		}
	}
	return res
}

// Sync ticks the unticked steps of every task of plan.md that a commit for
// the change names. With check it writes nothing and reports each such task
// as plan-unsynced. The error is for failures not tied to the change.
func Sync(e Env, slug string, check bool) (SyncReport, error) {
	rep := SyncReport{Change: slug, Ticked: []TaskTicks{}}
	head, ok, err := e.Git.Head()
	if err != nil {
		return rep, err
	}
	if !ok {
		rep.Findings = append(rep.Findings, Finding{Rule: "no-commits", Change: slug,
			Message: "the repository has no commits; commit first"})
		return rep, nil
	}
	c, finds, unread := loadChange(e, slug, true, true)
	if c == nil {
		rep.Findings, rep.Unreadable = finds, unread
		return rep, nil
	}
	commits, err := e.Git.Commits(c.Meta.Base, head)
	if err != nil {
		return rep, err
	}
	res := computeSync(c, commits)
	rep.Findings = append(rep.Findings, res.Findings...)
	rep.Ticked = append(rep.Ticked, res.Ticks()...)

	var lines []int
	for _, w := range res.Tasks {
		if len(w.Lines) == 0 {
			continue
		}
		lines = append(lines, w.Lines...)
		if check {
			rep.Findings = append(rep.Findings, Finding{Rule: "plan-unsynced", Change: slug, File: c.path("plan.md"), Line: w.Task.Line,
				Message: fmt.Sprintf("Task %d has commits and %d unticked steps; run dkt sync %s", w.Task.Number, len(w.Lines), slug)})
		}
	}
	if check || len(lines) == 0 {
		return rep, nil
	}
	out, err := change.Tick(c.PlanBytes, lines)
	if err != nil {
		return rep, err
	}
	if err := writeFile(e.Root, c.path("plan.md"), out); err != nil {
		var sl *safepath.SymlinkError
		if errors.As(err, &sl) {
			rep.Ticked = []TaskTicks{}
			rep.Findings = append(rep.Findings, Finding{Rule: "symlink-refused", Change: slug, File: c.path("plan.md"), Message: err.Error()})
			return rep, nil
		}
		return rep, err
	}
	return rep, nil
}

// writeFile replaces the content of an existing file below root, refusing a
// link anywhere on its path.
func writeFile(root, rel string, b []byte) error {
	if err := safepath.Check(root, rel); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(root, filepath.FromSlash(rel)), os.O_WRONLY|os.O_TRUNC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
