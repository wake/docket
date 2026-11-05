package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pl = "openspec/changes/add-x/plan.md"

const reuseHead = "| Need | Existing (path:symbol) | Decision | Reason | Evidence |\n|---|---|---|---|---|\n"

func reusePlan(rows string) string {
	return "# Plan\n\n## Reuse\n\n" + reuseHead + rows + "\n### Task 1: One\n\n**Scenarios:** none\n"
}

const goodRow = "| Walk | `internal/a.go:Walk` | reuse | | rg Walk |\n"

var goodPlan = reusePlan(goodRow)

func lintPlan(t *testing.T, plan string) Report {
	t.Helper()
	m := map[string]string{
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\n",
		"openspec/changes/add-x/proposal.md":    goodProposal,
		"internal/a.go":                         "package a\n",
	}
	if plan != "" {
		m["openspec/changes/add-x/plan.md"] = plan
	}
	in, _ := repo(t, m)
	in.Slugs = []string{"add-x"}
	return run(t, in)
}

func lineOf(doc, s string) int {
	for i, l := range strings.Split(doc, "\n") {
		if strings.Contains(l, s) {
			return i + 1
		}
	}
	return -1
}

func TestGoodPlanIsClean(t *testing.T) {
	if r := lintPlan(t, goodPlan); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestNoPlanNoPlanRules(t *testing.T) {
	if r := lintPlan(t, ""); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestTaskHeadingAtTheWrongLevel(t *testing.T) {
	doc := goodPlan + "\n## Task 3: Parser\n"
	if r := lintPlan(t, doc); !has(r, "task-heading", pl, lineOf(doc, "## Task 3"), "Task 3: Parser") || len(r.Findings) != 1 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestDuplicateTaskNumber(t *testing.T) {
	doc := goodPlan + "\n### Task 2: A\n\n**Scenarios:** none\n\n### Task 2: B\n\n**Scenarios:** none\n\n## Cancelled Tasks\n\n- Task 1: Gone - because\n"
	r := lintPlan(t, doc)
	a, b := lineOf(doc, "Task 2: A"), lineOf(doc, "Task 2: B")
	l1, l2 := lineOf(doc, "Task 1: One"), lineOf(doc, "- Task 1: Gone")
	if len(r.Findings) != 2 ||
		!has(r, "task-heading", pl, b, fmt.Sprint(a)) ||
		!has(r, "task-heading", pl, l2, fmt.Sprint(l1)) {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestReuseTable(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		line int // -1: any
		msg  string
	}{
		{"missing path", reusePlan("| Run | `internal/gitx/run.go:Run` | reuse | | rg |\n"), 7, "does not exist"},
		{"empty table", reusePlan(""), 5, "no rows"},
		{"no section", "# Plan\n\n### Task 1: One\n\n**Scenarios:** none\n", 0, "no `## Reuse`"},
		{"no table", "# Plan\n\n## Reuse\n\ntext\n\n### Task 1: One\n\n**Scenarios:** none\n", 3, "no table"},
		{"wrong header", "# Plan\n\n## Reuse\n\n| A | B | C | D | E |\n|---|---|---|---|---|\n" + goodRow, 5, "header"},
		{"bad decision", reusePlan("| Walk | `internal/a.go` | borrow | | rg |\n"), 7, "must be reuse, extend or new"},
		{"new without reason", reusePlan("| Walk | | new | | rg |\n"), 7, "needs a Reason"},
		{"no evidence", reusePlan("| Walk | `internal/a.go` | extend | | |\n"), 7, "Evidence is empty"},
		{"no path", reusePlan("| Walk | `:Walk` | reuse | | rg |\n"), 7, "repository path"},
		{"missing dir", reusePlan("| Walk | `no/such/` | reuse | | rg |\n"), 7, "does not exist"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := lintPlan(t, tc.doc)
			if !has(r, "reuse-table", pl, tc.line, tc.msg) || len(r.Findings) != 1 {
				t.Fatalf("findings = %+v", r.Findings)
			}
		})
	}
}

func TestReuseRowsThatPass(t *testing.T) {
	rows := "| A | external: git CLI | reuse | | rg |\n" +
		"| B | `internal/` | extend | | rg |\n" +
		"| C | | new | nothing fits | rg |\n" +
		"| D | `internal/a.go:Walk` | `reuse` | | rg |\n"
	if r := lintPlan(t, reusePlan(rows)); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestReusePathThroughALinkOutIsMissing(t *testing.T) {
	skipRoot(t)
	dir, outside := t.TempDir(), t.TempDir()
	for p, c := range map[string]string{
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\n",
		"openspec/changes/add-x/proposal.md":    goodProposal,
		"openspec/changes/add-x/plan.md":        reusePlan("| Out | `leak/secret.go:X` | reuse | | rg |\n"),
	} {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "leak")); err != nil {
		t.Fatal(err)
	}
	in, _ := open(t, dir)
	in.Slugs = []string{"add-x"}
	r := run(t, in)
	if !has(r, "reuse-table", pl, 7, "does not exist") {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestCancelledAndCompensates(t *testing.T) {
	doc := goodPlan +
		"\n### Task 2: B\n\n**Scenarios:** none\n\n**Compensates:** Task 5\n\n**Compensates:** Task 9\n\n**Compensates:** Task 1 please\n\n**Compensates:** Task 1\n" +
		"\n## Cancelled Tasks\n\n- Task 5: Front desk screen\n- Task 6: Other - dropped\n"
	r := lintPlan(t, doc)
	if len(r.Findings) != 4 ||
		!has(r, "cancelled-task", pl, lineOf(doc, "- Task 5:"), "reason") ||
		!has(r, "compensates", pl, lineOf(doc, "Compensates:** Task 5"), "cancelled") ||
		!has(r, "compensates", pl, lineOf(doc, "Compensates:** Task 9"), "not a task") ||
		!has(r, "compensates", pl, lineOf(doc, "Task 1 please"), "must read") {
		t.Fatalf("findings = %+v", r.Findings)
	}
}
