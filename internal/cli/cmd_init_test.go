package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/testutil"
)

const schemaDir = "openspec/schemas/docket/"

func runInit(t *testing.T, dir, openspec string, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+openspec)
	code := Run(Options{Args: append([]string{"init"}, args...), Dir: dir, Env: env, Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

func committedRepo(t *testing.T) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{"README.md": "x\n"})
	return r
}

func assertSchemaInstalled(t *testing.T, dir string) {
	t.Helper()
	n := 0
	err := fs.WalkDir(docket.Schema(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		n++
		want, _ := fs.ReadFile(docket.Schema(), p)
		got, rerr := os.ReadFile(filepath.Join(dir, schemaDir, p))
		if rerr != nil || !bytes.Equal(got, want) {
			t.Errorf("%s not identical (err %v)", p, rerr)
		}
		return nil
	})
	if err != nil || n == 0 {
		t.Fatalf("walk: %v n=%d", err, n)
	}
}

func resultOf(t *testing.T, so string) map[string]any {
	t.Helper()
	res, ok := decode(t, so)["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not an object: %s", so)
	}
	return res
}

func TestFreshRepository(t *testing.T) {
	r := committedRepo(t)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t))
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "openspec/config.yaml")); err != nil {
		t.Error(err)
	}
	assertSchemaInstalled(t, r.Dir)
}

func TestJSONOnSuccess(t *testing.T) {
	r := committedRepo(t)
	code, so, _ := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	m := decode(t, so)
	if code != 0 || m["ok"] != true || resultOf(t, so)["openspec_initialized"] != true {
		t.Fatalf("exit %d: %s", code, so)
	}
}

func TestExistingOpenSpecProject(t *testing.T) {
	r := committedRepo(t)
	cmd, log := testutil.FakeOpenSpec(t, "1.14.3")
	r.Write("openspec/config.yaml", "schema: x\n")
	code, so, se := runInit(t, r.Dir, cmd)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	b, _ := os.ReadFile(log)
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "init") {
			t.Errorf("init was run: %q", l)
		}
	}
	assertSchemaInstalled(t, r.Dir)
}

func TestLocallyEditedTemplate(t *testing.T) {
	r := committedRepo(t)
	cmd := testutil.OpenSpecCmd(t)
	if code, so, se := runInit(t, r.Dir, cmd); code != 0 {
		t.Fatalf("%d %s %s", code, so, se)
	}
	rel := schemaDir + "templates/proposal.md"
	p := filepath.Join(r.Dir, rel)
	if err := os.WriteFile(p, []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, so, _ := runInit(t, r.Dir, cmd, "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s", code, so)
	}
	if b, _ := os.ReadFile(p); string(b) != "mine\n" {
		t.Errorf("file changed")
	}
	fnds, _ := decode(t, so)["findings"].([]any)
	if len(fnds) != 1 {
		t.Fatalf("findings: %s", so)
	}
	f := fnds[0].(map[string]any)
	if f["rule"] != "schema-differs" || f["file"] != rel {
		t.Errorf("finding: %v", f)
	}
}

func TestForceOverwrites(t *testing.T) {
	r := committedRepo(t)
	cmd := testutil.OpenSpecCmd(t)
	runInit(t, r.Dir, cmd)
	rel := schemaDir + "templates/proposal.md"
	if err := os.WriteFile(filepath.Join(r.Dir, rel), []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, so, _ := runInit(t, r.Dir, cmd, "--force", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, so)
	}
	assertSchemaInstalled(t, r.Dir)
	found := false
	for _, f := range resultOf(t, so)["files"].([]any) {
		m := f.(map[string]any)
		if m["path"] == rel {
			found = m["status"] == "overwritten"
		}
	}
	if !found {
		t.Errorf("no overwritten status: %s", so)
	}
}

