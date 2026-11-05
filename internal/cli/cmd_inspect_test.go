package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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
		Files []struct{ Path, Kind, Status string }
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
	exclude := filepath.Join(r.Dir, ".git/info/exclude")
	b0, _ := os.ReadFile(exclude)
	code, _, _ := inspect(t, r, "1.14.0", false)
	b1, _ := os.ReadFile(exclude)
	if code != 1 || snapshot(r.Dir) != before || string(b0) != string(b1) {
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
	r.Write("tools/openspec", string(b))
	if err := os.Chmod(filepath.Join(r.Dir, "tools/openspec"), 0o755); err != nil {
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
