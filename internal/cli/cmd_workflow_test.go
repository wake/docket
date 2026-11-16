package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

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

type modeOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     struct {
		Path   string
		Mode   string
		Synced []string
	}
}

// modeRun runs `workflow mode <args> --json` with a fixed clock.
func modeRun(t *testing.T, dir string, env []string, args ...string) (int, modeOut, string) {
	t.Helper()
	now := func() time.Time { return time.Date(2026, 11, 16, 1, 2, 3, 0, time.UTC) }
	var out, errb bytes.Buffer
	code := Run(Options{Args: append(append([]string{"workflow", "mode"}, args...), "--json"), Dir: dir, Env: env, Stdout: &out, Stderr: &errb, Now: now})
	var m modeOut
	if out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &m); err != nil {
			t.Fatalf("%q: %v", out.String(), err)
		}
	}
	return code, m, out.String()
}

// entryOf is the index entry for path.
func entryOf(t *testing.T, idx, path string) projects.Entry {
	t.Helper()
	x, err := projects.Load(idx)
	if err != nil {
		t.Fatal(err)
	}
	i := x.Find(path)
	if i < 0 {
		t.Fatalf("no entry for %s", path)
	}
	return x.Projects[i]
}

func fileBytes(idx string) []byte { b, _ := os.ReadFile(idx); return b }

func TestWorkflowModeSyncRecorded(t *testing.T) {
	tpl, hs := shippedWorkflow(t), headings(t)
	r, idx := syncFixture(t, tpl, tpl, "independent", nil)
	code, m, out := modeRun(t, r.Dir, r.Env, "sync")
	e := entryOf(t, idx, prReal(t, r.Dir))
	if code != 0 || m.Result.Path != prReal(t, r.Dir) || m.Result.Mode != "sync" || e.Mode != "sync" || !reflect.DeepEqual(m.Result.Synced, hs) || len(e.Synced) != len(hs) {
		t.Fatalf("code=%d %s %+v", code, out, e)
	}
	for _, h := range hs {
		if e.Synced[h] != hashHex(sectionText(t, tpl, h)) {
			t.Errorf("hash of %q: %q", h, e.Synced[h])
		}
	}
}

func TestWorkflowModeKeepsEarlierHashes(t *testing.T) {
	tpl, hs := shippedWorkflow(t), headings(t)
	user := editSection(tpl, hs[0], "user edit")
	proj := editSection(tpl, hs[3], "owned edit")
	proj = strings.Replace(proj, "template:", "project-owned: ["+hs[3]+"]\ntemplate:", 1)
	r, idx := syncFixture(t, user, proj, "sync", map[string]string{hs[0]: "old0", hs[3]: "old3", "Only in index": "old"})
	code, m, out := modeRun(t, r.Dir, r.Env, "sync")
	e := entryOf(t, idx, prReal(t, r.Dir))
	want := hs[1:3]
	want = append(slices.Clone(want), hs[4:]...)
	if code != 0 || !reflect.DeepEqual(m.Result.Synced, want) || e.Synced[hs[0]] != "old0" || e.Synced[hs[3]] != "old3" || e.Synced["Only in index"] != "old" || len(e.Synced) != len(want)+3 {
		t.Fatalf("code=%d %s %+v", code, out, e.Synced)
	}
}

func TestWorkflowModeVersionGap(t *testing.T) {
	tpl, hs := shippedWorkflow(t), headings(t)
	older := strings.Replace(tpl, "template: "+strconv.Itoa(shippedVersion(t)), "template: 1", 1)
	r, idx := syncFixture(t, older, tpl, "sync", map[string]string{hs[0]: "old0"})
	code, m, out := modeRun(t, r.Dir, r.Env, "sync")
	e := entryOf(t, idx, prReal(t, r.Dir))
	if code != 0 || m.Result.Synced == nil || len(m.Result.Synced) != 0 || !strings.Contains(out, `"synced":[]`) || !reflect.DeepEqual(e.Synced, map[string]string{hs[0]: "old0"}) {
		t.Fatalf("code=%d %s %+v", code, out, e.Synced)
	}
}

