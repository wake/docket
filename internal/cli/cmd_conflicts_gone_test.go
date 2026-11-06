package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConflictsOpenSpecGoneAfterVersion(t *testing.T) {
	e := newCfEnv(t)
	put(t, e.r.Dir, spec("add-x", "x"))
	put(t, e.worktree(t, "other"), spec("add-y", "y"))
	script := filepath.Join(t.TempDir(), "openspec")
	body := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 1.14.3; rm -f \"$0\"; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	code, m, raw, _ := e.run(t, script)
	if code != 4 || len(m.Findings) != 1 || m.Findings[0].Rule != "openspec-missing" || len(m.Unreadable) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}
