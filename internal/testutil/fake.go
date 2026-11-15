package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// exe is the suffix of an executable's name on this system.
var exe = func() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}()

// Program says what a fake made by FakeProgram does, in this order: append
// its arguments, joined by spaces, as a line to Log; exit with Code when an
// argument is ExitIfArg; run Real with the arguments and the same standard
// streams, and exit with its code; else print Print, remove its own file when
// RemoveSelf, and exit with Code.
type Program struct {
	Log        string
	ExitIfArg  string
	Real       string
	Print      string
	RemoveSelf bool
	Code       int
}

// FakeProgram writes an executable at path that does what p says, on every
// system. A copy of its bytes elsewhere behaves the same.
func FakeProgram(t testing.TB, path string, p Program) {
	t.Helper()
	writeFake(t, path, map[string]any{"Kind": "program", "Log": p.Log, "ExitIfArg": p.ExitIfArg,
		"Real": p.Real, "Print": p.Print, "RemoveSelf": p.RemoveSelf, "Code": p.Code})
}

// The marker and the field names must match internal/testutil/fakebin.
const fakeMarker = "\x00docket-fake-payload\x00"

var fakebin struct {
	once sync.Once
	path string
	err  error
}

// writeFake writes fakebin to path with payload appended.
func writeFake(t testing.TB, path string, payload map[string]any) {
	t.Helper()
	fakebin.once.Do(buildFakebin)
	if fakebin.err != nil {
		t.Fatalf("build the fake program: %v", fakebin.err)
	}
	b, err := os.ReadFile(fakebin.path)
	if err != nil {
		t.Fatal(err)
	}
	j, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	b = append(append(b, fakeMarker...), j...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o755); err != nil {
		t.Fatalf("write fake %s: %v", path, err)
	}
}

// buildFakebin builds fakebin once per test process into the temporary
// folder, named for its source, its system and the Go version, so later
// processes reuse it.
func buildFakebin() {
	_, here, _, _ := runtime.Caller(0)
	src := filepath.Join(filepath.Dir(here), "fakebin")
	code, err := os.ReadFile(filepath.Join(src, "main.go"))
	if err != nil {
		fakebin.err = err
		return
	}
	sum := sha256.Sum256(fmt.Appendf(code, "\n%s/%s/%s", runtime.GOOS, runtime.GOARCH, runtime.Version()))
	fakebin.path = filepath.Join(os.TempDir(), "docket-fakebin-"+hex.EncodeToString(sum[:8])+exe)
	if _, err := os.Stat(fakebin.path); err == nil {
		return
	}
	tmp, err := os.CreateTemp(os.TempDir(), "docket-fakebin-*"+exe)
	if err != nil {
		fakebin.err = err
		return
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	goTool := filepath.Join(runtime.GOROOT(), "bin", "go"+exe)
	if _, err := os.Stat(goTool); err != nil {
		goTool = "go"
	}
	c := exec.Command(goTool, "build", "-o", tmp.Name(), ".")
	c.Dir = src
	if out, err := c.CombinedOutput(); err != nil {
		fakebin.err = fmt.Errorf("%v: %s", err, out)
		return
	}
	// a concurrent process may have renamed its own build here first; either is the same program
	if err := os.Rename(tmp.Name(), fakebin.path); err != nil {
		if _, serr := os.Stat(fakebin.path); serr != nil {
			fakebin.err = err
		}
	}
}
