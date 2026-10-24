package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func runConflicts(t *testing.T, dir string, asJSON bool, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	a := append([]string{"conflicts"}, args...)
	if asJSON {
		a = append(a, "--json")
	}
	code := Run(Options{Args: a, Dir: dir, Env: testutil.Env(t), Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

func deltaFor(op, name string) string {
	body := "### Requirement: " + name + "\nThe system SHALL work.\n\n#### Scenario: s\n- **WHEN** a\n- **THEN** b\n\n"
	return "## " + op + " Requirements\n\n" + body
}

// addChange writes a delta spec for slug into a new worktree of r.
func addChange(t *testing.T, r *testutil.Repo, slug, capability, op, req string) *testutil.Repo {
	t.Helper()
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt-"+slug), "br-"+slug)
	w.Write("openspec/changes/"+slug+"/specs/"+capability+"/spec.md", deltaFor(op, req))
	return w
}

func overlapsOf(t *testing.T, so string) []any {
	t.Helper()
	res := decode(t, so)["result"].(map[string]any)
	return res["overlaps"].([]any)
}

func TestConflictsBothModify(t *testing.T) {
	r := cleanRepo(t)
	addChange(t, r, "add-x", "booking", "MODIFIED", "Cancel booking")
	addChange(t, r, "add-y", "booking", "MODIFIED", "Cancel booking")
	code, so, se := runConflicts(t, r.Dir, true)
	if code != 1 || !hasRule(t, so, "requirement-overlap") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	ov := overlapsOf(t, so)
	if len(ov) != 1 {
		t.Fatalf("overlaps = %v", ov)
	}
	entries := ov[0].(map[string]any)["changes"].([]any)
	if len(entries) != 2 {
		t.Fatalf("entries = %v", entries)
	}
	for i, slug := range []string{"add-x", "add-y"} {
		e := entries[i].(map[string]any)
		ops := e["operations"].([]any)
		if e["slug"] != slug || len(ops) != 1 || ops[0] != "MODIFIED" {
			t.Errorf("entry %d = %v", i, e)
		}
		srcs := e["sources"].([]any)
		if len(srcs) == 0 || len(srcs[0].(map[string]any)["commit"].(string)) != 40 {
			t.Errorf("sources of %s = %v", slug, srcs)
		}
	}
}

func TestConflictsNoOverlaps(t *testing.T) {
	r := cleanRepo(t)
	addChange(t, r, "add-x", "booking", "MODIFIED", "Cancel booking")
	addChange(t, r, "add-y", "booking", "MODIFIED", "Move booking")
	code, so, se := runConflicts(t, r.Dir, true)
	if code != 0 || decode(t, so)["ok"] != true {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	if ov := res["overlaps"].([]any); len(ov) != 0 {
		t.Errorf("overlaps = %v", ov)
	}
	ch := res["changes"].([]any)
	if len(ch) != 2 || ch[0] != "add-x" || ch[1] != "add-y" {
		t.Errorf("changes = %v", ch)
	}
	code, so, _ = runConflicts(t, r.Dir, false)
	if code != 0 || !strings.HasSuffix(so, "dkt conflicts: ok\n") {
		t.Errorf("text exit %d\n%s", code, so)
	}
}

func overlapRepo(t *testing.T) *testutil.Repo {
	r := cleanRepo(t)
	addChange(t, r, "add-x", "booking", "MODIFIED", "Cancel booking")
	addChange(t, r, "add-y", "booking", "MODIFIED", "Cancel booking")
	addChange(t, r, "add-z", "booking", "ADDED", "Unrelated")
	return r
}

func TestConflictsUnrelatedOverlapHidden(t *testing.T) {
	r := overlapRepo(t)
	code, so, se := runConflicts(t, r.Dir, true, "add-z")
	if code != 0 || len(overlapsOf(t, so)) != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestConflictsFocusShowsOwnOverlap(t *testing.T) {
	r := overlapRepo(t)
	code, so, se := runConflicts(t, r.Dir, true, "add-x")
	if code != 1 || len(overlapsOf(t, so)) != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestConflictsSlugInUnreadableSource(t *testing.T) {
	r := newRepo(t)
	addChange(t, r, "add-y", "booking", "MODIFIED", "Cancel booking")
	gone := r.AddWorktree(filepath.Join(t.TempDir(), "wt-gone"), "gone")
	if err := os.RemoveAll(gone.Dir); err != nil {
		t.Fatal(err)
	}
	code, so, se := runConflicts(t, r.Dir, true, "add-q")
	if code != 3 || hasRule(t, so, "change-not-active") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	u, _ := decode(t, so)["unreadable"].([]any)
	if len(u) != 1 || !strings.HasSuffix(u[0].(map[string]any)["source"].(string), "wt-gone") {
		t.Fatalf("unreadable = %v", u)
	}
}

func TestConflictsSlugNotActive(t *testing.T) {
	r := newRepo(t)
	addChange(t, r, "add-y", "booking", "MODIFIED", "Cancel booking")
	code, so, se := runConflicts(t, r.Dir, true, "add-q")
	if code != 1 || !hasRule(t, so, "change-not-active") || !strings.Contains(so, "add-q") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	code, so, se = runConflicts(t, r.Dir, true, "add-x")
	if code != 1 || !hasRule(t, so, "change-not-active") || !strings.Contains(so, "archived as 2025-01-02-add-x") {
		t.Fatalf("archived: exit %d\n%s\n%s", code, so, se)
	}
	for _, slug := range []string{"add-q", "add-x"} {
		_, so, _ = runConflicts(t, r.Dir, true, slug)
		res, _ := decode(t, so)["result"].(map[string]any)
		if res == nil {
			t.Fatalf("%s: result is null\n%s", slug, so)
		}
		if ch, ok := res["changes"].([]any); !ok || len(ch) != 1 || ch[0] != "add-y" {
			t.Errorf("%s: result.changes = %v, want [add-y]", slug, res["changes"])
		}
		if ov, ok := res["overlaps"].([]any); !ok || len(ov) != 0 {
			t.Errorf("%s: result.overlaps = %#v, want []", slug, res["overlaps"])
		}
	}
}

func TestConflictsSlugInvalid(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runConflicts(t, r.Dir, true, "Bad_Slug")
	if code != 1 || !hasRule(t, so, "slug-invalid") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestConflictsTwoSlugs(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runConflicts(t, r.Dir, true, "add-x", "add-y")
	if code != 2 || !hasRule(t, so, "usage") || !strings.Contains(so, "at most one slug") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestConflictsUnreadableWorktree(t *testing.T) {
	r := cleanRepo(t)
	addChange(t, r, "add-x", "booking", "MODIFIED", "Cancel booking")
	addChange(t, r, "add-y", "booking", "MODIFIED", "Cancel booking")
	gone := r.AddWorktree(filepath.Join(t.TempDir(), "wt-gone"), "gone")
	if err := os.RemoveAll(gone.Dir); err != nil {
		t.Fatal(err)
	}
	code, so, se := runConflicts(t, r.Dir, true)
	if code != 3 || len(overlapsOf(t, so)) != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	u, _ := decode(t, so)["unreadable"].([]any)
	if len(u) != 1 || !strings.HasSuffix(u[0].(map[string]any)["source"].(string), "wt-gone") {
		t.Fatalf("unreadable = %v", u)
	}
}

func TestConflictsSourcesListedInText(t *testing.T) {
	r := cleanRepo(t)
	addChange(t, r, "add-x", "booking", "MODIFIED", "Cancel booking")
	y := addChange(t, r, "add-y", "booking", "MODIFIED", "Cancel booking")
	code, so, se := runConflicts(t, r.Dir, false)
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	head := strings.TrimSpace(y.Git("rev-parse", "HEAD"))[:7]
	for _, want := range []string{
		`booking / "Cancel booking"`,
		"  add-y MODIFIED: ",
		y.Dir + "@" + head,
		`requirement-overlap: booking / "Cancel booking": add-x (MODIFIED), add-y (MODIFIED)`,
	} {
		if !strings.Contains(so, want) {
			t.Errorf("missing %q in:\n%s", want, so)
		}
	}
}

func TestConflictsWritesNothing(t *testing.T) {
	r := overlapRepo(t)
	before := snapshot(t, r.Dir)
	for _, c := range []struct {
		json bool
		args []string
	}{{true, nil}, {false, []string{"add-x"}}} {
		code, so, se := runConflicts(t, r.Dir, c.json, c.args...)
		if code != 1 {
			t.Fatalf("args %v: exit %d\n%s\n%s", c.args, code, so, se)
		}
	}
	sameSnapshot(t, before, snapshot(t, r.Dir))
}