func TestWorkflowModeIndependent(t *testing.T) {
	tpl := shippedWorkflow(t)
	r, idx := syncFixture(t, tpl, tpl, "sync", map[string]string{"A": "x"})
	code, m, out := modeRun(t, r.Dir, r.Env, "independent")
	e := entryOf(t, idx, prReal(t, r.Dir))
	if code != 0 || m.Result.Mode != "independent" || e.Mode != "independent" || e.Synced != nil || m.Result.Synced == nil || !strings.Contains(out, `"synced":[]`) {
		t.Fatalf("code=%d %s %+v", code, out, e)
	}
}

func TestWorkflowModeAddsEntry(t *testing.T) {
	tpl := shippedWorkflow(t)
	r, idx := syncFixture(t, tpl, tpl, "", nil)
	if code, _, out := modeRun(t, r.Dir, r.Env, "independent"); code != 0 {
		t.Fatalf("%d %s", code, out)
	}
	e := entryOf(t, idx, prReal(t, r.Dir))
	if e.Mode != "independent" || e.Ignored || e.Added != "2026-11-16T01:02:03Z" || e.Origin != nil {
		t.Errorf("%+v", e)
	}
	// an ignored entry stays ignored
	x := &projects.Index{Projects: []projects.Entry{{Path: prReal(t, r.Dir), Added: "2026-01-01T00:00:00Z", Ignored: true}}}
	if err := x.Save(idx); err != nil {
		t.Fatal(err)
	}
	modeRun(t, r.Dir, r.Env, "independent")
	if e := entryOf(t, idx, prReal(t, r.Dir)); !e.Ignored || e.Mode != "independent" || e.Added != "2026-01-01T00:00:00Z" {
		t.Errorf("%+v", e)
	}
}

func TestWorkflowModeLinkedWorktree(t *testing.T) {
	tpl, hs := shippedWorkflow(t), headings(t)
	r, idx := syncFixture(t, tpl, tpl, "independent", nil)
	r.Commit("base", nil)
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-b", "other", wt)
	r.Write("docket/workflow.md", editSection(tpl, hs[0], "main edit"))
	code, m, out := modeRun(t, wt, r.Env, "sync")
	e := entryOf(t, idx, prReal(t, r.Dir))
	if code != 0 || m.Result.Path != prReal(t, r.Dir) || e.Mode != "sync" || e.Synced[hs[0]] == "" || len(m.Result.Synced) != len(hs) {
		t.Fatalf("code=%d %s %+v", code, out, e)
	}
	if x, _ := projects.Load(idx); len(x.Projects) != 1 {
		t.Errorf("entries: %+v", x.Projects)
	}
}

func TestWorkflowModeRefusals(t *testing.T) {
	tpl, hs := shippedWorkflow(t), headings(t)
	badOwned := strings.Replace(tpl, "template:", "project-owned: "+hs[0]+"\ntemplate:", 1)
	cases := []struct {
		name, user, proj, rule, file string
	}{
		{"no user template", "", tpl, "user-workflow-missing", "USER"},
		{"owned invalid", tpl, badOwned, "workflow-project-owned-invalid", "docket/workflow.md"},
	}
	for _, c := range cases {
		r, idx := syncFixture(t, c.user, c.proj, "independent", nil)
		before := fileBytes(idx)
		code, m, out := modeRun(t, r.Dir, r.Env, "sync")
		file := c.file
		if file == "USER" {
			file = userEnvPath(r)
		}
		if code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != c.rule || m.Findings[0].File != file || !bytes.Equal(before, fileBytes(idx)) {
			t.Errorf("%s: code=%d %s", c.name, code, out)
		}
	}
	r, _ := syncFixture(t, tpl, tpl, "", nil)
	if code, m, out := modeRun(t, t.TempDir(), r.Env, "independent"); code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != "not-a-repository" {
		t.Errorf("outside: code=%d %s", code, out)
	}
	if err := os.Remove(filepath.Join(r.Dir, "docket", "workflow.md")); err != nil {
		t.Fatal(err)
	}
	if code, m, out := modeRun(t, r.Dir, r.Env, "independent"); code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != "not-a-project" {
		t.Errorf("not a project: code=%d %s", code, out)
	}
}

