package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wake/docket/internal/projects"
	"github.com/wake/docket/internal/setup"
	"github.com/wake/docket/internal/testutil"
)

type inspectOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     struct {
		DefaultBranch *string `json:"default_branch"`
		OpenSpec      struct {
			Version   *string
			Minimum   string
			Supported bool
		}
		Files        []struct{ Path, Kind, Status string }
		UserWorkflow *struct {
			Path     string
			Exists   bool
			Version  int
			Default  *string
			Mode     string
			Sections []struct{ Heading, State string }
		} `json:"user_workflow"`
	}
}

func inspect(t *testing.T, r *testutil.Repo, version string, text bool) (int, inspectOut, string) {
	t.Helper()
	env := append([]string{}, r.Env...)
	if version != "" {
		cmd, _ := testutil.FakeOpenSpec(t, version)
		env = append(env, "DOCKET_OPENSPEC="+cmd)
	} else {
		env = append(env, "DOCKET_OPENSPEC="+filepath.Join(t.TempDir(), "none"))
	}
	args := []string{"inspect", "--json"}
	if text {
		args = args[:1]
	}
	code, out, _ := runIn(t, r.Dir, env, args...)
	var m inspectOut
	if !text {
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatalf("%q: %v", out, err)
		}
	}
	return code, m, out
}

func installed(t *testing.T, r *testutil.Repo) {
	t.Helper()
	ds, ps, err := setup.Decide(r.Dir, shippedFS(), false)
	if err != nil || len(ps) > 0 {
		t.Fatal(err, ps)
	}
	if err := setup.Write(r.Dir, ds); err != nil {
		t.Fatal(err)
	}
}

// rules lists the files each rule names, once per finding.
func rules(m inspectOut) map[string][]string {
	out := map[string][]string{}
	for _, f := range m.Findings {
		out[f.Rule] = append(out[f.Rule], f.File)
	}
	return out
}

// one is the file of the only finding with the rule, or "" if there is not exactly one.
func one(m inspectOut, rule string) string {
	if fs := rules(m)[rule]; len(fs) == 1 {
		return fs[0]
	}
	return ""
}

func snapshot(dir string) string {
	var b strings.Builder
	filepath.Walk(dir, func(p string, i os.FileInfo, err error) error {
		if err == nil && !strings.Contains(p, "/.git/") {
			b.WriteString(p + i.ModTime().String() + "\n")
		}
		return nil
	})
	return b.String()
}

func TestInspectNothingWritten(t *testing.T) {
	r := testutil.NewRepo(t)
	before := snapshot(r.Dir)
	home := ""
	for _, kv := range r.Env {
		if v, ok := strings.CutPrefix(kv, "HOME="); ok {
			home = v
		}
	}
	homeBefore := snapshot(home)
	exclude := filepath.Join(r.Dir, ".git/info/exclude")
	b0, _ := os.ReadFile(exclude)
	code, _, _ := inspect(t, r, "1.14.0", false)
	b1, _ := os.ReadFile(exclude)
	if code != 1 || snapshot(r.Dir) != before || snapshot(home) != homeBefore || string(b0) != string(b1) {
		t.Errorf("code=%d, tree changed", code)
	}
}

func TestInspectCleanSetup(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 0 || len(m.Findings) != 0 || len(m.Unreadable) != 0 {
		t.Fatalf("code=%d %+v", code, m)
	}
	o := m.Result.OpenSpec
	if o.Version == nil || *o.Version != "1.14.0" || o.Minimum != "1.14.0" || !o.Supported {
		t.Errorf("openspec = %+v", o)
	}
	if len(m.Result.Files) == 0 {
		t.Fatal("no files")
	}
	for _, f := range m.Result.Files {
		if f.Status != "same" || f.Kind == "" {
			t.Errorf("file %+v", f)
		}
	}
}

func TestInspectBeforeInit(t *testing.T) {
	r := testutil.NewRepo(t)
	code, m, _ := inspect(t, r, "1.14.0", false)
	got := rules(m)
	for _, k := range []string{"schema-missing", "skill-missing", "config-schema-default", "guidance-missing"} {
		if _, ok := got[k]; !ok {
			t.Errorf("no %s in %v", k, got)
		}
	}
	ds, _, _ := setup.Decide(r.Dir, shippedFS(), false)
	var want []string
	for _, d := range ds {
		if d.Kind == setup.KindSchema {
			want = append(want, d.Path)
		}
	}
	if len(want) < 2 || !slices.Equal(got["schema-missing"], want) {
		t.Errorf("schema-missing = %v, want %v", got["schema-missing"], want)
	}
	if one(m, "config-schema-default") != "openspec/config.yaml" || one(m, "guidance-missing") != "openspec/config.yaml" || code != 1 {
		t.Errorf("code=%d %v", code, got)
	}
}

func TestInspectEditedSkill(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	r.Write(".claude/skills/docket/SKILL.md", "mine")
	_, m, _ := inspect(t, r, "1.14.0", false)
	if one(m, "skill-differs") != ".claude/skills/docket/SKILL.md" {
		t.Errorf("%+v", m.Findings)
	}
	for _, f := range m.Result.Files {
		if f.Path == ".claude/skills/docket/SKILL.md" && f.Status != "differs" {
			t.Errorf("%+v", f)
		}
	}
}

