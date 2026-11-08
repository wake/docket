package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wake/docket/internal/testutil"
)

func regEnv(env []string, kv ...string) []string { return append(append([]string{}, env...), kv...) }

func regHome(env []string) string {
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "HOME="); ok {
			return v
		}
	}
	return ""
}

func regSettings(t *testing.T, env []string, content string) string {
	t.Helper()
	p := filepath.Join(regHome(env), ".config", "docket", "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// regBaseline runs args in a fresh environment with registration off.
func regBaseline(t *testing.T, dir string, args ...string) (int, string) {
	t.Helper()
	env := testutil.Env(t)
	regSettings(t, env, `{"projects":{"autoRegister":false}}`)
	code, out, _ := runIn(t, dir, env, args...)
	return code, out
}

func regRun(t *testing.T, env []string, dir string, args ...string) (int, string, string) {
	t.Helper()
	return runIn(t, dir, env, args...)
}

func regIndex(t *testing.T, env []string) string {
	t.Helper()
	b, err := os.ReadFile(prIndexPath(env))
	if err != nil {
		return ""
	}
	return string(b)
}

func regSame(t *testing.T, code int, out string, dir string, args ...string) {
	t.Helper()
	bc, bo := regBaseline(t, dir, args...)
	if code != bc || out != bo {
		t.Errorf("code=%d want %d; stdout differs: %v", code, bc, out != bo)
	}
}

func regWarnings(stderr string) []string {
	var w []string
	for _, l := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(l, "dkt: project index:") {
			w = append(w, l)
		}
	}
	return w
}

func TestRegisterFirstCommand(t *testing.T) {
	r := prRepo(t, true)
	code, out, errs := regRun(t, r.Env, r.Dir, "status", "--json")
	regSame(t, code, out, r.Dir, "status", "--json")
	if strings.Contains(errs, "project index") {
		t.Errorf("stderr %q", errs)
	}
	idx := regIndex(t, r.Env)
	if !strings.Contains(idx, prReal(t, r.Dir)) || !strings.Contains(idx, `"ignored": false`) {
		t.Errorf("index %q", idx)
	}
}

func TestRegisterAlreadyRegistered(t *testing.T) {
	r := prRepo(t, true)
	if code, _, _, _ := prRun(t, r.Env, r.Dir, "add"); code != 0 {
		t.Fatal(code)
	}
	p := prIndexPath(r.Env)
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	before := regIndex(t, r.Env)
	regRun(t, r.Env, r.Dir, "lint", "--json")
	fi, _ := os.Stat(p)
	if regIndex(t, r.Env) != before || fi.ModTime().After(old.Add(time.Minute)) {
		t.Error("index rewritten")
	}
}

func TestRegisterFromLinkedWorktree(t *testing.T) {
	r := prRepo(t, true)
	prRun(t, r.Env, r.Dir, "add")
	before := regIndex(t, r.Env)
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "-b", "other", wt)
	regRun(t, r.Env, wt, "status", "--json")
	if regIndex(t, r.Env) != before {
		t.Errorf("index changed: %q", regIndex(t, r.Env))
	}
}

func TestRegisterFromLinkedWorktreeFirst(t *testing.T) {
	r := prRepo(t, true)
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "-b", "other", wt)
	regRun(t, r.Env, wt, "status", "--json")
	idx := regIndex(t, r.Env)
	main, _ := filepath.EvalSymlinks(r.Dir)
	linked, _ := filepath.EvalSymlinks(wt)
	if strings.Count(idx, `"path"`) != 1 || !strings.Contains(idx, main) || strings.Contains(idx, linked) {
		t.Errorf("index %q", idx)
	}
}

func TestRegisterIgnoredKept(t *testing.T) {
	r := prRepo(t, true)
	prRun(t, r.Env, r.Dir, "ignore", r.Dir)
	before := regIndex(t, r.Env)
	regRun(t, r.Env, r.Dir, "status", "--json")
	if before == "" || regIndex(t, r.Env) != before {
		t.Errorf("index %q", regIndex(t, r.Env))
	}
}

func TestRegisterNoDocket(t *testing.T) {
	r := prRepo(t, false)
	regRun(t, r.Env, r.Dir, "status", "--json")
	if regIndex(t, r.Env) != "" {
		t.Error("index written")
	}
}

func TestRegisterOff(t *testing.T) {
	r := prRepo(t, true)
	regSettings(t, r.Env, `{"projects":{"autoRegister":false}}`)
	regRun(t, r.Env, r.Dir, "status", "--json")
	if regIndex(t, r.Env) != "" {
		t.Error("index written")
	}
}

func TestRegisterInit(t *testing.T) {
	r := prRepo(t, false)
	code, _, out, _ := doInit(t, r, r.Dir, "1.14.0", nil, "--tools", "claude")
	if code != 0 {
		t.Fatal(code, out)
	}
	if !strings.Contains(regIndex(t, r.Env), prReal(t, r.Dir)) {
		t.Errorf("index %q", regIndex(t, r.Env))
	}
}

