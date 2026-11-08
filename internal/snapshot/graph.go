// Package snapshot reads the state of a project's work from git history.
package snapshot

import (
	"sort"
	"strings"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
)

// Graph is the commit graph of a history, held in memory.
type Graph struct {
	bySHA  map[string]gitx.Commit
	intro  map[string]introduced     // per tip, see introducers
	byWork map[Work][]string         // commits naming each work
	rel    map[string]map[string]int // per tag list, commit -> first tag
}

type introduced struct {
	line []string
	at   map[string]int
}

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
// the task numbers it lists. A malformed task list names the work with no
// task.
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
	line, intro := g.introducers(tip)
	best := -1
	for _, s := range g.named()[w] {
		if i, ok := intro[s]; ok && i > best {
			best = i
		}
	}
	if best < 0 {
		return nil
	}
	return ref(g.bySHA[line[best]])
}

// introducers lists the first-parent line of tip, oldest first, and maps each
// commit of its history to the index of the line commit that brought it in.
// It is computed once per tip.
func (g *Graph) introducers(tip string) ([]string, map[string]int) {
	if r, ok := g.intro[tip]; ok {
		return r.line, r.at
	}
	var line []string
	for sha := tip; ; {
		c, ok := g.bySHA[sha]
		if !ok {
			break
		}
		line = append(line, sha)
		if len(c.Parents) == 0 {
			break
		}
		sha = c.Parents[0]
	}
	for i, j := 0, len(line)-1; i < j; i, j = i+1, j-1 {
		line[i], line[j] = line[j], line[i]
	}
	at := map[string]int{}
	for i, l := range line {
		g.markFrom(l, at, i)
	}
	if g.intro == nil {
		g.intro = map[string]introduced{}
	}
	g.intro[tip] = introduced{line, at}
	return line, at
}

// markFrom sets mark[c] = v for sha and its ancestors not yet marked.
func (g *Graph) markFrom(sha string, mark map[string]int, v int) {
	stack := []string{sha}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		c, ok := g.bySHA[s]
		if _, done := mark[s]; !ok || done {
			continue
		}
		mark[s] = v
		stack = append(stack, c.Parents...)
	}
}

// named indexes, once, the commits whose trailers name each work.
func (g *Graph) named() map[Work][]string {
	if g.byWork == nil {
		g.byWork = map[Work][]string{}
		for sha, c := range g.bySHA {
			for _, t := range c.Trailers {
				slug, _, _ := strings.Cut(t.Value, "/")
				w := Work{t.Key, strings.TrimSpace(slug)}
				if l := g.byWork[w]; len(l) == 0 || l[len(l)-1] != sha {
					g.byWork[w] = append(l, sha)
				}
			}
		}
	}
	return g.byWork
}

// Release returns the first tag, in the given order, whose commit reaches
// commit; nil when none does.
func (g *Graph) Release(tags []gitx.Tag, commit string) *gitx.Tag {
	key := make([]string, len(tags))
	for i, t := range tags {
		key[i] = t.Commit
	}
	k := strings.Join(key, " ")
	at, ok := g.rel[k]
	if !ok {
		at = map[string]int{}
		for i, t := range tags {
			g.markFrom(t.Commit, at, i)
		}
		if g.rel == nil {
			g.rel = map[string]map[string]int{}
		}
		g.rel[k] = at
	}
	if i, ok := at[commit]; ok {
		return &tags[i]
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
