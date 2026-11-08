package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

type snapshotOut struct {
	Command    string
	Findings   []Finding
	Unreadable []Unreadable
	Result     *struct{ Works []struct{ Slug string } }
}

const snChange = "openspec/changes/add-x/"

func snFiles() map[string]string {
	return map[string]string{
		snChange + "proposal.md":    "---\ntitle: Add X\n---\n\nWhy\n",
		snChange + "plan.md":        "## Tasks\n\n### Task 1: One\n\n- [ ] **Step 1: a**\n",
		snChange + ".openspec.yaml": "schema: docket\n",
	}
}

func snRun(t *testing.T, r *testutil.Repo, json_ bool) (int, snapshotOut, string) {
	t.Helper()
	env := append(slices.Clone(r.Env), "DOCKET_OPENSPEC="+filepath.Join(t.TempDir(), "nope"))
	a := []string{"snapshot"}
	if json_ {
		a = append(a, "--json")
	}
	var out, errb bytes.Buffer
	code := Run(Options{Args: a, Dir: r.Dir, Env: env, Stdout: &out, Stderr: &errb})
	var m snapshotOut
	if json_ && out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &m); err != nil {
			t.Fatalf("%q: %v", out.String(), err)
		}
	}
	return code, m, out.String() + errb.String()
}

func TestSnapshotCommand(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", snFiles())
	code, m, raw := snRun(t, r, true)
	if code != 0 || m.Command != "snapshot" || m.Result == nil || len(m.Result.Works) != 1 || m.Result.Works[0].Slug != "add-x" {
		t.Errorf("%d %s", code, raw)
	}
}

func TestSnapshotCommandFindings(t *testing.T) {
	r := stRepo(t, stOrder("a"), map[string]string{"a": "---\ndropped: false\n---\n"})
	code, m, raw := snRun(t, r, true)
	if code != 1 || m.Result == nil || len(m.Findings) == 0 || m.Findings[0].Rule != "idea-front-matter" || m.Findings[0].File != ideaA || m.Findings[0].Line != 0 {
		t.Errorf("%d %s", code, raw)
	}
}

func TestSnapshotCommandUnreadable(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", snFiles())
	wt := r.Dir + "-gone"
	r.Git("worktree", "add", "-q", "--detach", wt)
	if err := os.RemoveAll(wt); err != nil {
		t.Fatal(err)
	}
	code, m, raw := snRun(t, r, true)
	if code != 3 || len(m.Unreadable) == 0 || !strings.Contains(m.Unreadable[0].Source, filepath.Base(wt)) || m.Unreadable[0].Reason == "" {
		t.Errorf("%d %s", code, raw)
	}
}

func TestSnapshotOutsideRepository(t *testing.T) {
	var out bytes.Buffer
	code := Run(Options{Args: []string{"snapshot", "--json"}, Dir: t.TempDir(), Env: testutil.Env(t), Stdout: &out, Stderr: &bytes.Buffer{}})
	var m snapshotOut
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatalf("%q: %v", out.String(), err)
	}
	if code != 4 || len(m.Findings) != 1 || m.Findings[0].Rule != "no-repository" {
		t.Errorf("code=%d out=%q", code, out.String())
	}
}

func TestSnapshotText(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", nil)
	r.Git("checkout", "-q", "-b", "add-x") // on main the change would count as merged
	r.Commit("plan", snFiles())
	code, _, raw := snRun(t, r, false)
	if want := "change add-x  plan  Add X\n0 ideas, 0 issues, 0 specs\n"; code != 0 || raw != want {
		t.Errorf("%d %q", code, raw)
	}
}
