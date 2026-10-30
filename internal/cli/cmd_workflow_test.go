package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/testutil"
)

const wfFour = "## Full review\n## Light review\n## Check\n## Direct merge\n"

func wfFM(h string) string { return "---\ntemplate: " + h + "\n---\n" }

func wfHash() string {
	sum := sha256.Sum256([]byte(docket.Template()))
	return hex.EncodeToString(sum[:])[:12]
}

// runWorkflow runs dkt workflow in a repository with testutil.Env plus extra.
func runWorkflow(t *testing.T, home func() (string, error), extra []string, args ...string) (int, string, string) {
	t.Helper()
	repo := testutil.NewRepo(t)
	var so, se bytes.Buffer
	code := Run(Options{
		Args: args, Dir: repo.Dir, Env: append(testutil.Env(t), extra...),
		UserHome: home, Stdout: &so, Stderr: &se,
	})
	return code, so.String(), se.String()
}

func wfResult(t *testing.T, so string) map[string]any {
	t.Helper()
	r, ok := decode(t, so)["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not an object: %s", so)
	}
	return r
}

func wfKeys(m map[string]any) []string {
	var k []string
	for key := range m {
		k = append(k, key)
	}
	sort.Strings(k)
	return k
}

func wfFindings(t *testing.T, so string) []map[string]any {
	t.Helper()
	raw, _ := decode(t, so)["findings"].([]any)
	var out []map[string]any
	for _, f := range raw {
		out = append(out, f.(map[string]any))
	}
	return out
}

var wfFiveKeys = []string{"missing_sections", "path", "recorded", "status", "template"}

// wfConfig returns a config home x and the workflow path under it.
func wfConfig(t *testing.T) (xdgEnv []string, path string) {
	t.Helper()
	x := t.TempDir()
	return []string{"XDG_CONFIG_HOME=" + x}, filepath.Join(x, "docket", "workflow.md")
}

func wfWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowPathInConfigHome(t *testing.T) {
	h := t.TempDir()
	code, so, _ := runWorkflow(t, nil, []string{"HOME=" + h}, "workflow", "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	if got, want := wfResult(t, so)["path"], filepath.Join(h, ".config", "docket", "workflow.md"); got != want {
		t.Errorf("path = %v, want %v", got, want)
	}
}

func TestWorkflowPathUnderXDG(t *testing.T) {
	x := t.TempDir()
	_, so, _ := runWorkflow(t, nil, []string{"XDG_CONFIG_HOME=" + x}, "workflow", "--json")
	if got, want := wfResult(t, so)["path"], filepath.Join(x, "docket", "workflow.md"); got != want {
		t.Errorf("path = %v, want %v", got, want)
	}
}

func TestWorkflowXDGRelativeIgnored(t *testing.T) {
	h := t.TempDir()
	_, so, _ := runWorkflow(t, nil, []string{"XDG_CONFIG_HOME=rel", "HOME=" + h}, "workflow", "--json")
	if got, want := wfResult(t, so)["path"], filepath.Join(h, ".config", "docket", "workflow.md"); got != want {
		t.Errorf("path = %v, want %v", got, want)
	}
}

func TestWorkflowTemplatePrinted(t *testing.T) {
	code, so, _ := runWorkflow(t, nil, nil, "workflow", "template", "--json")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, so)
	}
	r := wfResult(t, so)
	if r["text"] != docket.Template() || r["template"] != wfHash() {
		t.Errorf("result = %v", r)
	}
	if !reflect.DeepEqual(wfKeys(r), []string{"template", "text"}) {
		t.Errorf("keys = %v", wfKeys(r))
	}
	code, so, _ = runWorkflow(t, nil, nil, "workflow", "template")
	if code != 0 || so != docket.Template()+"dkt workflow: ok\n" {
		t.Errorf("text mode: exit %d, stdout %q", code, so)
	}
}