func TestRegisterNotForVersionHelpUsage(t *testing.T) {
	r := prRepo(t, true)
	for _, args := range [][]string{{"--version"}, {"--help"}, {"status", "--help"}, {"status", "--bogus"}} {
		regRun(t, r.Env, r.Dir, args...)
		if regIndex(t, r.Env) != "" {
			t.Errorf("%v wrote the index", args)
		}
	}
}

func TestRegisterNotForProjects(t *testing.T) {
	r := prRepo(t, true)
	prRun(t, r.Env, r.Dir, "add")
	if code, _, _, _ := prRun(t, r.Env, r.Dir, "rm", r.Dir); code != 0 {
		t.Fatal(code)
	}
	if strings.Contains(regIndex(t, r.Env), "path") {
		t.Errorf("index %q", regIndex(t, r.Env))
	}
}

func TestRegisterBadSettings(t *testing.T) {
	r := prRepo(t, true)
	p := regSettings(t, r.Env, "{")
	code, out, errs := regRun(t, r.Env, r.Dir, "status", "--json")
	regSame(t, code, out, r.Dir, "status", "--json")
	w := regWarnings(errs)
	if len(w) != 1 || !strings.Contains(w[0], p) || regIndex(t, r.Env) != "" {
		t.Errorf("warnings %v", w)
	}
}

func TestRegisterUnwritable(t *testing.T) {
	r := prRepo(t, true)
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, []byte("x"), 0o644)
	env := regEnv(r.Env, "XDG_STATE_HOME="+filepath.Join(file, "state"))
	code, out, errs := regRun(t, env, r.Dir, "status", "--json")
	regSame(t, code, out, r.Dir, "status", "--json")
	w := regWarnings(errs)
	if len(w) != 1 || !strings.Contains(w[0], filepath.Join(file, "state", "docket", "projects.json")) {
		t.Errorf("warnings %v", w)
	}
}

func TestRegisterBadIndex(t *testing.T) {
	r := prRepo(t, true)
	p := prIndexPath(r.Env)
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte("not json"), 0o600)
	code, out, errs := regRun(t, r.Env, r.Dir, "status", "--json")
	regSame(t, code, out, r.Dir, "status", "--json")
	w := regWarnings(errs)
	b, _ := os.ReadFile(p)
	if len(w) != 1 || !strings.Contains(w[0], p) || string(b) != "not json" {
		t.Errorf("warnings %v, index %q", w, b)
	}
}

func TestRegisterFailingCommand(t *testing.T) {
	r := prRepo(t, true)
	code, _, _ := regRun(t, r.Env, r.Dir, "sync", "add-none", "--json")
	if code != 1 || !strings.Contains(regIndex(t, r.Env), prReal(t, r.Dir)) {
		t.Errorf("code=%d index %q", code, regIndex(t, r.Env))
	}
}

func TestRegisterNoHome(t *testing.T) {
	r := prRepo(t, true)
	var env []string
	for _, kv := range r.Env {
		if !strings.HasPrefix(kv, "HOME=") {
			env = append(env, kv)
		}
	}
	code, out, errs := regRun(t, env, r.Dir, "status", "--json")
	regSame(t, code, out, r.Dir, "status", "--json")
	w := regWarnings(errs)
	if len(w) != 1 || !strings.Contains(w[0], "HOME") {
		t.Errorf("warnings %v", w)
	}
}

func TestRegisterNotAfterCommandUsageError(t *testing.T) {
	r := prRepo(t, true)
	code, _, _ := regRun(t, r.Env, r.Dir, "new", "add-x", "--json")
	if code != 2 || regIndex(t, r.Env) != "" {
		t.Errorf("code=%d index %q", code, regIndex(t, r.Env))
	}
}

func TestRegisterNoDocketNoWarning(t *testing.T) {
	r := prRepo(t, false)
	regSettings(t, r.Env, "{")
	_, _, errs := regRun(t, r.Env, r.Dir, "status", "--json")
	if w := regWarnings(errs); len(w) != 0 || regIndex(t, r.Env) != "" {
		t.Errorf("warnings %v index %q", w, regIndex(t, r.Env))
	}
}

func TestRegisterNoStateHome(t *testing.T) {
	r := prRepo(t, true)
	var env []string
	for _, kv := range r.Env {
		if !strings.HasPrefix(kv, "HOME=") {
			env = append(env, kv)
		}
	}
	env = append(env, "XDG_CONFIG_HOME="+t.TempDir())
	_, _, errs := regRun(t, env, r.Dir, "status", "--json")
	w := regWarnings(errs)
	if len(w) != 1 || !strings.Contains(w[0], "XDG_STATE_HOME") {
		t.Errorf("warnings %v", w)
	}
}

func TestInspectInNewProject(t *testing.T) {
	r := prRepo(t, true)
	inspect(t, r, "1.14.0", false)
	if out := r.Git("status", "--porcelain"); out != "" {
		t.Errorf("status %q", out)
	}
	if !strings.Contains(regIndex(t, r.Env), prReal(t, r.Dir)) {
		t.Errorf("index %q", regIndex(t, r.Env))
	}
}
