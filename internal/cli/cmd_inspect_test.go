package cli

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
	"github.com/wake/docket/internal/workflow"
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
	res := resultOf(t, so)
	if got := res["changes"]; !reflect.DeepEqual(got, []any{}) {
		t.Errorf("changes: %v", got)
	}
	if o, _ := res["openspec"].(map[string]any); o["version"] != "1.14.3" {
		t.Errorf("openspec: %v", res["openspec"])
	}
	if wf, _ := res["workflow"].(map[string]any); wf["status"] != "ok" {
		t.Errorf("workflow: %v", res["workflow"])
	}
}

func TestInspectCleanSetup(t *testing.T) {
	r := committedRepo(t)
	if code, so, _ := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("init: %d %s", code, so)
	}
	env, _ := inspectEnv(t)
	code, so, _ := runInspect(t, r.Dir, env, "--json")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, so)
	}
	if fs := decode(t, so)["findings"].([]any); len(fs) != 0 {
		t.Errorf("findings: %v", fs)
	}
	res := resultOf(t, so)
	if got := res["openspec"]; !reflect.DeepEqual(got, map[string]any{"version": "1.14.3", "supported": true}) {
		t.Errorf("openspec: %v", got)
	}
	if wf, _ := res["workflow"].(map[string]any); wf["status"] != "ok" {
		t.Errorf("workflow: %v", res["workflow"])
	}
	if got, want := wfKeys(res), []string{"changes", "files", "openspec", "workflow"}; !reflect.DeepEqual(got, want) {
		t.Errorf("keys %v, want %v", got, want)
	}
	if got := res["changes"]; !reflect.DeepEqual(got, []any{}) {
		t.Errorf("changes: %v", got)
	}
}

// inspectOpenSpecMissing checks the result of an inspect run where no OpenSpec
// could be started.
func inspectOpenSpecMissing(t *testing.T, env []string) {
	t.Helper()
	r := committedRepo(t)
	code, so, _ := runInspect(t, r.Dir, env, "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	if got := resultOf(t, so)["openspec"]; !reflect.DeepEqual(got, map[string]any{"version": nil, "supported": false}) {
		t.Errorf("openspec: %v", got)
	}
	found := false
	for _, x := range inspectRules(t, so) {
		if x == "openspec-missing" {
			found = true
		}
	}
	if !found {
		t.Errorf("no openspec-missing finding: %s", so)
	}
}

func TestInspectUnsupportedOpenSpec(t *testing.T) {
	r := committedRepo(t)
	env, _ := inspectEnv(t)
	fake, _ := testutil.FakeOpenSpec(t, "1.3.0")
	env = append(env, "DOCKET_OPENSPEC="+fake)
	code, so, _ := runInspect(t, r.Dir, env, "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	if got := resultOf(t, so)["openspec"]; !reflect.DeepEqual(got, map[string]any{"version": "1.3.0", "supported": false}) {
		t.Errorf("openspec: %v", got)
	}
	found := false
	for _, f := range wfFindings(t, so) {
		if f["rule"] == "openspec-unsupported" {
			found = true
			msg, _ := f["message"].(string)
			if !strings.Contains(msg, "1.3.0") || !strings.Contains(msg, "1.14.x") {
				t.Errorf("message: %q", msg)
			}
		}
	}
	if !found {
		t.Errorf("no openspec-unsupported finding: %s", so)
	}
}

func TestInspectNoOpenSpec(t *testing.T) {
	env, _ := inspectEnv(t)
	inspectOpenSpecMissing(t, append(env, "DOCKET_OPENSPEC="+filepath.Join(t.TempDir(), "missing")))
}

func TestInspectOpenSpecVersionFails(t *testing.T) {
	script := filepath.Join(t.TempDir(), "openspec")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	env, _ := inspectEnv(t)
	inspectOpenSpecMissing(t, append(env, "DOCKET_OPENSPEC="+script))
}