func TestWorkflowMissing(t *testing.T) {
	env, path := wfConfig(t)
	code, so, _ := runWorkflow(t, nil, env, "workflow", "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	r := wfResult(t, so)
	if !reflect.DeepEqual(wfKeys(r), wfFiveKeys) {
		t.Errorf("keys = %v", wfKeys(r))
	}
	if r["status"] != "missing" || r["recorded"] != nil || r["template"] != wfHash() || r["path"] != path {
		t.Errorf("result = %v", r)
	}
	if ms, ok := r["missing_sections"].([]any); !ok || len(ms) != 0 {
		t.Errorf("missing_sections = %#v", r["missing_sections"])
	}
	fs := wfFindings(t, so)
	if len(fs) != 1 || fs[0]["rule"] != "workflow-missing" || fs[0]["file"] != path {
		t.Errorf("findings = %v", fs)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("config directory was created: %v", err)
	}
}

func TestWorkflowComplete(t *testing.T) {
	env, path := wfConfig(t)
	wfWrite(t, path, wfFM(wfHash())+wfFour)
	code, so, _ := runWorkflow(t, nil, env, "workflow", "--json")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, so)
	}
	r := wfResult(t, so)
	if !reflect.DeepEqual(wfKeys(r), wfFiveKeys) {
		t.Errorf("keys = %v", wfKeys(r))
	}
	if r["status"] != "ok" || r["recorded"] != wfHash() {
		t.Errorf("result = %v", r)
	}
	if fs := wfFindings(t, so); len(fs) != 0 {
		t.Errorf("findings = %v", fs)
	}
	_, so, _ = runWorkflow(t, nil, env, "workflow")
	if !strings.HasSuffix(so, "dkt workflow: ok\n") {
		t.Errorf("text mode: %q", so)
	}
}

func TestWorkflowSectionMissing(t *testing.T) {
	env, path := wfConfig(t)
	wfWrite(t, path, wfFM("0123456789ab")+"## Full review\n## Light review\n## Direct merge\n")
	code, so, _ := runWorkflow(t, nil, env, "workflow", "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	r := wfResult(t, so)
	if r["status"] != "incomplete" || !reflect.DeepEqual(r["missing_sections"], []any{"Check"}) {
		t.Errorf("result = %v", r)
	}
	fs := wfFindings(t, so)
	if len(fs) != 1 || fs[0]["rule"] != "workflow-incomplete" || fs[0]["file"] != path ||
		!strings.Contains(fs[0]["message"].(string), "## Check") {
		t.Errorf("findings = %v", fs)
	}

	t.Run("two", func(t *testing.T) {
		env, path := wfConfig(t)
		wfWrite(t, path, wfFM("0123456789ab")+"## Full review\n## Light review\n")
		_, so, _ := runWorkflow(t, nil, env, "workflow", "--json")
		if got := wfResult(t, so)["missing_sections"]; !reflect.DeepEqual(got, []any{"Check", "Direct merge"}) {
			t.Errorf("missing_sections = %v", got)
		}
		fs := wfFindings(t, so)
		if len(fs) != 2 {
			t.Fatalf("findings = %v", fs)
		}
		for i, name := range []string{"## Check", "## Direct merge"} {
			if fs[i]["rule"] != "workflow-incomplete" || !strings.Contains(fs[i]["message"].(string), name) {
				t.Errorf("finding %d = %v", i, fs[i])
			}
		}
	})
}

func TestWorkflowStale(t *testing.T) {
	env, path := wfConfig(t)
	wfWrite(t, path, wfFM("0123456789ab")+wfFour)
	code, so, _ := runWorkflow(t, nil, env, "workflow", "--json")
	if code != 1 {
		t.Fatalf("exit %d: %s", code, so)
	}
	r := wfResult(t, so)
	if r["status"] != "stale" || r["recorded"] != "0123456789ab" || r["template"] != wfHash() {
		t.Errorf("result = %v", r)
	}
	fs := wfFindings(t, so)
	if len(fs) != 1 || fs[0]["rule"] != "workflow-stale" || fs[0]["file"] != r["path"] {
		t.Errorf("findings = %v", fs)
	}
}

func TestWorkflowNoFrontmatter(t *testing.T) {
	env, path := wfConfig(t)
	wfWrite(t, path, wfFour)
	_, so, _ := runWorkflow(t, nil, env, "workflow", "--json")
	r := wfResult(t, so)
	if r["status"] != "stale" || r["recorded"] != nil {
		t.Errorf("result = %v", r)
	}
	fs := wfFindings(t, so)
	if len(fs) != 1 || fs[0]["rule"] != "workflow-stale" || fs[0]["file"] != r["path"] {
		t.Errorf("findings = %v", fs)
	}
}

func TestWorkflowUnreadable(t *testing.T) {
	check := func(t *testing.T, so, path string, code int) {
		t.Helper()
		if code != 3 {
			t.Fatalf("exit %d: %s", code, so)
		}
		m := decode(t, so)
		u, _ := m["unreadable"].([]any)
		if len(u) != 1 || u[0].(map[string]any)["source"] != path {
			t.Errorf("unreadable = %v", m["unreadable"])
		}
		if m["result"] != nil {
			t.Errorf("result = %v", m["result"])
		}
	}
	t.Run("mode 000", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root reads any file")
		}
		env, path := wfConfig(t)
		wfWrite(t, path, wfFM(wfHash())+wfFour)
		if err := os.Chmod(path, 0); err != nil {
			t.Fatal(err)
		}
		code, so, _ := runWorkflow(t, nil, env, "workflow", "--json")
		check(t, so, path, code)
	})
	t.Run("directory", func(t *testing.T) {
		env, path := wfConfig(t)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		code, so, _ := runWorkflow(t, nil, env, "workflow", "--json")
		check(t, so, path, code)
	})
}