func TestRerunNoDifferences(t *testing.T) {
	r := committedRepo(t)
	cmd := testutil.OpenSpecCmd(t)
	runInit(t, r.Dir, cmd)
	times := map[string]int64{}
	err := filepath.WalkDir(filepath.Join(r.Dir, schemaDir), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			i, ierr := d.Info()
			if ierr != nil {
				return ierr
			}
			times[p] = i.ModTime().UnixNano()
		}
		return err
	})
	if err != nil || len(times) == 0 {
		t.Fatalf("walk: %v", err)
	}
	code, so, _ := runInit(t, r.Dir, cmd, "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, so)
	}
	for _, f := range resultOf(t, so)["files"].([]any) {
		if s := f.(map[string]any)["status"]; s != "unchanged" {
			t.Errorf("status %v", s)
		}
	}
	for p, want := range times {
		i, err := os.Stat(p)
		if err != nil || i.ModTime().UnixNano() != want {
			t.Errorf("%s mtime changed (err %v)", p, err)
		}
	}
}

func TestRoutingSectionPrinted(t *testing.T) {
	r := committedRepo(t)
	r.Write("AGENTS.md", "# agents\n")
	r.Write("CLAUDE.md", "# claude\n")
	r.Write("openspec/config.yaml", "schema: x\n")
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t))
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, se)
	}
	if !strings.Contains(so, "Add this section to AGENTS.md:") || !strings.Contains(so, docket.AgentsSection()) {
		t.Errorf("section missing:\n%s", so)
	}
	for f, want := range map[string]string{"AGENTS.md": "# agents\n", "CLAUDE.md": "# claude\n", "openspec/config.yaml": "schema: x\n"} {
		if b, _ := os.ReadFile(filepath.Join(r.Dir, f)); string(b) != want {
			t.Errorf("%s changed", f)
		}
	}
	_, so, _ = runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if got := resultOf(t, so)["agents_section"]; got != docket.AgentsSection() {
		t.Errorf("agents_section = %v", got)
	}
}

func TestCLIMissing(t *testing.T) {
	r := committedRepo(t)
	code, _, se := runInit(t, r.Dir, "/nonexistent/openspec")
	if code != 4 || !strings.Contains(se, "DOCKET_OPENSPEC") {
		t.Fatalf("exit %d: %s", code, se)
	}
}

func TestExistingOpenSpecWithoutCLI(t *testing.T) {
	r := committedRepo(t)
	r.Write("openspec/config.yaml", "schema: x\n")
	code, so, se := runInit(t, r.Dir, "/nonexistent/openspec")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	assertSchemaInstalled(t, r.Dir)
}

func TestInitUnbornRepository(t *testing.T) {
	r := testutil.NewRepo(t)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if repoOf(t, so)["head"] != nil {
		t.Errorf("head not null: %s", so)
	}
}

func TestInitOpenSpecFails(t *testing.T) {
	r := committedRepo(t)
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.3")
	var so, se bytes.Buffer
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+cmd, "FAKE_OPENSPEC_FAIL=init")
	code := Run(Options{Args: []string{"init", "--json"}, Dir: r.Dir, Env: env, Stdout: &so, Stderr: &se})
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so.String(), se.String())
	}
	findings, _ := decode(t, so.String())["findings"].([]any)
	if len(findings) != 1 {
		t.Fatalf("findings: %s", so.String())
	}
	f := findings[0].(map[string]any)
	msg, _ := f["message"].(string)
	if f["rule"] != "openspec-failed" || !strings.Contains(msg, "3") || !strings.Contains(msg, "fake failure") {
		t.Errorf("finding: %v", f)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "openspec/schemas/docket")); err == nil {
		t.Error("schema directory was written")
	}
}

func TestInitUnsupportedVersion(t *testing.T) {
	r := committedRepo(t)
	cmd, _ := testutil.FakeOpenSpec(t, "1.3.0")
	code, so, se := runInit(t, r.Dir, cmd)
	if code != 4 || !strings.Contains(se, "1.3.0") || !strings.Contains(se, "1.14.x") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "openspec")); err == nil {
		t.Error("openspec/ was created")
	}
}

