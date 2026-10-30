package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

// runInspect runs dkt inspect in dir with env and returns the exit code,
// stdout and stderr.
func runInspect(t *testing.T, dir string, env []string, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	code := Run(Options{Args: append([]string{"inspect"}, args...), Dir: dir, Env: env, Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

// inspectEnv builds one clean environment for inspect: a complete workflow
// file in a fresh config home and a fake OpenSpec 1.14.3. A test changes one
// part by appending a variable (the last value counts) or by rewriting the
// workflow file.
func inspectEnv(t *testing.T) (env []string, workflowPath string) {
	t.Helper()
	xdg, path := wfConfig(t)
	wfWrite(t, path, wfFM(wfHash())+wfFour)
	fake, _ := testutil.FakeOpenSpec(t, "1.14.3")
	env = append(testutil.Env(t), xdg...)
	env = append(env, "DOCKET_OPENSPEC="+fake)
	return env, path
}

// inspectFiles returns result.files keyed by path.
func inspectFiles(t *testing.T, so string) map[string]map[string]any {
	t.Helper()
	raw, ok := resultOf(t, so)["files"].([]any)
	if !ok {
		t.Fatalf("result.files is not an array: %s", so)
	}
	out := map[string]map[string]any{}
	for _, f := range raw {
		m := f.(map[string]any)
		out[m["path"].(string)] = m
	}
	return out
}

// inspectRules returns the rule of every finding.
func inspectRules(t *testing.T, so string) []string {
	t.Helper()
	var rules []string
	for _, f := range decode(t, so)["findings"].([]any) {
		rules = append(rules, f.(map[string]any)["rule"].(string))
	}
	return rules
}

func TestInspectBeforeInit(t *testing.T) {
	r := committedRepo(t)
	env, _ := inspectEnv(t)
	code, so, _ := runInspect(t, r.Dir, env, "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	rules := map[string]bool{}
	for _, x := range inspectRules(t, so) {
		rules[x] = true
	}
	for _, want := range []string{"schema-missing", "skill-missing", "agents-block-missing"} {
		if !rules[want] {
			t.Errorf("no %s finding: %s", want, so)
		}
	}
	kinds := map[any]bool{}
	for p, f := range inspectFiles(t, so) {
		if f["status"] != "missing" {
			t.Errorf("%s: status %v", p, f["status"])
		}
		kinds[f["kind"]] = true
	}
	for _, k := range []string{"schema", "skill", "block"} {
		if !kinds[k] {
			t.Errorf("no file of kind %s", k)
		}
	}
}

func TestInspectEditedSkill(t *testing.T) {
	r := committedRepo(t)
	if code, so, _ := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("init: %d %s", code, so)
	}
	const rel = ".claude/skills/docket/SKILL.md"
	p := filepath.Join(r.Dir, rel)
	if err := os.WriteFile(p, []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env, _ := inspectEnv(t)
	code, so, _ := runInspect(t, r.Dir, env, "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	if f := inspectFiles(t, so)[rel]; f["status"] != "differs" || f["kind"] != "skill" {
		t.Errorf("entry: %v", f)
	}
	fs := decode(t, so)["findings"].([]any)
	if len(fs) != 1 {
		t.Fatalf("findings: %v", fs)
	}
	if f := fs[0].(map[string]any); f["rule"] != "skill-differs" || f["file"] != rel {
		t.Errorf("finding: %v", f)
	}
	if b, _ := os.ReadFile(p); string(b) != "mine\n" {
		t.Errorf("file changed: %q", b)
	}
}

func TestInspectClaudeMDImportsAgentsMD(t *testing.T) {
	for _, linked := range []bool{false, true} {
		name := "import"
		if linked {
			name = "link"
		}
		t.Run(name, func(t *testing.T) {
			r := committedRepo(t)
			if code, so, _ := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
				t.Fatalf("init: %d %s", code, so)
			}
			p := filepath.Join(r.Dir, "CLAUDE.md")
			if linked {
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("AGENTS.md", p); err != nil {
					t.Fatal(err)
				}
			} else if b, _ := os.ReadFile(p); string(b) != "@AGENTS.md\n" {
				t.Fatalf("CLAUDE.md: %q", b)
			}
			env, _ := inspectEnv(t)
			_, so, _ := runInspect(t, r.Dir, env, "--json")
			if _, ok := inspectFiles(t, so)["CLAUDE.md"]; ok {
				t.Error("CLAUDE.md is listed")
			}
			for _, f := range decode(t, so)["findings"].([]any) {
				if f.(map[string]any)["file"] == "CLAUDE.md" {
					t.Errorf("finding for CLAUDE.md: %v", f)
				}
			}
		})
	}
}

func TestInspectSymlinkedSkillDirectory(t *testing.T) {
	r := committedRepo(t)
	if code, so, _ := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("init: %d %s", code, so)
	}
	link := filepath.Join(r.Dir, ".claude/skills/docket")
	if err := os.RemoveAll(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Fatal(err)
	}
	env, _ := inspectEnv(t)
	code, so, _ := runInspect(t, r.Dir, env, "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	fs := decode(t, so)["findings"].([]any)
	if len(fs) != 1 {
		t.Fatalf("findings: %v", fs)
	}
	if f := fs[0].(map[string]any); f["rule"] != "symlink-refused" || f["file"] != ".claude/skills/docket" {
		t.Errorf("finding: %v", f)
	}
	if f := inspectFiles(t, so)["openspec/schemas/docket/schema.yaml"]; f["status"] != "same" {
		t.Errorf("schema entry: %v", f)
	}
}

func TestInspectUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every file")
	}
	r := committedRepo(t)
	if code, so, _ := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("init: %d %s", code, so)
	}
	const rel = ".agents/skills/docket/SKILL.md"
	p := filepath.Join(r.Dir, rel)
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	env, _ := inspectEnv(t)
	code, so, _ := runInspect(t, r.Dir, env, "--json")
	if code != 3 {
		t.Fatalf("exit %d: %s", code, so)
	}
	found := false
	for _, u := range decode(t, so)["unreadable"].([]any) {
		if u.(map[string]any)["source"] == rel {
			found = true
		}
	}
	if !found {
		t.Errorf("unreadable does not name %s: %s", rel, so)
	}
	if f := inspectFiles(t, so)["openspec/schemas/docket/schema.yaml"]; f["status"] != "same" {
		t.Errorf("schema entry: %v", f)
	}
}

func TestInspectUsage(t *testing.T) {
	r := committedRepo(t)
	env, _ := inspectEnv(t)
	if code, so, _ := runInspect(t, r.Dir, env, "extra"); code != 2 {
		t.Errorf("exit %d: %s", code, so)
	}
}
