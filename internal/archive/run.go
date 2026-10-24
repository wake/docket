package archive

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/lint"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/safepath"
)

// Runner is the part of OpenSpec that Archive drives; *openspec.Runner
// satisfies it.
type Runner interface {
	CheckVersion() (string, error)
	Run(args ...string) (string, error)
}

// Input says what to archive. Version is Docket's version and Origin the
// origin URL, "" when there is none.
type Input struct {
	Env
	OpenSpec Runner
	Lint     func(slug string) lint.Report
	Version  string
	Origin   string
	Slugs    []string
}

// Archived is a change that OpenSpec archived; Folder and Evidence are
// repository-relative, and Evidence is "" when writing it failed.
type Archived struct {
	Change   string `json:"change"`
	Folder   string `json:"folder"`
	Evidence string `json:"evidence"`
}

// Stop says where a run stopped; Change is "" for preconditions.
type Stop struct {
	Change string `json:"change"`
	Step   string `json:"step"`
}

// ArchiveReport is the outcome of Archive.
type ArchiveReport struct {
	Archived   []Archived
	Stopped    *Stop
	Drifts     []Drift
	Readings   *Readings
	Findings   []Finding
	Unreadable []Unreadable
}

type archiveRun struct {
	in      Input
	rep     *ArchiveReport
	head    string
	version string
}

// Archive checks and archives the changes one after another, stopping at
// the first step of the first change that has a finding or an unreadable
// source. The error is for OpenSpec being unavailable and for failures not
// tied to a change.
func Archive(in Input) (ArchiveReport, error) {
	rep := ArchiveReport{Archived: []Archived{}, Drifts: []Drift{}, Findings: []Finding{}, Unreadable: []Unreadable{}}
	ver, err := in.OpenSpec.CheckVersion()
	if err != nil {
		return rep, err
	}
	r := &archiveRun{in: in, rep: &rep, version: ver}
	if err := r.preconditions(); err != nil || rep.Stopped != nil {
		return rep, err
	}
	for _, slug := range in.Slugs {
		if stopped, err := r.change(slug); err != nil || stopped {
			return rep, err
		}
	}
	return rep, nil
}

// stop records the findings and unreadable sources of a step; it reports
// whether there were any.
func (r *archiveRun) stop(slug, step string, finds []Finding, unread []Unreadable) bool {
	if len(finds) == 0 && len(unread) == 0 {
		return false
	}
	r.rep.Findings = append(r.rep.Findings, finds...)
	r.rep.Unreadable = append(r.rep.Unreadable, unread...)
	r.rep.Stopped = &Stop{Change: slug, Step: step}
	return true
}

func (r *archiveRun) preconditions() error {
	e := r.in.Env
	var finds []Finding
	add := func(rule, slug, format string, args ...any) {
		finds = append(finds, Finding{Rule: rule, Change: slug, Message: fmt.Sprintf(format, args...)})
	}
	head, done, err := headOrNoCommits(e, "", &finds)
	if err != nil {
		return err
	}
	if done {
		r.stop("", "preconditions", finds, nil)
		return nil
	}
	r.head = head
	dirty, err := e.Git.Uncommitted()
	if err != nil {
		return err
	}
	if len(dirty) > 0 {
		shown := dirty
		if len(shown) > 5 {
			shown = shown[:5]
		}
		more := ""
		if len(dirty) > len(shown) {
			more = fmt.Sprintf(" and %d more", len(dirty)-len(shown))
		}
		add("uncommitted-changes", "", "the working tree has uncommitted changes (%s%s); commit or stash them", strings.Join(shown, ", "), more)
	}
	behind, err := e.Git.IsAncestor(e.Default.Commit, head)
	if err != nil {
		return err
	}
	if !behind {
		add("behind-default-branch", "", "HEAD does not contain %s; merge %s first", e.Default.Name, e.Default.Name)
	}
	merged, err := e.Git.ListDirs(e.Default.Commit, "openspec/changes")
	if err != nil {
		return err
	}
	for _, slug := range r.in.Slugs {
		found := false
		for _, m := range merged {
			found = found || m == slug
		}
		if !found {
			add("change-not-merged", slug, "change %q is not on %s yet; merge its pull request first", slug, e.Default.Name)
		}
	}
	r.stop("", "preconditions", finds, nil)
	return nil
}

