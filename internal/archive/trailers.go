package archive

import (
	"fmt"
	"strings"

	"github.com/wake/docket/internal/change"
)

// TrailerCheck says whether a change on the default branch had its commit
// trailers checked. Skipped holds the reason when it was not, nil when it was.
type TrailerCheck struct {
	Slug    string  `json:"slug"`
	Skipped *string `json:"skipped"`
}

// Trailers checks the changes in the default branch's tree the way Archive
// does: a change whose plan has tasks needs a commit since its base with a
// Change and a Task trailer. A change without a plan, or whose plan has no
// task, is not listed. A change whose base cannot be used is listed as
// skipped. It reads the default branch only, never the working tree.
func Trailers(e Env) ([]TrailerCheck, []Finding, error) {
	checks := []TrailerCheck{}
	var finds []Finding
	names, err := e.Git.ListDirs(e.Default.Commit, "openspec/changes")
	if err != nil {
		return nil, nil, err
	}
	for _, name := range names {
		if name == "archive" {
			continue
		}
		dir := "openspec/changes/" + name
		ob, hasOS, err := e.Git.ShowIfExists(e.Default.Commit, dir+"/"+change.OpenSpecFile)
		if err != nil {
			return nil, nil, err
		}
		if hasOS {
			schema, serr := change.ReadSchema(map[string][]byte{change.OpenSpecFile: ob})
			if serr == nil && schema != change.DocketSchema {
				reason := "made with the " + schema + " schema"
				checks = append(checks, TrailerCheck{Slug: name, Skipped: &reason})
				continue
			}
		}
		pb, ok, err := e.Git.ShowIfExists(e.Default.Commit, dir+"/plan.md")
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			continue
		}
		plan := change.ParsePlan(pb)
		if len(plan.Tasks) == 0 {
			continue
		}
		skip := func(format string, args ...any) {
			reason := fmt.Sprintf(format, args...)
			checks = append(checks, TrailerCheck{Slug: name, Skipped: &reason})
		}
		files := map[string][]byte{}
		for _, f := range []string{change.DocketFile, change.OpenSpecFile} {
			b, ok, err := e.Git.ShowIfExists(e.Default.Commit, dir+"/"+f)
			if err != nil {
				return nil, nil, err
			}
			if ok {
				files[f] = b
			}
		}
		m, merr := change.ReadMetadata(files)
		if merr != nil {
			skip("cannot read the base: %v", merr)
			continue
		}
		if m.Base == "" {
			skip("no base recorded")
			continue
		}
		isCommit := !strings.HasPrefix(m.Base, "-")
		if isCommit {
			if isCommit, err = e.Git.RefExists(m.Base + "^{commit}"); err != nil {
				return nil, nil, err
			}
		}
		if !isCommit {
			skip("the base %s is not a commit in this repository", m.Base)
			continue
		}
		anc, err := e.Git.IsAncestor(m.Base, e.Default.Commit)
		if err != nil {
			return nil, nil, err
		}
		if !anc {
			skip("the base %s is not an ancestor of %s", m.Base, e.Default.Name)
			continue
		}
		commits, err := e.Git.Commits(m.Base, e.Default.Commit)
		if err != nil {
			return nil, nil, err
		}
		checks = append(checks, TrailerCheck{Slug: name})
		if _, _, carried := change.NamedTasks(plan, name, commits); !carried {
			finds = append(finds, Finding{Rule: "no-task-commits", Change: name, File: dir + "/plan.md",
				Message: fmt.Sprintf("no commit on %s since the base carries Change: %s with a Task: trailer; was the pull request squashed? dkt archive will refuse this change", e.Default.Name, name)})
		}
	}
	return checks, finds, nil
}