func TestInitVersionCommandFails(t *testing.T) {
	r := committedRepo(t)
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.3")
	var so, se bytes.Buffer
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+cmd, "FAKE_OPENSPEC_FAIL=--version")
	code := Run(Options{Args: []string{"init", "--json"}, Dir: r.Dir, Env: env, Stdout: &so, Stderr: &se})
	if code != 4 || !strings.Contains(se.String(), "DOCKET_OPENSPEC") || !strings.Contains(se.String(), "fake failure") {
		t.Fatalf("exit %d\n%s\n%s", code, so.String(), se.String())
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "openspec")); err == nil {
		t.Error("openspec directory was created")
	}
}

// snapshot maps every path under dir except .git (links not followed) to its
// content or link target.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == ".git" {
			return filepath.SkipDir
		}
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			tgt, _ := os.Readlink(p)
			m[rel] = "-> " + tgt
		case d.IsDir():
			m[rel] = "dir"
		default:
			b, _ := os.ReadFile(p)
			m[rel] = string(b)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func sameSnapshot(t *testing.T, before, after map[string]string) {
	t.Helper()
	for p, v := range after {
		if w, ok := before[p]; !ok || w != v {
			t.Errorf("%s changed or appeared", p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			t.Errorf("%s vanished", p)
		}
	}
}

func TestSymlinkedSchemaFileRefused(t *testing.T) {
	r := committedRepo(t)
	r.Write("AGENTS.md", "# agents\n")
	r.Write("openspec/config.yaml", "schema: x\n")
	rel := schemaDir + "templates/proposal.md"
	if err := os.MkdirAll(filepath.Join(r.Dir, schemaDir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(r.Dir, "AGENTS.md"), filepath.Join(r.Dir, rel)); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, r.Dir)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--force", "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	fnds, _ := decode(t, so)["findings"].([]any)
	if len(fnds) != 1 {
		t.Fatalf("findings: %s", so)
	}
	f := fnds[0].(map[string]any)
	if f["rule"] != "symlink-refused" || f["file"] != rel {
		t.Errorf("finding: %v", f)
	}
	sameSnapshot(t, before, snapshot(t, r.Dir))
}

func TestSymlinkedSchemasDirectoryRefused(t *testing.T) {
	r := committedRepo(t)
	r.Write("openspec/config.yaml", "schema: x\n")
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(r.Dir, "openspec/schemas")); err != nil {
		t.Fatal(err)
	}
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--force", "--json")
	if code != 1 || !strings.Contains(so, "symlink-refused") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if ents, _ := os.ReadDir(outside); len(ents) != 0 {
		t.Errorf("outside directory written: %v", ents)
	}
}

func TestSymlinkedOpenSpecRefused(t *testing.T) {
	r := committedRepo(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(r.Dir, "openspec")); err != nil {
		t.Fatal(err)
	}
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 1 || !strings.Contains(so, "symlink-refused") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if ents, _ := os.ReadDir(outside); len(ents) != 0 {
		t.Errorf("outside directory written: %v", ents)
	}
}

func TestPartialFailureKeepsResult(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	r := committedRepo(t)
	r.Write("openspec/config.yaml", "schema: x\n")
	// An empty, read-only skill directory passes the checks and fails its
	// write; the files written before it must still be reported.
	dir := filepath.Join(r.Dir, ".agents/skills/docket")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	files, _ := resultOf(t, so)["files"].([]any)
	if len(files) < 2 {
		t.Errorf("files missing on partial failure: %s", so)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json"); code != 0 {
		t.Fatalf("rerun exit %d\n%s\n%s", code, so, se)
	}
}

const skillRels = ".claude/skills/docket/SKILL.md .agents/skills/docket/SKILL.md"

