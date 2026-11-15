package testutil

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// run runs prog with args and returns its stdout and exit code.
func run(t *testing.T, prog string, args ...string) (string, int) {
	t.Helper()
	out, err := exec.Command(prog, args...).Output()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return string(out), ee.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(out), 0
}

func TestFakeProgram(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	printer := filepath.Join(dir, "printer"+Exe)
	FakeProgram(t, printer, Program{Print: "hi\n", Code: 4})
	if out, code := run(t, printer); out != "hi\n" || code != 4 {
		t.Errorf("print: %q %d", out, code)
	}
	wrapper := filepath.Join(dir, "wrapper"+Exe)
	FakeProgram(t, wrapper, Program{Log: log, ExitIfArg: "stop", Code: 3, Real: printer})
	if out, code := run(t, wrapper, "a", "b"); out != "hi\n" || code != 4 {
		t.Errorf("real: %q %d", out, code)
	}
	if out, code := run(t, wrapper, "x", "stop"); out != "" || code != 3 {
		t.Errorf("exit if arg: %q %d", out, code)
	}
	if b, _ := os.ReadFile(log); string(b) != "a b\nx stop\n" {
		t.Errorf("log %q", b)
	}
	// a copy of the bytes behaves the same
	b, err := os.ReadFile(printer)
	if err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(dir, "gone"+Exe)
	FakeProgram(t, gone, Program{Print: "bye\n", RemoveSelf: true})
	copied := filepath.Join(t.TempDir(), "copied"+Exe)
	if err := os.WriteFile(copied, b, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, _ := run(t, copied); out != "hi\n" {
		t.Errorf("copy: %q", out)
	}
	if out, code := run(t, gone); out != "bye\n" || code != 0 {
		t.Errorf("remove self: %q %d", out, code)
	}
	if _, err := os.Stat(gone); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("still there: %v", err)
	}
}

func TestFakeOpenSpecVersionAndLog(t *testing.T) {
	cmd, log := FakeOpenSpec(t, "1.14.0")
	if out, code := run(t, cmd, "-y", "@fission-ai/openspec@1.14.0", "--version"); out != "1.14.0\n" || code != 0 {
		t.Errorf("version: %q %d", out, code)
	}
	if b, _ := os.ReadFile(log); !strings.HasSuffix(string(b), "\t-y\t@fission-ai/openspec@1.14.0\t--version\n") {
		t.Errorf("log %q", b)
	}
}
