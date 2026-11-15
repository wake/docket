package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pl = "openspec/changes/add-x/plan.md"

const (
	reuseHead   = "| Need | Existing (path:symbol) | Decision | Reason |\n|---|---|---|---|\n"
	earlierHead = "| Need | Existing (path:symbol) | Decision | Reason | Evidence |\n|---|---|---|---|---|\n"
	planTail    = "\n### Task 1: One\n\n**Scenarios:** none\n"
)

func reusePlan(rows string) string {
	return "# Plan\n\n## Reuse\n\n" + reuseHead + rows + planTail
}

// earlierPlan puts rows, written with four cells, in the earlier five-column
// table, each with an empty Evidence cell.
func earlierPlan(rows string) string {
	return "# Plan\n\n## Reuse\n\n" + earlierHead + strings.ReplaceAll(rows, "|\n", "| |\n") + planTail
}

const goodRow = "| Walk | `internal/a.go:Walk` | extend | |\n"

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

func TestReuseSection(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		line int
		msg  string
	}{
		{"no section", "# Plan\n" + planTail, 0, "no `## Reuse`"},
		{"no table", "# Plan\n\n## Reuse\n\ntext\n" + planTail, 3, "no table"},
		{"wrong header", "# Plan\n\n## Reuse\n\n| A | B | C | D |\n|---|---|---|---|\n" + goodRow + planTail, 5, "header"},
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

// TestReuseRows runs each row in both tables: the earlier one, its Evidence
// empty, gives the same result, except that it accepts `reuse` rows.
func TestReuseRows(t *testing.T) {
	tests := []struct {
		name    string
		row     string
		msg     string // "": no finding
		earlier string // the earlier table's result when it differs; "-": no finding
	}{
		{"extend", "| B | `internal/` | extend | |\n", "", ""},
		{"extend in backticks", "| D | `internal/a.go:Walk` | `extend` | |\n", "", ""},
		{"external", "| A | external: git CLI | extend | |\n", "", ""},
		{"new with a reason", "| C | | new | nothing fits |\n", "", ""},
		{"missing path", "| Run | `internal/gitx/run.go:Run` | extend | |\n", "does not exist", ""},
		{"missing dir", "| Walk | `no/such/` | extend | |\n", "does not exist", ""},
		{"no path", "| Walk | `:Walk` | extend | |\n", "repository path", ""},
		{"bad decision", "| Walk | `internal/a.go` | borrow | |\n", "must be extend or new", ""},
		{"new without reason", "| Walk | | new | |\n", "needs a Reason", ""},
		{"reuse", "| Walk | `internal/a.go:Walk` | reuse | |\n", "must be extend or new", "-"},
		{"reuse external", "| A | external: git CLI | reuse | |\n", "must be extend or new", "-"},
		{"reuse missing path", "| Run | `internal/gitx/run.go:Run` | reuse | |\n", "must be extend or new", "does not exist"},
	}
	for _, tc := range tests {
		for _, f := range []struct {
			name string
			plan func(string) string
		}{{"four", reusePlan}, {"earlier", earlierPlan}} {
			want := tc.msg
			if f.name == "earlier" && tc.earlier != "" {
				want = strings.TrimPrefix(tc.earlier, "-")
			}
			t.Run(tc.name+"/"+f.name, func(t *testing.T) {
				r := lintPlan(t, f.plan(tc.row))
				if want == "" && len(r.Findings) != 0 ||
					want != "" && (!has(r, "reuse-table", pl, 7, want) || len(r.Findings) != 1) {
					t.Fatalf("want %q, findings = %+v", want, r.Findings)
				}
			})
		}
	}
}

func TestEmptyReuseTable(t *testing.T) {
	for _, doc := range []string{reusePlan(""), earlierPlan("")} {
		if r := lintPlan(t, doc); len(r.Findings) != 0 {
			t.Fatalf("%q: findings = %+v", doc, r.Findings)
		}
	}
}

func TestReusePathThroughALinkOutIsMissing(t *testing.T) {
	skipRoot(t)
	dir, outside := t.TempDir(), t.TempDir()
	for p, c := range map[string]string{
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\n",
		"openspec/changes/add-x/proposal.md":    goodProposal,
		"openspec/changes/add-x/plan.md":        reusePlan("| Out | `leak/secret.go:X` | extend | |\n"),
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

func TestNonTaskCancelledBullets(t *testing.T) {
	r := lintPlan(t, goodPlan+"\n## Cancelled Tasks\n\n- other\n- more\n")
	if count(r, "cancelled-task") != 2 || len(r.Findings) != 2 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestReuseHeaderCaseAndWidth(t *testing.T) {
	for _, h := range []string{
		"| need | existing (path:symbol) | decision | reason |\n|---|---|---|---|\n",
		"| need | existing (path:symbol) | decision | reason | evidence |\n|---|---|---|---|---|\n",
	} {
		if r := lintPlan(t, "# Plan\n\n## Reuse\n\n"+h+goodRow+planTail); len(r.Findings) != 0 {
			t.Fatalf("lower case %q: %+v", h, r.Findings)
		}
	}
	for _, h := range []string{
		"| Need | Existing (path:symbol) | Decision | Reason | Notes |\n|---|---|---|---|---|\n",
		"| Need | Evidence | Existing (path:symbol) | Decision | Reason |\n|---|---|---|---|---|\n",
		"| Need | Existing (path:symbol) | Decision | Reason | Evidence | Extra |\n|---|---|---|---|---|---|\n",
		"| Need | Existing (path:symbol) | Decision |\n|---|---|---|\n",
	} {
		r := lintPlan(t, "# Plan\n\n## Reuse\n\n"+h+planTail)
		if !has(r, "reuse-table", pl, 5, "header must be: Need | Existing (path:symbol) | Decision | Reason") ||
			len(r.Findings) != 1 || strings.Contains(r.Findings[0].Message, "Evidence") {
			t.Fatalf("%q: %+v", h, r.Findings)
		}
	}
}

func TestReuseExistingOutsideTheRepository(t *testing.T) {
	r := lintPlan(t, reusePlan("| Out | `../outside.go:X` | extend | |\n"))
	if !has(r, "reuse-table", pl, 7, "must start with a repository path or `external:`") || len(r.Findings) != 1 {
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
