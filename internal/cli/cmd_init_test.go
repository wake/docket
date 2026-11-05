package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/setup"
	"github.com/wake/docket/internal/testutil"
)

type initOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     struct {
		OpenspecInit *string `json:"openspec_init"`
		Files        []struct{ Path, Kind, Status string }
		NextSteps    []string `json:"next_steps"`
	}
}

// doInit runs dkt init in dir with the fake OpenSpec; extra is added to the environment.
func doInit(t *testing.T, r *testutil.Repo, dir, version string, extra []string, args ...string) (int, initOut, string, string) {
	t.Helper()
	cmd, log := testutil.FakeOpenSpec(t, version)
	return doInitCmd(t, r, dir, cmd, log, extra, args...)
}

// doInitCmd returns the exit code, the parsed JSON (unless args start with "text"), the output and the fake's call log.
func doInitCmd(t *testing.T, r *testutil.Repo, dir, cmd, log string, extra []string, args ...string) (int, initOut, string, string) {
	t.Helper()
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd)
	env = append(env, extra...)
	text := len(args) > 0 && args[0] == "text"
	if text {
		args = args[1:]
	} else {
		args = append([]string{"--json"}, args...)
	}
	code, out, errs := runIn(t, dir, env, append([]string{"init"}, args...)...)
	var m initOut
	if !text && out != "" {
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatalf("%q: %v", out, err)
		}
	}
	calls, _ := os.ReadFile(log)
	return code, m, out + errs, string(calls)
}

func status(m initOut, path string) string {
	for _, f := range m.Result.Files {
		if f.Path == path {
			return f.Status
		}
	}
	return ""
}

func readFile(t *testing.T, r *testutil.Repo, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.Dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func hasFinding(m initOut, rule, file string) bool {
	for _, f := range m.Findings {
		if f.Rule == rule && f.File == file {
			return true
		}
	}
	return false
}

const skillPath = ".claude/skills/docket/SKILL.md"

func TestInitFresh(t *testing.T) {
	r := testutil.NewRepo(t)
	code, m, _, calls := doInit(t, r, r.Dir, "1.7.0", nil, "--tools", "claude")
	if code != 0 || len(m.Findings) != 0 || len(m.Unreadable) != 0 {
		t.Fatalf("code=%d %+v", code, m)
	}
	if !strings.Contains(calls, "init\t--tools\tclaude") {
		t.Errorf("calls %q", calls)
	}
	if m.Result.OpenspecInit == nil || *m.Result.OpenspecInit != "claude" {
		t.Errorf("openspec_init %v", m.Result.OpenspecInit)
	}
	if status(m, ".claude/skills/openspec-explore/SKILL.md") != "created" {
		t.Errorf("%+v", m.Result.Files)
	}
	n := 0
	for _, f := range m.Result.Files {
		if f.Path == "openspec/config.yaml" {
			n++
			if f.Kind != "config" || f.Status != "created" {
				t.Errorf("config %+v", f)
			}
		}
		if f.Path == ".claude/skills/openspec-explore/SKILL.md" && f.Kind != "openspec" {
			t.Errorf("%+v", f)
		}
	}
	if n != 1 {
		t.Errorf("config listed %d times", n)
	}
	ds, _, _ := setup.Decide(r.Dir, shippedFS(), false)
	for _, d := range ds {
		if d.Action != setup.ActionNone {
			t.Errorf("%s still %s", d.Path, d.Action)
		}
	}
	if readFile(t, r, skillPath) != readFile(t, r, ".agents/skills/docket/SKILL.md") {
		t.Error("skills differ")
	}
	if c := readFile(t, r, "openspec/config.yaml"); !strings.HasPrefix(c, "schema: docket\n") || strings.Contains(c, "spec-driven") {
		t.Errorf("config %q", c)
	}
	ns := m.Result.NextSteps
	if len(ns) != 2 || !strings.Contains(ns[0], "Commit") || !strings.Contains(ns[1], "reload") {
		t.Errorf("next_steps %q", ns)
	}
}

func TestInitFromSubdirectory(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write("src/a.txt", "x")
	sub := filepath.Join(r.Dir, "src")
	if code, _, _, _ := doInit(t, r, sub, "1.7.0", nil, "--tools", "claude"); code != 0 {
		t.Fatalf("code %d", code)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, skillPath)); err != nil {
		t.Error(err)
	}
	for _, p := range []string{".claude", "openspec"} {
		if _, err := os.Stat(filepath.Join(sub, p)); err == nil {
			t.Errorf("wrote %s under src", p)
		}
	}
}

