// Package snapshot reads the state of a project's work from git history.
package snapshot

import (
	"sort"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
)

// Graph is the commit graph of a history, held in memory.
type Graph struct{ bySHA map[string]gitx.Commit }

// NewGraph indexes commits by SHA.
func NewGraph(commits []gitx.Commit) *Graph {
	g := &Graph{bySHA: make(map[string]gitx.Commit, len(commits))}
	for _, c := range commits {
		g.bySHA[c.SHA] = c
	}
	return g
}

// Reachable returns the commits reachable from the given ones, themselves
// included. A SHA the graph does not hold adds nothing.
func (g *Graph) Reachable(from ...string) map[string]bool {
	seen := map[string]bool{}
	stack := append([]string(nil), from...)
	for len(stack) > 0 {
		sha := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		c, ok := g.bySHA[sha]
		if !ok || seen[sha] {
			continue
		}
		seen[sha] = true
		stack = append(stack, c.Parents...)
	}
	return seen
}

// Work is a Change or a Record, by Key, and its slug.
type Work struct{ Key, Slug string }

// Names reports whether a trailer of c names w, with or without tasks, and
// the task numbers it lists.
func Names(c gitx.Commit, w Work) (named bool, tasks []int) {
	for _, t := range c.Trailers {
		if t.Key != w.Key {
			continue
		}
		ts, _, mine := change.TaskRefs(t.Value, w.Slug)
		if mine {
			named = true
			tasks = append(tasks, ts...)
		}
	}
	return named, tasks
}

// Ref is a commit with its time and subject.
type Ref struct{ Commit, Time, Subject string }

func ref(c gitx.Commit) *Ref { return &Ref{c.SHA, c.Time, c.Subject} }

// Merge returns the newest commit on the first-parent line of tip that
// brought in a commit naming w: a merge commit, or the commit itself on the
// line. It is nil when none did.
func (g *Graph) Merge(tip string, w Work) *Ref {
	for sha := tip; ; {
		c, ok := g.bySHA[sha]
		if !ok {
			return nil
		}
		if named, _ := Names(c, w); named {
			return ref(c)
		}
		if len(c.Parents) == 0 {
			return nil
		}
		if len(c.Parents) > 1 {
			in := g.Reachable(c.Parents[1:]...)
			for s := range g.Reachable(c.Parents[0]) {
				delete(in, s)
			}
			for s := range in {
				if named, _ := Names(g.bySHA[s], w); named {
					return ref(c)
				}
			}
		}
		sha = c.Parents[0]
	}
}

// Release returns the first tag, in the given order, whose commit reaches
// commit; nil when none does.
func (g *Graph) Release(tags []gitx.Tag, commit string) *gitx.Tag {
	for i := range tags {
		if g.Reachable(tags[i].Commit)[commit] {
			return &tags[i]
		}
	}
	return nil
}

// Last returns, among the commits reachable from from that name w, the newest
// by Time, ties broken by SHA (the larger wins); nil when none.
func (g *Graph) Last(from []string, w Work) *Ref {
	var best *gitx.Commit
	for s := range g.Reachable(from...) {
		c := g.bySHA[s]
		if named, _ := Names(c, w); !named {
			continue
		}
		if best == nil || c.Time > best.Time || c.Time == best.Time && c.SHA > best.SHA {
			best = &c
		}
	}
	if best == nil {
		return nil
	}
	return ref(*best)
}

// TaskCommits maps each task number to the commits reachable from from that
// name it, oldest first.
func (g *Graph) TaskCommits(from string, w Work) map[int][]string {
	var cs []gitx.Commit
	for s := range g.Reachable(from) {
		cs = append(cs, g.bySHA[s])
	}
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].Time != cs[j].Time {
			return cs[i].Time < cs[j].Time
		}
		return cs[i].SHA < cs[j].SHA
	})
	res := map[int][]string{}
	for _, c := range cs {
		_, tasks := Names(c, w)
		seen := map[int]bool{}
		for _, n := range tasks {
			if !seen[n] {
				seen[n] = true
				res[n] = append(res[n], c.SHA)
			}
		}
	}
	return res
}
