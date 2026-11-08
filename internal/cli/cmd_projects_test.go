package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wake/docket/internal/testutil"
)

type projectsOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Repo       *Repo
	Result     struct {
		State   string
		Project struct {
			Path    string
			Ignored bool
			Added   string
		}
		Projects []struct {
			Path             string
			Ignored, Missing bool
		}
	}
}

// prRun runs `projects <args> --json` with a fixed clock.
func prRun(t *testing.T, env []string, dir string, args ...string) (int, projectsOut, string, string) {
	t.Helper()
	now := func() time.Time { return time.Date(2026, 11, 8, 17, 10, 11, 500_000_000, time.FixedZone("x", 8*3600)) }
	var out, errb bytes.Buffer
	code := Run(Options{Args: append(append([]string{"projects"}, args...), "--json"), Dir: dir, Env: env, Stdout: &out, Stderr: &errb, Now: now})
	var m projectsOut
	if out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &m); err != nil {
			t.Fatalf("%q: %v", out.String(), err)
		}
	}
	return code, m, out.String(), errb.String()
}

func prRepo(t *testing.T, docket bool) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	files := map[string]string{"README.md": "x"}
	if docket {
		files["docket/workflow.md"] = "# w\n"
	}
	r.Commit("base", files)
	return r
}

func prIndexPath(env []string) string {
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "XDG_STATE_HOME="); ok {
			return filepath.Join(v, "docket", "projects.json")
		}
	}
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "HOME="); ok {
			return filepath.Join(v, ".local", "state", "docket", "projects.json")
		}
	}
	return ""
}

