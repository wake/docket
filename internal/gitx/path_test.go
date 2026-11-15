package gitx

import (
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func writeStub(t *testing.T, path string) {
	t.Helper()
	testutil.FakeProgram(t, path, testutil.Program{Print: "stub-marker\n"})
}

func TestRelativePathEntryResolvesAgainstDir(t *testing.T) {
	dir := t.TempDir()
	writeStub(t, filepath.Join(dir, "tools", "git"))
	g := Git{Dir: dir, Env: []string{"PATH=tools"}}
	out, err := g.Run("anything")
	if err != nil {
		t.Fatal(err)
	}
	if out != "stub-marker" {
		t.Fatalf("out = %q, want stub-marker", out)
	}
}

func TestEmptyPathEntryResolvesAgainstDir(t *testing.T) {
	dir := t.TempDir()
	writeStub(t, filepath.Join(dir, "git"))
	g := Git{Dir: dir, Env: []string{"PATH=:/nonexistent-dir"}}
	out, err := g.Run("anything")
	if err != nil {
		t.Fatal(err)
	}
	if out != "stub-marker" {
		t.Fatalf("out = %q, want stub-marker", out)
	}
}
