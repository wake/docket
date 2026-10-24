package lint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket"
)

func TestLinkedChangeFolderIsNotAChange(t *testing.T) {
	root := t.TempDir()
	changes := filepath.Join(root, "openspec", "changes")
	real := filepath.Join(root, "elsewhere")
	for _, d := range []string{changes, real} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(real, "proposal.md"), []byte(validProposal), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(changes, "add-x")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	in := Input{Repo: os.DirFS(root), Templates: docket.Schema()}
	r := Run(in)
	if len(r.Changes) != 0 || len(r.Findings) != 0 {
		t.Fatalf("unnamed run = %+v", r)
	}
	in.Slugs = []string{"add-x"}
	r = Run(in)
	if len(r.Changes) != 0 {
		t.Fatalf("named run Changes = %v", r.Changes)
	}
	wantOne(t, r, "change-missing", 0)
}
