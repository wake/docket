package change

// Task is one `### Task N: <name>` section of plan.md. Line is 1-based.
type Task struct {
	Number int
	Name   string
	Line   int
	Steps  []Step
}

// Plan is what ticking needs from plan.md: its tasks in order, and the numbers
// listed under `## Cancelled Tasks`.
type Plan struct {
	Tasks     []Task
	Cancelled []int
}

func ParsePlan(b []byte) Plan { return Plan{} }