func prReal(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func prWrite(t *testing.T, env []string, body string) string {
	t.Helper()
	p := prIndexPath(env)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProjectsAddNew(t *testing.T) {
	r := prRepo(t, true)
	code, m, raw, _ := prRun(t, r.Env, r.Dir, "add")
	if code != 0 || m.Result.State != "added" {
		t.Fatalf("%d %s", code, raw)
	}
	p := m.Result.Project
	if p.Path != prReal(t, r.Dir) || p.Ignored || p.Added != "2026-11-08T09:10:11Z" {
		t.Fatalf("%+v", p)
	}
	data, err := os.ReadFile(prIndexPath(r.Env))
	if err != nil || !strings.Contains(string(data), p.Path) {
		t.Fatalf("%v %s", err, data)
	}
}

func TestProjectsAddUnderStateHome(t *testing.T) {
	r := prRepo(t, true)
	s := filepath.Join(t.TempDir(), "s")
	env := append(append([]string{}, r.Env...), "XDG_STATE_HOME="+s)
	if code, _, raw, _ := prRun(t, env, r.Dir, "add"); code != 0 {
		t.Fatalf("%d %s", code, raw)
	}
	if _, err := os.Stat(filepath.Join(s, "docket", "projects.json")); err != nil {
		t.Fatal(err)
	}
}

func TestProjectsAddExists(t *testing.T) {
	r := prRepo(t, true)
	prRun(t, r.Env, r.Dir, "add")
	idx := prIndexPath(r.Env)
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(idx, old, old); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(idx)
	code, m, raw, _ := prRun(t, r.Env, r.Dir, "add")
	after, _ := os.ReadFile(idx)
	fi, _ := os.Stat(idx)
	if code != 0 || m.Result.State != "exists" || !bytes.Equal(before, after) || !fi.ModTime().Equal(old) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestProjectsAddUnignores(t *testing.T) {
	r := prRepo(t, true)
	prWrite(t, r.Env, `{"projects":[{"path":"`+prReal(t, r.Dir)+`","origin":null,"added":"2026-01-01T00:00:00Z","ignored":true}]}`)
	code, m, raw, _ := prRun(t, r.Env, r.Dir, "add")
	if code != 0 || m.Result.State != "unignored" || m.Result.Project.Ignored || m.Result.Project.Added != "2026-01-01T00:00:00Z" {
		t.Fatalf("%d %s", code, raw)
	}
	data, _ := os.ReadFile(prIndexPath(r.Env))
	if strings.Contains(string(data), `"ignored": true`) {
		t.Fatalf("%s", data)
	}
}

func TestProjectsAddNotProject(t *testing.T) {
	r := prRepo(t, false)
	code, m, raw, _ := prRun(t, r.Env, r.Dir, "add")
	if code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != "not-a-project" {
		t.Fatalf("%d %s", code, raw)
	}
	if _, err := os.Stat(prIndexPath(r.Env)); err == nil {
		t.Fatal("index created")
	}
}

func TestProjectsAddNotRepository(t *testing.T) {
	env := testutil.Env(t)
	code, m, raw, _ := prRun(t, env, t.TempDir(), "add", t.TempDir())
	if code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != "not-a-repository" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestProjectsUnreadableIndex(t *testing.T) {
	r := prRepo(t, true)
	idx := prWrite(t, r.Env, "not json")
	code, m, raw, _ := prRun(t, r.Env, r.Dir, "add")
	data, _ := os.ReadFile(idx)
	if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != idx || string(data) != "not json" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestProjectsLs(t *testing.T) {
	env := testutil.Env(t)
	dir := t.TempDir()
	gone := filepath.Join(dir, "gone")
	body := `{"projects":[{"path":"` + gone + `","origin":null,"added":"2026-01-01T00:00:00Z","ignored":false},` +
		`{"path":"` + dir + `","origin":null,"added":"2026-01-01T00:00:00Z","ignored":true}]}`
	idx := prWrite(t, env, body)
	before, _ := os.ReadFile(idx)
	code, m, raw, _ := prRun(t, env, dir, "ls")
	ps := m.Result.Projects
	if code != 0 || len(ps) != 2 || ps[0].Path != dir || !ps[0].Ignored || ps[0].Missing || ps[1].Path != gone || ps[1].Ignored || !ps[1].Missing {
		t.Fatalf("%d %s", code, raw)
	}
	after, _ := os.ReadFile(idx)
	if !bytes.Equal(before, after) {
		t.Fatal("index changed")
	}
	var out bytes.Buffer
	Run(Options{Args: []string{"projects", "ls"}, Dir: dir, Env: env, Stdout: &out, Stderr: &bytes.Buffer{}})
	if want := dir + "  ignored\n" + gone + "  missing\n"; out.String() != want {
		t.Fatalf("%q want %q", out.String(), want)
	}
}

func TestProjectsLsUncheckable(t *testing.T) {
	skipRoot(t)
	env := testutil.Env(t)
	locked := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	app := filepath.Join(locked, "app")
	prWrite(t, env, `{"projects":[{"path":"`+app+`","origin":null,"added":"2026-01-01T00:00:00Z","ignored":false}]}`)
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	code, m, raw, _ := prRun(t, env, t.TempDir(), "ls")
	if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != app || len(m.Result.Projects) != 1 || m.Result.Projects[0].Missing {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestProjectsLsEmpty(t *testing.T) {
	env := testutil.Env(t)
	code, _, raw, _ := prRun(t, env, t.TempDir(), "ls")
	if code != 0 || !strings.Contains(raw, `"projects":[]`) {
		t.Fatalf("%d %s", code, raw)
	}
	if _, err := os.Stat(prIndexPath(env)); err == nil {
		t.Fatal("index created")
	}
}

func TestProjectsLsOutsideRepo(t *testing.T) {
	code, m, raw, _ := prRun(t, testutil.Env(t), t.TempDir(), "ls")
	if code != 0 || m.Repo != nil || !strings.Contains(raw, `"repo":null`) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestProjectsUsage(t *testing.T) {
	env := testutil.Env(t)
	for _, a := range [][]string{{}, {"prune"}, {"ls", "x"}, {"add", "a", "b"}, {"rm"}} {
		code, m, raw, _ := prRun(t, env, t.TempDir(), a...)
		if code != 2 || len(m.Findings) != 1 || m.Findings[0].Rule != "usage" {
			t.Fatalf("%v: %d %s", a, code, raw)
		}
	}
}

func TestProjectsNoHome(t *testing.T) {
	var env []string
	for _, kv := range testutil.Env(t) {
		if strings.HasPrefix(kv, "PATH=") {
			env = append(env, kv)
		}
	}
	code, m, raw, _ := prRun(t, env, t.TempDir(), "ls")
	if code != 4 || len(m.Findings) != 1 || m.Findings[0].Rule != "no-home" {
		t.Fatalf("%d %s", code, raw)
	}
}

func prEntry(path string, ignored bool) string {
	i := "false"
	if ignored {
		i = "true"
	}
	return `{"path":"` + path + `","origin":null,"added":"2026-01-01T00:00:00Z","ignored":` + i + `}`
}

func TestProjectsRm(t *testing.T) {
	r := prRepo(t, true)
	prRun(t, r.Env, r.Dir, "add")
	code, m, raw, _ := prRun(t, r.Env, r.Dir, "rm", r.Dir)
	data, _ := os.ReadFile(prIndexPath(r.Env))
	if code != 0 || m.Result.State != "removed" || m.Result.Project.Path != prReal(t, r.Dir) || strings.Contains(string(data), m.Result.Project.Path) {
		t.Fatalf("%d %s %s", code, raw, data)
	}
}

func TestProjectsRmSubdirectory(t *testing.T) {
	r := prRepo(t, true)
	r.Commit("sub", map[string]string{"internal/x.go": "x"})
	prRun(t, r.Env, r.Dir, "add")
	code, m, raw, _ := prRun(t, r.Env, r.Dir, "rm", filepath.Join(r.Dir, "internal"))
	if code != 0 || m.Result.State != "removed" || m.Result.Project.Path != prReal(t, r.Dir) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestProjectsRmGone(t *testing.T) {
	env := testutil.Env(t)
	tmp := t.TempDir()
	old := filepath.Join(tmp, "old")
	for _, arg := range []string{old + "/", filepath.Join(tmp, "x", "..", "old")} {
		prWrite(t, env, `{"projects":[`+prEntry(old, false)+`]}`)
		code, m, raw, _ := prRun(t, env, tmp, "rm", arg)
		if code != 0 || m.Result.State != "removed" {
			t.Fatalf("%s: %d %s", arg, code, raw)
		}
	}
}

func TestProjectsRmUnknown(t *testing.T) {
	env := testutil.Env(t)
	tmp := t.TempDir()
	idx := prWrite(t, env, `{"projects":[`+prEntry(filepath.Join(tmp, "a"), false)+`]}`)
	before, _ := os.ReadFile(idx)
	code, m, raw, _ := prRun(t, env, tmp, "rm", filepath.Join(tmp, "none"))
	after, _ := os.ReadFile(idx)
	if code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != "not-in-index" || !bytes.Equal(before, after) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestProjectsRmGitMissing(t *testing.T) {
	var env []string
	for _, kv := range testutil.Env(t) {
		if !strings.HasPrefix(kv, "PATH=") {
			env = append(env, kv)
		}
	}
	env = append(env, "PATH="+t.TempDir())
	tmp := t.TempDir()
	idx := prWrite(t, env, `{"projects":[`+prEntry(tmp, false)+`]}`)
	before, _ := os.ReadFile(idx)
	code, m, raw, _ := prRun(t, env, tmp, "rm", tmp) // an existing directory, so git is asked
	after, _ := os.ReadFile(idx)
	if code != 4 || len(m.Findings) != 1 || m.Findings[0].Rule != "git-missing" || !bytes.Equal(before, after) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestProjectsIgnoreRegistered(t *testing.T) {
	r := prRepo(t, true)
	prRun(t, r.Env, r.Dir, "add")
	code, m, raw, _ := prRun(t, r.Env, r.Dir, "ignore", r.Dir)
	if code != 0 || m.Result.State != "ignored" || !m.Result.Project.Ignored {
		t.Fatalf("%d %s", code, raw)
	}
	before, _ := os.ReadFile(prIndexPath(r.Env))
	code, m, raw, _ = prRun(t, r.Env, r.Dir, "ignore", r.Dir)
	after, _ := os.ReadFile(prIndexPath(r.Env))
	if code != 0 || m.Result.State != "exists" || !bytes.Equal(before, after) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestProjectsIgnoreNew(t *testing.T) {
	r := prRepo(t, true)
	r.Git("remote", "add", "origin", "https://example.com/a/b.git")
	code, m, raw, _ := prRun(t, r.Env, r.Dir, "ignore", r.Dir)
	data, _ := os.ReadFile(prIndexPath(r.Env))
	if code != 0 || m.Result.State != "ignored" || !m.Result.Project.Ignored || m.Result.Project.Added != "2026-11-08T09:10:11Z" ||
		!strings.Contains(string(data), "example.com/a/b") {
		t.Fatalf("%d %s %s", code, raw, data)
	}
}

func TestProjectsIgnoreNotRepository(t *testing.T) {
	env := testutil.Env(t)
	dir := t.TempDir()
	code, m, raw, _ := prRun(t, env, dir, "ignore", dir)
	if code != 0 || m.Result.State != "ignored" || m.Result.Project.Path != dir || !strings.Contains(raw, `"origin":null`) {
		t.Fatalf("%d %s", code, raw)
	}
}
