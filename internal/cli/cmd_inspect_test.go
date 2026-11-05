package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/setup"
	"github.com/wake/docket/internal/testutil"
)

type inspectOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     struct {
		OpenSpec struct {
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

func rules(m inspectOut) map[string]string {
	out := map[string]string{}
	for _, f := range m.Findings {
		out[f.Rule] = f.File
	}
	return out
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
	code, _, _ := inspect(t, r, "1.7.0", false)
	b1, _ := os.ReadFile(exclude)
	if code != 1 || snapshot(r.Dir) != before || string(b0) != string(b1) {
		t.Errorf("code=%d, tree changed", code)
	}
}

func TestInspectCleanSetup(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	code, m, _ := inspect(t, r, "1.7.0", false)
	if code != 0 || len(m.Findings) != 0 || len(m.Unreadable) != 0 {
		t.Fatalf("code=%d %+v", code, m)
	}
	o := m.Result.OpenSpec
	if o.Version == nil || *o.Version != "1.7.0" || o.Minimum != "1.7.0" || !o.Supported {
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
	code, m, _ := inspect(t, r, "1.7.0", false)
	got := rules(m)
	for _, k := range []string{"schema-missing", "skill-missing", "config-schema-default", "guidance-missing"} {
		if _, ok := got[k]; !ok {
			t.Errorf("no %s in %v", k, got)
		}
	}
	if got["config-schema-default"] != "openspec/config.yaml" || got["guidance-missing"] != "openspec/config.yaml" || code != 1 {
		t.Errorf("code=%d %v", code, got)
	}
}

func TestInspectEditedSkill(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	r.Write(".claude/skills/docket/SKILL.md", "mine")
	_, m, _ := inspect(t, r, "1.7.0", false)
	if rules(m)["skill-differs"] != ".claude/skills/docket/SKILL.md" {
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
	_, m, _ := inspect(t, r, "1.7.0", false)
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
	_, m, _ = inspect(t, r, "1.7.0", false)
	if rules(m)["config-schema-other"] != "openspec/config.yaml" {
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
	code, m, _ := inspect(t, r, "1.7.0", false)
	if code != 1 || rules(m)["symlink-refused"] != ".claude/skills/docket" {
		t.Errorf("code=%d %+v", code, m.Findings)
	}
}

func TestInspectOpenSpecVersions(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	code, m, _ := inspect(t, r, "1.6.2", false)
	o := m.Result.OpenSpec
	if code != 1 || o.Version == nil || *o.Version != "1.6.2" || o.Supported {
		t.Errorf("code=%d %+v", code, o)
	}
	if _, ok := rules(m)["openspec-unsupported"]; !ok {
		t.Errorf("%+v", m.Findings)
	}
	code, m, _ = inspect(t, r, "", false)
	if code != 1 || m.Result.OpenSpec.Version != nil || m.Result.OpenSpec.Supported || m.Result.OpenSpec.Minimum != "1.7.0" {
		t.Errorf("code=%d %+v", code, m.Result.OpenSpec)
	}
	if _, ok := rules(m)["openspec-missing"]; !ok {
		t.Errorf("%+v", m.Findings)
	}
}

func TestInspectIgnored(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write(".git/info/exclude", "# 1\n# 2\n# 3\n# 4\n# 5\n# 6\nopenspec/\n")
	_, m, _ := inspect(t, r, "1.7.0", false)
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
	code, m, _ := inspect(t, r, "1.7.0", false)
	if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != ".agents/skills/docket/SKILL.md" {
		t.Errorf("code=%d %+v", code, m.Unreadable)
	}
}

func TestInspectUnparsableConfig(t *testing.T) {
	for _, c := range []string{"schema:\n  docket\n", "operations:\n  a:\noperations:\n  b:\n"} {
		r := testutil.NewRepo(t)
		installed(t, r)
		r.Write("openspec/config.yaml", c)
		code, m, _ := inspect(t, r, "1.7.0", false)
		got := rules(m)
		_, a := got["config-schema-default"]
		_, b := got["guidance-missing"]
		if code != 3 || a || b || len(m.Unreadable) != 1 || m.Unreadable[0].Source != "openspec/config.yaml" {
			t.Errorf("%q: code=%d %+v %+v", c, code, m.Findings, m.Unreadable)
		}
	}
}

func TestInspectText(t *testing.T) {
	r := testutil.NewRepo(t)
	installed(t, r)
	ds, _, _ := setup.Decide(r.Dir, shippedFS(), false)
	code, _, out := inspect(t, r, "1.7.0", true)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if code != 0 || len(lines) != 1+len(ds) || lines[0] != "OpenSpec 1.7.0 (minimum 1.7.0)" || lines[1] != "same  "+ds[0].Path {
		t.Errorf("code=%d %q", code, out)
	}
	_, _, out = inspect(t, r, "", true)
	if !strings.HasPrefix(out, "OpenSpec: not found (minimum 1.7.0)\n") {
		t.Errorf("%q", out)
	}
	_, _, out = inspect(t, r, "1.6.2", true)
	if !strings.HasPrefix(out, "OpenSpec 1.6.2: unsupported (minimum 1.7.0)\n") {
		t.Errorf("%q", out)
	}
}
