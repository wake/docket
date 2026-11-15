package cli

import (
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestConflictsOpenSpecGoneAfterVersion(t *testing.T) {
	e := newCfEnv(t)
	put(t, e.r.Dir, spec("add-x", "x"))
	put(t, e.worktree(t, "other"), spec("add-y", "y"))
	script := filepath.Join(t.TempDir(), "openspec")
	testutil.FakeProgram(t, script, testutil.Program{Print: "1.14.3\n", RemoveSelf: true}) // gone after --version
	code, m, raw, _ := e.run(t, script)
	if code != 4 || len(m.Findings) != 1 || m.Findings[0].Rule != "openspec-missing" || len(m.Unreadable) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}
