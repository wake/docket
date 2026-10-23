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
	if _, err := os.Stat(filepath.Join(r.Dir, ".claude")); err == nil {
		t.Error(".claude created")
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
