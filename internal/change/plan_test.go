package change

import (
	"reflect"
	"strings"
	"testing"
)

func TestParsePlanTasks(t *testing.T) {
	src := "# Plan\n\n### Task 1: Parser\n\n**Scenarios:** `a#b`, `c#d`\n\n### Task 2: Docs\n\n**Scenarios:** x#y,z#w\n\n### Task 3: Misc\n\n**Scenarios:** none\n\n### Task 4: Bare\n\ntext\n"
	p := ParsePlan([]byte(src))
	if len(p.Tasks) != 4 {
		t.Fatalf("tasks = %+v", p.Tasks)
	}
	t1 := p.Tasks[0]
	if t1.Number != 1 || t1.Name != "Parser" || t1.Line != 3 || t1.ScenariosLine != 5 || !reflect.DeepEqual(t1.Scenarios, []string{"a#b", "c#d"}) {
		t.Fatalf("t1 = %+v", t1)
	}
	if !reflect.DeepEqual(p.Tasks[1].Scenarios, []string{"x#y", "z#w"}) {
		t.Fatalf("t2 = %+v", p.Tasks[1])
	}
	if t3 := p.Tasks[2]; len(t3.Scenarios) != 0 || t3.ScenariosLine != 13 {
		t.Fatalf("t3 = %+v", t3)
	}
	if t4 := p.Tasks[3]; t4.ScenariosLine != 0 || t4.Scenarios != nil {
		t.Fatalf("t4 = %+v", t4)
	}
}

func TestBadHeadings(t *testing.T) {
	p := ParsePlan([]byte("## Task 3: Parser\n\n### Task 0: x\n\n### Task 2 Parser\n\n### Task 1: ok\n"))
	want := []BadHeading{{"Task 3: Parser", 2, 1}, {"Task 0: x", 3, 3}, {"Task 2 Parser", 3, 5}}
	if !reflect.DeepEqual(p.BadHeadings, want) {
		t.Fatalf("bad = %+v", p.BadHeadings)
	}
	if len(p.Tasks) != 1 || p.Tasks[0].Number != 1 {
		t.Fatalf("tasks = %+v", p.Tasks)
	}
}

func TestCancelledBullets(t *testing.T) {
	p := ParsePlan([]byte("## Cancelled Tasks\n\n- Task 5: Front desk screen - moved to add-x\n- Task 5: Front desk screen\n- other\n\n## Reuse\n"))
	if len(p.Cancelled) != 3 {
		t.Fatalf("cancelled = %+v", p.Cancelled)
	}
	want := Cancelled{Number: 5, Name: "Front desk screen", Reason: "moved to add-x", Line: 3, OK: true}
	if p.Cancelled[0] != want {
		t.Fatalf("c0 = %+v", p.Cancelled[0])
	}
	if c := p.Cancelled[1]; c.OK || c.Line != 4 || c.Number != 5 {
		t.Fatalf("c1 = %+v", c)
	}
}

func TestCompensates(t *testing.T) {
	p := ParsePlan([]byte("### Task 6: Fix\n\n**Compensates:** Task 2\n**Compensates:** Task two\n"))
	got := p.Tasks[0].Compensates
	want := []Ref{{2, 3, true}, {0, 4, false}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("compensates = %+v", got)
	}
}

func TestReuseTable(t *testing.T) {
	src := "## Reuse\n\n| Capability | Existing | Evidence |\n|---|---|---|\n| lint | `AGENTS.md`: \"Change workflow\" | rg a\\|b |\n| new | none | rg c |\n\n## Other\n"
	p := ParsePlan([]byte(src))
	if p.ReuseLine != 1 || p.Reuse == nil || len(p.Reuse.Rows) != 2 {
		t.Fatalf("plan = %+v", p)
	}
	r := p.Reuse.Rows[0].Cells
	if r[1] != "`AGENTS.md`: \"Change workflow\"" || r[2] != "rg a|b" {
		t.Fatalf("cells = %q", r)
	}
	if q := ParsePlan([]byte("### Task 1: a\n")); q.ReuseLine != 0 || q.Reuse != nil {
		t.Fatalf("no reuse = %+v", q)
	}
}

func TestPlanCRLF(t *testing.T) {
	src := "# Plan\n\n### Task 1: Parser\n\n**Scenarios:** `a#b`, c#d\n\n### Task 2: Fix\n\n**Compensates:** Task 1\n**Scenarios:** none\n\n## Cancelled Tasks\n\n- Task 5: Desk - moved to add-x\n- other\n\n## Reuse\n\n| A | B |\n|---|---|\n| x | y |\n"
	crlf := "\xef\xbb\xbf" + strings.ReplaceAll(src, "\n", "\r\n")
	want := ParsePlan([]byte(src))
	got := ParsePlan([]byte(crlf))
	if len(got.Tasks) != 2 || len(got.Cancelled) != 2 || got.Reuse == nil {
		t.Fatalf("crlf plan = %+v", got)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("crlf = %+v\nlf   = %+v", got, want)
	}
}
