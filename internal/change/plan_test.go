package change

import (
	"reflect"
	"strings"
	"testing"
)

func TestParsePlan(t *testing.T) {
	src := strings.Join([]string{
		"# Plan",               // 1
		"",                     // 2
		"### Task 1: A ",       // 3
		"- [ ] one",            // 4
		"- [x] two",            // 5
		"- [X] three",          // 6
		"* [ ] four",           // 7
		"#### Notes",           // 8
		"  - [ ] nested",       // 9
		"```",                  // 10
		"- [ ] fenced",         // 11
		"```",                  // 12
		"<!--",                 // 13
		"- [ ] commented",      // 14
		"-->",                  // 15
		"### Task 2 - x",       // 16
		"- [ ] not in a task",  // 17
		"### Task 3: B",        // 18
		"- [ ] b1",             // 19
		"## Review Focus",      // 20
		"- [ ] focus",          // 21
		"## Cancelled Tasks",   // 22
		"",                     // 23
		"- Task 2: x - why",    // 24
		"* Task 7",             // 25
		"  - Task 8: indented", // 26
		"- other",              // 27
		"```",                  // 28
		"- Task 9: fenced",     // 29
		"```",                  // 30
		"",
	}, "\n")
	want := []Task{
		{Number: 1, Name: "A", Line: 3, Steps: []Step{{4, false}, {5, true}, {6, true}, {7, false}, {9, false}}},
		{Number: 3, Name: "B", Line: 18, Steps: []Step{{19, false}}},
	}
	for _, s := range []string{src, "\xef\xbb\xbf" + strings.ReplaceAll(src, "\n", "\r\n")} {
		p := ParsePlan([]byte(s))
		if !reflect.DeepEqual(p.Tasks, want) {
			t.Fatalf("tasks = %+v\nwant %+v", p.Tasks, want)
		}
		if !reflect.DeepEqual(p.Cancelled, []int{2, 7}) {
			t.Fatalf("cancelled = %v", p.Cancelled)
		}
	}
}

func TestParsePlanTaskHeadingLevel(t *testing.T) {
	p := ParsePlan([]byte("## Task 1: a\n- [ ] x\n#### Task 2: b\n- [ ] y\n### Task 3: c\n- [ ] z\n"))
	want := []Task{{Number: 3, Name: "c", Line: 5, Steps: []Step{{6, false}}}}
	if !reflect.DeepEqual(p.Tasks, want) {
		t.Fatalf("tasks = %+v, want %+v", p.Tasks, want)
	}
}

func TestParsePlanCancelledHeadingCase(t *testing.T) {
	p := ParsePlan([]byte("## cancelled tasks\n\n- Task 4: x - dropped\n"))
	if !reflect.DeepEqual(p.Cancelled, []int{4}) {
		t.Fatalf("cancelled = %v", p.Cancelled)
	}
}

func TestParsePlanBodyEndsAtLevelThree(t *testing.T) {
	p := ParsePlan([]byte("### Task 1: A\n- [ ] a\n### Task 2: B\n- [ ] b\n"))
	if len(p.Tasks) != 2 || len(p.Tasks[0].Steps) != 1 || p.Tasks[0].Steps[0].Line != 2 {
		t.Fatalf("tasks = %+v", p.Tasks)
	}
	p = ParsePlan([]byte("## Intro\n- [ ] x\n### Task 1: A\n"))
	if len(p.Tasks) != 1 || len(p.Tasks[0].Steps) != 0 {
		t.Fatalf("tasks = %+v", p.Tasks)
	}
}

func TestParsePlanHeadingsInComments(t *testing.T) {
	// multi-line and one-line comment headings are not tasks
	p := ParsePlan([]byte("<!--\n### Task 1: hidden\n-->\n- [ ] unrelated\n<!-- ### Task 2: x -->\n- [ ] more\n"))
	if len(p.Tasks) != 0 {
		t.Fatalf("%+v", p.Tasks)
	}
	// a heading in a comment inside a task does not end its body
	p = ParsePlan([]byte("### Task 1: A\n- [ ] a\n<!--\n### Task 2: x\n-->\n- [ ] b\n"))
	if len(p.Tasks) != 1 || len(p.Tasks[0].Steps) != 2 {
		t.Fatalf("%+v", p.Tasks)
	}
	// Cancelled Tasks inside a comment gives nothing
	p = ParsePlan([]byte("<!--\n## Cancelled Tasks\n- Task 3: gone\n-->\n"))
	if len(p.Cancelled) != 0 {
		t.Fatalf("%v", p.Cancelled)
	}
}