func TestFreshRepositoryGetsTheSkill(t *testing.T) {
	r := committedRepo(t)
	if code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	want, err := fs.ReadFile(docket.Skill(), "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range strings.Fields(skillRels) {
		got, err := os.ReadFile(filepath.Join(r.Dir, rel))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s differs (err %v)", rel, err)
		}
	}
}

func TestEditedSkillFile(t *testing.T) {
	r := committedRepo(t)
	cmd := testutil.OpenSpecCmd(t)
	if code, so, se := runInit(t, r.Dir, cmd); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	const rel = ".agents/skills/docket/SKILL.md"
	p := filepath.Join(r.Dir, rel)
	if err := os.WriteFile(p, []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, so, _ := runInit(t, r.Dir, cmd, "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s", code, so)
	}
	fnds, _ := decode(t, so)["findings"].([]any)
	if len(fnds) != 1 {
		t.Fatalf("findings: %s", so)
	}
	if f := fnds[0].(map[string]any); f["rule"] != "skill-differs" || f["file"] != rel {
		t.Errorf("finding: %v", f)
	}
	if b, _ := os.ReadFile(p); string(b) != "mine\n" {
		t.Error("file changed")
	}
	code, so, _ = runInit(t, r.Dir, cmd, "--force", "--json")
	if code != 0 {
		t.Fatalf("force exit %d\n%s", code, so)
	}
	found := false
	for _, f := range resultOf(t, so)["files"].([]any) {
		m := f.(map[string]any)
		if m["path"] == rel {
			found = m["status"] == "overwritten"
		}
	}
	if !found {
		t.Errorf("not overwritten: %s", so)
	}
}

func TestSymlinkedSkillDirectoryRefused(t *testing.T) {
	r := committedRepo(t)
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(r.Dir, ".claude/skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(r.Dir, ".claude/skills/docket")); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, r.Dir)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--force", "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	fnds, _ := decode(t, so)["findings"].([]any)
	if len(fnds) != 1 {
		t.Fatalf("findings: %s", so)
	}
	if f := fnds[0].(map[string]any); f["rule"] != "symlink-refused" || f["file"] != ".claude/skills/docket" {
		t.Errorf("finding: %v", f)
	}
	sameSnapshot(t, before, snapshot(t, r.Dir))
	if ents, _ := os.ReadDir(outside); len(ents) != 0 {
		t.Errorf("outside directory written: %v", ents)
	}
}

func TestUnreadableSkillFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	r := committedRepo(t)
	r.Write(".agents/skills/docket/SKILL.md", "x\n")
	p := filepath.Join(r.Dir, ".agents/skills/docket/SKILL.md")
	if err := os.Chmod(p, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	before := snapshot(t, r.Dir)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 3 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	un, _ := decode(t, so)["unreadable"].([]any)
	if len(un) != 1 || un[0].(map[string]any)["source"] != ".agents/skills/docket/SKILL.md" {
		t.Errorf("unreadable: %s", so)
	}
	sameSnapshot(t, before, snapshot(t, r.Dir))
}

func TestLinkAndUnreadableTogether(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	r := committedRepo(t)
	r.Write("openspec/config.yaml", "schema: x\n")
	r.Write(schemaDir+"schema.yaml", "x\n")
	bad := filepath.Join(r.Dir, schemaDir, "schema.yaml")
	if err := os.Chmod(bad, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(bad, 0o644) })
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(r.Dir, ".claude/skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(r.Dir, ".claude/skills/docket")); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, r.Dir)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 3 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	env := decode(t, so)
	fnds, _ := env["findings"].([]any)
	un, _ := env["unreadable"].([]any)
	if len(fnds) != 1 || fnds[0].(map[string]any)["rule"] != "symlink-refused" || len(un) != 1 {
		t.Errorf("envelope: %s", so)
	}
	sameSnapshot(t, before, snapshot(t, r.Dir))
	if ents, _ := os.ReadDir(outside); len(ents) != 0 {
		t.Errorf("outside written")
	}
}
