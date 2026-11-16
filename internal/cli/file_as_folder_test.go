package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

// A file standing where dkt expects a folder makes what is under it unreadable, never missing,
// on every system (docket/issues/2026-11-16-file-as-folder-windows.md). Windows reports such a
// read like a missing path, so these tests run in its CI job.

// homeWithFile returns a new folder, for a config or state home, holding a file named docket.
func homeWithFile(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	if err := os.WriteFile(filepath.Join(h, "docket"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return h
}

func unreadableHas(us []Unreadable, src string) bool {
	return slices.ContainsFunc(us, func(u Unreadable) bool { return u.Source == src })
}

func TestFileAsFolderIndex(t *testing.T) {
	state := homeWithFile(t)
	env := append(testutil.Env(t), "XDG_STATE_HOME="+state)
	code, m, raw, _ := prRun(t, env, t.TempDir(), "ls")
	if code != 3 || !unreadableHas(m.Unreadable, filepath.Join(state, "docket", "projects.json")) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestFileAsFolderUserFiles(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	cfg := homeWithFile(t)
	r.Env = append(r.Env, "XDG_CONFIG_HOME="+cfg)
	code, m, raw := inspect(t, r, "1.14.0", false)
	for _, f := range []string{"workflow.md", "config.json"} {
		if !unreadableHas(m.Unreadable, filepath.Join(cfg, "docket", f)) {
			t.Errorf("%s not unreadable: %d %s", f, code, raw)
		}
	}
	if code != 3 {
		t.Errorf("code %d", code)
	}
}

func TestFileAsFolderSettingsForRegistration(t *testing.T) {
	r := prRepo(t, true)
	cfg := homeWithFile(t)
	env := regEnv(r.Env, "XDG_CONFIG_HOME="+cfg)
	_, _, errs := regRun(t, env, r.Dir, "status", "--json")
	if w := regWarnings(errs); len(w) != 1 || !strings.Contains(w[0], filepath.Join(cfg, "docket", "config.json")) {
		t.Errorf("warnings %v", w)
	}
	if idx := regIndex(t, env); idx != "" {
		t.Errorf("registered: %s", idx)
	}
}

func TestFileAsFolderUserTemplateForMode(t *testing.T) {
	tpl := shippedWorkflow(t)
	r, _ := syncFixture(t, tpl, tpl, "independent", nil)
	cfg := homeWithFile(t)
	env := append(slices.Clone(r.Env), "XDG_CONFIG_HOME="+cfg)
	code, m, out := modeRun(t, r.Dir, env, "sync")
	if code != 3 || len(m.Findings) != 0 || !unreadableHas(m.Unreadable, filepath.Join(cfg, "docket", "workflow.md")) {
		t.Errorf("code=%d %s", code, out)
	}
}

func TestFileAsFolderUserTemplateForRefs(t *testing.T) {
	env, _, _ := refsEnv(t, testutil.Env(t))
	cfg := homeWithFile(t)
	env = append(env, "XDG_CONFIG_HOME="+cfg)
	code, m, out := refsRun(t, t.TempDir(), env, "refs")
	if code != 3 || !unreadableHas(m.Unreadable, filepath.Join(cfg, "docket", "workflow.md")) {
		t.Errorf("code=%d %s", code, out)
	}
}

func TestFileAsFolderSessions(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", snFiles())
	state := homeWithFile(t)
	r.Env = append(r.Env, "XDG_STATE_HOME="+state)
	code, m, raw := snRun(t, r, true)
	if code != 3 || !unreadableHas(m.Unreadable, filepath.Join(state, "docket", "sessions")) {
		t.Errorf("%d %s", code, raw)
	}
}

func TestFileAsFolderLintChanges(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write("openspec/changes", "x")
	code, m, raw, _ := doLint(t, r.Dir, r.Env, "add-x")
	if code != 3 || !unreadableHas(m.Unreadable, "openspec/changes") || len(m.Findings) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestFileAsFolderLintOpenspec(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write("openspec", "x")
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 3 || !unreadableHas(m.Unreadable, "openspec/changes") || !unreadableHas(m.Unreadable, "openspec/specs") {
		t.Fatalf("%d %s", code, raw)
	}
}

// A change's capability whose main spec folder is a file: the main spec is unreadable, so the
// change is not told that the capability needs a name.
func TestFileAsFolderLintMainSpec(t *testing.T) {
	r := lintRepo(t, "docket")
	r.Write("openspec/changes/add-x/proposal.md", strings.Replace(lintProposal, "specs:\n  x-cap: X capability\n", "", 1))
	r.Write("openspec/specs/x-cap", "x")
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.0")
	dir := t.TempDir()
	testutil.OpenSpecAnswer(t, dir, 0, `{"deltas":[{"spec":"x-cap","operation":"ADDED","requirements":[{"name":"X","text":"The system SHALL x.","scenarios":[]}]}]}`,
		"show", "add-x", "--type", "change", "--json", "--deltas-only")
	testutil.OpenSpecAnswer(t, dir, 0, `{"items":[{"id":"add-x","valid":true,"issues":[]}]}`,
		"validate", "add-x", "--type", "change", "--json")
	env := append(slices.Clone(r.Env), "DOCKET_OPENSPEC="+cmd, "FAKE_OPENSPEC_JSON="+dir)
	code, out, _ := runIn(t, r.Dir, env, "lint", "add-x", "--json")
	var m lintOut
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("%q: %v", out, err)
	}
	if code != 3 || !unreadableHas(m.Unreadable, "openspec/specs/x-cap/spec.md") || strings.Contains(out, "title-missing") {
		t.Fatalf("%d %s", code, out)
	}
}

func TestFileAsFolderDomainsOpenspec(t *testing.T) {
	r := dmRepo(t, map[string]string{"docket/domains.md": "- A\n  - `gone`\n", "openspec": "x"})
	code, m, raw := dmRun(t, r, r.Dir, true)
	if code != 3 || !unreadableHas(m.Unreadable, "openspec/changes") || !unreadableHas(m.Unreadable, "openspec/specs") {
		t.Errorf("%d %s", code, raw)
	}
}

func TestFileAsFolderConflictsSpecs(t *testing.T) {
	e := newCfEnv(t)
	put(t, e.r.Dir, map[string]string{"openspec/changes/add-x/specs": "x"})
	e.answer(t, "add-x", [3]string{"booking", "ADDED", "A"})
	code, m, raw, _ := e.do(t)
	if code != 3 || len(m.Unreadable) == 0 {
		t.Fatalf("%d %s", code, raw)
	}
}
