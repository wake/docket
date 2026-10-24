// Package status turns a parsed plan and the commits that name its tasks into
// the phase, task states and progress of a change.
package status

import "github.com/wake/docket/internal/change"

// Phases of a change.
const (
	Design       = "design"
	Ready        = "ready"
	Implementing = "implementing"
	Done         = "done"
)

// States of a task.
const (
	Pending   = "pending"
	Started   = "started"
	TaskDone  = "done"
	Cancelled = "cancelled"
)

// Task is one task of a change with its progress.
type Task struct {
	Number   int      `json:"number"`
	Name     string   `json:"name"`
	State    string   `json:"state"`
	Steps    int      `json:"steps"`
	Ticked   int      `json:"ticked"`
	Commits  []string `json:"commits"`
	Unsynced bool     `json:"unsynced"`
}

// Progress counts tasks and steps, leaving cancelled tasks out.
type Progress struct {
	TasksDone   int `json:"tasks_done"`
	TasksTotal  int `json:"tasks_total"`
	StepsTicked int `json:"steps_ticked"`
	StepsTotal  int `json:"steps_total"`
}

// Summarize gives the phase, the tasks (plan order, then cancelled) and the
// progress of a change. plan is nil when the change has no plan.md; named
// holds the tasks some commit names.
func Summarize(plan *change.Plan, named []change.NamedTask) (phase string, tasks []Task, progress Progress) {
	tasks = []Task{}
	if plan == nil {
		return Design, tasks, progress
	}
	commits := map[int][]string{}
	for _, n := range named {
		if _, ok := commits[n.Task.Number]; ok {
			continue
		}
		shas := make([]string, 0, len(n.Commits))
		for _, c := range n.Commits {
			shas = append(shas, c.SHA)
		}
		commits[n.Task.Number] = shas
	}
	seen := map[int]bool{}
	allDone, anyStarted := true, false
	for _, pt := range plan.Tasks {
		if seen[pt.Number] {
			continue
		}
		seen[pt.Number] = true
		t := Task{Number: pt.Number, Name: pt.Name, Steps: len(pt.Steps), Commits: []string{}}
		for _, s := range pt.Steps {
			if s.Done {
				t.Ticked++
			}
		}
		shas, isNamed := commits[pt.Number]
		t.Commits = append(t.Commits, shas...)
		switch {
		case t.Steps > 0 && t.Ticked == t.Steps:
			t.State = TaskDone
		case t.Ticked > 0 || isNamed:
			t.State = Started
		default:
			t.State = Pending
		}
		t.Unsynced = len(t.Commits) > 0 && t.Ticked < t.Steps
		if t.State == TaskDone {
			progress.TasksDone++
		} else {
			allDone = false
		}
		if t.State != Pending {
			anyStarted = true
		}
		progress.TasksTotal++
		progress.StepsTicked += t.Ticked
		progress.StepsTotal += t.Steps
		tasks = append(tasks, t)
	}
	for _, c := range plan.Cancelled {
		if c.Number == 0 {
			continue // a bullet that names no task is a note, not a task
		}
		tasks = append(tasks, Task{Number: c.Number, Name: c.Name, State: Cancelled, Commits: []string{}})
	}
	switch {
	case len(plan.Tasks) == 0:
		phase = Design
	case allDone:
		phase = Done
	case anyStarted:
		phase = Implementing
	default:
		phase = Ready
	}
	return phase, tasks, progress
}