func TestInspectWorkflowReported(t *testing.T) {
	r := committedRepo(t)
	env, path := inspectEnv(t)
	wfWrite(t, path, wfFM(wfHash())+strings.Replace(wfFour, "## Check", "## Other", 1))
	code, so, _ := runInspect(t, r.Dir, env, "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	if wf, _ := resultOf(t, so)["workflow"].(map[string]any); wf["status"] != "incomplete" {
		t.Errorf("workflow: %v", resultOf(t, so)["workflow"])
	}
	n := 0
	for _, f := range wfFindings(t, so) {
		if f["rule"] == "workflow-incomplete" {
			n++
			if msg, _ := f["message"].(string); !strings.Contains(msg, `"## Check"`) {
				t.Errorf("message: %q", msg)
			}
		}
	}
	if n != 1 {
		t.Errorf("%d workflow-incomplete findings: %s", n, so)
	}
}

func TestInspectWorkflowNoHome(t *testing.T) {
	r := committedRepo(t)
	env, _ := inspectEnv(t)
	var bare []string
	for _, e := range env {
		if !strings.HasPrefix(e, "HOME=") && !strings.HasPrefix(e, "XDG_CONFIG_HOME=") {
			bare = append(bare, e)
		}
	}
	var so, se bytes.Buffer
	code := Run(Options{Args: []string{"inspect", "--json"}, Dir: r.Dir, Env: bare, UserHome: nil, Stdout: &so, Stderr: &se})
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so.String())
	}
	want := map[string]any{"path": nil, "status": "missing", "template": wfHash(), "recorded": nil, "missing_sections": []any{}}
	if got := resultOf(t, so.String())["workflow"]; !reflect.DeepEqual(got, want) {
		t.Errorf("workflow: %v, want %v", got, want)
	}
	found := false
	for _, f := range wfFindings(t, so.String()) {
		if f["rule"] == "workflow-missing" {
			found = true
			if f["message"] != workflow.ErrNoHome.Error() {
				t.Errorf("message: %v", f["message"])
			}
		}
	}
	if !found {
		t.Errorf("no workflow-missing finding: %s", so.String())
	}
	if len(inspectFiles(t, so.String())) == 0 {
		t.Error("no files")
	}
}

func TestInspectWorkflowUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every file")
	}
	r := committedRepo(t)
	env, path := inspectEnv(t)
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o644) })
	code, so, _ := runInspect(t, r.Dir, env, "--json")
	if code != 3 {
		t.Fatalf("exit %d: %s", code, so)
	}
	found := false
	for _, u := range decode(t, so)["unreadable"].([]any) {
		if u.(map[string]any)["source"] == path {
			found = true
		}
	}
	if !found {
		t.Errorf("unreadable does not name %s: %s", path, so)
	}
	res := resultOf(t, so)
	if res["workflow"] != nil {
		t.Errorf("workflow: %v", res["workflow"])
	}
	if len(inspectFiles(t, so)) == 0 {
		t.Error("no files")
	}
	if got := res["changes"]; !reflect.DeepEqual(got, []any{}) {
		t.Errorf("changes: %v", got)
	}
	if o, _ := res["openspec"].(map[string]any); o["version"] != "1.14.3" {
		t.Errorf("openspec: %v", res["openspec"])
	}
}

func hasLine(out, want string) bool {
	for _, l := range strings.Split(out, "\n") {
		if l == want {
			return true
		}
	}
	return false
}