func TestInspectSkillFolderFromEarlierRelease(t *testing.T) {
	withSideFile(t)
	r := testutil.NewRepo(t)
	installed(t, r)
	earlier(t, "old skill", skillDirs[0]+"SKILL.md", skillDirs[1]+"SKILL.md")
	for _, d := range skillDirs {
		r.Write(d+"SKILL.md", "old skill")
		if err := os.Remove(filepath.Join(r.Dir, d+sideFile)); err != nil {
			t.Fatal(err)
		}
	}
	code, m, _ := inspect(t, r, "1.14.0", false)
	got := rules(m)
	slices.Sort(got["skill-outdated"])
	slices.Sort(got["skill-missing"])
	if code != 1 ||
		!slices.Equal(got["skill-outdated"], []string{skillDirs[1] + "SKILL.md", skillDirs[0] + "SKILL.md"}) ||
		!slices.Equal(got["skill-missing"], []string{skillDirs[1] + sideFile, skillDirs[0] + sideFile}) {
		t.Errorf("code=%d %v", code, got)
	}
}

func TestInspectSideFileRules(t *testing.T) {
	withSideFile(t)
	r := testutil.NewRepo(t)
	installed(t, r)
	side := skillDirs[0] + sideFile

	r.Write(side, "mine")
	if code, m, _ := inspect(t, r, "1.14.0", false); code != 1 || one(m, "skill-differs") != side {
		t.Errorf("edited: code=%d %+v", code, m.Findings)
	}

	earlier(t, "old side", side)
	r.Write(side, "old side")
	code, m, _ := inspect(t, r, "1.14.0", false)
	listed := false
	for _, f := range m.Result.Files {
		listed = listed || f.Path == side && f.Status == "outdated"
	}
	if code != 1 || one(m, "skill-outdated") != side || !listed {
		t.Errorf("earlier: code=%d %+v", code, m)
	}

	if err := os.Remove(filepath.Join(r.Dir, side)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "nowhere"), filepath.Join(r.Dir, side)); err != nil {
		t.Fatal(err)
	}
	if code, m, _ := inspect(t, r, "1.14.0", false); code != 1 || one(m, "symlink-refused") != side {
		t.Errorf("link: code=%d %+v", code, m.Findings)
	}
}

func TestInspectConfigOutdatedAndOther(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	r.Write("openspec/config.yaml", "schema: spec-driven\n")
	_, m, _ := inspect(t, r, "1.14.0", false)
	got := rules(m)
	if _, ok := got["config-schema-default"]; !ok {
		t.Errorf("%v", got)
	}
	if _, ok := got["guidance-missing"]; !ok {
		t.Errorf("%v", got)
	}
	for _, f := range m.Result.Files {
		if f.Path == "openspec/config.yaml" && f.Status != "outdated" {
			t.Errorf("%+v", f)
		}
	}
	r.Write("openspec/config.yaml", "schema: team-flow\n")
	_, m, _ = inspect(t, r, "1.14.0", false)
	if one(m, "config-schema-other") != "openspec/config.yaml" {
		t.Errorf("%+v", m.Findings)
	}
}

// setup keeps no earlier release's operations section yet, so the mapping is
// tested on a constructed decision.
func TestConfigFindingsGuidance(t *testing.T) {
	d := setup.Decision{Path: "openspec/config.yaml", Kind: setup.KindConfig, Config: &setup.ConfigState{Schema: "docket", Guidance: "outdated"}}
	f := decisionFindings(d)
	if len(f) != 1 || f[0].Rule != "guidance-outdated" || f[0].File != d.Path {
		t.Errorf("%+v", f)
	}
	d.Config.Guidance = "differs"
	if f = decisionFindings(d); len(f) != 1 || f[0].Rule != "guidance-differs" {
		t.Errorf("%+v", f)
	}
}

func TestInspectSymlinkedSkillFolder(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	if err := os.RemoveAll(filepath.Join(r.Dir, ".claude/skills/docket")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(r.Dir, ".claude/skills/docket")); err != nil {
		t.Fatal(err)
	}
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 1 || one(m, "symlink-refused") != ".claude/skills/docket" {
		t.Errorf("code=%d %+v", code, m.Findings)
	}
}

func TestInspectOpenSpecVersions(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	code, m, _ := inspect(t, r, "1.13.2", false)
	o := m.Result.OpenSpec
	if code != 1 || o.Version == nil || *o.Version != "1.13.2" || o.Supported {
		t.Errorf("code=%d %+v", code, o)
	}
	if _, ok := rules(m)["openspec-unsupported"]; !ok {
		t.Errorf("%+v", m.Findings)
	}
	code, m, _ = inspect(t, r, "", false)
	if code != 1 || m.Result.OpenSpec.Version != nil || m.Result.OpenSpec.Supported || m.Result.OpenSpec.Minimum != "1.14.0" {
		t.Errorf("code=%d %+v", code, m.Result.OpenSpec)
	}
	if _, ok := rules(m)["openspec-missing"]; !ok {
		t.Errorf("%+v", m.Findings)
	}
}

func TestInspectIgnored(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write(".git/info/exclude", "# 1\n# 2\n# 3\n# 4\n# 5\n# 6\nopenspec/\n")
	_, m, _ := inspect(t, r, "1.14.0", false)
	for _, f := range m.Findings {
		if f.Rule == "install-ignored" && f.File == "openspec/config.yaml" {
			if !strings.Contains(f.Message, ".git/info/exclude:7: openspec/") {
				t.Errorf("message %q", f.Message)
			}
			return
		}
	}
	t.Errorf("no install-ignored: %+v", m.Findings)
}

func TestInspectUnreadable(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	p := filepath.Join(r.Dir, ".agents/skills/docket/SKILL.md")
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(p, 0o755); err != nil {
		t.Fatal(err)
	}
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != ".agents/skills/docket/SKILL.md" {
		t.Errorf("code=%d %+v", code, m.Unreadable)
	}
}