// change runs the steps for one change; stopped says a step had findings.
func (r *archiveRun) change(slug string) (stopped bool, err error) {
	e := r.in.Env
	c, finds, unread := loadChange(e, slug, loadOpts{Plan: true, BaseIsCommit: true, Deltas: true})
	if c == nil {
		return r.stop(slug, "plan", finds, unread), nil
	}
	commits, err := e.Git.Commits(c.Meta.Base, r.head)
	if err != nil {
		return false, err
	}
	res := computeSync(c, commits)
	if r.stop(slug, "plan", planFindings(c, res), nil) {
		return true, nil
	}

	lr := r.in.Lint(slug)
	for _, f := range lr.Findings {
		finds = append(finds, Finding(f))
	}
	for _, u := range lr.Unreadable {
		unread = append(unread, Unreadable(u))
	}
	if r.stop(slug, "lint", finds, unread) {
		return true, nil
	}

	drifts, finds, unread, checked := checkBase(e, c)
	r.rep.Drifts = append(r.rep.Drifts, drifts...)
	if r.stop(slug, "base", finds, unread) {
		return true, nil
	}

	out, err := r.in.OpenSpec.Run("validate", slug, "--strict")
	if f, err := r.failed(slug, err, out); err != nil || f != nil {
		return r.stop(slug, "validate", wrap(f), nil), err
	}
	validated := firstLine(out)

	show, err := r.in.OpenSpec.Run("show", slug, "--type", "change", "--json", "--deltas-only")
	if f, err := r.failed(slug, err, show); err != nil || f != nil {
		return r.stop(slug, "cross-check", wrap(f), nil), err
	}
	rd, same, cerr := CrossCheck(c.Deltas, show)
	if cerr != nil || !same {
		msg := "Docket and OpenSpec read the delta specs differently"
		if cerr != nil {
			msg = cerr.Error()
		}
		r.rep.Readings = &rd
		return r.stop(slug, "cross-check", []Finding{{Rule: "readings-disagree", Change: slug, Message: msg}}, nil), nil
	}

	folder, as, finds, err := r.archive(c)
	if err != nil || r.stop(slug, "archive", finds, nil) {
		return err == nil, err
	}
	a := Archived{Change: slug, Folder: folder}
	evPath := path.Join(folder, "evidence.md")
	ev := evidence{Slug: slug, Date: as[:10], Docket: r.in.Version, OpenSpec: r.version, Origin: r.in.Origin, Head: r.head,
		Meta: c.Meta, Tasks: c.Plan.Tasks, Commits: map[int][]gitx.Commit{}, Validate: validated, Checked: checked}
	for _, w := range res.Tasks {
		ev.Commits[w.Task.Number] = w.Commits
	}
	f, err := refuseLink(slug, evPath, createFile(e.Root, evPath, []byte(ev.render())))
	if f == nil && err != nil {
		f = &Finding{Rule: "evidence-failed", Change: slug, File: evPath, Message: "cannot write evidence.md: " + err.Error()}
	}
	if f == nil {
		a.Evidence = evPath
	}
	r.rep.Archived = append(r.rep.Archived, a)
	return r.stop(slug, "evidence", wrap(f), nil), nil
}

func wrap(f *Finding) []Finding {
	if f == nil {
		return nil
	}
	return []Finding{*f}
}

// failed turns an OpenSpec exit failure into an openspec-failed finding;
// any other error is returned as is.
func (r *archiveRun) failed(slug string, err error, stdout string) (*Finding, error) {
	var fe *openspec.FailedError
	switch {
	case err == nil:
		return nil, nil
	case errors.As(err, &fe):
		return &Finding{Rule: "openspec-failed", Change: slug,
			Message: fmt.Sprintf("`openspec %s` exited with %d\nstdout: %s\nstderr: %s", strings.Join(fe.Command, " "), fe.Code, strings.TrimSpace(stdout), strings.TrimSpace(fe.Stderr))}, nil
	}
	return nil, err
}