func TestInspectText(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		r := committedRepo(t)
		env, path := inspectEnv(t)
		_, so, _ := runInspect(t, r.Dir, env)
		for _, want := range []string{"openspec 1.14.3", "workflow ok " + path} {
			if !hasLine(so, want) {
				t.Errorf("no line %q in %q", want, so)
			}
		}
	})
	t.Run("unsupported", func(t *testing.T) {
		r := committedRepo(t)
		env, _ := inspectEnv(t)
		fake, _ := testutil.FakeOpenSpec(t, "1.3.0")
		_, so, _ := runInspect(t, r.Dir, append(env, "DOCKET_OPENSPEC="+fake))
		if want := "openspec 1.3.0 (unsupported)"; !hasLine(so, want) {
			t.Errorf("no line %q in %q", want, so)
		}
	})
	t.Run("missing", func(t *testing.T) {
		r := committedRepo(t)
		env, _ := inspectEnv(t)
		_, so, _ := runInspect(t, r.Dir, append(env, "DOCKET_OPENSPEC="+filepath.Join(t.TempDir(), "missing")))
		if want := "openspec not found"; !hasLine(so, want) {
			t.Errorf("no line %q in %q", want, so)
		}
	})
	t.Run("no home", func(t *testing.T) {
		r := committedRepo(t)
		env, _ := inspectEnv(t)
		var bare []string
		for _, e := range env {
			if !strings.HasPrefix(e, "HOME=") && !strings.HasPrefix(e, "XDG_CONFIG_HOME=") {
				bare = append(bare, e)
			}
		}
		var so, se bytes.Buffer
		Run(Options{Args: []string{"inspect"}, Dir: r.Dir, Env: bare, Stdout: &so, Stderr: &se})
		if want := "workflow missing"; !hasLine(so.String(), want) {
			t.Errorf("no line %q in %q", want, so.String())
		}
	})
	t.Run("edited skill", func(t *testing.T) {
		r := committedRepo(t)
		if code, so, _ := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
			t.Fatalf("init: %d %s", code, so)
		}
		const rel = ".claude/skills/docket/SKILL.md"
		if err := os.WriteFile(filepath.Join(r.Dir, rel), []byte("mine\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		env, _ := inspectEnv(t)
		_, so, _ := runInspect(t, r.Dir, env)
		if want := fmt.Sprintf("%-8s %s", "differs", rel); !hasLine(so, want) {
			t.Errorf("no line %q in %q", want, so)
		}
	})
}

func TestInspectUsage(t *testing.T) {
	r := committedRepo(t)
	env, _ := inspectEnv(t)
	if code, so, _ := runInspect(t, r.Dir, env, "extra"); code != 2 {
		t.Errorf("exit %d: %s", code, so)
	}
}

const squashedPlan = "# Plan\n\n### Task 1: One\n\n- [x] **Step 1: a**\n"

// commitSquashed commits a change add-x with no trailers, as a squashed pull
// request leaves it, and returns the commit.
func commitSquashed(t *testing.T, r *testutil.Repo) string {
	t.Helper()
	base := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	return r.Commit("squashed (#1)", map[string]string{
		"openspec/changes/add-x/docket.yaml": "title: X\nkind: feature\nsource: s\nbase: " + base + "\n",
		"openspec/changes/add-x/plan.md":     squashedPlan,
	})
}

// hasChangeFinding reports whether the findings in so include rule for change.
func hasChangeFinding(t *testing.T, so, rule, change string) bool {
	t.Helper()
	for _, f := range decode(t, so)["findings"].([]any) {
		m := f.(map[string]any)
		if m["rule"] == rule && m["change"] == change {
			return true
		}
	}
	return false
}

func TestInspectSquashedChange(t *testing.T) {
	r := committedRepo(t)
	if code, so, _ := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("init: %d %s", code, so)
	}
	r.Commit("install", nil)
	commitSquashed(t, r)
	env, _ := inspectEnv(t)
	code, so, _ := runInspect(t, r.Dir, env, "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	if !hasChangeFinding(t, so, "no-task-commits", "add-x") {
		t.Errorf("no no-task-commits finding: %s", so)
	}
	want := []any{map[string]any{"slug": "add-x", "skipped": nil}}
	if got := resultOf(t, so)["changes"]; !reflect.DeepEqual(got, want) {
		t.Errorf("changes: %v", got)
	}
}