func TestWorkflowModeUsage(t *testing.T) {
	tpl := shippedWorkflow(t)
	r, _ := syncFixture(t, tpl, tpl, "", nil)
	for _, args := range [][]string{{"always"}, {}, {"sync", "extra"}} {
		if code, m, out := modeRun(t, r.Dir, r.Env, args...); code != 2 || len(m.Findings) != 1 || m.Findings[0].Rule != "usage" {
			t.Errorf("%v: code=%d %s", args, code, out)
		}
	}
}

func TestWorkflowModeUnreadable(t *testing.T) {
	tpl := shippedWorkflow(t)
	r, idx := syncFixture(t, tpl, tpl, "independent", nil)
	if err := os.WriteFile(idx, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, m, out := modeRun(t, r.Dir, r.Env, "sync")
	if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != idx || string(fileBytes(idx)) != "not json" {
		t.Errorf("index: code=%d %s", code, out)
	}
	for name, file := range map[string]func(*testutil.Repo) string{
		"user template":    userEnvPath,
		"project workflow": func(r *testutil.Repo) string { return filepath.Join(r.Dir, "docket", "workflow.md") },
	} {
		r, idx := syncFixture(t, tpl, tpl, "independent", nil)
		before := fileBytes(idx)
		p := file(r)
		want := strings.Replace(p, r.Dir, prReal(t, r.Dir), 1) // dkt names the real path
		if err := os.Chmod(p, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(p, 0o644) })
		if f, err := os.Open(p); err == nil {
			f.Close()
			t.Skip("file readable despite mode 0")
		}
		code, m, out := modeRun(t, r.Dir, r.Env, "sync")
		if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != want || !bytes.Equal(before, fileBytes(idx)) {
			t.Errorf("%s: code=%d %s", name, code, out)
		}
	}
}

func TestWorkflowModeNoHome(t *testing.T) {
	tpl := shippedWorkflow(t)
	without := func(env []string, keys ...string) []string {
		return slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
			for _, k := range keys {
				if strings.HasPrefix(kv, k+"=") {
					return true
				}
			}
			return false
		})
	}
	r, _ := syncFixture(t, tpl, tpl, "", nil)
	noState := without(testutil.WithoutHome(r.Env), "XDG_STATE_HOME")
	for _, mode := range []string{"independent", "sync"} {
		if code, m, out := modeRun(t, r.Dir, noState, mode); code != 4 || len(m.Findings) != 1 || m.Findings[0].Rule != "no-home" {
			t.Errorf("%s: code=%d %s", mode, code, out)
		}
	}
	state := t.TempDir()
	onlyState := append(without(testutil.WithoutHome(r.Env), "XDG_CONFIG_HOME"), "XDG_STATE_HOME="+state)
	idx := filepath.Join(state, "docket", "projects.json")
	if code, m, out := modeRun(t, r.Dir, onlyState, "sync"); code != 4 || len(m.Findings) != 1 || m.Findings[0].Rule != "no-home" {
		t.Errorf("sync: code=%d %s", code, out)
	}
	if _, err := os.Stat(idx); err == nil {
		t.Error("index written")
	}
	if code, _, out := modeRun(t, r.Dir, onlyState, "independent"); code != 0 || entryOf(t, idx, prReal(t, r.Dir)).Mode != "independent" {
		t.Errorf("independent: code=%d %s", code, out)
	}
}

func TestWorkflowModeSyncNoHashWithoutAgreement(t *testing.T) {
	tpl, hs := shippedWorkflow(t), headings(t)
	user := editSection(tpl, hs[0], "user edit") + "\n### Only in user\n\ntext\n"
	proj := editSection(tpl, hs[1], "project edit")
	r, idx := syncFixture(t, user, proj, "independent", nil)
	code, m, out := modeRun(t, r.Dir, r.Env, "sync")
	e := entryOf(t, idx, prReal(t, r.Dir))
	if code != 0 {
		t.Fatalf("code=%d %s", code, out)
	}
	for _, h := range []string{hs[0], hs[1], "Only in user"} {
		if _, ok := e.Synced[h]; ok || slices.Contains(m.Result.Synced, h) {
			t.Errorf("%q got a hash: %v %v", h, e.Synced, m.Result.Synced)
		}
	}
	if len(m.Result.Synced) != len(hs)-2 {
		t.Errorf("synced %v", m.Result.Synced)
	}
}
