package archive

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"

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
	head, c, finds, unread, err := loadAtHead(e, slug, loadOpts{Plan: true, BaseIsCommit: true})
	if err != nil {
		return rep, err
	}
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
	f, err := writeOrRefuse(e.Root, c.path("plan.md"), slug, out)
	if err != nil {
		return rep, err
	}
	if f != nil {
		rep.Ticked = []TaskTicks{}
		rep.Findings = append(rep.Findings, *f)
	}
	return rep, nil
}

// writeOrRefuse writes like writeFile; a link on the path comes back as a
// symlink-refused finding and nothing is written.
func writeOrRefuse(root, rel, slug string, b []byte) (*Finding, error) {
	return refuseLink(slug, rel, writeFile(root, rel, b))
}

// refuseLink turns a symlink error into a symlink-refused finding.
func refuseLink(slug, rel string, err error) (*Finding, error) {
	var sl *safepath.SymlinkError
	if errors.As(err, &sl) {
		return &Finding{Rule: "symlink-refused", Change: slug, File: rel, Message: err.Error()}, nil
	}
	return nil, err
}

// writeAll is the default writeTemp. Tests replace writeTemp to inject a
// write failure.
func writeAll(f *os.File, b []byte) error {
	_, err := f.Write(b)
	return err
}

var writeTemp = writeAll

// writeFile replaces the content of an existing regular file below root,
// refusing a link anywhere on its path. It writes a temporary file in the
// same directory and renames it onto the target, so a failure leaves the
// target untouched.
func writeFile(root, rel string, b []byte) error {
	if err := safepath.Check(root, rel); err != nil {
		return err
	}
	target := filepath.Join(root, filepath.FromSlash(rel))
	fi, err := os.Lstat(target)
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", rel)
	}
	return swapIn(target, fi.Mode().Perm(), b)
}

// createFile writes a new file below root, refusing a link anywhere on its
// path and a file that already exists. Like writeFile it goes through a
// temporary file and a rename.
func createFile(root, rel string, b []byte) error {
	if err := safepath.Check(root, rel); err != nil {
		return err
	}
	target := filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Lstat(target); err == nil {
		return &fs.PathError{Op: "create", Path: target, Err: fs.ErrExist}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return swapIn(target, 0o644, b)
}

// swapIn writes b to a temporary file next to target, gives it perm and
// renames it onto target. On any failure the temporary file is removed and
// target is left as it was.
func swapIn(target string, perm os.FileMode, b []byte) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".dkt-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name())
		}
	}()
	if err = writeTemp(tmp, b); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), target)
}
