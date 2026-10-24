package status

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
)

// planOf builds a plan from per-task step ticks: ticks[i] lists the steps of
// task i+1, true when ticked.
func planOf(ticks ...[]bool) *change.Plan {
	p := &change.Plan{}
	for i, steps := range ticks {
		t := change.Task{Number: i + 1, Name: fmt.Sprintf("Task name %d", i+1)}
		for j, done := range steps {
			t.Steps = append(t.Steps, change.Step{Line: 10*i + j + 1, Done: done})
		}
		p.Tasks = append(p.Tasks, t)
	}
	return p
}

func namedOf(p *change.Plan, n int, shas ...string) change.NamedTask {
	nt := change.NamedTask{Task: p.Tasks[n-1]}
	for _, s := range shas {
		nt.Commits = append(nt.Commits, gitx.Commit{SHA: s})
	}
	return nt
}

func states(ts []Task) string {
	var out []string
	for _, t := range ts {
		out = append(out, t.State)
	}
	return strings.Join(out, ",")
}

func TestPlanNotWritten(t *testing.T) {
	phase, tasks, prog := Summarize(nil, nil)
	if phase != Design || tasks == nil || len(tasks) != 0 || prog != (Progress{}) {
		t.Fatalf("got %q %#v %+v", phase, tasks, prog)
	}
}

func TestEmptyPlan(t *testing.T) {
	phase, tasks, _ := Summarize(&change.Plan{}, nil)
	if phase != Design || tasks == nil || len(tasks) != 0 {
		t.Fatalf("got %q %#v", phase, tasks)
	}
}

func TestNothingStarted(t *testing.T) {
	p := planOf([]bool{false, false}, []bool{false}, []bool{false, false, false})
	phase, tasks, prog := Summarize(p, nil)
	if phase != Ready || states(tasks) != "pending,pending,pending" {
		t.Fatalf("got %q %s", phase, states(tasks))
	}
	if prog != (Progress{TasksDone: 0, TasksTotal: 3, StepsTicked: 0, StepsTotal: 6}) {
		t.Fatalf("progress %+v", prog)
	}
	if tasks[0].Commits == nil || tasks[0].Name != "Task name 1" || tasks[0].Steps != 2 {
		t.Fatalf("task %+v", tasks[0])
	}
}

func TestCommitNamesATask(t *testing.T) {
	p := planOf([]bool{false}, []bool{false})
	phase, tasks, _ := Summarize(p, []change.NamedTask{namedOf(p, 1, "c1")})
	if phase != Implementing || tasks[0].State != Started || tasks[1].State != Pending {
		t.Fatalf("got %q %s", phase, states(tasks))
	}
	if !reflect.DeepEqual(tasks[0].Commits, []string{"c1"}) || !tasks[0].Unsynced {
		t.Fatalf("task %+v", tasks[0])
	}
}

func TestEveryTaskDone(t *testing.T) {
	p := planOf([]bool{true, true}, []bool{true})
	phase, tasks, prog := Summarize(p, nil)
	if phase != Done || states(tasks) != "done,done" {
		t.Fatalf("got %q %s", phase, states(tasks))
	}
	if prog != (Progress{TasksDone: 2, TasksTotal: 2, StepsTicked: 3, StepsTotal: 3}) {
		t.Fatalf("progress %+v", prog)
	}
}

func TestCancelledTaskLeftOut(t *testing.T) {
	p := planOf([]bool{true}, []bool{true})
	p.Cancelled = []change.Cancelled{{Number: 3, Name: "Dropped", Reason: "no need", OK: true}}
	phase, tasks, prog := Summarize(p, nil)
	if phase != Done || len(tasks) != 3 {
		t.Fatalf("got %q %s", phase, states(tasks))
	}
	last := tasks[2]
	if last.Number != 3 || last.Name != "Dropped" || last.State != Cancelled || last.Steps != 0 || last.Commits == nil {
		t.Fatalf("cancelled %+v", last)
	}
	if prog.TasksDone != 2 || prog.TasksTotal != 2 {
		t.Fatalf("progress %+v", prog)
	}
}

func TestCommitsNotYetSynced(t *testing.T) {
	p := planOf([]bool{true}, []bool{true, false})
	phase, tasks, _ := Summarize(p, []change.NamedTask{namedOf(p, 2, "c1", "c2")})
	if phase != Implementing {
		t.Fatalf("phase %q", phase)
	}
	t2 := tasks[1]
	if t2.State != Started || !t2.Unsynced || t2.Ticked != 1 || t2.Steps != 2 || len(t2.Commits) != 2 {
		t.Fatalf("task %+v", t2)
	}
}

func TestStepsTickedNoCommits(t *testing.T) {
	p := planOf([]bool{true, false})
	phase, tasks, _ := Summarize(p, nil)
	if phase != Implementing || tasks[0].State != Started || tasks[0].Unsynced {
		t.Fatalf("got %q %+v", phase, tasks[0])
	}
}

func TestTaskWithoutSteps(t *testing.T) {
	p := planOf([]bool{true}, nil)
	phase, tasks, _ := Summarize(p, nil)
	if phase != Implementing || states(tasks) != "done,pending" {
		t.Fatalf("got %q %s", phase, states(tasks))
	}
}

func TestDuplicatedHeadingListedOnce(t *testing.T) {
	p := planOf([]bool{true}, []bool{false})
	p.Tasks = append(p.Tasks, change.Task{Number: 1, Name: "Again", Steps: []change.Step{{Line: 99}}})
	_, tasks, prog := Summarize(p, nil)
	if len(tasks) != 2 || tasks[0].Name != "Task name 1" || prog.TasksTotal != 2 {
		t.Fatalf("got %+v %+v", tasks, prog)
	}
}

func TestCancelledEntryWithoutNumberLeftOut(t *testing.T) {
	p := planOf([]bool{true})
	p.Cancelled = []change.Cancelled{
		{Number: 3, Name: "Old", Reason: "dropped", OK: true},
		{Number: 4},
		{},
	}
	_, tasks, _ := Summarize(p, nil)
	if len(tasks) != 3 || tasks[1].Number != 3 || tasks[2].Number != 4 || states(tasks[1:]) != "cancelled,cancelled" {
		t.Fatalf("got %+v", tasks)
	}
}

func TestOnlyCancelledTasksIsDesign(t *testing.T) {
	p := &change.Plan{Cancelled: []change.Cancelled{{Number: 1, Name: "Old", Reason: "dropped", OK: true}}}
	phase, tasks, prog := Summarize(p, nil)
	if phase != Design || len(tasks) != 1 || tasks[0].State != Cancelled || prog != (Progress{}) {
		t.Fatalf("got %q %+v %+v", phase, tasks, prog)
	}
}
