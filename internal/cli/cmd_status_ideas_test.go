package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

const (
	stIdeaB = "docket/ideas/2026-11-02-b.md"
	stIdeaC = "docket/ideas/2026-11-02-c.md"
)

func stIdea(title string) string { return "---\ntitle: " + title + "\n---\n\nbody\n" }

func statusResult(t *testing.T, so string) map[string]any {
	t.Helper()
	res, _ := decode(t, so)["result"].(map[string]any)
	if res == nil {
		t.Fatalf("result is null\n%s", so)
	}
	return res
}

// ideaRepo holds ideas a, b and c, the order a, b, and a record that finishes a.
func ideaRepo(t *testing.T) *testutil.Repo {
	t.Helper()
	r := cleanRepo(t)
	r.Commit("ideas", map[string]string{
		"docket/ideas/2026-11-02-a.md": stIdea("A"), stIdeaB: stIdea("B"), stIdeaC: stIdea("C"),
		"docket/order.md":                    "# Order\n\n1. `docket/ideas/2026-11-02-a.md`\n2. `" + stIdeaB + "`\n",
		"docket/records/2026-11-02-fix-a.md": "---\ntitle: R\nkind: fix\nsource: docket/ideas/2026-11-02-a.md\n---\n\n## What\n\nx\n",
	})
	return r
}

func TestStatusOrderListed(t *testing.T) {
	r := ideaRepo(t)
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "x")
	w.Commit("change", map[string]string{"openspec/changes/add-x/docket.yaml": "title: X\nsource: " + stIdeaB + "\n"})
	code, so, se := runStatus(t, r.Dir, true)
	res := statusResult(t, so)
	order, _ := res["order"].([]any)
	ideas, ok := res["ideas"].([]any)
	if code != 0 || len(order) != 2 || res["next"] != float64(2) || !ok || len(ideas) != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	_, hs, _ := runStatus(t, r.Dir, false)
	for _, want := range []string{"order 2 in-progress " + stIdeaB + " \"B\" (change add-x)\n", "next 2 " + stIdeaB + "\n"} {
		if !strings.Contains(hs, want) {
			t.Errorf("text lacks %q:\n%s", want, hs)
		}
	}
	if strings.Contains(hs, "order 1 ") {
		t.Errorf("a done line is listed:\n%s", hs)
	}
}

func TestStatusIdeaInProgressOutsideOrder(t *testing.T) {
	r := ideaRepo(t)
	_, so, _ := runStatus(t, r.Dir, true)
	if ideas, _ := statusResult(t, so)["ideas"].([]any); len(ideas) != 1 {
		t.Fatalf("ideas = %v", ideas)
	}
	_, hs, _ := runStatus(t, r.Dir, false)
	if strings.Contains(hs, "idea ") {
		t.Errorf("a pending idea has a text line:\n%s", hs)
	}
	r.Commit("record", map[string]string{"docket/records/2026-11-02-fix-c.md": "---\ntitle: R\nkind: fix\nsource: " + stIdeaC + "\n---\n\n## What\n\nx\n"})
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "x")
	w.Commit("change", map[string]string{"openspec/changes/add-c/docket.yaml": "title: X\nsource: " + stIdeaC + "\n"})
	_, hs, _ = runStatus(t, r.Dir, false)
	if want := "idea " + stIdeaC + " in-progress \"C\" (change add-c, record 2026-11-02-fix-c merged)\n"; !strings.Contains(hs, want) {
		t.Errorf("text lacks %q:\n%s", want, hs)
	}
}

func TestStatusOrderNotListedWithSlugs(t *testing.T) {
	r := ideaRepo(t)
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt"), "x")
	w.Commit("change", map[string]string{"openspec/changes/add-x/docket.yaml": "title: X\nsource: " + stIdeaB + "\n"})
	code, so, se := runStatus(t, r.Dir, true, "add-x")
	res := statusResult(t, so)
	order, ok1 := res["order"].([]any)
	ideas, ok2 := res["ideas"].([]any)
	if code != 0 || !ok1 || !ok2 || len(order) != 0 || len(ideas) != 0 || res["next"] != nil {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestStatusNoOrder(t *testing.T) {
	r := cleanRepo(t)
	code, so, se := runStatus(t, r.Dir, true)
	res := statusResult(t, so)
	order, ok := res["order"].([]any)
	if code != 0 || !ok || len(order) != 0 || res["next"] != nil {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}
