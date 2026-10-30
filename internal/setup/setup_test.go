package setup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

var schema = fstest.MapFS{
	"schema.yaml":           {Data: []byte("name: docket\n")},
	"templates/proposal.md": {Data: []byte("# proposal\n")},
}

var skill = fstest.MapFS{"SKILL.md": {Data: []byte("# skill\n")}}

var assets = Assets{Schema: schema, Skill: skill, Block: "<!-- docket:start -->\nblock\n<!-- docket:end -->"}

func statuses(r Result) map[string]FileStatus {
	m := map[string]FileStatus{}
	for _, f := range r.Files {
		m[f.Path] = f.Status
	}
	return m
}

func inspectPaths(fs []File) map[string]File {
	m := map[string]File{}
	for _, f := range fs {
		m[f.Path] = f
	}
	return m
}

func TestInspect(t *testing.T) {
	t.Run("empty root", func(t *testing.T) {
		root := t.TempDir()
		files, problems, err := Inspect(root, assets)
		if err != nil || len(problems) != 0 {
			t.Fatalf("err %v problems %v", err, problems)
		}
		got := inspectPaths(files)
		for _, p := range []string{"AGENTS.md", "CLAUDE.md", "openspec/schemas/docket/schema.yaml", ".claude/skills/docket/SKILL.md"} {
			if _, ok := got[p]; !ok {
				t.Errorf("%s not listed", p)
			}
		}
		for _, f := range files {
			if f.Status != Written {
				t.Errorf("%s: status %q, want written", f.Path, f.Status)
			}
		}
		entries, _ := os.ReadDir(root)
		if len(entries) != 0 {
			t.Errorf("root not empty: %v", entries)
		}
	})
}

func TestInspectAfterInit(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, assets, false, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	files, problems, err := Inspect(root, assets)
	if err != nil || len(problems) != 0 {
		t.Fatalf("err %v problems %v", err, problems)
	}
	if len(files) == 0 {
		t.Fatal("no files")
	}
	for _, f := range files {
		if f.Status != Unchanged {
			t.Errorf("%s: status %q, want unchanged", f.Path, f.Status)
		}
		if f.Path == "CLAUDE.md" {
			t.Error("CLAUDE.md listed although it imports AGENTS.md")
		}
	}
}

func initedRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := Init(root, assets, false, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestInspectEditedSkill(t *testing.T) {
	root := initedRoot(t)
	p := filepath.Join(root, ".agents/skills/docket/SKILL.md")
	if err := os.WriteFile(p, []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, problems, err := Inspect(root, assets)
	if err != nil || len(problems) != 0 {
		t.Fatalf("err %v problems %v", err, problems)
	}
	f := inspectPaths(files)[".agents/skills/docket/SKILL.md"]
	if f.Status != Differs || f.Kind != KindSkill {
		t.Errorf("got %+v", f)
	}
	if b, _ := os.ReadFile(p); string(b) != "mine\n" {
		t.Errorf("file changed: %q", b)
	}
}

func TestInspectClaudeMDLinkedToAgents(t *testing.T) {
	root := initedRoot(t)
	p := filepath.Join(root, "CLAUDE.md")
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("AGENTS.md", p); err != nil {
		t.Fatal(err)
	}
	files, problems, err := Inspect(root, assets)
	if err != nil || len(problems) != 0 {
		t.Fatalf("err %v problems %v", err, problems)
	}
	if _, ok := inspectPaths(files)["CLAUDE.md"]; ok {
		t.Error("CLAUDE.md listed although it links to AGENTS.md")
	}
}

func TestInspectSymlinkedSkillDirectory(t *testing.T) {
	root := initedRoot(t)
	other := t.TempDir()
	link := filepath.Join(root, ".claude/skills/docket")
	if err := os.RemoveAll(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, link); err != nil {
		t.Fatal(err)
	}
	files, problems, err := Inspect(root, assets)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) == 0 {
		t.Fatal("no problems")
	}
	for _, p := range problems {
		if p.Kind != ProblemSymlink || !strings.HasPrefix(p.Path, ".claude/skills/docket/") {
			t.Errorf("unexpected problem %+v", p)
		}
	}
	got := inspectPaths(files)
	if _, ok := got["openspec/schemas/docket/schema.yaml"]; !ok {
		t.Error("schema not returned")
	}
	if _, ok := got[".agents/skills/docket/SKILL.md"]; !ok {
		t.Error("agents skill not returned")
	}
	if entries, _ := os.ReadDir(other); len(entries) != 0 {
		t.Errorf("wrote through the link: %v", entries)
	}
}

func TestInitCallsOpenSpecOnlyWhenMissing(t *testing.T) {
	root := t.TempDir()
	calls := 0
	initOS := func() error { calls++; return os.MkdirAll(filepath.Join(root, "openspec"), 0o755) }
	r, err := Init(root, assets, false, initOS)
	if err != nil || calls != 1 || !r.OpenSpecInitialized {
		t.Fatalf("first: %v calls=%d %+v", err, calls, r)
	}
	r, err = Init(root, assets, false, initOS)
	if err != nil || calls != 1 || r.OpenSpecInitialized {
		t.Fatalf("second: %v calls=%d %+v", err, calls, r)
	}
	for p, s := range statuses(r) {
		if s != Unchanged {
			t.Errorf("%s = %s", p, s)
		}
	}
}

func TestInitStatuses(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "openspec"), 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := Init(root, assets, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	const rel = "openspec/schemas/docket/templates/proposal.md"
	if got := statuses(r)[rel]; got != Written {
		t.Fatalf("first status = %s", got)
	}
	p := filepath.Join(root, rel)
	if err := os.WriteFile(p, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err = Init(root, assets, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := statuses(r)[rel]; got != Differs {
		t.Fatalf("status = %s", got)
	}
	if b, _ := os.ReadFile(p); string(b) != "edited\n" {
		t.Fatalf("file changed: %q", b)
	}
	r, err = Init(root, assets, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := statuses(r)[rel]; got != Overwritten {
		t.Fatalf("status = %s", got)
	}
	if b, _ := os.ReadFile(p); string(b) != "# proposal\n" {
		t.Fatalf("not overwritten: %q", b)
	}
}

func TestInitPropagatesOpenSpecError(t *testing.T) {
	boom := errors.New("boom")
	_, err := Init(t.TempDir(), assets, false, func() error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

func TestInitReadOnlyUnchangedFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	root := t.TempDir()
	initOS := func() error { return os.MkdirAll(filepath.Join(root, "openspec"), 0o755) }
	if _, err := Init(root, assets, false, initOS); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, schemaDir, "schema.yaml"), 0o444); err != nil {
		t.Fatal(err)
	}
	r, err := Init(root, assets, false, initOS)
	if err != nil {
		t.Fatal(err)
	}
	if s := statuses(r)[schemaDir+"/schema.yaml"]; s != Unchanged {
		t.Fatalf("status = %s, want %s", s, Unchanged)
	}
}

func openspecDir(t *testing.T, root string) func() error {
	t.Helper()
	return func() error { return os.MkdirAll(filepath.Join(root, "openspec"), 0o755) }
}

func TestInitInstallsSkill(t *testing.T) {
	root := t.TempDir()
	r, err := Init(root, assets, false, openspecDir(t, root))
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{".claude/skills/docket/SKILL.md", ".agents/skills/docket/SKILL.md"} {
		if got := statuses(r)[rel]; got != Written {
			t.Errorf("%s = %q", rel, got)
		}
		if b, _ := os.ReadFile(filepath.Join(root, rel)); string(b) != "# skill\n" {
			t.Errorf("%s content %q", rel, b)
		}
		for _, f := range r.Files {
			if f.Path == rel && f.Kind != KindSkill {
				t.Errorf("%s kind %q", rel, f.Kind)
			}
			if f.Path == schemaDir+"/schema.yaml" && f.Kind != KindSchema {
				t.Errorf("schema kind %q", f.Kind)
			}
		}
	}
	r, err = Init(root, assets, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	for p, s := range statuses(r) {
		if s != Unchanged {
			t.Errorf("%s = %s", p, s)
		}
	}
}

func TestInitLeavesUserSkillFiles(t *testing.T) {
	root := t.TempDir()
	notes := filepath.Join(root, ".claude/skills/docket/notes.md")
	if err := os.MkdirAll(filepath.Dir(notes), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(notes, []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Init(root, assets, true, openspecDir(t, root))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := statuses(r)[".claude/skills/docket/notes.md"]; ok {
		t.Error("notes.md listed")
	}
	if b, _ := os.ReadFile(notes); string(b) != "mine\n" {
		t.Errorf("notes.md = %q", b)
	}
}

func TestInitRefusesBeforeWriting(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, schemaDir), 0o755); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(root, schemaDir, "schema.yaml")
	if err := os.WriteFile(bad, []byte("x"), 0o000); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".agents/skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".agents/skills/docket")); err != nil {
		t.Fatal(err)
	}
	called := false
	r, err := Init(root, assets, true, func() error { called = true; return nil })
	var ref *Refused
	if !errors.As(err, &ref) {
		t.Fatalf("err = %v", err)
	}
	kinds := map[ProblemKind]string{}
	for _, p := range ref.Problems {
		kinds[p.Kind] = p.Path
	}
	if len(ref.Problems) != 2 || kinds[ProblemSymlink] == "" || kinds[ProblemUnreadable] != schemaDir+"/schema.yaml" {
		t.Errorf("problems = %+v", ref.Problems)
	}
	if r.Files == nil || len(r.Files) != 0 || called {
		t.Errorf("files=%v called=%v", r.Files, called)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); err == nil {
		t.Error(".claude created")
	}
	if ents, _ := os.ReadDir(outside); len(ents) != 0 {
		t.Error("outside written")
	}
}

func TestInitWriteFailureThenRerun(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	root := t.TempDir()
	dir := filepath.Join(root, ".agents/skills/docket")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	r, err := Init(root, assets, false, openspecDir(t, root))
	if err == nil || !strings.Contains(err.Error(), ".agents/skills/docket/SKILL.md") {
		t.Fatalf("err = %v", err)
	}
	st := statuses(r)
	if st[schemaDir+"/schema.yaml"] != Written || st[".claude/skills/docket/SKILL.md"] != Written || len(st) != 3 {
		t.Fatalf("files = %v", st)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	r, err = Init(root, assets, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	st = statuses(r)
	if st[".agents/skills/docket/SKILL.md"] != Written || st[schemaDir+"/schema.yaml"] != Unchanged || st[".claude/skills/docket/SKILL.md"] != Unchanged {
		t.Errorf("rerun = %v", st)
	}
}