func TestWorkflowBrokenLink(t *testing.T) {
	env, path := wfConfig(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone.md"), path); err != nil {
		t.Fatal(err)
	}
	code, so, _ := runWorkflow(t, nil, env, "workflow", "--json")
	if code != 3 {
		t.Fatalf("exit %d: %s", code, so)
	}
	u, _ := decode(t, so)["unreadable"].([]any)
	if len(u) != 1 || u[0].(map[string]any)["source"] != path {
		t.Errorf("unreadable = %v", u)
	}
}

func TestWorkflowBrokenParentLink(t *testing.T) {
	env, _ := wfConfig(t)
	x := t.TempDir()
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone"), filepath.Join(x, "docket")); err != nil {
		t.Fatal(err)
	}
	var e []string
	for _, v := range env {
		if !strings.HasPrefix(v, "XDG_CONFIG_HOME=") {
			e = append(e, v)
		}
	}
	e = append(e, "XDG_CONFIG_HOME="+x)
	code, so, _ := runWorkflow(t, nil, e, "workflow", "--json")
	if code != 3 {
		t.Fatalf("exit %d: %s", code, so)
	}
	m := decode(t, so)
	u, _ := m["unreadable"].([]any)
	if len(u) != 1 || u[0].(map[string]any)["source"] != filepath.Join(x, "docket", "workflow.md") {
		t.Errorf("unreadable = %v", u)
	}
	if m["result"] != nil {
		t.Errorf("result = %v", m["result"])
	}
}

func TestWorkflowThroughLink(t *testing.T) {
	env, path := wfConfig(t)
	target := filepath.Join(t.TempDir(), "real.md")
	wfWrite(t, target, wfFM(wfHash())+wfFour)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	code, so, _ := runWorkflow(t, nil, env, "workflow", "--json")
	if code != 0 || wfResult(t, so)["status"] != "ok" {
		t.Errorf("exit %d: %s", code, so)
	}
}

func TestWorkflowNoHome(t *testing.T) {
	repo := testutil.NewRepo(t)
	var env []string
	for _, e := range testutil.Env(t) {
		if !strings.HasPrefix(e, "HOME=") && !strings.HasPrefix(e, "XDG_CONFIG_HOME=") {
			env = append(env, e)
		}
	}
	run := func(home func() (string, error)) (int, string) {
		var so, se bytes.Buffer
		code := Run(Options{Args: []string{"workflow", "--json"}, Dir: repo.Dir, Env: env, UserHome: home, Stdout: &so, Stderr: &se})
		return code, so.String()
	}
	code, so := run(nil)
	if code != 4 {
		t.Fatalf("exit %d: %s", code, so)
	}
	if fs := wfFindings(t, so); len(fs) != 1 || fs[0]["rule"] != "environment" {
		t.Errorf("findings = %v", fs)
	}
	u := t.TempDir()
	_, so = run(func() (string, error) { return u, nil })
	if got, want := wfResult(t, so)["path"], filepath.Join(u, ".config", "docket", "workflow.md"); got != want {
		t.Errorf("path = %v, want %v", got, want)
	}
}

func TestWorkflowUsage(t *testing.T) {
	for _, args := range [][]string{{"workflow", "other"}, {"workflow", "template", "extra"}} {
		if code, so, _ := runWorkflow(t, nil, nil, args...); code != 2 {
			t.Errorf("%v: exit %d: %s", args, code, so)
		}
	}
}