func TestInitMissingTools(t *testing.T) {
	r := testutil.NewRepo(t)
	code, _, out, calls := doInit(t, r, r.Dir, "1.7.0", nil)
	if code != 2 || strings.Contains(calls, "init") {
		t.Errorf("code=%d calls=%q", code, calls)
	}
	if !strings.Contains(out, "--tools") || !strings.Contains(out, "openspec init") {
		t.Errorf("out %q", out)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "openspec")); err == nil {
		t.Error("openspec written")
	}
}

func TestInitExistingOpenSpec(t *testing.T) {
	for _, args := range [][]string{nil, {"--tools", "claude"}} {
		r := testutil.NewRepo(t)
		r.Write("openspec/specs/.keep", "")
		code, m, _, calls := doInit(t, r, r.Dir, "1.7.0", nil, args...)
		if code != 0 || strings.Contains(calls, "init") || m.Result.OpenspecInit != nil {
			t.Errorf("args %v: code=%d calls=%q init=%v", args, code, calls, m.Result.OpenspecInit)
		}
		if status(m, "openspec/config.yaml") != "created" || readFile(t, r, "openspec/config.yaml") == "" {
			t.Errorf("%+v", m.Result.Files)
		}
	}
}

func TestInitOtherConfigUnchanged(t *testing.T) {
	r := testutil.NewRepo(t)
	files := map[string]string{".claude/settings.json": "{}", ".claude/skills/mine/SKILL.md": "mine", "AGENTS.md": "a", "CLAUDE.md": "c", "openspec/specs/.keep": ""}
	for p, c := range files {
		r.Write(p, c)
	}
	if code, _, _, _ := doInit(t, r, r.Dir, "1.7.0", nil); code != 0 {
		t.Fatal(code)
	}
	for p, c := range files {
		if readFile(t, r, p) != c {
			t.Errorf("%s changed", p)
		}
	}
}

func TestInitRerunSame(t *testing.T) {
	r := testutil.NewRepo(t)
	doInit(t, r, r.Dir, "1.7.0", nil, "--tools", "claude")
	code, m, _, calls := doInit(t, r, r.Dir, "1.7.0", nil, "--tools", "claude")
	if code != 0 || m.Result.OpenspecInit != nil || strings.Contains(calls, "init") {
		t.Fatalf("code=%d %+v calls=%q", code, m, calls)
	}
	if len(m.Result.Files) == 0 || len(m.Result.NextSteps) != 0 || m.Result.NextSteps == nil {
		t.Errorf("%+v", m.Result)
	}
	for _, f := range m.Result.Files {
		if f.Status != "same" {
			t.Errorf("%+v", f)
		}
	}
}

func TestInitEarlierRelease(t *testing.T) {
	r := testutil.NewRepo(t)
	doInit(t, r, r.Dir, "1.7.0", nil, "--tools", "claude")
	const p = "openspec/schemas/docket/templates/plan.md"
	old := "old plan"
	sum := sha256.Sum256([]byte(old))
	saved := setup.Earlier
	setup.Earlier = map[string][]string{p: {hex.EncodeToString(sum[:])}}
	defer func() { setup.Earlier = saved }()
	r.Write(p, old)
	code, m, _, _ := doInit(t, r, r.Dir, "1.7.0", nil)
	if code != 0 || status(m, p) != "updated" || readFile(t, r, p) == old {
		t.Errorf("code=%d %q", code, status(m, p))
	}
}