func TestInspectUnparsableConfig(t *testing.T) {
	for _, c := range []string{"schema:\n  docket\n", "operations:\n  a:\noperations:\n  b:\n"} {
		r := testutil.NewRepo(t)
		installed(t, r)
		r.Write("openspec/config.yaml", c)
		code, m, _ := inspect(t, r, "1.14.0", false)
		got := rules(m)
		_, a := got["config-schema-default"]
		_, b := got["guidance-missing"]
		if code != 3 || a || b || len(m.Unreadable) != 1 || m.Unreadable[0].Source != "openspec/config.yaml" {
			t.Errorf("%q: code=%d %+v %+v", c, code, m.Findings, m.Unreadable)
		}
	}
}

func withEarlier(t *testing.T, key, text string) {
	t.Helper()
	saved := setup.Earlier
	t.Cleanup(func() { setup.Earlier = saved })
	sum := sha256.Sum256([]byte(text))
	setup.Earlier = map[string][]string{key: {hex.EncodeToString(sum[:])}}
}

func TestInspectOutdatedTemplate(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	ds, _, _ := setup.Decide(r.Dir, shippedFS(), false)
	var path string
	for _, d := range ds {
		if d.Kind == setup.KindSchema && strings.Contains(d.Path, "templates/") {
			path = d.Path
			break
		}
	}
	r.Write(path, "an earlier release\n")
	withEarlier(t, path, "an earlier release\n")
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 1 || one(m, "schema-outdated") != path {
		t.Fatalf("code=%d %+v", code, m.Findings)
	}
	for _, f := range m.Result.Files {
		if f.Path == path && f.Status != "outdated" {
			t.Errorf("%+v", f)
		}
	}
}

func TestInspectOutdatedGuidance(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	sec := "operations:\n  apply:\n    guidance:\n      - old\n"
	r.Write("openspec/config.yaml", "schema: docket\n\n"+sec)
	withEarlier(t, "openspec/config.yaml#operations", sec)
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 1 || len(m.Findings) != 1 || one(m, "guidance-outdated") != "openspec/config.yaml" {
		t.Errorf("code=%d %+v", code, m.Findings)
	}
}

func TestInspectRelativeOpenSpecFromSubdirectory(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	fake, _ := testutil.FakeOpenSpec(t, "1.14.0")
	b, err := os.ReadFile(fake)
	if err != nil {
		t.Fatal(err)
	}
	r.Write("tools/openspec"+testutil.Exe, string(b))
	if err := os.Chmod(filepath.Join(r.Dir, "tools/openspec"+testutil.Exe), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(r.Dir, "sub/dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC=./tools/openspec")
	code, out, _ := runIn(t, sub, env, "inspect", "--json")
	var m inspectOut
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatal(err)
	}
	if m.Result.OpenSpec.Version == nil || *m.Result.OpenSpec.Version != "1.14.0" || len(rules(m)["openspec-missing"]) != 0 {
		t.Errorf("code=%d %s", code, out)
	}
}

func TestInspectStartsOpenSpecOnlyForVersion(t *testing.T) {
	r := testutil.NewRepo(t)
	fake, log := testutil.FakeOpenSpec(t, "1.14.0")
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+fake)
	runIn(t, r.Dir, env, "inspect", "--json")
	b, _ := os.ReadFile(log)
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) != 1 || lines[0] != "OPENSPEC_TELEMETRY=0\t--version" {
		t.Errorf("calls = %q", lines)
	}
}

func TestInspectText(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	ds, _, _ := setup.Decide(r.Dir, shippedFS(), false)
	code, _, out := inspect(t, r, "1.14.0", true)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if code != 0 || len(lines) != 2+len(ds) || lines[0] != "OpenSpec 1.14.0 (minimum 1.14.0)" || lines[1] != "same  "+ds[0].Path || lines[len(lines)-1] != "No default branch found; the trailer check was skipped." {
		t.Errorf("code=%d %q", code, out)
	}
	_, _, out = inspect(t, r, "", true)
	if !strings.HasPrefix(out, "OpenSpec: not found (minimum 1.14.0)\n") {
		t.Errorf("%q", out)
	}
	_, _, out = inspect(t, r, "1.13.2", true)
	if !strings.HasPrefix(out, "OpenSpec 1.13.2: unsupported (minimum 1.14.0)\n") {
		t.Errorf("%q", out)
	}
}

const planOne = "# Plan\n\n### Task 1: x\n"

// trailerRepo is a repository on main whose first commit holds the given change folders.
func trailerRepo(t *testing.T, msg string, slugs ...string) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	files := map[string]string{"README.md": "x\n"}
	for _, s := range slugs {
		files["openspec/changes/"+s+"/.openspec.yaml"] = "schema: docket\ncreated: 2026-01-01\n"
		files["openspec/changes/"+s+"/plan.md"] = planOne
	}
	r.Commit(msg, files)
	return r
}

func trailerFiles(m inspectOut) string {
	return strings.Join(rules(m)["trailer-missing"], ",")
}

func TestInspectTrailerMissing(t *testing.T) { // Squash merge dropped the trailer
	r := trailerRepo(t, "squashed", "add-x")
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 1 || trailerFiles(m) != "openspec/changes/add-x/plan.md" || m.Findings[len(m.Findings)-1].Change != "add-x" {
		t.Fatalf("%d %+v", code, m.Findings)
	}
	if m.Result.DefaultBranch == nil || *m.Result.DefaultBranch != "main" {
		t.Errorf("default_branch = %v", m.Result.DefaultBranch)
	}
}

func TestInspectTrailerText(t *testing.T) {
	r := trailerRepo(t, "squashed", "add-x")
	_, _, out := inspect(t, r, "1.14.0", true)
	for _, w := range []string{"Default branch: main", "trailer-missing", "no commit on main carries a Change: trailer for add-x"} {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in %q", w, out)
		}
	}
}

