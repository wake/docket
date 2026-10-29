package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/testutil"
)

func TestAgentsMDLinkRefused(t *testing.T) {
	r := committedRepo(t)
	if err := os.Symlink("README.md", filepath.Join(r.Dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, r.Dir)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	fnds, _ := decode(t, so)["findings"].([]any)
	if len(fnds) != 1 || fnds[0].(map[string]any)["rule"] != "symlink-refused" || fnds[0].(map[string]any)["file"] != "AGENTS.md" {
		t.Errorf("findings: %s", so)
	}
	sameSnapshot(t, before, snapshot(t, r.Dir))
	if got := readFile(t, r.Dir, "README.md"); got != "x\n" {
		t.Errorf("README.md = %q", got)
	}
}

func TestAgentsMDDirectoryUnreadable(t *testing.T) {
	r := committedRepo(t)
	if err := os.Mkdir(filepath.Join(r.Dir, "AGENTS.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, r.Dir)
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--json")
	if code != 3 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	un, _ := decode(t, so)["unreadable"].([]any)
	if len(un) != 1 || un[0].(map[string]any)["source"] != "AGENTS.md" {
		t.Errorf("unreadable: %s", so)
	}
	sameSnapshot(t, before, snapshot(t, r.Dir))
}

func TestClaudeMDEditedBlockForced(t *testing.T) {
	r := committedRepo(t)
	r.Write("CLAUDE.md", "# claude\n<!-- docket:start -->\nold\n<!-- docket:end -->\nafter\n")
	code, so, se := runInit(t, r.Dir, testutil.OpenSpecCmd(t), "--force", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if s := fileStatus(t, so, "CLAUDE.md"); s != "overwritten" {
		t.Errorf("status %s", s)
	}
	if got, want := readFile(t, r.Dir, "CLAUDE.md"), "# claude\n"+docket.Block()+"\nafter\n"; got != want {
		t.Errorf("CLAUDE.md = %q", got)
	}
}
