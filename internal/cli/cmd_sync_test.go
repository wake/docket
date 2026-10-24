package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

const syncPlan = "# Plan\n\n### Task 1: One\n\n- [ ] **Step 1: a**\n- [ ] **Step 2: b**\n- [ ] **Step 3: c**\n- [ ] **Step 4: d**\n"

func runSync(t *testing.T, r *testutil.Repo, env []string, args ...string) (int, string, string) {
	t.Helper()
	if env == nil {
		env = r.Env
	}
	var so, se bytes.Buffer
	code := Run(Options{Args: append([]string{"sync"}, args...), Dir: r.Dir, Env: env, Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

func syncFixture(t *testing.T) *testutil.Repo {
	t.Helper()
	r := cleanRepo(t)
	base := r.Git("rev-parse", "HEAD")
	r.Commit("add the change", map[string]string{
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\nx-base: " + base + "\n",
		"openspec/changes/add-x/plan.md":        syncPlan,
	})
	r.Commit("work\n\nChange: add-x\nTask: 1", map[string]string{"a.txt": "a\n"})
	return r
}

func TestSyncCommand(t *testing.T) {
	r := syncFixture(t)
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC=/nonexistent/openspec")
	code, so, se := runSync(t, r, env, "add-x", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	tk := res["ticked"].([]any)
	if res["change"] != "add-x" || res["check"] != false || len(tk) != 1 ||
		tk[0].(map[string]any)["task"] != float64(1) || tk[0].(map[string]any)["steps"] != float64(4) {
		t.Fatalf("result %v", res)
	}
	b, _ := os.ReadFile(filepath.Join(r.Dir, "openspec/changes/add-x/plan.md"))
	if strings.Contains(string(b), "[ ]") {
		t.Fatalf("not ticked:\n%s", b)
	}
	code, so, _ = runSync(t, r, nil, "add-x", "--json")
	if code != 0 || decode(t, so)["result"].(map[string]any)["ticked"] == nil {
		t.Fatalf("second run: %d %s", code, so)
	}
}

func TestSyncCheckExitsOne(t *testing.T) {
	r := syncFixture(t)
	code, so, se := runSync(t, r, nil, "add-x", "--check", "--json")
	if code != 1 || !hasRule(t, so, "plan-unsynced") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	b, _ := os.ReadFile(filepath.Join(r.Dir, "openspec/changes/add-x/plan.md"))
	if string(b) != syncPlan {
		t.Fatal("plan written")
	}
}

func TestSyncNeedsOneSlug(t *testing.T) {
	r := syncFixture(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{}, "sync needs a slug"},
		{[]string{"add-x", "add-y"}, "sync takes one slug"},
	} {
		code, so, se := runSync(t, r, nil, tc.args...)
		if code != 2 || !strings.Contains(se, tc.want) {
			t.Fatalf("%v: exit %d, want 2 with %q\n%s\n%s", tc.args, code, tc.want, so, se)
		}
	}
}

func TestSyncSymlinkedPlanRefused(t *testing.T) {
	r := syncFixture(t)
	p := filepath.Join(r.Dir, "openspec/changes/add-x/plan.md")
	if err := os.Rename(p, filepath.Join(r.Dir, "real-plan.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../../real-plan.md", p); err != nil {
		t.Fatal(err)
	}
	code, so, se := runSync(t, r, nil, "add-x", "--json")
	if code != 1 || !hasRule(t, so, "symlink-refused") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	b, _ := os.ReadFile(filepath.Join(r.Dir, "real-plan.md"))
	if string(b) != syncPlan {
		t.Fatal("link target written")
	}
}
