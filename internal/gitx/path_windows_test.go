package gitx

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestPathSpelledAsWindowsDoes(t *testing.T) {
	dir, empty := t.TempDir(), t.TempDir()
	writeStub(t, filepath.Join(dir, "git"))
	if out, err := (Git{Dir: t.TempDir(), Env: []string{"Path=" + dir}}).Run("anything"); err != nil || out != "stub-marker" {
		t.Errorf("Path: %q %v", out, err)
	}
	if out, err := (Git{Dir: t.TempDir(), Env: []string{"PATH=" + empty, "Path=" + dir}}).Run("anything"); err != nil || out != "stub-marker" {
		t.Errorf("last spelling holds git: %q %v", out, err)
	}
	if _, err := (Git{Dir: t.TempDir(), Env: []string{"Path=" + dir, "PATH=" + empty}}).Run("anything"); !errors.Is(err, ErrGitMissing) {
		t.Errorf("last spelling has no git: %v", err)
	}
}