func TestInspectTrailerCarried(t *testing.T) {
	r := trailerRepo(t, "work\n\nChange: add-x/1", "add-x")
	_, m, _ := inspect(t, r, "1.14.0", false)
	if trailerFiles(m) != "" {
		t.Errorf("%+v", m.Findings)
	}
}

func TestInspectTrailerOnLaterCommitAndSlugAlone(t *testing.T) {
	r := trailerRepo(t, "plain", "add-x", "add-y")
	r.Commit("later\n\nChange: add-x", nil)
	r.Commit("other\n\nChange: add-other/1", nil)
	_, m, _ := inspect(t, r, "1.14.0", false)
	if trailerFiles(m) != "openspec/changes/add-y/plan.md" {
		t.Errorf("%+v", m.Findings)
	}
}

func TestInspectTrailerSlugPrefixDoesNotCount(t *testing.T) {
	r := trailerRepo(t, "work\n\nChange: add-x-2/1", "add-x")
	_, m, _ := inspect(t, r, "1.14.0", false)
	if trailerFiles(m) != "openspec/changes/add-x/plan.md" {
		t.Errorf("%+v", m.Findings)
	}
}

func TestInspectTrailerSkips(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	r.Commit("c", map[string]string{
		"openspec/changes/archive/2026-01-01-old/plan.md": planOne,
		"openspec/changes/other-schema/.openspec.yaml":    "schema: spec-driven\n",
		"openspec/changes/other-schema/plan.md":           planOne,
		"openspec/changes/no-plan/.openspec.yaml":         "schema: docket\n",
		"openspec/changes/no-tasks/plan.md":               "# Plan\n",
	})
	_, m, _ := inspect(t, r, "1.14.0", false)
	if trailerFiles(m) != "" {
		t.Errorf("%+v", m.Findings)
	}
}

func TestInspectTrailerMissingSchemaMeansDocket(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	r.Commit("c", map[string]string{"openspec/changes/add-x/plan.md": planOne})
	_, m, _ := inspect(t, r, "1.14.0", false)
	if trailerFiles(m) != "openspec/changes/add-x/plan.md" {
		t.Errorf("%+v", m.Findings)
	}
}

func TestInspectTrailerReadsDefaultBranchNotCheckout(t *testing.T) {
	r := trailerRepo(t, "squashed", "add-x")
	r.Git("checkout", "-q", "-b", "feat")
	r.Commit("feat\n\nChange: add-x/1", map[string]string{"f": "1"})
	r.Git("rm", "-q", "-r", "openspec")
	r.Git("commit", "-q", "-m", "drop")
	_, m, _ := inspect(t, r, "1.14.0", false)
	if trailerFiles(m) != "openspec/changes/add-x/plan.md" {
		t.Errorf("%+v", m.Findings)
	}
}

func TestInspectDefaultBranchFromOrigin(t *testing.T) {
	bare := testutil.NewBare(t)
	src := trailerRepo(t, "squashed", "add-x")
	src.Git("remote", "add", "origin", bare)
	src.Git("push", "-q", "origin", "main")
	dir := filepath.Join(t.TempDir(), "clone")
	cl := &testutil.Repo{T: t, Dir: filepath.Dir(dir), Env: src.Env}
	cl.Git("clone", "-q", bare, dir)
	r := &testutil.Repo{T: t, Dir: dir, Env: src.Env}
	r.Git("checkout", "-q", "-b", "feat")
	_, m, _ := inspect(t, r, "1.14.0", false)
	if m.Result.DefaultBranch == nil || *m.Result.DefaultBranch != "main" || trailerFiles(m) != "openspec/changes/add-x/plan.md" {
		t.Errorf("%v %+v", m.Result.DefaultBranch, m.Findings)
	}
}

func TestInspectShallowCloneSkipsTrailerCheck(t *testing.T) {
	src := trailerRepo(t, "add change\n\nChange: add-x/1", "add-x")
	src.Commit("later, no trailer", map[string]string{"b.txt": "b\n"})
	dir := filepath.Join(t.TempDir(), "clone")
	src.Git("clone", "-q", "--depth", "1", "file://"+src.Dir, dir)
	r := &testutil.Repo{T: t, Dir: dir, Env: src.Env}
	_, m, _ := inspect(t, r, "1.14.0", false)
	if trailerFiles(m) != "" || m.Result.DefaultBranch != nil {
		t.Errorf("%v %+v", m.Result.DefaultBranch, m.Findings)
	}
	_, _, text := inspect(t, r, "1.14.0", true)
	if !strings.Contains(text, "The repository is a shallow clone; the trailer check was skipped.") || strings.Contains(text, "Default branch:") {
		t.Errorf("text: %s", text)
	}
}

func TestInspectNoDefaultBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 0 || m.Result.DefaultBranch != nil || len(m.Findings) != 0 {
		t.Errorf("%d %+v", code, m)
	}
}

func TestInspectMainSpecTitles(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	r.Write("openspec/specs/alpha/spec.md", "no title\n")
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 1 || one(m, "title-missing") != "openspec/specs/alpha/spec.md" || m.Findings[0].Line != 1 {
		t.Errorf("%d %+v", code, m.Findings)
	}
	r.Write("openspec/specs/alpha/spec.md", "# beta Specification\n")
	if code, m, _ = inspect(t, r, "1.14.0", false); code != 0 || len(m.Findings) != 0 {
		t.Errorf("%d %+v", code, m.Findings)
	}
}

func TestInspectMainSpecUnreadable(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	r.Write("openspec/specs/alpha/spec.md", "x\n")
	p := filepath.Join(r.Dir, "openspec/specs/alpha/spec.md")
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	if f, err := os.Open(p); err == nil {
		f.Close()
		t.Skip("file readable despite mode 0")
	}
	_, m, _ := inspect(t, r, "1.14.0", false)
	if len(m.Unreadable) != 1 || m.Unreadable[0].Source != "openspec/specs/alpha/spec.md" {
		t.Errorf("%+v", m.Unreadable)
	}
}