func TestInitEditedSkill(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write("openspec/specs/.keep", "")
	r.Write(skillPath, "mine")
	code, m, _, _ := doInit(t, r, r.Dir, "1.7.0", nil)
	if code != 1 || status(m, skillPath) != "kept" || readFile(t, r, skillPath) != "mine" {
		t.Fatalf("code=%d %+v", code, m.Result.Files)
	}
	if !hasFinding(m, "skill-differs", skillPath) {
		t.Errorf("%+v", m.Findings)
	}
	// the other files are still written, and the doctor step comes last
	if status(m, ".agents/skills/docket/SKILL.md") != "created" || status(m, "openspec/config.yaml") != "created" {
		t.Errorf("%+v", m.Result.Files)
	}
	ns := m.Result.NextSteps
	if len(ns) == 0 || !strings.Contains(ns[len(ns)-1], "docket-doctor") {
		t.Errorf("next_steps %q", ns)
	}
	code, m, _, _ = doInit(t, r, r.Dir, "1.7.0", nil, "--force")
	if code != 0 || status(m, skillPath) != "overwritten" || readFile(t, r, skillPath) == "mine" {
		t.Errorf("force: code=%d %+v", code, m.Result.Files)
	}
}

func TestInitConfigCases(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write("openspec/config.yaml", "schema: team-flow\n")
	code, m, _, _ := doInit(t, r, r.Dir, "1.7.0", nil, "--force")
	if code != 1 || !hasFinding(m, "config-schema-other", "openspec/config.yaml") || !strings.HasPrefix(readFile(t, r, "openspec/config.yaml"), "schema: team-flow\n") {
		t.Errorf("other: code=%d %+v", code, m.Findings)
	}

	r = testutil.NewRepo(t)
	r.Write("openspec/config.yaml", "schema: docket\noperations:\n  archive:\n    guidance:\n      - mine\n")
	code, m, _, _ = doInit(t, r, r.Dir, "1.7.0", nil, "--force")
	if code != 1 || !hasFinding(m, "guidance-differs", "openspec/config.yaml") || !strings.Contains(readFile(t, r, "openspec/config.yaml"), "- mine") {
		t.Errorf("guidance: code=%d %+v", code, m.Findings)
	}

	r = testutil.NewRepo(t)
	r.Write("openspec/config.yaml", "schema:\n  docket\n")
	before := snapshot(r.Dir)
	code, m, _, calls := doInit(t, r, r.Dir, "1.7.0", nil)
	if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != "openspec/config.yaml" || snapshot(r.Dir) != before || strings.Contains(calls, "init") {
		t.Errorf("unreadable: code=%d %+v", code, m.Unreadable)
	}
}

func TestInitUnreadableStopsEverything(t *testing.T) {
	r := testutil.NewRepo(t)
	if err := os.MkdirAll(filepath.Join(r.Dir, skillPath), 0o755); err != nil {
		t.Fatal(err)
	}
	code, m, _, calls := doInit(t, r, r.Dir, "1.7.0", nil, "--tools", "claude")
	if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != skillPath || strings.Contains(calls, "init") {
		t.Errorf("code=%d %+v calls=%q", code, m.Unreadable, calls)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "openspec")); err == nil {
		t.Error("openspec written")
	}
	if m.Result.Files == nil || m.Result.OpenspecInit != nil {
		t.Errorf("%+v", m.Result)
	}
}

func TestInitSymlinkRefused(t *testing.T) {
	r := testutil.NewRepo(t)
	other := t.TempDir()
	if err := os.MkdirAll(filepath.Join(r.Dir, ".claude/skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, filepath.Join(r.Dir, ".claude/skills/docket")); err != nil {
		t.Fatal(err)
	}
	code, m, _, calls := doInit(t, r, r.Dir, "1.7.0", nil, "--tools", "claude", "--force")
	if code != 1 || strings.Contains(calls, "init") || !hasFinding(m, "symlink-refused", ".claude/skills/docket") {
		t.Fatalf("code=%d calls=%q %+v", code, calls, m.Findings)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "openspec")); err == nil {
		t.Error("openspec written")
	}
	if es, _ := os.ReadDir(other); len(es) != 0 {
		t.Error("wrote through link")
	}
}

