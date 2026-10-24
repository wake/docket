package lint

import (
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wake/docket/internal/change"
)

const reuseHead = "| Need | Existing (path:symbol) | Decision | Reason | Evidence |\n|---|---|---|---|---|\n"

const planPrefix = "# Plan\n\n## Reuse\n\n" + reuseHead +
	"| Walk | `internal/a.go:Walk` | reuse | | rg Walk |\n\n"

const goodPlan = planPrefix + "### Task 1: One\n\n**Scenarios:** none\n"

func planRepo(plan string, extra ...string) fstest.MapFS {
	m := repoWith(map[string]string{
		"openspec/changes/add-x/proposal.md": validProposal,
		"openspec/changes/add-x/plan.md":     plan,
		"internal/a.go":                      "package a\n",
		"AGENTS.md":                          "x\n",
	})
	for _, e := range extra {
		m[e] = &fstest.MapFile{Data: []byte("x")}
	}
	return m
}

func lintPlan(plan string) Report { return runOn(planRepo(plan)) }

func TestGoodPlanIsClean(t *testing.T) {
	if r := lintPlan(goodPlan); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestNoPlanNoPlanRules(t *testing.T) {
	r := runOn(repoWith(map[string]string{"openspec/changes/add-x/proposal.md": validProposal}))
	if len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestTaskHeadingAtTheWrongLevel(t *testing.T) {
	doc := goodPlan + "\n## Task 3: Parser\n"
	f := wantOne(t, lintPlan(doc), "task-heading", lineOf(doc, "## Task 3"))
	if f.File != "openspec/changes/add-x/plan.md" {
		t.Fatalf("finding = %+v", f)
	}
}

func TestDuplicateTaskNumber(t *testing.T) {
	doc := goodPlan + "\n### Task 2: A\n\n### Task 2: B\n\n## Cancelled Tasks\n\n- Task 1: Gone - because\n"
	r := lintPlan(doc)
	if len(r.Findings) != 2 {
		t.Fatalf("findings = %+v", r.Findings)
	}
	a, b := r.Findings[0], r.Findings[1]
	l1, l2 := lineOf(doc, "Task 1: One"), lineOf(doc, "- Task 1: Gone")
	if a.Rule != "task-heading" || a.Line != lineOf(doc, "Task 2: B") ||
		!strings.Contains(a.Message, strconvI(lineOf(doc, "Task 2: A"))) {
		t.Fatalf("a = %+v", a)
	}
	if b.Rule != "task-heading" || b.Line != l2 || !strings.Contains(b.Message, strconvI(l1)) {
		t.Fatalf("b = %+v", b)
	}
}

func strconvI(n int) string { return strconv.Itoa(n) }

func reusePlan(rows string) string {
	return "# Plan\n\n## Reuse\n\n" + reuseHead + rows + "\n### Task 1: One\n\n**Scenarios:** none\n"
}

func TestReuseRowWithAMissingPath(t *testing.T) {
	doc := reusePlan("| Run | `internal/gitx/run.go:Run` | reuse | | rg Run |\n")
	f := wantOne(t, lintPlan(doc), "reuse-table", lineOf(doc, "internal/gitx"))
	if !strings.Contains(f.Message, "internal/gitx/run.go") {
		t.Fatalf("finding = %+v", f)
	}
}

func TestNewRowWithoutAReason(t *testing.T) {
	doc := reusePlan("| Run | none | new | | rg Run |\n")
	wantOne(t, lintPlan(doc), "reuse-table", lineOf(doc, "| Run |"))
	ok := reusePlan("| Run | none | new | nothing fits | rg Run |\n")
	if r := lintPlan(ok); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestExternalDependencyRow(t *testing.T) {
	doc := reusePlan("| Git | external: git CLI | reuse | | git --help |\n| Lock | external: `syscall.Flock` | extend | | go doc |\n")
	if r := lintPlan(doc); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestEmptyReuseTable(t *testing.T) {
	doc := reusePlan("")
	wantOne(t, lintPlan(doc), "reuse-table", lineOf(doc, "| Need |"))
}

func TestReuseSectionProblems(t *testing.T) {
	none := "# Plan\n\n### Task 1: One\n\n**Scenarios:** none\n"
	wantOne(t, lintPlan(none), "reuse-table", 0)
	notable := "# Plan\n\n## Reuse\n\nnothing\n\n### Task 1: One\n"
	wantOne(t, lintPlan(notable), "reuse-table", 3)
	bad := strings.Replace(reusePlan(""), "Evidence", "Proof", 1)
	wantOne(t, lintPlan(bad), "reuse-table", 5)
	dec := reusePlan("| Run | none | maybe | r | e |\n")
	wantOne(t, lintPlan(dec), "reuse-table", lineOf(dec, "maybe"))
	abs := reusePlan("| Run | `/etc/passwd:x` | reuse | | e |\n")
	wantOne(t, lintPlan(abs), "reuse-table", lineOf(abs, "passwd"))
}

func TestRowWithoutEvidence(t *testing.T) {
	doc := reusePlan("| Run | `internal/a.go:Walk` | reuse | | |\n")
	wantOne(t, lintPlan(doc), "reuse-table", lineOf(doc, "| Run |"))
}

func TestCancelledTaskWithoutAReason(t *testing.T) {
	doc := goodPlan + "\n## Cancelled Tasks\n\n- Task 5: Front desk screen\n"
	wantOne(t, lintPlan(doc), "cancelled-task", lineOf(doc, "- Task 5"))
}

func TestCompensatingACancelledTask(t *testing.T) {
	doc := goodPlan + "\n### Task 2: Fix\n\n**Compensates:** Task 5\n\n## Cancelled Tasks\n\n- Task 5: Desk - dropped\n"
	f := wantOne(t, lintPlan(doc), "compensates", lineOf(doc, "**Compensates:**"))
	if !strings.Contains(f.Message, "cancelled") {
		t.Fatalf("finding = %+v", f)
	}
}

func TestCompensatingAMissingTask(t *testing.T) {
	doc := goodPlan + "\n### Task 2: Fix\n\n**Compensates:** Task 7\n"
	f := wantOne(t, lintPlan(doc), "compensates", lineOf(doc, "**Compensates:**"))
	if !strings.Contains(f.Message, "not a task") {
		t.Fatalf("finding = %+v", f)
	}
	bad := goodPlan + "\n### Task 2: Fix\n\n**Compensates:** Task one\n"
	wantOne(t, lintPlan(bad), "compensates", lineOf(bad, "**Compensates:**"))
	ok := goodPlan + "\n### Task 2: Fix\n\n**Compensates:** Task 1\n"
	if r := lintPlan(ok); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestFencedPlanContent(t *testing.T) {
	fenced := "```\n### Task 9: x\n\n## Reuse\n\n" + reuseHead + "| a | none | new | r | e |\n```\n"
	none := "# Plan\n\n" + fenced + "\n### Task 1: One\n\n```\n**Scenarios:** none\n```\n"
	r := lintPlan(none)
	if len(r.Findings) != 1 || r.Findings[0].Rule != "reuse-table" || r.Findings[0].Line != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
	p := change.ParsePlan([]byte(none))
	if len(p.Tasks) != 1 || p.Tasks[0].ScenariosLine != 0 || p.ReuseLine != 0 {
		t.Fatalf("plan = %+v", p)
	}
}

func TestCRLFPlan(t *testing.T) {
	doc := reusePlan("| Run | `internal/gitx/run.go:Run` | reuse | | |\n") + "\n## Task 4: Bad\n\n## Cancelled Tasks\n\n- Task 6: Nope\n"
	lf := lintPlan(doc)
	crlf := lintPlan("\xef\xbb\xbf" + strings.ReplaceAll(doc, "\n", "\r\n"))
	if len(lf.Findings) < 4 || len(lf.Findings) != len(crlf.Findings) {
		t.Fatalf("lf = %+v, crlf = %+v", lf.Findings, crlf.Findings)
	}
	for i := range lf.Findings {
		if lf.Findings[i] != crlf.Findings[i] {
			t.Fatalf("finding %d: %+v != %+v", i, lf.Findings[i], crlf.Findings[i])
		}
	}
}

func TestRepositoryStylePlan(t *testing.T) {
	doc := reusePlan(
		"| Walk | `internal/a.go:Walk` | reuse | | `rg Walk` \\| hit |\n" +
			"| Rules | `AGENTS.md`: \"Change workflow (superpowers × OpenSpec)\" | extend | | read it |\n" +
			"| Root | external: Go os.Root and Root.FS | reuse | | go doc |\n" +
			"| Lock | external: `syscall.Flock` | reuse | | go doc |\n" +
			"| Fresh | none | new | nothing fits | rg |\n")
	if r := lintPlan(doc); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}