func TestInspectLintFromSubdirectoryRootRelative(t *testing.T) {
	r := trailerRepo(t, "squashed", "add-x")
	r.Git("checkout", "-q", "-b", "feat")
	r.Write("openspec/specs/alpha/spec.md", "no title\n")
	r.Write("docs/superpowers/plans/p.md", "p\n")
	sub := filepath.Join(r.Dir, "sub/dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.0")
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd)
	for name, want := range map[string][]string{
		"inspect": {"trailer-missing:openspec/changes/add-x/plan.md", "title-missing:openspec/specs/alpha/spec.md"},
		"lint":    {"title-missing:openspec/specs/alpha/spec.md", "routing:docs/superpowers/plans/p.md"},
	} {
		_, out, _ := runIn(t, sub, env, name, "--json")
		var m inspectOut
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatal(err)
		}
		got := map[string]bool{}
		for _, f := range m.Findings {
			got[f.Rule+":"+f.File] = true
		}
		for _, w := range want {
			if !got[w] {
				t.Errorf("%s: missing %s in %s", name, w, out)
			}
		}
	}
}

const wfPath = "docket/workflow.md"

// withShippedWorkflow makes dkt see a shipped workflow template with the given text.
func withShippedWorkflow(t *testing.T, text string) {
	t.Helper()
	withShipped(t, "workflow/workflow.md", text)
}

// withShipped makes the shipped tree, for the rest of the test, the real one
// with the file at p, relative to shipped/, set to text.
func withShipped(t *testing.T, p, text string) {
	t.Helper()
	saved := shippedFS
	t.Cleanup(func() { shippedFS = saved })
	m := fstest.MapFS{}
	fs.WalkDir(saved(), ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			b, _ := fs.ReadFile(saved(), p)
			m[p] = &fstest.MapFile{Data: b}
		}
		return nil
	})
	m[p] = &fstest.MapFile{Data: []byte(text)}
	shippedFS = func() fs.FS { return m }
}

func wfStatus(m inspectOut) string {
	for _, f := range m.Result.Files {
		if f.Path == wfPath {
			return f.Status
		}
	}
	return ""
}

func TestInspectWorkflowMissing(t *testing.T) {
	r := testutil.NewRepo(t)
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 1 || one(m, "workflow-missing") != wfPath || wfStatus(m) != "missing" {
		t.Errorf("code=%d %v %q", code, rules(m), wfStatus(m))
	}
}

func TestInspectWorkflowCustomized(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	r.Write(wfPath, strings.ReplaceAll(shippedWorkflow(t), "Before any code", "Before any of our code"))
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 0 || wfStatus(m) != "same" || len(m.Findings) != 0 {
		t.Errorf("code=%d %v %q", code, rules(m), wfStatus(m))
	}
}

func TestInspectWorkflowHigherVersion(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	r.Write(wfPath, withVersion(t, shippedWorkflow(t), 99))
	_, m, _ := inspect(t, r, "1.14.0", false)
	if wfStatus(m) != "differs" || one(m, "workflow-template-unknown") != wfPath {
		t.Errorf("%v %q", rules(m), wfStatus(m))
	}
}

func TestInspectWorkflowNotANumber(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	r.Write(wfPath, withVersion(t, shippedWorkflow(t), "one"))
	_, m, _ := inspect(t, r, "1.14.0", false)
	if wfStatus(m) != "differs" || one(m, "workflow-template-unknown") != wfPath {
		t.Errorf("%v %q", rules(m), wfStatus(m))
	}
}

func TestInspectWorkflowNoFrontMatter(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	r.Write(wfPath, "just words\n")
	_, m, _ := inspect(t, r, "1.14.0", false)
	if one(m, "workflow-template-unknown") != wfPath || len(rules(m)["workflow-section-missing"]) != 0 {
		t.Errorf("%v", rules(m))
	}
}

func TestInspectWorkflowOutdated(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	withShippedWorkflow(t, withVersion(t, shippedWorkflow(t), shippedVersion(t)+1))
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 1 || wfStatus(m) != "outdated" || one(m, "workflow-outdated") != wfPath {
		t.Errorf("code=%d %v %q", code, rules(m), wfStatus(m))
	}
}

func TestWorkflowFindings(t *testing.T) {
	cases := []struct {
		name string
		d    setup.Decision
		want []string // rules
		in   [][]string
	}{
		{"outdated", setup.Decision{Status: setup.StatusOutdated, Workflow: &setup.WorkflowState{Version: 1, Shipped: 2}}, []string{"workflow-outdated"}, [][]string{{"template 1", "template 2"}}},
		{"unknown", setup.Decision{Status: setup.StatusDiffers, Workflow: &setup.WorkflowState{Version: 0, Shipped: 1}}, []string{"workflow-template-unknown"}, [][]string{{"template 1"}}},
		{"sections", setup.Decision{Status: setup.StatusDiffers, Workflow: &setup.WorkflowState{Version: 1, Shipped: 1, Missing: []string{"Plan review", "Reviewers"}}}, []string{"workflow-section-missing", "workflow-section-missing"}, [][]string{{`"Plan review"`}, {`"Reviewers"`}}},
		{"same", setup.Decision{Status: setup.StatusSame, Workflow: &setup.WorkflowState{Version: 1, Shipped: 1}}, nil, nil},
	}
	for _, c := range cases {
		got := workflowFindings(c.d)
		if len(got) != len(c.want) {
			t.Errorf("%s: %+v", c.name, got)
			continue
		}
		for i, f := range got {
			if f.Rule != c.want[i] || f.File != wfPath {
				t.Errorf("%s: %+v", c.name, f)
			}
			for _, s := range c.in[i] {
				if !strings.Contains(f.Message, s) {
					t.Errorf("%s: %q lacks %q", c.name, f.Message, s)
				}
			}
		}
	}
}

