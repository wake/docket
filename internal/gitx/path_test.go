package gitx

import (
	"os"
	"path/filepath"
	"testing"
)

func writeStub(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho stub-marker\n"), 0o755); err != nil {
		t.Fatal(err)
	}
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
