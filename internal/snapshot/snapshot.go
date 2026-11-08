package snapshot

import (
	"path"
	"slices"
	"strings"

	"github.com/wake/docket/internal/domain"
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/idea"
	"github.com/wake/docket/internal/scan"
)

const (
	remotePrefix = "refs/remotes/"
	headsPrefix  = "refs/heads/"
	ideasDir     = "docket/ideas"
	zeroSHA      = "0000000000000000000000000000000000000000"
)

// Collect reads the project once and puts the result together. The problems
// are those of the order, ideas, issues and domain tree on the tip; an
// unreadable source is listed once and the rest is still reported. The error
// is for git failing as a whole.
func Collect(g gitx.Git) (Result, []idea.Problem, []scan.Unreadable, error) {
	var res Result
	var unreadable []scan.Unreadable
	ws, err := scan.Scan(g, "openspec/specs", domain.File, idea.OrderFile, ideasDir)
	if err != nil {
		return res, nil, nil, err
	}
	invs := [3]scan.Inventory{ws.Changes, ws.Records, ws.Issues}
	for _, inv := range invs {
		unreadable = append(unreadable, inv.Unreadable...)
	}
	wts, refs := ws.Worktrees, ws.Refs
	def, hasDef, err := g.DefaultBranchIn(refs)
	if err != nil {
		return res, nil, nil, err
	}
	tags, err := g.Tags()
	if err != nil {
		return res, nil, nil, err
	}

	// default_branch.ref: the remote-tracking ref first, as the tip is.
	defRef := ""
	if hasDef {
		defRef = remotePrefix + "origin/" + def.Name
		if !slices.ContainsFunc(refs, func(r gitx.Ref) bool { return r.Name == defRef }) {
			defRef = headsPrefix + def.Name
		}
	}

	// A ref the scan could not read stays out of the history, which a git log
	// of a non-commit would fail as a whole.
	bad := map[string]bool{}
	for _, u := range unreadable {
		bad[u.Source] = true
	}
	places := map[string]Place{}
	var revs []string
	addRev := func(c string) {
		if c != "" && !slices.Contains(revs, c) {
			revs = append(revs, c)
		}
	}
	for _, w := range wts {
		if w.Bare {
			continue
		}
		c := w.Head
		if c == zeroSHA {
			c = ""
		}
		places[w.Path] = Place{Kind: "worktree", Branch: strings.TrimPrefix(w.Branch, headsPrefix), Commit: c}
		addRev(c)
	}
	for _, r := range refs {
		p := Place{Kind: "branch", Branch: strings.TrimPrefix(r.Name, headsPrefix), Commit: r.Commit, Default: r.Name == defRef}
		if strings.HasPrefix(r.Name, remotePrefix) {
			p.Kind, p.Branch = "remote-branch", strings.TrimPrefix(r.Name, remotePrefix)
		}
		places[r.Name] = p
		if !bad[r.Name] {
			addRev(r.Commit)
		}
	}
	if hasDef && !bad[defRef] {
		addRev(def.Commit)
	}
	for _, t := range tags {
		addRev(t.Commit)
	}
	commits, err := g.History(revs...)
	if err != nil {
		return res, nil, nil, err
	}
	graph := NewGraph(commits)
	for s, p := range places {
		if c, ok := graph.bySHA[p.Commit]; ok {
			p.Time = c.Time
			places[s] = p
		}
	}

	// One Wanted per use, so that every copy of a work is read.
	var want []Wanted
	for _, inv := range invs {
		for _, u := range inv.Uses {
			want = append(want, Wanted{Source: u.Source, Commit: u.Commit, Paths: []string{u.Path}})
		}
	}
	// The tip is read only when its ref could be listed; its files are wanted
	// one by one, so that a link costs only itself.
	tipIdeas := map[string]bool{}
	var dates map[string]string
	if hasDef && !bad[defRef] {
		var paths []string
		for _, f := range ws.Trees[def.Commit] {
			switch {
			case f.Link && (f.Path == ideasDir || f.Path == specsDir[:len(specsDir)-1] || strings.HasPrefix(f.Path, specsDir)):
				// a linked folder is listed by Read as unreadable, never read through
			case f.Path == domain.File || f.Path == idea.OrderFile:
			case path.Dir(f.Path) == ideasDir && strings.HasSuffix(f.Path, ".md"):
				tipIdeas[f.Path] = true
			case strings.HasPrefix(f.Path, specsDir) && strings.HasSuffix(f.Path, "/spec.md"):
			default:
				continue
			}
			paths = append(paths, f.Path)
		}
		want = append(want, Wanted{Source: defRef, Commit: def.Commit, Paths: paths})
		all, err := g.AddedDates(def.Commit, ideasDir)
		if err != nil {
			return res, nil, nil, err
		}
		dates = map[string]string{} // only the ideas the tip still holds
		for p, d := range all {
			if tipIdeas[p] {
				dates[p] = d
			}
		}
	}
	files, readBad, err := Read(g, want, ws.Trees)
	if err != nil {
		return res, nil, nil, err
	}
	unreadable = append(unreadable, readBad...)

	works, worksBad := Works(WorksInput{Changes: invs[0].Uses, Records: invs[1].Uses, Places: places, Files: files,
		Graph: graph, Tip: def.Commit, Tags: tags, Dates: dates})
	unreadable = append(unreadable, worksBad...)

	tip := files[defRef] // nil without a default branch
	ideas := map[string][]byte{}
	for p := range tipIdeas {
		if b, ok := tip[p]; ok {
			ideas[p] = b
		}
	}
	ideaEntries, issues, order, problems := Backlog(BacklogInput{Order: tip[idea.OrderFile], Ideas: ideas,
		Issues: invs[2].Uses, IssueFiles: files, Dates: dates, Works: works, Places: places, DefaultRef: defRef})
	res.Works, res.Ideas, res.Issues, res.Order = works, ideaEntries, issues, order

	res.Specs = Specs(tip)
	var caps []string
	for _, s := range res.Specs {
		caps = append(caps, s.Capability)
	}
	for _, w := range works {
		if w.Archived == nil {
			for _, d := range w.Deltas {
				caps = append(caps, d.Capability)
			}
		}
	}
	tree, found := tip[domain.File]
	var dprobs []domain.Problem
	res.Domains, dprobs = DomainTree(tree, found, caps)
	for _, p := range dprobs {
		problems = append(problems, idea.Problem{Rule: p.Rule, File: domain.File, Line: p.Line, Message: p.Message})
	}

	if hasDef {
		b := &Branch{Name: def.Name, Ref: defRef, Commit: def.Commit}
		if c, ok := graph.bySHA[def.Commit]; ok {
			b.Time, b.Subject = c.Time, c.Subject
		}
		res.DefaultBranch = b
	}

	var merged []scan.Unreadable
	seen := map[string]bool{}
	for _, u := range unreadable {
		if !seen[u.Source] {
			seen[u.Source] = true
			merged = append(merged, u)
		}
	}
	return res, problems, merged, nil
}