// userEnv puts the user template (when body is not empty) under a temporary config home.
func userEnv(t *testing.T, r *testutil.Repo, body string) string {
	t.Helper()
	cfg := t.TempDir()
	r.Env = append(r.Env, "XDG_CONFIG_HOME="+cfg)
	p := filepath.Join(cfg, "docket", "workflow.md")
	if body != "" {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func userTemplate(v string) string { return "---\ntemplate: " + v + "\n---\n\n# x\n" }

func TestInspectUserWorkflow(t *testing.T) {
	ship := shippedVersion(t)
	cases := []struct {
		name, body, rule string
		version          int
		code             int
	}{
		{"same", userTemplate(strconv.Itoa(ship)), "", ship, 0},
		{"older", userTemplate(strconv.Itoa(ship - 1)), "user-workflow-outdated", ship - 1, 1},
		{"newer", userTemplate(strconv.Itoa(ship + 1)), "user-workflow-newer", ship + 1, 1},
		{"no front matter", "# x\n", "user-workflow-template-unknown", 0, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := testutil.NewRepo(t)
			installed(t, r)
			p := userEnv(t, r, c.body)
			code, m, _ := inspect(t, r, "1.14.0", false)
			u := m.Result.UserWorkflow
			if code != c.code || u == nil || u.Path != p || !u.Exists || u.Version != c.version {
				t.Fatalf("code=%d %+v %v", code, u, rules(m))
			}
			got := map[string][]string{}
			for k, v := range rules(m) {
				if strings.HasPrefix(k, "user-workflow") {
					got[k] = v
				}
			}
			want := map[string][]string{}
			if c.rule != "" {
				want[c.rule] = []string{p}
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestInspectNoUserWorkflow(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	p := userEnv(t, r, "")
	code, m, _ := inspect(t, r, "1.14.0", false)
	u := m.Result.UserWorkflow
	if code != 0 || u == nil || u.Path != p || u.Exists || u.Version != 0 || len(m.Findings) != 0 {
		t.Errorf("code=%d %+v %v", code, u, m.Findings)
	}
}

func TestInspectUserWorkflowNoHome(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	r.Env = testutil.WithoutHome(r.Env)
	code, m, out := inspect(t, r, "1.14.0", false)
	if code != 0 || m.Result.UserWorkflow != nil || len(m.Findings) != 0 || !strings.Contains(out, `"user_workflow": null`) && !strings.Contains(out, `"user_workflow":null`) {
		t.Errorf("code=%d %+v %s", code, m, out)
	}
}

func TestInspectUserWorkflowUnderHomeFolder(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	home, other := homeVars()
	h := t.TempDir()
	r.Env = append(testutil.WithoutHome(r.Env), home+"="+h, other+"="+t.TempDir())
	p := filepath.Join(h, ".config", "docket", "workflow.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(userTemplate(strconv.Itoa(shippedVersion(t)))), 0o644); err != nil {
		t.Fatal(err)
	}
	code, m, _ := inspect(t, r, "1.14.0", false)
	if u := m.Result.UserWorkflow; code != 0 || u == nil || u.Path != p || !u.Exists {
		t.Errorf("code=%d %+v %v", code, u, rules(m))
	}
}

func TestInspectUserWorkflowUnreadable(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	p := userEnv(t, r, userTemplate("1"))
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	if f, err := os.Open(p); err == nil {
		f.Close()
		t.Skip("file readable despite mode 0")
	}
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != p || m.Result.UserWorkflow == nil || m.Result.UserWorkflow.Path != p {
		t.Errorf("code=%d %+v %+v", code, m.Unreadable, m.Result.UserWorkflow)
	}
}

func TestInspectUserWorkflowSymlink(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	p := userEnv(t, r, "")
	target := filepath.Join(t.TempDir(), "mine.md")
	if err := os.WriteFile(target, []byte(userTemplate(strconv.Itoa(shippedVersion(t)))), 0o644); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.Symlink(target, p); err != nil {
		t.Fatal(err)
	}
	code, m, _ := inspect(t, r, "1.14.0", false)
	if u := m.Result.UserWorkflow; code != 0 || u == nil || !u.Exists || u.Version != shippedVersion(t) {
		t.Errorf("code=%d %+v", code, u)
	}
}

// editSection adds a line under the heading of the shipped-style workflow text.
func editSection(text, heading, line string) string {
	return strings.Replace(text, "### "+heading+"\n", "### "+heading+"\n\n"+line+"\n", 1)
}

func hashHex(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

// sectionText is the text of a section as setup.Sections gives it.
func sectionText(t *testing.T, doc, heading string) string {
	t.Helper()
	for _, s := range setup.Sections([]byte(doc)) {
		if s.Heading == heading {
			return s.Text
		}
	}
	t.Fatalf("no section %q", heading)
	return ""
}

// syncFixture is a Docket project whose workflow file is project, beside a user template user,
// with the project's index entry holding mode and synced (mode "" gives no entry).
func syncFixture(t *testing.T, user, project, mode string, synced map[string]string) (*testutil.Repo, string) {
	t.Helper()
	r := testutil.NewRepo(t)
	installed(t, r)
	userEnv(t, r, user)
	r.Write("docket/workflow.md", project)
	idx := prIndexPath(r.Env)
	if mode != "" {
		x := &projects.Index{Projects: []projects.Entry{{Path: prReal(t, r.Dir), Added: "2026-01-01T00:00:00Z", Mode: mode, Synced: synced}}}
		if err := x.Save(idx); err != nil {
			t.Fatal(err)
		}
	}
	return r, idx
}

func headings(t *testing.T) []string {
	var hs []string
	for _, s := range setup.Sections([]byte(shippedWorkflow(t))) {
		hs = append(hs, s.Heading)
	}
	if len(hs) < 4 {
		t.Fatal("the shipped template has too few sections")
	}
	return hs
}

func TestInspectModeIndependent(t *testing.T) {
	tpl, hs := shippedWorkflow(t), headings(t)
	for _, mode := range []string{"", "independent"} {
		r, _ := syncFixture(t, editSection(tpl, hs[0], "user line"), tpl, mode, nil)
		code, m, out := inspect(t, r, "1.14.0", false)
		u := m.Result.UserWorkflow
		if code != 0 || u == nil || u.Mode != "independent" || u.Default != nil || u.Sections == nil || len(u.Sections) != 0 || !strings.Contains(out, `"default":null`) || !strings.Contains(out, `"sections":[]`) {
			t.Errorf("%q: code=%d %+v %s", mode, code, u, out)
		}
		for rule := range rules(m) {
			if strings.HasPrefix(rule, "workflow-sync") {
				t.Errorf("%q: %s", mode, rule)
			}
		}
	}
}

func TestInspectSyncInStep(t *testing.T) {
	tpl := shippedWorkflow(t)
	r, _ := syncFixture(t, tpl, tpl, "sync", nil)
	code, m, _ := inspect(t, r, "1.14.0", false)
	u := m.Result.UserWorkflow
	if code != 0 || u.Mode != "sync" || len(u.Sections) != len(headings(t)) || len(m.Findings) != 0 {
		t.Fatalf("code=%d %+v %v", code, u, m.Findings)
	}
	for _, s := range u.Sections {
		if s.State != "in-step" {
			t.Errorf("%+v", s)
		}
	}
}

func TestInspectSyncStates(t *testing.T) {
	tpl, hs := shippedWorkflow(t), headings(t)
	user := editSection(editSection(tpl, hs[0], "user edit"), hs[2], "user two")
	proj := editSection(editSection(editSection(tpl, hs[1], "project edit"), hs[2], "project two"), hs[3], "owned edit")
	proj = strings.Replace(proj, "template:", "project-owned: ["+hs[3]+"]\ntemplate:", 1)
	synced := map[string]string{
		hs[0]: hashHex(sectionText(t, tpl, hs[0])),
		hs[1]: hashHex(sectionText(t, tpl, hs[1])),
		hs[2]: hashHex(sectionText(t, tpl, hs[2])),
	}
	r, idx := syncFixture(t, user, proj, "sync", synced)
	before, _ := os.ReadFile(idx)
	snap := snapshot(r.Dir)
	code, m, _ := inspect(t, r, "1.14.0", false)
	got := map[string]string{}
	for _, s := range m.Result.UserWorkflow.Sections {
		got[s.Heading] = s.State
	}
	want := map[string]string{hs[0]: "template-changed", hs[1]: "project-changed", hs[2]: "conflict"}
	for _, h := range hs[4:] {
		want[h] = "in-step"
	}
	if code != 1 || !reflect.DeepEqual(got, want) {
		t.Fatalf("code=%d got %v want %v", code, got, want)
	}
	for rule, h := range map[string]string{"workflow-sync-template-changed": hs[0], "workflow-sync-project-changed": hs[1], "workflow-sync-conflict": hs[2]} {
		n := 0
		for _, f := range m.Findings {
			if f.Rule == rule && f.File == "docket/workflow.md" && strings.Contains(f.Message, strconv.Quote(h)) {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%s: %d findings, %v", rule, n, m.Findings)
		}
	}
	for _, f := range m.Findings {
		if strings.Contains(f.Message, strconv.Quote(hs[3])) {
			t.Errorf("owned section reported: %+v", f)
		}
	}
	after, _ := os.ReadFile(idx)
	if !bytes.Equal(before, after) || snapshot(r.Dir) != snap {
		t.Error("inspect wrote")
	}
}

func TestInspectProjectOwnedInvalid(t *testing.T) {
	tpl, hs := shippedWorkflow(t), headings(t)
	proj := strings.Replace(tpl, "template:", "project-owned: "+hs[0]+"\ntemplate:", 1)
	r, _ := syncFixture(t, tpl, proj, "sync", nil)
	code, m, _ := inspect(t, r, "1.14.0", false)
	if code != 1 || one(m, "workflow-project-owned-invalid") != "docket/workflow.md" || len(m.Result.UserWorkflow.Sections) != 0 {
		t.Errorf("code=%d %v %+v", code, rules(m), m.Result.UserWorkflow)
	}
}

func TestInspectNoSectionsWithoutBoth(t *testing.T) {
	tpl, hs := shippedWorkflow(t), headings(t)
	edited := editSection(tpl, hs[0], "x")
	older := strings.Replace(edited, "template: "+strconv.Itoa(shippedVersion(t)), "template: 1", 1)
	if older == edited {
		t.Fatal("version line not replaced")
	}
	for name, c := range map[string]struct{ user, proj string }{
		"no user template": {"", edited},
		"version gap":      {older, edited},
	} {
		r, _ := syncFixture(t, c.user, c.proj, "sync", nil)
		_, m, _ := inspect(t, r, "1.14.0", false)
		if u := m.Result.UserWorkflow; u == nil || len(u.Sections) != 0 || u.Mode != "sync" {
			t.Errorf("%s: %+v", name, u)
		}
	}
	r, _ := syncFixture(t, tpl, tpl, "sync", nil)
	if err := os.Remove(filepath.Join(r.Dir, "docket", "workflow.md")); err != nil {
		t.Fatal(err)
	}
	_, m, _ := inspect(t, r, "1.14.0", false)
	if u := m.Result.UserWorkflow; u == nil || len(u.Sections) != 0 {
		t.Errorf("no project file: %+v", u)
	}
}

func TestInspectDefaultSetting(t *testing.T) {
	tpl := shippedWorkflow(t)
	for body, want := range map[string]string{`{"workflow":{"default":"sync"}}`: "sync", `{"workflow":{"default":"always"}}`: "", `not json`: ""} {
		r, _ := syncFixture(t, tpl, tpl, "", nil)
		cfg := filepath.Join(filepath.Dir(userEnvPath(r)), "config.json")
		if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		code, m, _ := inspect(t, r, "1.14.0", false)
		u := m.Result.UserWorkflow
		if want != "" {
			if code != 0 || u.Default == nil || *u.Default != want {
				t.Errorf("%s: code=%d %+v", body, code, u)
			}
			continue
		}
		if code != 3 || u == nil || u.Default != nil || len(m.Unreadable) != 1 || m.Unreadable[0].Source != cfg {
			t.Errorf("%s: code=%d %+v %+v", body, code, u, m.Unreadable)
		}
	}
}

// userEnvPath is the user template's path in the environment userEnv set.
func userEnvPath(r *testutil.Repo) string {
	p, _ := projects.UserWorkflowPath(r.Env)
	return p
}

func TestInspectUnreadableIndex(t *testing.T) {
	tpl := shippedWorkflow(t)
	r, idx := syncFixture(t, tpl, tpl, "", nil)
	if err := os.MkdirAll(filepath.Dir(idx), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(idx, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, m, _ := inspect(t, r, "1.14.0", false)
	if u := m.Result.UserWorkflow; code != 3 || u == nil || u.Mode != "independent" || !slices.ContainsFunc(m.Unreadable, func(x Unreadable) bool { return x.Source == idx }) {
		t.Errorf("code=%d %+v %+v", code, u, m.Unreadable)
	}
}

func TestInspectIndexWithoutLocation(t *testing.T) {
	tpl := shippedWorkflow(t)
	r, _ := syncFixture(t, tpl, tpl, "", nil)
	cfg := t.TempDir()
	r.Env = append(testutil.WithoutHome(r.Env), "XDG_CONFIG_HOME="+cfg)
	code, m, _ := inspect(t, r, "1.14.0", false)
	u := m.Result.UserWorkflow
	if code != 0 || u == nil || u.Path != filepath.Join(cfg, "docket", "workflow.md") || u.Mode != "independent" || len(m.Findings) != 0 || len(m.Unreadable) != 0 {
		t.Errorf("code=%d %+v %v %v", code, u, m.Findings, m.Unreadable)
	}
}

func TestInspectSyncLinkedWorktree(t *testing.T) {
	tpl, hs := shippedWorkflow(t), headings(t)
	r, _ := syncFixture(t, tpl, tpl, "sync", nil)
	r.Commit("base", nil)
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-b", "other", wt)
	if err := os.WriteFile(filepath.Join(wt, "docket", "workflow.md"), []byte(editSection(tpl, hs[1], "worktree edit")), 0o644); err != nil {
		t.Fatal(err)
	}
	env := append(slices.Clone(r.Env), "DOCKET_OPENSPEC="+filepath.Join(t.TempDir(), "none"))
	code, out, _ := runIn(t, wt, env, "inspect", "--json")
	var m inspectOut
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("%d %q: %v", code, out, err)
	}
	u := m.Result.UserWorkflow
	if u == nil || u.Mode != "sync" || len(u.Sections) != len(hs) {
		t.Fatalf("code=%d %+v", code, u)
	}
	for _, s := range u.Sections {
		if (s.Heading == hs[1]) == (s.State == "in-step") {
			t.Errorf("worktree file not compared: %+v", s)
		}
	}
}

func TestInspectSyncProjectWorkflowUnreadable(t *testing.T) {
	tpl := shippedWorkflow(t)
	r, _ := syncFixture(t, tpl, tpl, "sync", nil)
	p := filepath.Join(r.Dir, "docket", "workflow.md")
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	if f, err := os.Open(p); err == nil {
		f.Close()
		t.Skip("file readable despite mode 0")
	}
	code, m, _ := inspect(t, r, "1.14.0", false)
	u := m.Result.UserWorkflow
	if code != 3 || !slices.ContainsFunc(m.Unreadable, func(x Unreadable) bool { return strings.HasPrefix(x.Reason, "openat docket/workflow.md") }) || u == nil || len(u.Sections) != 0 {
		t.Errorf("code=%d %+v %+v", code, m.Unreadable, u)
	}
}

func TestInspectSyncDocketIsAFile(t *testing.T) {
	tpl := shippedWorkflow(t)
	r, _ := syncFixture(t, tpl, tpl, "sync", nil)
	if err := os.RemoveAll(filepath.Join(r.Dir, "docket")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Dir, "docket"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, m, _ := inspect(t, r, "1.14.0", false)
	u := m.Result.UserWorkflow
	if u == nil || u.Mode != "sync" || len(u.Sections) != 0 {
		t.Fatalf("%+v", u)
	}
	// other checks may report the file; this path must add nothing to what independent mode reports
	r2, _ := syncFixture(t, tpl, tpl, "independent", nil)
	os.RemoveAll(filepath.Join(r2.Dir, "docket"))
	os.WriteFile(filepath.Join(r2.Dir, "docket"), []byte("x"), 0o644)
	_, m2, _ := inspect(t, r2, "1.14.0", false)
	if len(m.Unreadable) != len(m2.Unreadable) {
		t.Errorf("sync %+v, independent %+v", m.Unreadable, m2.Unreadable)
	}
}
