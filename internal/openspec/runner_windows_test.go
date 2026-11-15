package openspec_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/testutil"
)

// twoOpenSpecs makes a folder holding openspec.exe and openspec.cmd, copies of the fake.
func twoOpenSpecs(t *testing.T) string {
	t.Helper()
	fake, _ := testutil.FakeOpenSpec(t, "1.14.0")
	b, err := os.ReadFile(fake)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, n := range []string{"openspec.exe", "openspec.cmd"} {
		if err := os.WriteFile(filepath.Join(dir, n), b, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func resolved(t *testing.T, env ...string) (string, error) {
	t.Helper()
	r, err := openspec.New(t.TempDir(), append(testutil.Env(t), append([]string{"DOCKET_OPENSPEC="}, env...)...))
	if err != nil {
		return "", err
	}
	return r.Cmd[0], nil
}

func TestExtensionsInOrder(t *testing.T) {
	dir := twoOpenSpecs(t)
	for pathext, want := range map[string]string{".CMD;.EXE": "openspec.cmd", ".EXE;.CMD": "openspec.exe"} {
		if got, err := resolved(t, "PATH="+dir, "PATHEXT="+pathext); err != nil || got != filepath.Join(dir, want) {
			t.Errorf("PATHEXT=%s: %q %v, want %s", pathext, got, err, want)
		}
	}
}

func TestOpenSpecPathSpellings(t *testing.T) {
	dir, empty := twoOpenSpecs(t), t.TempDir()
	if got, err := resolved(t, "PATH="+empty, "Path="+dir); err != nil || filepath.Dir(got) != dir {
		t.Errorf("last spelling holds OpenSpec: %q %v", got, err)
	}
	var ua *openspec.UnavailableError
	if _, err := resolved(t, "Path="+dir, "PATH="+empty); !errors.As(err, &ua) {
		t.Errorf("last spelling has no OpenSpec: %v", err)
	}
}

func TestBatchFileGetsNoUnsafeArgument(t *testing.T) {
	fake, log := testutil.FakeOpenSpec(t, "1.14.0")
	b, err := os.ReadFile(fake)
	if err != nil {
		t.Fatal(err)
	}
	cmd := filepath.Join(t.TempDir(), "openspec.cmd") // never started: the check comes first
	if err := os.WriteFile(cmd, b, 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := openspec.New(t.TempDir(), append(testutil.Env(t), "DOCKET_OPENSPEC="+cmd))
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Run("show", "add-x&calc", "--type", "change")
	var ue *openspec.UnsafeArgError
	if !errors.As(err, &ue) || ue.Arg != "add-x&calc" {
		t.Errorf("err = %v, want UnsafeArgError for add-x&calc", err)
	}
	if calls, _ := os.ReadFile(log); len(calls) != 0 {
		t.Errorf("started: %q", calls)
	}
}

func TestOverrideWithExtensionAndWithout(t *testing.T) {
	dir := twoOpenSpecs(t)
	exe := filepath.Join(dir, "openspec.exe")
	r, err := openspec.New(t.TempDir(), append(testutil.Env(t), "DOCKET_OPENSPEC="+exe))
	if err != nil || r.Cmd[0] != exe {
		t.Fatalf("extension given: %v %v", r, err)
	}
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(exe)
	if err := os.WriteFile(filepath.Join(repo, "bin", "openspec.exe"), b, 0o755); err != nil {
		t.Fatal(err)
	}
	r, err = openspec.New(repo, append(testutil.Env(t), "DOCKET_OPENSPEC=./bin/openspec", "PATHEXT=.exe"))
	if err != nil || r.Cmd[0] != filepath.Join(repo, "bin", "openspec.exe") {
		t.Fatalf("no extension: %v %v", r, err)
	}
}