func planFindings(c *loadedChange, res syncResult) []Finding {
	file := c.path("plan.md")
	finds := append([]Finding(nil), res.Findings...)
	named := map[int]bool{}
	for _, w := range res.Tasks {
		named[w.Task.Number] = true
		if len(w.Lines) > 0 {
			finds = append(finds, Finding{Rule: "plan-unsynced", Change: c.Slug, File: file, Line: w.Task.Line,
				Message: fmt.Sprintf("Task %d has commits and %d unticked steps; run dkt sync %s", w.Task.Number, len(w.Lines), c.Slug)})
		}
	}
	for _, t := range c.Plan.Tasks {
		if named[t.Number] {
			continue
		}
		for _, s := range t.Steps {
			if !s.Done {
				finds = append(finds, Finding{Rule: "plan-unticked", Change: c.Slug, File: file, Line: t.Line,
					Message: fmt.Sprintf("Task %d: %s has unticked steps and no commit names it", t.Number, t.Name)})
				break
			}
		}
	}
	if !res.AnyTrailer {
		finds = append(finds, Finding{Rule: "no-task-commits", Change: c.Slug, File: file,
			Message: "no commit since x-base carries Change: " + c.Slug + " with a Task: trailer; was the pull request squashed?"})
	}
	return finds
}

// archive refuses links on the paths OpenSpec writes, runs `openspec
// archive` and returns the archived folder and its archivedAs name.
func (r *archiveRun) archive(c *loadedChange) (folder, as string, finds []Finding, err error) {
	e := r.in.Env
	paths := []string{c.Dir, "openspec/changes/archive"}
	for _, capability := range sortedCaps(c) {
		paths = append(paths, mainSpecPath(capability))
	}
	for _, p := range paths {
		if f, err := refuseLink(c.Slug, p, safepath.Check(e.Root, p)); f != nil || err != nil {
			return "", "", wrap(f), err
		}
	}
	out, err := r.in.OpenSpec.Run("archive", c.Slug, "-y", "--json")
	if f, err := r.failed(c.Slug, err, out); f != nil || err != nil {
		return "", "", wrap(f), err
	}
	var res struct {
		Archive struct {
			ArchivedAs string `json:"archivedAs"`
		} `json:"archive"`
	}
	bad := json.Unmarshal([]byte(out), &res) != nil
	as = res.Archive.ArchivedAs
	if bad || !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`+regexp.QuoteMeta(c.Slug)+`$`).MatchString(as) {
		return "", "", r.movedUnexpectedly(c.Slug, as, "reported the archive name "+strconv.Quote(as)), nil
	}
	folder = "openspec/changes/archive/" + as
	if !isDir(e.Root, folder) {
		return "", "", r.movedUnexpectedly(c.Slug, as, "reported "+strconv.Quote(as)+" but that folder does not exist"), nil
	}
	return folder, as, nil, nil
}

// movedUnexpectedly records a change that OpenSpec archived although what it
// reported cannot be used: the change appears in the report, with a folder
// only when the reported name is a local directory under the archive, and
// the returned finding says that no evidence was written.
func (r *archiveRun) movedUnexpectedly(slug, as, what string) []Finding {
	a := Archived{Change: slug}
	if as != "" && as != "." && as != ".." && filepath.Base(as) == as {
		if folder := "openspec/changes/archive/" + as; isDir(r.in.Env.Root, folder) {
			a.Folder = folder
		}
	}
	r.rep.Archived = append(r.rep.Archived, a)
	return []Finding{{Rule: "evidence-failed", Change: slug, File: a.Folder,
		Message: fmt.Sprintf("OpenSpec archived the change, but `openspec archive` %s; evidence.md was not written", what)}}
}

func isDir(root, rel string) bool {
	fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil && fi.IsDir()
}
