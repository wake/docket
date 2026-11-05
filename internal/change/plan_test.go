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
		wantC := []Cancelled{
			{Number: 2, Name: "x", Reason: "why", Line: 24, OK: true},
			{Number: 7, Line: 25},
			{Line: 27},
		}
		if !reflect.DeepEqual(p.Cancelled, wantC) {
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
	if want := []Cancelled{{Number: 4, Name: "x", Reason: "dropped", Line: 3, OK: true}}; !reflect.DeepEqual(p.Cancelled, want) {
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

func TestParsePlanScenarios(t *testing.T) {
	cases := []struct {
		line string
		want []string
	}{
		{"**Scenarios:** `a`, b ,`c d`", []string{"a", "b", "c d"}},
		{"**Scenarios:** none", nil},
		{"**Scenarios:**", []string{""}},
	}
	for _, c := range cases {
		p := ParsePlan([]byte("### Task 1: A\n\n" + c.line + "\n"))
		if p.Tasks[0].ScenariosLine != 3 || !reflect.DeepEqual(p.Tasks[0].Scenarios, c.want) {
			t.Errorf("%q: %+v", c.line, p.Tasks[0])
		}
	}
	p := ParsePlan([]byte("### Task 1: A\n- [ ] s\n"))
	if p.Tasks[0].ScenariosLine != 0 || p.Tasks[0].Scenarios != nil {
		t.Errorf("%+v", p.Tasks[0])
	}
}

func TestParsePlanCompensates(t *testing.T) {
	p := ParsePlan([]byte("### Task 1: A\n**Compensates:** Task 3\n**Compensates:** the third\n"))
	want := []Ref{{Number: 3, Line: 2, OK: true}, {Line: 3}}
	if !reflect.DeepEqual(p.Tasks[0].Compensates, want) {
		t.Fatalf("%+v", p.Tasks[0].Compensates)
	}
}

func TestParsePlanBadHeadings(t *testing.T) {
	p := ParsePlan([]byte("## Task 3: x\n#### Task 4: y\n### Task 0: z\n### Task 2 - x\n### Tasks left\n### Task 5: ok\n"))
	want := []BadHeading{
		{"Task 3: x", 2, 1}, {"Task 4: y", 4, 2}, {"Task 0: z", 3, 3}, {"Task 2 - x", 3, 4},
	}
	if !reflect.DeepEqual(p.BadHeadings, want) {
		t.Fatalf("bad = %+v", p.BadHeadings)
	}
	if len(p.Tasks) != 1 || p.Tasks[0].Number != 5 {
		t.Fatalf("tasks = %+v", p.Tasks)
	}
}

func TestParsePlanCancelledBullets(t *testing.T) {
	p := ParsePlan([]byte("## Cancelled Tasks\n- Task 5: Name - reason\n- Task 5: Name\n"))
	want := []Cancelled{
		{Number: 5, Name: "Name", Reason: "reason", Line: 2, OK: true},
		{Number: 5, Line: 3},
	}
	if !reflect.DeepEqual(p.Cancelled, want) {
		t.Fatalf("%+v", p.Cancelled)
	}
}

func TestParsePlanReuse(t *testing.T) {
	p := ParsePlan([]byte("# Plan\n\n## reuse\n\n| A | B |\n|---|---|\n| 1 | 2 |\n"))
	if p.ReuseLine != 3 || p.Reuse == nil || !reflect.DeepEqual(p.Reuse.Header, []string{"A", "B"}) ||
		len(p.Reuse.Rows) != 1 || p.Reuse.Rows[0].Index != 6 {
		t.Fatalf("%d %+v", p.ReuseLine, p.Reuse)
	}
	p = ParsePlan([]byte("## Reuse\n\nnothing\n"))
	if p.ReuseLine != 1 || p.Reuse != nil {
		t.Fatalf("%d %+v", p.ReuseLine, p.Reuse)
	}
	p = ParsePlan([]byte("# Plan\n"))
	if p.ReuseLine != 0 || p.Reuse != nil {
		t.Fatalf("%d %+v", p.ReuseLine, p.Reuse)
	}
}

func TestParsePlanHiddenStructure(t *testing.T) {
	src := "### Task 1: A\n<!--\n**Scenarios:** x\n**Compensates:** Task 2\n-->\n```\n**Scenarios:** y\n**Compensates:** Task 3\n```\n" +
		"<!--\n## Task 9: h\n## Reuse\n-->\n```\n## Reuse\n## Task 8: f\n- Task 7: a - b\n```\n<!-- ## Cancelled Tasks -->\n- Task 6: c - d\n"
	p := ParsePlan([]byte(src))
	if t0 := p.Tasks[0]; t0.ScenariosLine != 0 || len(t0.Compensates) != 0 {
		t.Fatalf("%+v", t0)
	}
	if len(p.BadHeadings) != 0 || p.ReuseLine != 0 || len(p.Cancelled) != 0 {
		t.Fatalf("%+v", p)
	}
}
