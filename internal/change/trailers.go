package change

import (
	"regexp"
	"slices"
	"strconv"

	"github.com/wake/docket/internal/gitx"
)

var taskNumberRe = regexp.MustCompile(`^[1-9][0-9]*$`)

// NamedTask is a task of the plan and the commits that name it.
type NamedTask struct {
	Task    Task
	Commits []gitx.Commit // oldest first
}

// UnknownTrailer is a Task value on a commit for the change that is not a
// task heading of the plan.
type UnknownTrailer struct {
	Commit gitx.Commit
	Value  string
}

// NamedTasks reads the Task trailers of the commits that carry
// Change: <slug>. A value counts once per commit and names a task only when
// it is a plain positive number that matches a task heading; the first
// heading of a duplicated number wins. named lists, in plan order, the tasks
// some commit names; unknown lists the other values in commit order.
// anyTrailer is true when any commit for the change carries a Task trailer.
func NamedTasks(p Plan, slug string, commits []gitx.Commit) (named []NamedTask, unknown []UnknownTrailer, anyTrailer bool) {
	first := map[int]Task{}
	for _, t := range p.Tasks {
		if _, dup := first[t.Number]; !dup {
			first[t.Number] = t
		}
	}
	commitsOf := map[int][]gitx.Commit{}
	for _, cm := range commits {
		if !slices.Contains(cm.Change, slug) {
			continue
		}
		seen := map[string]bool{}
		for _, v := range cm.Task {
			anyTrailer = true
			if seen[v] {
				continue
			}
			seen[v] = true
			n := 0
			if taskNumberRe.MatchString(v) {
				n, _ = strconv.Atoi(v)
			}
			if _, ok := first[n]; !ok {
				unknown = append(unknown, UnknownTrailer{Commit: cm, Value: v})
				continue
			}
			commitsOf[n] = append(commitsOf[n], cm)
		}
	}
	for _, t := range p.Tasks {
		if cs, ok := commitsOf[t.Number]; ok {
			named = append(named, NamedTask{Task: first[t.Number], Commits: cs})
			delete(commitsOf, t.Number)
		}
	}
	return named, unknown, anyTrailer
}
