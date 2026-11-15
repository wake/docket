package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/projects"
	"github.com/wake/docket/internal/setup"
	"github.com/wake/docket/internal/testutil"
)

type refsOut struct {
	Repo       *Repo
	Findings   []Finding
	Unreadable []Unreadable
	Result     struct {
		Template int
		Sources  []struct {
			Path, Kind string
			Template   int
			Missing    []string
			LeftOut    []string `json:"left_out"`
		}
		Sections []struct {
			Heading, Template string
			Variants          []struct {
				Text           string
				SameAsTemplate bool `json:"same_as_template"`
				Sources        []string
			}
		}
	}
}

// refsEnv gives dkt its own config and state homes, so no real one is read.
func refsEnv(t *testing.T, base []string) (env []string, cfg, state string) {
	t.Helper()
	cfg, state = t.TempDir(), t.TempDir()
	return append(slices.Clone(base), "XDG_CONFIG_HOME="+cfg, "XDG_STATE_HOME="+state), cfg, state
}

// refsProject makes a registered-looking project directory holding a workflow file; its real path is returned.
func refsProject(t *testing.T, workflow string) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if workflow != "" {
		if err := os.MkdirAll(filepath.Join(d, "docket"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "docket", "workflow.md"), []byte(workflow), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

type refEntry struct {
	path    string
	ignored bool
}

func refsIndex(t *testing.T, state string, entries ...refEntry) {
	t.Helper()
	x := &projects.Index{Projects: []projects.Entry{}}
	for _, e := range entries {
		x.Projects = append(x.Projects, projects.Entry{Path: e.path, Added: "2026-11-01T00:00:00Z", Ignored: e.ignored})
	}
	if err := x.Save(filepath.Join(state, "docket", "projects.json")); err != nil {
		t.Fatal(err)
	}
}

func refsRun(t *testing.T, dir string, env []string, args ...string) (int, refsOut, string) {
	t.Helper()
	code, out, _ := runIn(t, dir, env, append([]string{"workflow"}, append(args, "--json")...)...)
	var m refsOut
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("%q: %v", out, err)
	}
	return code, m, out
}

// wf builds a workflow file with the given level-three sections under one level-two heading.
func wf(template string, sections ...string) string {
	s := "---\ntemplate: " + template + "\n---\n\n## Part\n"
	for i := 0; i+1 < len(sections); i += 2 {
		s += "\n### " + sections[i] + "\n\n" + sections[i+1] + "\n"
	}
	return s
}

func shippedSections(t *testing.T) []setup.Section {
	t.Helper()
	return setup.Sections([]byte(shippedWorkflow(t)))
}

func (m refsOut) variants(heading string) map[string][]string {
	out := map[string][]string{}
	for _, s := range m.Result.Sections {
		if s.Heading == heading {
			for _, v := range s.Variants {
				out[v.Text] = v.Sources
			}
		}
	}
	return out
}

func TestWorkflowRefsVariants(t *testing.T) {
	env, cfg, state := refsEnv(t, testutil.Env(t))
	ver := shippedVersion(t)
	tpl := shippedSections(t)[0] // the first shipped section, "Plan review"
	a := refsProject(t, wf("3", tpl.Heading, "Same way"))
	b := refsProject(t, strings.ReplaceAll(wf("3", tpl.Heading, "Same way  \n\n"), "\n", "\r\n")+"\r\n\r\n")
	c := refsProject(t, wf("3", tpl.Heading, "Another way"))
	refsIndex(t, state, refEntry{path: a}, refEntry{path: b}, refEntry{path: c})
	userDir := filepath.Join(cfg, "docket")
	os.MkdirAll(userDir, 0o755)
	user := filepath.Join(userDir, "workflow.md")
	os.WriteFile(user, []byte(wf("3", tpl.Heading, tpl.Text)), 0o644)

	code, m, out := refsRun(t, t.TempDir(), env, "refs")
	if code != 0 || m.Repo != nil || m.Result.Template != ver {
		t.Fatalf("code=%d %s", code, out)
	}
	if len(m.Result.Sections) != len(shippedSections(t)) || m.Result.Sections[0].Heading != tpl.Heading || m.Result.Sections[0].Template != tpl.Text {
		t.Fatalf("sections: %s", out)
	}
	want := map[string][]string{"Same way": {a, b}, "Another way": {c}, tpl.Text: {user}}
	if got := m.variants(tpl.Heading); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	for _, v := range m.Result.Sections[0].Variants {
		if v.SameAsTemplate != (v.Text == tpl.Text) {
			t.Errorf("same_as_template of %q is %v", v.Text, v.SameAsTemplate)
		}
	}
	var kinds []string
	for _, s := range m.Result.Sources {
		kinds = append(kinds, s.Kind)
	}
	if !reflect.DeepEqual(kinds, []string{"user", "project", "project", "project"}) || m.Result.Sources[0].Path != user {
		t.Errorf("sources: %s", out)
	}
}

func TestWorkflowRefsNotReferences(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"docket/workflow.md": wf("3", "Plan review", "mine")})
	env, _, state := refsEnv(t, r.Env)
	self, err := filepath.EvalSymlinks(r.Dir)
	if err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "-b", "other", wt)
	ignored := refsProject(t, wf("3", "Plan review", "ignored"))
	gone := filepath.Join(t.TempDir(), "gone")
	bare := refsProject(t, "")
	fileDocket := t.TempDir() // docket is a file, so there is no docket/workflow.md
	if err := os.WriteFile(filepath.Join(fileDocket, "docket"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok := refsProject(t, wf("3", "Plan review", "kept"))
	refsIndex(t, state, refEntry{path: self}, refEntry{path: ignored, ignored: true}, refEntry{path: gone}, refEntry{path: bare}, refEntry{path: fileDocket}, refEntry{path: ok})
	for _, dir := range []string{r.Dir, wt} {
		code, m, out := refsRun(t, dir, env, "refs")
		if code != 0 || len(m.Result.Sources) != 1 || m.Result.Sources[0].Path != ok {
			t.Errorf("%s: code=%d %s", dir, code, out)
		}
	}
}

func TestWorkflowRefsOlderTemplate(t *testing.T) {
	env, _, state := refsEnv(t, testutil.Env(t))
	old := refsProject(t, wf("2", "Plan review", "kept", "Old step", "gone"))
	nofm := refsProject(t, "## Part\n\n### Plan review\n\nkept\n")
	refsIndex(t, state, refEntry{path: old}, refEntry{path: nofm})
	code, m, out := refsRun(t, t.TempDir(), env, "refs")
	if code != 0 || len(m.Result.Sources) != 2 {
		t.Fatalf("code=%d %s", code, out)
	}
	src := m.Result.Sources[0]
	if src.Path != old || src.Template != 2 || !reflect.DeepEqual(src.LeftOut, []string{"Old step"}) || !slices.Contains(src.Missing, "Reviewers") || slices.Contains(src.Missing, "Plan review") {
		t.Errorf("%+v", src)
	}
	if src := m.Result.Sources[1]; src.Path != nofm || src.Template != 0 {
		t.Errorf("%+v", src)
	}
	if got := m.variants("Plan review"); !reflect.DeepEqual(got, map[string][]string{"kept": {old, nofm}}) {
		t.Errorf("got %v", got)
	}
	if strings.Contains(out, "gone") {
		t.Errorf("the extra section was compared: %s", out)
	}
}

func TestWorkflowRefsUnreadable(t *testing.T) {
	env, _, state := refsEnv(t, testutil.Env(t))
	locked := refsProject(t, wf("3", "Plan review", "locked"))
	lp := filepath.Join(locked, "docket", "workflow.md")
	os.Chmod(lp, 0)
	t.Cleanup(func() { os.Chmod(lp, 0o644) })
	if f, err := os.Open(lp); err == nil {
		f.Close()
		t.Skip("file readable despite mode 0")
	}
	good := refsProject(t, wf("3", "Plan review", "good"))
	outside := filepath.Join(t.TempDir(), "outside.md")
	os.WriteFile(outside, []byte(wf("3", "Plan review", "outside")), 0o644)
	linked := refsProject(t, wf("3", "Plan review", "x"))
	lk := filepath.Join(linked, "docket", "workflow.md")
	os.Remove(lk)
	if err := os.Symlink(outside, lk); err != nil {
		t.Fatal(err)
	}
	refsIndex(t, state, refEntry{path: locked}, refEntry{path: good}, refEntry{path: linked})
	code, m, out := refsRun(t, t.TempDir(), env, "refs")
	var un []string
	for _, u := range m.Unreadable {
		un = append(un, u.Source)
	}
	slices.Sort(un)
	if code != 3 || !reflect.DeepEqual(un, []string{lk, lp}) && !reflect.DeepEqual(un, []string{lp, lk}) {
		t.Fatalf("code=%d %v %s", code, un, out)
	}
	if len(m.Result.Sources) != 1 || m.Result.Sources[0].Path != good || strings.Contains(out, "outside") || strings.Contains(out, "locked") {
		t.Errorf("%s", out)
	}
}

func TestWorkflowRefsUnreadableIndex(t *testing.T) {
	env, cfg, state := refsEnv(t, testutil.Env(t))
	os.MkdirAll(filepath.Join(state, "docket"), 0o755)
	idx := filepath.Join(state, "docket", "projects.json")
	os.WriteFile(idx, []byte("{"), 0o600)
	os.MkdirAll(filepath.Join(cfg, "docket"), 0o755)
	user := filepath.Join(cfg, "docket", "workflow.md")
	os.WriteFile(user, []byte(wf("3", "Plan review", "mine")), 0o644)
	code, m, out := refsRun(t, t.TempDir(), env, "refs")
	if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != idx || len(m.Result.Sources) != 1 || m.Result.Sources[0].Path != user {
		t.Errorf("code=%d %s", code, out)
	}
}

func TestWorkflowRefsEmptyOutsideRepository(t *testing.T) {
	env, _, state := refsEnv(t, testutil.Env(t))
	p := refsProject(t, wf("3", "Plan review", "x"))
	refsIndex(t, state, refEntry{path: p, ignored: true})
	before := snapshot(p)
	code, m, out := refsRun(t, t.TempDir(), env, "refs")
	if code != 0 || m.Repo != nil || !strings.Contains(out, `"findings":[]`) || !strings.Contains(out, `"unreadable":[]`) || !strings.Contains(out, `"sources":[]`) {
		t.Errorf("code=%d %s", code, out)
	}
	if strings.Contains(out, `"variants":null`) || strings.Contains(out, `:null,`) && strings.Contains(out, `"sections":null`) {
		t.Errorf("null array: %s", out)
	}
	for _, s := range m.Result.Sections {
		if s.Variants == nil {
			t.Errorf("%s: null variants", s.Heading)
		}
	}
	if snapshot(p) != before {
		t.Error("a project changed")
	}
}

func TestWorkflowRefsUsage(t *testing.T) {
	env, _, _ := refsEnv(t, testutil.Env(t))
	for _, args := range [][]string{{"workflow", "nosuch"}, {"workflow"}} {
		code, out, _ := runIn(t, t.TempDir(), env, append(args, "--json")...)
		var m refsOut
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatal(err)
		}
		if code != 2 || len(m.Findings) != 1 || m.Findings[0].Rule != "usage" {
			t.Errorf("%v: code=%d %s", args, code, out)
		}
	}
}

func TestWorkflowRefsText(t *testing.T) {
	env, _, state := refsEnv(t, testutil.Env(t))
	a := refsProject(t, wf("3", "Plan review", "one"))
	b := refsProject(t, wf("3", "Plan review", "two"))
	refsIndex(t, state, refEntry{path: a}, refEntry{path: b})
	code, out, _ := runIn(t, t.TempDir(), env, "workflow", "refs")
	if code != 0 || !strings.Contains(out, "Plan review") || !strings.Contains(out, a) || !strings.Contains(out, b) {
		t.Errorf("code=%d %q", code, out)
	}
}
