package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"testing/fstest"

	docket "github.com/wake/docket"
)

func shipped(t *testing.T) fs.FS {
	t.Helper()
	sub, err := fs.Sub(docket.Shipped, "shipped")
	if err != nil {
		t.Fatal(err)
	}
	return sub
}

func put(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func get(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func find(ds []Decision, path string) *Decision {
	for i := range ds {
		if ds[i].Path == path {
			return &ds[i]
		}
	}
	return nil
}

func decideT(t *testing.T, root string, force bool) ([]Decision, []Problem) {
	t.Helper()
	ds, ps, err := Decide(root, shipped(t), force)
	if err != nil {
		t.Fatal(err)
	}
	return ds, ps
}

func umask() os.FileMode {
	m := syscall.Umask(0)
	syscall.Umask(m)
	return os.FileMode(m)
}

func TestDecideEmptyRootIsAllMissingInOrder(t *testing.T) {
	ds, ps := decideT(t, t.TempDir(), false)
	if len(ps) != 0 {
		t.Fatal(ps)
	}
	var paths []string
	for _, d := range ds {
		if d.Status != StatusMissing || d.Action != ActionCreate {
			t.Errorf("%s: %s/%s", d.Path, d.Status, d.Action)
		}
		paths = append(paths, d.Path)
	}
	want := []string{
		"openspec/schemas/docket/schema.yaml",
		"openspec/schemas/docket/templates/design.md",
		"openspec/schemas/docket/templates/plan.md",
		"openspec/schemas/docket/templates/proposal.md",
		"openspec/schemas/docket/templates/spec.md",
		"openspec/config.yaml",
		".claude/skills/docket/SKILL.md",
		".agents/skills/docket/SKILL.md",
	}
	if len(paths) != len(want) {
		t.Fatalf("got %v", paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("got %v, want %v", paths, want)
		}
	}
	if find(ds, ".claude/skills/docket/SKILL.md").Kind != KindSkill || ds[0].Kind != KindSchema {
		t.Error("kinds")
	}
}

func TestDecideSameAfterWrite(t *testing.T) {
	root := t.TempDir()
	ds, _ := decideT(t, root, false)
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	ds, ps := decideT(t, root, false)
	if len(ps) != 0 {
		t.Fatal(ps)
	}
	for _, d := range ds {
		if d.Status != StatusSame || d.Action != ActionNone {
			t.Errorf("%s: %s/%s", d.Path, d.Status, d.Action)
		}
	}
}

func TestDecideDiffersKeepAndForce(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".claude/skills/docket/SKILL.md", "mine")
	ds, _ := decideT(t, root, false)
	if d := find(ds, ".claude/skills/docket/SKILL.md"); d.Status != StatusDiffers || d.Action != ActionKeep {
		t.Errorf("%+v", d)
	}
	if d := find(ds, ".agents/skills/docket/SKILL.md"); d.Status != StatusMissing {
		t.Errorf("skills decided separately: %+v", d)
	}
	ds, _ = decideT(t, root, true)
	d := find(ds, ".claude/skills/docket/SKILL.md")
	if d.Status != StatusDiffers || d.Action != ActionOverwrite {
		t.Errorf("%+v", d)
	}
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	if get(t, root, ".claude/skills/docket/SKILL.md") == "mine" {
		t.Error("not overwritten")
	}
}

func TestWriteKeepWritesNothing(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".claude/skills/docket/SKILL.md", "mine")
	ds, _ := decideT(t, root, false)
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	if get(t, root, ".claude/skills/docket/SKILL.md") != "mine" {
		t.Error("kept file changed")
	}
}

func TestDecideOutdated(t *testing.T) {
	old := earlier
	defer func() { earlier = old }()
	root := t.TempDir()
	put(t, root, ".claude/skills/docket/SKILL.md", "old release")
	sum := sha256.Sum256([]byte("old release"))
	earlier = map[string][]string{".claude/skills/docket/SKILL.md": {hex.EncodeToString(sum[:])}}
	ds, _ := decideT(t, root, false)
	if d := find(ds, ".claude/skills/docket/SKILL.md"); d.Status != StatusOutdated || d.Action != ActionUpdate {
		t.Errorf("%+v", d)
	}
	// the same hash recorded for another path does not count
	put(t, root, ".agents/skills/docket/SKILL.md", "old release")
	ds, _ = decideT(t, root, false)
	if d := find(ds, ".agents/skills/docket/SKILL.md"); d.Status != StatusDiffers {
		t.Errorf("%+v", d)
	}
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	if get(t, root, ".claude/skills/docket/SKILL.md") == "old release" {
		t.Error("outdated not updated")
	}
}

