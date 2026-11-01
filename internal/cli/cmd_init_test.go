package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/testutil"
	"github.com/wake/docket/internal/userdirs"
)

// schemaDir is the schema folder under the data home.
const schemaDir = "openspec/schemas/docket/"

// dataHome is the XDG_DATA_HOME the runs on the repository at dir share, a
// sibling of the repository so it never holds the real one.
func dataHome(dir string) string { return dir + ".data" }

// userHome is the HOME the runs on the repository at dir share, a sibling of
// the repository so it never holds the real one.
func userHome(dir string) string { return dir + ".home" }

// machineFile is the path of a schema file under the per-machine directory.
func machineFile(dir, rel string) string { return filepath.Join(dataHome(dir), schemaDir, rel) }

func runInit(t *testing.T, dir, openspec string, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+openspec, "XDG_DATA_HOME="+dataHome(dir), "HOME="+userHome(dir))
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
		got, rerr := os.ReadFile(machineFile(dir, p))
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
	if b, err := os.ReadFile(filepath.Join(r.Dir, "openspec/config.yaml")); err != nil || !strings.HasPrefix(string(b), "schema: docket\n") {
		t.Errorf("config.yaml = %q (err %v)", b, err)
	}
	if _, err := os.Lstat(filepath.Join(r.Dir, "openspec/schemas/docket")); err == nil {
		t.Error("the schema was installed in the repository")
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
	err := fs.WalkDir(docket.Schema(), ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if got := fileStatus(t, so, machineFile(r.Dir, p)); got != "written" {
				t.Errorf("%s: %v", p, got)
			}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := fileStatus(t, so, "openspec/config.yaml"); got != "written" {
		t.Errorf("config.yaml: %v", got)
	}
}

func TestExistingOpenSpecProject(t *testing.T) {
	r := committedRepo(t)
	cmd, log := testutil.FakeOpenSpec(t, "1.14.3")
	r.Write("openspec/config.yaml", "schema: docket\n")
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
	rel := machineFile(r.Dir, "templates/proposal.md")
	p := rel
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
	rel := machineFile(r.Dir, "templates/proposal.md")
	skill := ".agents/skills/docket/SKILL.md"
	for _, p := range []string{rel, filepath.Join(r.Dir, skill)} {
		if err := os.WriteFile(p, []byte("mine\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	code, so, _ := runInit(t, r.Dir, cmd, "--force", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, so)
	}
	assertSchemaInstalled(t, r.Dir)
	want, _ := fs.ReadFile(docket.Skills()[0].Files, "SKILL.md")
	if got := readFile(t, r.Dir, skill); got != string(want) {
		t.Errorf("skill not restored: %q", got)
	}
	for _, p := range []string{rel, skill} {
		if got := fileStatus(t, so, p); got != "overwritten" {
			t.Errorf("%s: %q", p, got)
		}
	}
}

func TestRerunNoDifferences(t *testing.T) {
	r := committedRepo(t)
	cmd := testutil.OpenSpecCmd(t)
	runInit(t, r.Dir, cmd)
	times := map[string]int64{}
	err := filepath.WalkDir(filepath.Join(dataHome(r.Dir), schemaDir), func(p string, d fs.DirEntry, err error) error {
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
	for _, rel := range append(skillRels(t), "AGENTS.md", "CLAUDE.md", "openspec/config.yaml") {
		i, err := os.Stat(filepath.Join(r.Dir, rel))
		if err != nil {
			t.Fatal(err)
		}
		times[filepath.Join(r.Dir, rel)] = i.ModTime().UnixNano()
	}
	for _, base := range []string{".claude", ".agents"} {
		for _, rel := range []string{"SKILL.md", "agents/openai.yaml"} {
			p := skillRel(filepath.Join(userHome(r.Dir), base), rel)
			i, err := os.Stat(p)
			if err != nil {
				t.Fatal(err)
			}
			times[p] = i.ModTime().UnixNano()
		}
	}
	code, so, _ := runInit(t, r.Dir, cmd, "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, so)
	}
	files := resultOf(t, so)["files"].([]any)
	if len(files) != len(times) {
		t.Errorf("%d files listed, %d tracked", len(files), len(times))
	}
	for _, f := range files {
		if s := f.(map[string]any)["status"]; s != "unchanged" {
			t.Errorf("status %v: %v", s, f)
		}
	}
	for p, want := range times {
		i, err := os.Stat(p)
		if err != nil || i.ModTime().UnixNano() != want {
			t.Errorf("%s mtime changed (err %v)", p, err)
		}
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
	r.Write("openspec/config.yaml", "schema: docket\n")
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
	r.Write("openspec/config.yaml", "schema: docket\n")
	rel := machineFile(r.Dir, "templates/proposal.md")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(r.Dir, "AGENTS.md"), rel); err != nil {
		t.Fatal(err)
	}
	before, machine := snapshot(t, r.Dir), snapshot(t, dataHome(r.Dir))
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
	sameSnapshot(t, machine, snapshot(t, dataHome(r.Dir)))
}

func TestSymlinkedSchemasDirectoryRefused(t *testing.T) {
	r := committedRepo(t)
	r.Write("openspec/config.yaml", "schema: docket\n")
	outside := t.TempDir()
	link := machineFile(r.Dir, "")
	if err := os.MkdirAll(filepath.Dir(filepath.Clean(link)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Clean(link)); err != nil {
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
	r.Write("openspec/config.yaml", "schema: docket\n")
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
	listed := map[string]bool{}
	for _, f := range files {
		listed[f.(map[string]any)["path"].(string)] = true
	}
	if !listed[machineFile(r.Dir, "schema.yaml")] || listed[".agents/skills/docket/SKILL.md"] {
		t.Errorf("wrong files on partial failure: %s", so)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json"); code != 0 {
		t.Fatalf("rerun exit %d\n%s\n%s", code, so, se)
	}
}

func TestFreshRepositoryGetsTheSkill(t *testing.T) {
	r := committedRepo(t)
	if code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	for _, root := range []string{".claude/skills", ".agents/skills"} {
		for _, sk := range docket.Skills() {
			err := fs.WalkDir(sk.Files, ".", func(p string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				want, err := fs.ReadFile(sk.Files, p)
				if err != nil {
					return err
				}
				rel := root + "/" + sk.Name + "/" + p
				got, err := os.ReadFile(filepath.Join(r.Dir, rel))
				if err != nil || !bytes.Equal(got, want) {
					t.Errorf("%s differs (err %v)", rel, err)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

// skillRels lists every installed skill file, relative to the repository.
func skillRels(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, root := range []string{".claude/skills", ".agents/skills"} {
		for _, sk := range docket.Skills() {
			err := fs.WalkDir(sk.Files, ".", func(p string, d fs.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					out = append(out, root+"/"+sk.Name+"/"+p)
				}
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	return out
}

func TestSkillCopiesAgree(t *testing.T) {
	r := committedRepo(t)
	if code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	files := func(root, name string) map[string][]byte {
		m := map[string][]byte{}
		base := filepath.Join(r.Dir, root, name)
		err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.Type().IsRegular() {
				return err
			}
			rel, _ := filepath.Rel(base, p)
			m[rel], err = os.ReadFile(p)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	for _, sk := range docket.Skills() {
		a, b := files(".claude/skills", sk.Name), files(".agents/skills", sk.Name)
		if len(a) == 0 || len(a) != len(b) {
			t.Errorf("%s: %d files vs %d", sk.Name, len(a), len(b))
		}
		for rel, data := range a {
			if !bytes.Equal(data, b[rel]) {
				t.Errorf("%s/%s differs between the copies", sk.Name, rel)
			}
		}
		var fm struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		}
		text := string(a["SKILL.md"])
		rest := strings.TrimPrefix(text, "---\n")
		end := strings.Index(rest, "\n---\n")
		if rest == text || end < 0 {
			t.Fatalf("%s: no frontmatter", sk.Name)
		}
		if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
			t.Fatal(err)
		}
		if fm.Name != sk.Name || len(fm.Description) > 1024 {
			t.Errorf("%s: name %q, description length %d", sk.Name, fm.Name, len(fm.Description))
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
	r.Write("openspec/config.yaml", "schema: docket\n")
	bad := machineFile(r.Dir, "schema.yaml")
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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
	before, machine := snapshot(t, r.Dir), snapshot(t, dataHome(r.Dir))
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
	sameSnapshot(t, machine, snapshot(t, dataHome(r.Dir)))
	if ents, _ := os.ReadDir(outside); len(ents) != 0 {
		t.Errorf("outside written")
	}
}

func fileStatus(t *testing.T, so, rel string) string {
	t.Helper()
	for _, f := range resultOf(t, so)["files"].([]any) {
		m := f.(map[string]any)
		if m["path"] == rel {
			return m["status"].(string)
		}
	}
	t.Fatalf("%s not listed: %s", rel, so)
	return ""
}

func readFile(t *testing.T, dir, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBlockAddedToExistingAgentsMD(t *testing.T) {
	r := committedRepo(t)
	r.Write("AGENTS.md", "# agents\n")
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got, want := readFile(t, r.Dir, "AGENTS.md"), "# agents\n\n"+docket.Block()+"\n"; got != want {
		t.Errorf("AGENTS.md = %q", got)
	}
	if s := fileStatus(t, so, "AGENTS.md"); s != "written" {
		t.Errorf("status %s", s)
	}
}

func TestAgentsMDCreated(t *testing.T) {
	r := committedRepo(t)
	if code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got := readFile(t, r.Dir, "AGENTS.md"); got != docket.Block()+"\n" {
		t.Errorf("AGENTS.md = %q", got)
	}
}

func TestEditedBlock(t *testing.T) {
	r := committedRepo(t)
	cmd := testutil.OpenSpecCmd(t)
	r.Write("AGENTS.md", "before\n")
	if code, so, se := runInit(t, r.Dir, cmd); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	lines := strings.Split(readFile(t, r.Dir, "AGENTS.md"), "\n")
	for i, l := range lines {
		if l == "<!-- docket:start -->" {
			lines[i+1] = "edited"
		}
	}
	edited := strings.Join(lines, "\n") + "after\n"
	r.Write("AGENTS.md", edited)
	code, so, _ := runInit(t, r.Dir, cmd, "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s", code, so)
	}
	fnds, _ := decode(t, so)["findings"].([]any)
	if len(fnds) != 1 {
		t.Fatalf("findings: %s", so)
	}
	f := fnds[0].(map[string]any)
	if f["rule"] != "agents-block-differs" || f["file"] != "AGENTS.md" {
		t.Errorf("finding: %v", f)
	}
	if got := readFile(t, r.Dir, "AGENTS.md"); got != edited {
		t.Errorf("AGENTS.md changed: %q", got)
	}
	code, so, _ = runInit(t, r.Dir, cmd, "--force", "--json")
	if code != 0 {
		t.Fatalf("force exit %d\n%s", code, so)
	}
	if s := fileStatus(t, so, "AGENTS.md"); s != "overwritten" {
		t.Errorf("status %s", s)
	}
	if got, want := readFile(t, r.Dir, "AGENTS.md"), "before\n\n"+docket.Block()+"\nafter\n"; got != want {
		t.Errorf("AGENTS.md = %q", got)
	}
}

func TestMalformedBlock(t *testing.T) {
	cases := []struct{ name, file, body string }{
		{"no end", "AGENTS.md", "<!-- docket:start -->\nx\n"},
		{"end first", "AGENTS.md", "<!-- docket:end -->\n<!-- docket:start -->\n"},
		{"two blocks", "AGENTS.md", "<!-- docket:start -->\n<!-- docket:end -->\n<!-- docket:start -->\n<!-- docket:end -->\n"},
		{"claude no end", "CLAUDE.md", "<!-- docket:start -->\nx\n"},
		{"claude imports and no end", "CLAUDE.md", "@AGENTS.md\n<!-- docket:start -->\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := committedRepo(t)
			r.Write(c.file, c.body)
			before := snapshot(t, r.Dir)
			code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
			if code != 1 {
				t.Fatalf("exit %d\n%s\n%s", code, so, se)
			}
			fnds, _ := decode(t, so)["findings"].([]any)
			if len(fnds) != 1 {
				t.Fatalf("findings: %s", so)
			}
			f := fnds[0].(map[string]any)
			if f["rule"] != "agents-block-malformed" || f["file"] != c.file {
				t.Errorf("finding: %v", f)
			}
			sameSnapshot(t, before, snapshot(t, r.Dir))
		})
	}
}

func TestUnreadableAgentsFiles(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		t.Run(name, func(t *testing.T) {
			r := committedRepo(t)
			r.Write(name, "x\n")
			p := filepath.Join(r.Dir, name)
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
			if len(un) != 1 || un[0].(map[string]any)["source"] != name {
				t.Errorf("unreadable: %s", so)
			}
			sameSnapshot(t, before, snapshot(t, r.Dir))
		})
	}
}

func TestClaudeMDLinkedElsewhereRefused(t *testing.T) {
	r := committedRepo(t)
	if err := os.Symlink("README.md", filepath.Join(r.Dir, "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, r.Dir)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	fnds, _ := decode(t, so)["findings"].([]any)
	if len(fnds) != 1 || fnds[0].(map[string]any)["rule"] != "symlink-refused" || fnds[0].(map[string]any)["file"] != "CLAUDE.md" {
		t.Errorf("findings: %s", so)
	}
	sameSnapshot(t, before, snapshot(t, r.Dir))
	if got := readFile(t, r.Dir, "README.md"); got != "x\n" {
		t.Errorf("README.md = %q", got)
	}
}

func TestClaudeMDImportsAgentsMD(t *testing.T) {
	r := committedRepo(t)
	r.Write("CLAUDE.md", "@AGENTS.md\n")
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got := readFile(t, r.Dir, "CLAUDE.md"); got != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md = %q", got)
	}
	if s := fileStatus(t, so, "CLAUDE.md"); s != "unchanged" {
		t.Errorf("status %s", s)
	}
	if got := readFile(t, r.Dir, "AGENTS.md"); got != docket.Block()+"\n" {
		t.Errorf("AGENTS.md = %q", got)
	}
}

func TestClaudeMDLinkedToAgentsMD(t *testing.T) {
	r := committedRepo(t)
	link := filepath.Join(r.Dir, "CLAUDE.md")
	if err := os.Symlink("AGENTS.md", link); err != nil {
		t.Fatal(err)
	}
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if tgt, err := os.Readlink(link); err != nil || tgt != "AGENTS.md" {
		t.Errorf("link: %q %v", tgt, err)
	}
	if got := readFile(t, r.Dir, "AGENTS.md"); got != docket.Block()+"\n" {
		t.Errorf("AGENTS.md = %q", got)
	}
	if s := fileStatus(t, so, "CLAUDE.md"); s != "unchanged" {
		t.Errorf("status %s", s)
	}
}

func TestClaudeMDCreated(t *testing.T) {
	r := committedRepo(t)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got := readFile(t, r.Dir, "CLAUDE.md"); got != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md = %q", got)
	}
	if s := fileStatus(t, so, "CLAUDE.md"); s != "written" {
		t.Errorf("status %s", s)
	}
}

func TestClaudeMDWithoutTheImport(t *testing.T) {
	r := committedRepo(t)
	r.Write("CLAUDE.md", "# claude\n")
	if code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got, want := readFile(t, r.Dir, "CLAUDE.md"), "# claude\n\n"+docket.Block()+"\n"; got != want {
		t.Errorf("CLAUDE.md = %q", got)
	}
}

func TestOtherConfigurationUnchanged(t *testing.T) {
	r := committedRepo(t)
	other := map[string]string{
		".claude/settings.json":        "{}\n",
		".claude/skills/mine/SKILL.md": "mine\n",
		"openspec/config.yaml":         "schema: docket\ncontext: |\n  x\n",
	}
	for p, c := range other {
		r.Write(p, c)
	}
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t))
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	for p, c := range other {
		if got := readFile(t, r.Dir, p); got != c {
			t.Errorf("%s changed: %q", p, got)
		}
	}
	if strings.Contains(so, "Add this section") {
		t.Errorf("printed section:\n%s", so)
	}
	_, so, _ = runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if _, ok := resultOf(t, so)["agents_section"]; ok {
		t.Errorf("agents_section present: %s", so)
	}
}

// initWith runs init on a bare repository with exactly the given environment
// variables added to a temporary HOME and no data home.
func initWith(t *testing.T, r *testutil.Repo, extra ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+testutil.OpenSpecCmd(t))
	code := Run(Options{Args: []string{"init", "--json"}, Dir: r.Dir, Env: append(env, extra...), Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

func nothingWritten(t *testing.T, r *testutil.Repo) {
	t.Helper()
	for _, p := range []string{"openspec", ".claude", "AGENTS.md"} {
		if _, err := os.Lstat(filepath.Join(r.Dir, p)); err == nil {
			t.Errorf("%s was written", p)
		}
	}
}

func TestInitDataHomeFromXDG(t *testing.T) {
	r := committedRepo(t)
	d := filepath.Join(t.TempDir(), "d")
	if code, so, se := initWith(t, r, "XDG_DATA_HOME="+d); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	want, _ := fs.ReadFile(docket.Schema(), "schema.yaml")
	if got, _ := os.ReadFile(filepath.Join(d, "openspec/schemas/docket/schema.yaml")); !bytes.Equal(got, want) {
		t.Errorf("schema.yaml = %q", got)
	}
}

func TestInitDataHomeDefault(t *testing.T) {
	r := committedRepo(t)
	h := t.TempDir()
	if code, so, se := initWith(t, r, "HOME="+h); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	want, _ := fs.ReadFile(docket.Schema(), "schema.yaml")
	if got, _ := os.ReadFile(filepath.Join(h, ".local/share/openspec/schemas/docket/schema.yaml")); !bytes.Equal(got, want) {
		t.Errorf("schema.yaml = %q", got)
	}
}

func TestInitRelativeDataHome(t *testing.T) {
	r := committedRepo(t)
	cmd, log := testutil.FakeOpenSpec(t, "1.14.3")
	var so, se bytes.Buffer
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+cmd, "XDG_DATA_HOME=data")
	code := Run(Options{Args: []string{"init"}, Dir: r.Dir, Env: env, Stdout: &so, Stderr: &se})
	if code != 4 || !strings.Contains(se.String(), "XDG_DATA_HOME") {
		t.Fatalf("exit %d\n%s\n%s", code, so.String(), se.String())
	}
	nothingWritten(t, r)
	if b, _ := os.ReadFile(log); len(bytes.TrimSpace(b)) != 0 {
		t.Errorf("openspec ran: %q", b)
	}
}

// bareEnv is env without HOME and XDG_DATA_HOME.
func bareEnv(env []string) []string {
	var bare []string
	for _, e := range env {
		if !strings.HasPrefix(e, "HOME=") && !strings.HasPrefix(e, "XDG_DATA_HOME=") {
			bare = append(bare, e)
		}
	}
	return bare
}

func TestInitNoHome(t *testing.T) {
	r := committedRepo(t)
	var so, se bytes.Buffer
	env := bareEnv(append(testutil.Env(t), "DOCKET_OPENSPEC="+testutil.OpenSpecCmd(t)))
	code := Run(Options{Args: []string{"init"}, Dir: r.Dir, Env: env, UserHome: nil, Stdout: &so, Stderr: &se})
	if code != 4 || !strings.Contains(se.String(), userdirs.ErrNoHome.Error()) {
		t.Fatalf("exit %d\n%s\n%s", code, so.String(), se.String())
	}
	nothingWritten(t, r)
}

func TestInitLinkedDataDirectoryFollowed(t *testing.T) {
	r := committedRepo(t)
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(tmp, "real"), filepath.Join(tmp, "link")); err != nil {
		t.Fatal(err)
	}
	if code, so, se := initWith(t, r, "XDG_DATA_HOME="+filepath.Join(tmp, "link")); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	want, _ := fs.ReadFile(docket.Schema(), "schema.yaml")
	if got, _ := os.ReadFile(filepath.Join(tmp, "real/openspec/schemas/docket/schema.yaml")); !bytes.Equal(got, want) {
		t.Errorf("schema.yaml = %q", got)
	}
}

func TestDefaultSchemaReplaced(t *testing.T) {
	r := committedRepo(t)
	r.Write("openspec/config.yaml", "schema: spec-driven\n\n# Project context (optional)\n")
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got := readFile(t, r.Dir, "openspec/config.yaml"); got != "schema: docket\n\n# Project context (optional)\n" {
		t.Errorf("config.yaml = %q", got)
	}
	if got := fileStatus(t, so, "openspec/config.yaml"); got != "written" {
		t.Errorf("status %s", got)
	}
}

func TestConfigFileCreated(t *testing.T) {
	r := committedRepo(t)
	r.Write("openspec/.keep", "")
	if code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got := readFile(t, r.Dir, "openspec/config.yaml"); got != "schema: docket\n" {
		t.Errorf("config.yaml = %q", got)
	}
}

func TestOtherSchemaLeft(t *testing.T) {
	r := committedRepo(t)
	r.Write("openspec/config.yaml", "schema: team-flow\n")
	code, so, _ := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--force", "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s", code, so)
	}
	if got := readFile(t, r.Dir, "openspec/config.yaml"); got != "schema: team-flow\n" {
		t.Errorf("config.yaml = %q", got)
	}
	fnds, _ := decode(t, so)["findings"].([]any)
	if len(fnds) != 1 {
		t.Fatalf("findings: %s", so)
	}
	if f := fnds[0].(map[string]any); f["rule"] != "config-schema-other" || f["file"] != "openspec/config.yaml" {
		t.Errorf("finding: %v", f)
	}
}

func TestUnreadableSchemaKey(t *testing.T) {
	r := committedRepo(t)
	r.Write("openspec/config.yaml", "schema:\n  docket\n")
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 3 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	un, _ := decode(t, so)["unreadable"].([]any)
	if len(un) != 1 || un[0].(map[string]any)["source"] != "openspec/config.yaml" {
		t.Errorf("unreadable: %s", so)
	}
	if _, err := os.Lstat(dataHome(r.Dir)); err == nil {
		t.Error("the per-machine directory exists")
	}
	if _, err := os.Lstat(filepath.Join(r.Dir, ".claude")); err == nil {
		t.Error("a skill was written")
	}
}

var oldCopyFiles = []string{"schema.yaml", "templates/design.md", "templates/plan.md", "templates/proposal.md", "templates/spec.md"}

func TestShippedProjectCopyRemoved(t *testing.T) {
	r := committedRepo(t)
	testutil.InstallOldSchema(t, r.Dir)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	assertSchemaInstalled(t, r.Dir)
	for _, p := range oldCopyFiles {
		if st := fileStatus(t, so, "openspec/schemas/docket/"+p); st != "removed" {
			t.Errorf("%s: %s", p, st)
		}
	}
	if _, err := os.Lstat(filepath.Join(r.Dir, "openspec/schemas/docket")); err == nil {
		t.Error("the project copy remains")
	}
	if left, _ := filepath.Glob(filepath.Join(r.Dir, "openspec", ".docket-removed-*")); len(left) != 0 {
		t.Errorf("left behind: %v", left)
	}
}

// appendLine adds a line to the file at rel under dir.
func appendLine(t *testing.T, dir, rel string) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(dir, rel), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString("edited\n"); err != nil {
		t.Fatal(err)
	}
}

// findingOf returns the finding with the rule, or nil.
func findingOf(t *testing.T, so, rule string) map[string]any {
	t.Helper()
	fnds, _ := decode(t, so)["findings"].([]any)
	for _, f := range fnds {
		if m := f.(map[string]any); m["rule"] == rule {
			return m
		}
	}
	return nil
}

func TestEditedProjectCopyKept(t *testing.T) {
	r := committedRepo(t)
	testutil.InstallSchema(t, r.Dir)
	appendLine(t, r.Dir, "openspec/schemas/docket/templates/proposal.md")
	before := snapshot(t, filepath.Join(r.Dir, "openspec/schemas/docket"))
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--force", "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	sameSnapshot(t, before, snapshot(t, filepath.Join(r.Dir, "openspec/schemas/docket")))
	f := findingOf(t, so, "schema-edited")
	if f == nil || f["file"] != "openspec/schemas/docket" || f["message"] != "differs from every schema Docket shipped, and is used instead of the per-machine one" {
		t.Errorf("finding: %v", f)
	}
}

func TestProjectCopyWithExtraFileKept(t *testing.T) {
	r := committedRepo(t)
	testutil.InstallSchema(t, r.Dir)
	r.Write("openspec/schemas/docket/templates/extra.md", "mine\n")
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 1 || findingOf(t, so, "schema-edited") == nil {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got := readFile(t, r.Dir, "openspec/schemas/docket/templates/extra.md"); got != "mine\n" {
		t.Errorf("extra.md = %q", got)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "openspec/schemas/docket/schema.yaml")); err != nil {
		t.Error(err)
	}
}

func TestRemovalWaitsForTheMachineCopy(t *testing.T) {
	r := committedRepo(t)
	if code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t)); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	testutil.InstallSchema(t, r.Dir)
	edited := machineFile(r.Dir, "schema.yaml")
	if err := os.WriteFile(edited, append(mustRead(t, edited), "# mine\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, filepath.Join(r.Dir, "openspec/schemas/docket"))
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	sameSnapshot(t, before, snapshot(t, filepath.Join(r.Dir, "openspec/schemas/docket")))
	for _, rule := range []string{"schema-differs", "schema-project-copy"} {
		if findingOf(t, so, rule) == nil {
			t.Errorf("no %s finding: %s", rule, so)
		}
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSymlinkInProjectCopyRefused(t *testing.T) {
	r := committedRepo(t)
	testutil.InstallSchema(t, r.Dir)
	r.Write("AGENTS.md", "# agents\n")
	link := filepath.Join(r.Dir, "openspec/schemas/docket/templates/proposal.md")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(r.Dir, "AGENTS.md"), link); err != nil {
		t.Fatal(err)
	}
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if f := findingOf(t, so, "symlink-refused"); f == nil || f["file"] != "openspec/schemas/docket/templates/proposal.md" {
		t.Errorf("finding: %v", f)
	}
	if _, err := os.Lstat(dataHome(r.Dir)); err == nil {
		t.Error("the per-machine directory was created")
	}
}

func TestFolderWithoutSchemaLeft(t *testing.T) {
	r := committedRepo(t)
	r.Write("openspec/schemas/docket/templates/proposal.md", "mine\n")
	before := snapshot(t, filepath.Join(r.Dir, "openspec/schemas/docket"))
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	sameSnapshot(t, before, snapshot(t, filepath.Join(r.Dir, "openspec/schemas/docket")))
	for _, f := range decode(t, so)["findings"].([]any) {
		if f.(map[string]any)["file"] == "openspec/schemas/docket" {
			t.Errorf("finding names the folder: %v", f)
		}
	}
}

// skillRel is a file of the docket-init skill under a user-level config folder.
func skillRel(base, rel string) string { return filepath.Join(base, "skills/docket-init", rel) }

// initIn runs init --json on r with extra variables added to the shared ones.
func initIn(t *testing.T, r *testutil.Repo, extra []string, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+testutil.OpenSpecCmd(t), "XDG_DATA_HOME="+dataHome(r.Dir), "HOME="+userHome(r.Dir))
	code := Run(Options{Args: append([]string{"init", "--json"}, args...), Dir: r.Dir, Env: append(env, extra...), Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

func TestInitUserSkill(t *testing.T) {
	r := committedRepo(t)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t))
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	h := userHome(r.Dir)
	for _, base := range []string{filepath.Join(h, ".claude"), filepath.Join(h, ".agents")} {
		for _, rel := range []string{"SKILL.md", "agents/openai.yaml"} {
			if p := skillRel(base, rel); !strings.Contains(so, "written     "+p+"\n") {
				t.Errorf("no written line for %s:\n%s", p, so)
			}
		}
	}
	r2 := committedRepo(t)
	code, so, se = initIn(t, r2, nil)
	if code != 0 {
		t.Fatalf("json exit %d\n%s\n%s", code, so, se)
	}
	files := map[string]string{}
	for _, f := range resultOf(t, so)["files"].([]any) {
		m := f.(map[string]any)
		files[m["path"].(string)] = m["status"].(string)
	}
	for _, rel := range []string{".claude/skills/docket-init/SKILL.md", ".claude/skills/docket-init/agents/openai.yaml", ".agents/skills/docket-init/SKILL.md", ".agents/skills/docket-init/agents/openai.yaml"} {
		if files[filepath.Join(userHome(r2.Dir), rel)] != "written" {
			t.Errorf("%s: %v", rel, files)
		}
	}
}

func TestInitClaudeConfigDir(t *testing.T) {
	r := committedRepo(t)
	cfg := filepath.Join(t.TempDir(), "claude")
	if code, so, se := initIn(t, r, []string{"CLAUDE_CONFIG_DIR=" + cfg}); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	for _, p := range []string{skillRel(cfg, "SKILL.md"), skillRel(filepath.Join(userHome(r.Dir), ".agents"), "SKILL.md")} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing %s", p)
		}
	}
	if _, err := os.Lstat(filepath.Join(userHome(r.Dir), ".claude")); err == nil {
		t.Error("home .claude was written")
	}
}

func TestInitRelativeClaudeConfigDir(t *testing.T) {
	r := committedRepo(t)
	code, so, se := initIn(t, r, []string{"CLAUDE_CONFIG_DIR=claude"})
	if code != 4 || !strings.Contains(se, "CLAUDE_CONFIG_DIR is not an absolute path") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	nothingWritten(t, r)
	for _, d := range []string{dataHome(r.Dir), filepath.Join(userHome(r.Dir), ".agents")} {
		if _, err := os.Lstat(d); err == nil {
			t.Errorf("%s was written", d)
		}
	}
}

func TestInitUserSkillDiffers(t *testing.T) {
	r := committedRepo(t)
	if code, so, se := initIn(t, r, nil); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	p := skillRel(filepath.Join(userHome(r.Dir), ".agents"), "SKILL.md")
	if err := os.WriteFile(p, []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, so, se := initIn(t, r, nil)
	f := findingOf(t, so, "user-skill-differs")
	if code != 1 || f == nil || f["file"] != p || f["message"] != "differs from the shipped skill; use --force to overwrite" {
		t.Fatalf("exit %d finding %v\n%s\n%s", code, f, so, se)
	}
	if code, so, se := initIn(t, r, nil, "--force"); code != 0 || !strings.Contains(so, `"overwritten"`) {
		t.Fatalf("force: exit %d\n%s\n%s", code, so, se)
	}
}

func TestInitSymlinkedCommandFolder(t *testing.T) {
	r := committedRepo(t)
	outside := t.TempDir()
	link := filepath.Join(userHome(r.Dir), ".claude/skills/docket-init")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, r.Dir)
	code, so, se := initIn(t, r, nil)
	fnds, _ := decode(t, so)["findings"].([]any)
	if code != 1 || len(fnds) != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if f := fnds[0].(map[string]any); f["rule"] != "symlink-refused" || f["file"] != link {
		t.Errorf("finding: %v", f)
	}
	sameSnapshot(t, before, snapshot(t, r.Dir))
	if ents, _ := os.ReadDir(outside); len(ents) != 0 {
		t.Errorf("outside written: %v", ents)
	}
}
