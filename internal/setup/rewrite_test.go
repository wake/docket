package setup

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

const oldBlock = "<!-- docket:start -->\nold\n<!-- docket:end -->\n"

func TestRewriteKeepsModeAndLeavesNoTemporaryFile(t *testing.T) {
	old := syscall.Umask(0o022)
	t.Cleanup(func() { syscall.Umask(old) })
	root := t.TempDir()
	p := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(p, []byte(oldBlock), 0o664); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o664); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, dirs(root), assets, true, openspecDir(t, root)); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != assets.Block+"\n" {
		t.Errorf("AGENTS.md = %q", b)
	}
	if i, err := os.Stat(p); err != nil || i.Mode().Perm() != 0o664 {
		t.Errorf("mode %v err %v", i.Mode(), err)
	}
	ents, _ := os.ReadDir(root)
	for _, e := range ents {
		switch e.Name() {
		case "AGENTS.md", "CLAUDE.md", "openspec", ".claude", ".agents":
		default:
			t.Errorf("stray file %s", e.Name())
		}
	}
}

func TestFailedRewriteKeepsTheFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file modes")
	}
	root := t.TempDir()
	p := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(p, []byte(oldBlock), 0o644); err != nil {
		t.Fatal(err)
	}
	// The directories exist, so the only write that can fail is the one in
	// the read-only root.
	for _, d := range []string{"openspec", ".claude/skills/docket", ".agents/skills/docket"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(root, 0o755) })
	if _, err := Init(root, dirs(root), assets, true, nil); err == nil {
		t.Fatal("Init succeeded in a read-only directory")
	}
	if b, _ := os.ReadFile(p); string(b) != oldBlock {
		t.Errorf("AGENTS.md = %q", b)
	}
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatal(err)
	}
	ents, _ := os.ReadDir(root)
	for _, e := range ents {
		switch e.Name() {
		case "AGENTS.md", "openspec", ".claude", ".agents":
		default:
			t.Errorf("stray file %s", e.Name())
		}
	}
}

func TestFailedRenameRemovesTheTemporaryFile(t *testing.T) {
	old := rename
	rename = func(string, string) error { return errors.New("rename refused") }
	t.Cleanup(func() { rename = old })
	root := t.TempDir()
	p := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(p, []byte(oldBlock), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, dirs(root), assets, true, openspecDir(t, root)); err == nil {
		t.Fatal("Init succeeded although rename failed")
	}
	if b, _ := os.ReadFile(p); string(b) != oldBlock {
		t.Errorf("AGENTS.md = %q", b)
	}
	if m, _ := filepath.Glob(filepath.Join(root, "*.docket-tmp-*")); len(m) != 0 {
		t.Errorf("temporary files left: %v", m)
	}
}
