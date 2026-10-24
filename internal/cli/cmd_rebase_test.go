package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func runRebase(t *testing.T, r *testutil.Repo, env []string, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	code := Run(Options{Args: append([]string{"rebase"}, args...), Dir: r.Dir, Env: env, Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

func TestRebaseNeedsOneSlug(t *testing.T) {
	r := cleanRepo(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{}, "rebase needs a slug"},
		{[]string{"add-x", "add-y"}, "rebase takes one slug"},
	} {
		code, so, se := runRebase(t, r, r.Env, tc.args...)
		if code != 2 || !strings.Contains(se, tc.want) {
			t.Fatalf("%v: exit %d, want 2 with %q\n%s\n%s", tc.args, code, tc.want, so, se)
		}
	}
}

const rebaseSpec = "# b\n\n## Purpose\nB.\n\n## Requirements\n\n### Requirement: Cancel\nThe system SHALL cancel.\n\n#### Scenario: ok\n- **WHEN** a\n- **THEN** b\n"

func TestRebaseCommand(t *testing.T) {
	r := cleanRepo(t)
	c1 := r.Commit("spec", map[string]string{"openspec/specs/b/spec.md": rebaseSpec})
	r.Git("checkout", "-q", "-b", "topic")
	r.Commit("add the change", map[string]string{
		"openspec/changes/add-x/.openspec.yaml":  "schema: docket\nx-base: " + c1 + "\n",
		"openspec/changes/add-x/specs/b/spec.md": "## MODIFIED Requirements\n\n### Requirement: Cancel\nThe system SHALL cancel.\n\n#### Scenario: ok\n- **WHEN** a\n- **THEN** c\n",
	})
	r.Git("checkout", "-q", "main")
	c2 := r.Commit("main moves", map[string]string{"openspec/specs/b/spec.md": strings.Replace(rebaseSpec, "THEN** b", "THEN** z", 1)})
	r.Git("checkout", "-q", "topic")
	r.Git("merge", "-q", "--no-edit", "main")
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC=/nonexistent/openspec")
	code, so, se := runRebase(t, r, env, "add-x", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	ch := res["changed"].([]any)
	if res["change"] != "add-x" || res["from"] != c1 || res["to"] != c2 || res["moved"] != true || len(ch) != 1 ||
		ch[0].(map[string]any)["requirement"] != "Cancel" || ch[0].(map[string]any)["capability"] != "b" {
		t.Fatalf("result %v", res)
	}
	code, so, _ = runRebase(t, r, env, "add-x")
	if code != 0 || !strings.Contains(so, "x-base is already "+c2[:7]) {
		t.Fatalf("second run: %d %s", code, so)
	}
}

func TestRebaseNoDefaultBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "topic")
	r.Commit("init", map[string]string{"openspec/changes/add-x/.openspec.yaml": "schema: docket\n"})
	code, so, se := runRebase(t, r, r.Env, "add-x")
	if code != 4 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}