// envValue returns the value of name in env, the last one counting.
func envValue(env []string, name string) string {
	v := ""
	for _, e := range env {
		if strings.HasPrefix(e, name+"=") {
			v = strings.TrimPrefix(e, name+"=")
		}
	}
	return v
}

func TestInspectNothingWritten(t *testing.T) {
	r := committedRepo(t)
	if code, so, _ := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("init: %d %s", code, so)
	}
	for _, root := range []string{"openspec/schemas/docket", ".claude/skills", ".agents/skills"} {
		err := filepath.WalkDir(filepath.Join(r.Dir, root), func(p string, d fs.DirEntry, err error) error {
			if err == nil && d.Type().IsRegular() {
				err = os.WriteFile(p, []byte("mine\n"), 0o644)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	agents := filepath.Join(r.Dir, "AGENTS.md")
	b, err := os.ReadFile(agents)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	i, j := strings.Index(s, "<!-- docket:start -->"), strings.Index(s, "<!-- docket:end -->")
	if i < 0 || j < i {
		t.Fatalf("no Docket block: %s", s)
	}
	if err := os.WriteFile(agents, []byte(s[:i]+"<!-- docket:start -->\nmine\n"+s[j:]), 0o644); err != nil {
		t.Fatal(err)
	}
	r.Commit("install", nil)
	commitSquashed(t, r)

	env, wf := inspectEnv(t)
	if err := os.Remove(wf); err != nil {
		t.Fatal(err)
	}
	fake, log := testutil.FakeOpenSpec(t, "1.14.3")
	env = append(env, "DOCKET_OPENSPEC="+fake)
	xdg, home := envValue(env, "XDG_CONFIG_HOME"), envValue(env, "HOME")
	snaps := [3]map[string]string{snapshot(t, r.Dir), snapshot(t, xdg), snapshot(t, home)}
	status := r.Git("status", "--porcelain")

	code, so, _ := runInspect(t, r.Dir, env, "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	rules := map[string]bool{}
	for _, x := range inspectRules(t, so) {
		rules[x] = true
		if strings.HasPrefix(x, "openspec-") {
			t.Errorf("unexpected %s", x)
		}
	}
	for _, want := range []string{"schema-differs", "skill-differs", "agents-block-differs", "workflow-missing", "no-task-commits"} {
		if !rules[want] {
			t.Errorf("no %s finding: %s", want, so)
		}
	}
	sameSnapshot(t, snaps[0], snapshot(t, r.Dir))
	sameSnapshot(t, snaps[1], snapshot(t, xdg))
	sameSnapshot(t, snaps[2], snapshot(t, home))
	if got := r.Git("status", "--porcelain"); got != status {
		t.Errorf("git status changed: %q -> %q", status, got)
	}
	lb, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSpace(string(lb)), "\n"); !reflect.DeepEqual(lines, []string{"--version"}) {
		t.Errorf("openspec calls: %q", lines)
	}
}

func TestInspectNoDefaultBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "work")
	r.Commit("init", map[string]string{"README.md": "x\n"})
	env, _ := inspectEnv(t)
	if code, so, _ := runInspect(t, r.Dir, env, "--json"); code != 4 {
		t.Fatalf("exit %d: %s", code, so)
	}
}

func TestInspectDefaultBranchFlag(t *testing.T) {
	r := committedRepo(t)
	r.Git("checkout", "-q", "-b", "trunk")
	commitSquashed(t, r)
	r.Git("checkout", "-q", "main")
	env, _ := inspectEnv(t)
	code, so, _ := runInspect(t, r.Dir, env, "--json", "--default-branch", "trunk")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	if !hasChangeFinding(t, so, "no-task-commits", "add-x") {
		t.Errorf("no no-task-commits finding: %s", so)
	}
	_, so, _ = runInspect(t, r.Dir, env, "--json")
	for _, f := range decode(t, so)["findings"].([]any) {
		if f.(map[string]any)["change"] == "add-x" {
			t.Errorf("finding names add-x: %s", so)
		}
	}
}