func TestDecideUserFilesAreNotTargets(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".claude/skills/docket/extra.md", "x")
	put(t, root, "openspec/schemas/docket/mine.md", "x")
	ds, _ := decideT(t, root, true)
	if find(ds, ".claude/skills/docket/extra.md") != nil || find(ds, "openspec/schemas/docket/mine.md") != nil {
		t.Error("user file is a target")
	}
}

func TestDecideSymlinkOneProblemPerLink(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "openspec/schemas"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, filepath.Join(root, "openspec/schemas/docket")); err != nil {
		t.Fatal(err)
	}
	ds, ps := decideT(t, root, false)
	if len(ps) != 1 || ps[0].Kind != ProblemSymlink || ps[0].Path != "openspec/schemas/docket" {
		t.Fatalf("%+v", ps)
	}
	if len(ds) != 3 {
		t.Errorf("config and skill targets should remain: %d", len(ds))
	}
	entries, _ := os.ReadDir(other)
	if len(entries) != 0 {
		t.Error("wrote through link")
	}
}

func TestDecideSymlinkAtFile(t *testing.T) {
	root := t.TempDir()
	put(t, root, "target.md", "x")
	if err := os.MkdirAll(filepath.Join(root, ".claude/skills/docket"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "target.md"), filepath.Join(root, ".claude/skills/docket/SKILL.md")); err != nil {
		t.Fatal(err)
	}
	ds, ps := decideT(t, root, true)
	if len(ps) != 1 || ps[0].Kind != ProblemSymlink || ps[0].Path != ".claude/skills/docket/SKILL.md" {
		t.Fatalf("%+v", ps)
	}
	if find(ds, ".claude/skills/docket/SKILL.md") != nil {
		t.Error("link target decided")
	}
}

func TestDecideUnreadable(t *testing.T) {
	root := t.TempDir()
	// a directory where a file should be cannot be read
	if err := os.MkdirAll(filepath.Join(root, ".agents/skills/docket/SKILL.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, ps := decideT(t, root, false)
	if len(ps) != 1 || ps[0].Kind != ProblemUnreadable || ps[0].Err == nil {
		t.Fatalf("%+v", ps)
	}
}

func TestDecideShippedReadError(t *testing.T) {
	bad := fstest.MapFS{"other": {Data: []byte("x")}}
	if _, _, err := Decide(t.TempDir(), bad, false); err == nil {
		t.Error("want an error for missing shipped files")
	}
}

func TestWriteModes(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".claude/skills/docket/SKILL.md", "mine")
	if err := os.Chmod(filepath.Join(root, ".claude/skills/docket/SKILL.md"), 0o600); err != nil {
		t.Fatal(err)
	}
	ds, _ := decideT(t, root, true)
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(filepath.Join(root, ".claude/skills/docket/SKILL.md"))
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("existing mode changed: %v", fi.Mode().Perm())
	}
	fi, _ = os.Stat(filepath.Join(root, ".agents/skills/docket/SKILL.md"))
	if fi.Mode().Perm() != 0o644&^umask() {
		t.Errorf("new file mode %v", fi.Mode().Perm())
	}
	di, _ := os.Stat(filepath.Join(root, "openspec/schemas/docket/templates"))
	if di.Mode().Perm() != 0o755&^umask() {
		t.Errorf("dir mode %v", di.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Join(root, ".claude/skills/docket"))
	if len(entries) != 1 {
		t.Error("temporary file left behind")
	}
}

func TestWriteFailureKeepsOldFile(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".claude/skills/docket/SKILL.md", "mine")
	ds, _ := decideT(t, root, true)
	old := rename
	defer func() { rename = old }()
	rename = func(string, string) error { return errors.New("boom") }
	if err := Write(root, ds); err == nil {
		t.Fatal("want error")
	}
	if get(t, root, ".claude/skills/docket/SKILL.md") != "mine" {
		t.Error("old file lost")
	}
	entries, _ := os.ReadDir(filepath.Join(root, ".claude/skills/docket"))
	if len(entries) != 1 {
		t.Error("temporary file left behind")
	}
}
