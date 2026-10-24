package archive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceWritten(t *testing.T) {
	r, e, f := archRepo(t, "add-x")
	r.Commit("plan\n\nChange: add-x\nTask: 1", map[string]string{
		"openspec/changes/add-x/plan.md": archPlan + "\n### Task 2: Two\n\n- [x] **Step 1: a**\n",
	})
	head := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	base := strings.TrimSpace(r.Git("rev-parse", "HEAD~2"))
	first := strings.TrimSpace(r.Git("rev-parse", "HEAD~1"))
	rep, err := Archive(f.input(e, nil, "add-x"))
	if err != nil || rep.Stopped != nil || len(rep.Archived) != 1 {
		t.Fatalf("%v %+v", err, rep)
	}
	got, err := os.ReadFile(filepath.Join(e.Root, rep.Archived[0].Evidence))
	if err != nil {
		t.Fatal(err)
	}
	want := "# Evidence: add-x\n\n| Field | Value |\n|---|---|\n| Date | 2026-11-02 |\n| Docket | 0.1.0 |\n| OpenSpec | 1.14.3 |\n" +
		"| Repository | none |\n| HEAD | " + head + " |\n| x-base | " + base + " |\n| x-kind | feature |\n| x-source | a \\| b |\n\n" +
		"## Task commits\n\n| Task | Commit | Subject |\n|---|---|---|\n" +
		"| 1 | " + first + " | work on add-x |\n| 1 | " + head + " | plan |\n| 2 | none | |\n\n" +
		"## Checks\n\n- `openspec validate add-x --strict`: Change is valid\n- Base check: 1 requirements checked, none changed since x-base\n"
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