func TestInitIgnored(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write(".gitignore", "a\nb\n.claude/\n")
	ign := readFile(t, r, ".gitignore")
	code, m, _, _ := doInit(t, r, r.Dir, "1.7.0", nil, "--tools", "claude")
	var msg string
	for _, f := range m.Findings {
		if f.Rule == "install-ignored" && f.File == skillPath {
			msg = f.Message
		}
	}
	if code != 1 || !strings.Contains(msg, ".gitignore:3: .claude/") {
		t.Fatalf("code=%d %+v", code, m.Findings)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, skillPath)); err != nil || readFile(t, r, ".gitignore") != ign {
		t.Error("write or ignore file")
	}
}

func TestInitOpenSpecUnusable(t *testing.T) {
	for _, missing := range []bool{false, true} {
		r := testutil.NewRepo(t)
		cmd, log := testutil.FakeOpenSpec(t, "1.6.2")
		if missing {
			cmd = filepath.Join(t.TempDir(), "none")
		}
		code, _, out, calls := doInitCmd(t, r, r.Dir, cmd, log, nil, "--tools", "claude")
		want := "openspec-unsupported"
		if missing {
			want = "openspec-missing"
		}
		if code != 4 || strings.Contains(calls, "init") || !strings.Contains(out, want) {
			t.Errorf("missing=%v: code=%d calls=%q out=%q", missing, code, calls, out)
		}
		if _, err := os.Stat(filepath.Join(r.Dir, "openspec")); err == nil {
			t.Errorf("missing=%v: wrote openspec", missing)
		}
	}
}

func TestInitOpenSpecFails(t *testing.T) {
	r := testutil.NewRepo(t)
	code, m, _, calls := doInit(t, r, r.Dir, "1.7.0", nil, "--tools", "nosuch")
	if code != 1 || !strings.Contains(calls, "init") {
		t.Fatalf("code=%d calls=%q", code, calls)
	}
	if len(m.Findings) != 1 || m.Findings[0].Rule != "openspec-failed" || !strings.Contains(m.Findings[0].Message, "nosuch") {
		t.Errorf("%+v", m.Findings)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, skillPath)); err == nil {
		t.Error("wrote docket files")
	}
	if m.Result.OpenspecInit == nil || *m.Result.OpenspecInit != "nosuch" || len(m.Result.Files) != 0 || m.Result.Files == nil {
		t.Errorf("%+v", m.Result)
	}
}

func TestInitListsWhatOpenSpecChanged(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"tracked.txt": "t\n"})
	r.Write("before.txt", "untracked before")
	const odd = `we "ird" name.txt`
	code, m, _, _ := doInit(t, r, r.Dir, "1.7.0", []string{"FAKE_OPENSPEC_TOUCH=tracked.txt", "FAKE_OPENSPEC_CREATE=" + odd}, "--tools", "claude")
	if code != 0 {
		t.Fatal(code)
	}
	kind := map[string]string{}
	for _, f := range m.Result.Files {
		kind[f.Path] = f.Kind
	}
	if status(m, "tracked.txt") != "updated" || kind["tracked.txt"] != "openspec" {
		t.Errorf("tracked %+v", m.Result.Files)
	}
	if status(m, odd) != "created" || kind[odd] != "openspec" {
		t.Errorf("odd %+v", m.Result.Files)
	}
	if status(m, "before.txt") != "" {
		t.Error("listed a file that existed before")
	}
}

func TestInitText(t *testing.T) {
	r := testutil.NewRepo(t)
	code, _, out, _ := doInit(t, r, r.Dir, "1.7.0", nil, "text", "--tools", "claude")
	if code != 0 {
		t.Fatal(code)
	}
	for _, want := range []string{"created  " + skillPath, "created  openspec/config.yaml", "Next steps:", "1. Commit", "2. "} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
}
