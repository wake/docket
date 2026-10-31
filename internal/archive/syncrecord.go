package archive

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/record"
	"github.com/wake/docket/internal/safepath"
)

// SyncRecord ticks the unticked steps of every task of the record that a
// commit between the default branch and HEAD names with Record: <slug>. With
// check it writes nothing and reports each such task as plan-unsynced.
func SyncRecord(e Env, slug string, check bool) (SyncReport, error) {
	rep := SyncReport{Change: slug, Ticked: []TaskTicks{}}
	if _, ok := record.SlugOf(slug + ".md"); !ok {
		rep.Findings = []Finding{{Rule: "slug-invalid", Change: slug, Message: fmt.Sprintf("%q is not a record slug", slug)}}
		return rep, nil
	}
	rel := record.Dir + "/" + slug + ".md"
	if err := safepath.Check(e.Root, rel); err != nil {
		f, ferr := refuseLink(slug, rel, err)
		if f != nil {
			rep.Findings = []Finding{*f}
		}
		return rep, ferr
	}
	b, err := os.ReadFile(filepath.Join(e.Root, filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		rep.Findings = []Finding{{Rule: "record-missing", Change: slug, File: rel, Message: fmt.Sprintf("record %q does not exist under %s", slug, record.Dir)}}
		return rep, nil
	}
	if err != nil {
		return rep, err
	}
	all, err := e.Git.Commits(e.Default.Commit, "HEAD")
	if err != nil {
		return rep, err
	}
	named, unknown, _ := change.MatchTasks(record.Parse(b).Plan, record.CommitsFor(slug, all))
	for _, u := range unknown {
		rep.Findings = append(rep.Findings, Finding{Rule: "task-unknown", Change: slug, File: rel,
			Message: fmt.Sprintf("commit %s has trailer Task: %s, which is not a task of %s", shortSHA(u.Commit.SHA), u.Value, rel)})
	}
	var lines []int
	for _, nt := range named {
		var n int
		for _, s := range nt.Task.Steps {
			if !s.Done {
				lines = append(lines, s.Line)
				n++
			}
		}
		if n == 0 {
			continue
		}
		rep.Ticked = append(rep.Ticked, TaskTicks{Task: nt.Task.Number, Steps: n})
		if check {
			rep.Findings = append(rep.Findings, Finding{Rule: "plan-unsynced", Change: slug, File: rel, Line: nt.Task.Line,
				Message: fmt.Sprintf("Task %d has commits and %d unticked steps; run dkt sync %s", nt.Task.Number, n, slug)})
		}
	}
	if check || len(lines) == 0 {
		return rep, nil
	}
	out, err := change.Tick(b, lines)
	if err != nil {
		return rep, err
	}
	f, err := writeOrRefuse(e.Root, rel, slug, out)
	if err != nil {
		return rep, err
	}
	if f != nil {
		rep.Ticked = []TaskTicks{}
		rep.Findings = append(rep.Findings, *f)
	}
	return rep, nil
}
