package openspec_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/testutil"
)

func readLog(t *testing.T, log string) [][]string {
	t.Helper()
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	var out [][]string
	for _, l := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		out = append(out, strings.Split(l, "\t"))
	}
	return out
}

func TestOverrideSplitsWords(t *testing.T) {
	fake, log := testutil.FakeOpenSpec(t, "1.14.3")
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+fake+" -y @fission-ai/openspec@1.14.3")
	r, err := openspec.New(t.TempDir(), env)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("new", "change", "x", "--schema", "docket"); err != nil {
		t.Fatal(err)
	}
	got := readLog(t, log)
	want := [][]string{
		{"-y", "@fission-ai/openspec@1.14.3", "--version"},
		{"-y", "@fission-ai/openspec@1.14.3", "new", "change", "x", "--schema", "docket"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("log = %q, want %q", got, want)
	}
}

func TestDefaultFromPath(t *testing.T) {
	fake, log := testutil.FakeOpenSpec(t, "1.14.3")
	bin := t.TempDir()
	b, err := os.ReadFile(fake)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "openspec"), b, 0o755); err != nil {
		t.Fatal(err)
	}
	env := append(testutil.Env(t), "DOCKET_OPENSPEC=", "PATH="+bin+":/usr/bin:/bin")
	r, err := openspec.New(t.TempDir(), env)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("--help"); err != nil {
		t.Fatal(err)
	}
	got := readLog(t, log)
	if len(got) != 2 || !reflect.DeepEqual(got[1], []string{"--help"}) {
		t.Fatalf("log = %q", got)
	}
}

func TestUnsupportedVersion(t *testing.T) {
	fake, log := testutil.FakeOpenSpec(t, "1.3.0")
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+fake)
	r, err := openspec.New(t.TempDir(), env)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Run("init", "--tools", "none")
	var ue *openspec.UnsupportedError
	if !errors.As(err, &ue) || ue.Found != "1.3.0" || ue.Range != "1.14.x" {
		t.Fatalf("err = %#v", err)
	}
	if got := readLog(t, log); len(got) != 1 || got[0][0] != "--version" {
		t.Fatalf("log = %q", got)
	}
	for _, want := range []string{"1.3.0", "1.14.x", "DOCKET_OPENSPEC"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q lacks %q", err.Error(), want)
		}
	}
}

func TestUnstartableProgramIsUnavailable(t *testing.T) {
	dir := t.TempDir()
	prog := filepath.Join(dir, "openspec")
	if err := os.WriteFile(prog, []byte("#!/nonexistent/interpreter\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+prog)
	r, err := openspec.New(dir, env)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Run("--help")
	var ue *openspec.UnavailableError
	var fe *openspec.FailedError
	if !errors.As(err, &ue) || errors.As(err, &fe) {
		t.Fatalf("err = %#v, want only UnavailableError", err)
	}
	if !strings.Contains(err.Error(), "DOCKET_OPENSPEC") {
		t.Errorf("message lacks DOCKET_OPENSPEC: %v", err)
	}
}

func TestFailingVersionIsUnavailable(t *testing.T) {
	fake, _ := testutil.FakeOpenSpec(t, "1.14.3")
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+fake, "FAKE_OPENSPEC_FAIL=--version")
	r, err := openspec.New(t.TempDir(), env)
	if err != nil {
		t.Fatal(err)
	}
	for name, call := range map[string]func() error{
		"CheckVersion": func() error { _, err := r.CheckVersion(); return err },
		"Run":          func() error { _, err := r.Run("--help"); return err },
	} {
		err := call()
		var ue *openspec.UnavailableError
		var fe *openspec.FailedError
		if !errors.As(err, &ue) || errors.As(err, &fe) {
			t.Fatalf("%s: err = %#v, want only UnavailableError", name, err)
		}
		for _, want := range []string{"fake failure", "3", "DOCKET_OPENSPEC"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: message %q lacks %q", name, err.Error(), want)
			}
		}
	}
}

func TestVersionMatching(t *testing.T) {
	cases := map[string]bool{
		"1.14.0": true, "1.14.3": true, "1.14.12": true,
		"1.15.0": false, "1.14": false, "1.14.bad": false, "v1.14.3": false,
		"1.14.3-beta": false, "": false,
	}
	for v, ok := range cases {
		fake, _ := testutil.FakeOpenSpec(t, v)
		env := append(testutil.Env(t), "DOCKET_OPENSPEC="+fake)
		r, err := openspec.New(t.TempDir(), env)
		if err != nil {
			t.Fatal(err)
		}
		got, err := r.CheckVersion()
		if ok && (err != nil || got != v) {
			t.Errorf("%q: got %q, %v", v, got, err)
		}
		var ue *openspec.UnsupportedError
		if !ok && !errors.As(err, &ue) {
			t.Errorf("%q: err = %v, want UnsupportedError", v, err)
		}
	}
}

func TestMissingProgram(t *testing.T) {
	env := append(testutil.Env(t), "DOCKET_OPENSPEC=/nonexistent/openspec")
	_, err := openspec.New(t.TempDir(), env)
	var ue *openspec.UnavailableError
	if !errors.As(err, &ue) || !strings.Contains(err.Error(), "DOCKET_OPENSPEC") {
		t.Fatalf("err = %v", err)
	}
}

func copyFake(t *testing.T, fake, dst string) {
	t.Helper()
	b, err := os.ReadFile(fake)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyPathEntryResolvesAgainstDir(t *testing.T) {
	fake, log := testutil.FakeOpenSpec(t, "1.14.3")
	dir := t.TempDir()
	copyFake(t, fake, filepath.Join(dir, "openspec"))
	env := append(testutil.Env(t), "DOCKET_OPENSPEC=", "PATH=:"+t.TempDir())
	r, err := openspec.New(dir, env)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(r.Cmd[0]) {
		t.Fatalf("Cmd[0] = %q, want absolute", r.Cmd[0])
	}
	if _, err := r.Run("--help"); err != nil {
		t.Fatal(err)
	}
	if got := readLog(t, log); len(got) != 2 {
		t.Fatalf("log = %q", got)
	}
}

func TestRelativeOverrideResolvesAgainstDir(t *testing.T) {
	fake, log := testutil.FakeOpenSpec(t, "1.14.3")
	dir := t.TempDir()
	copyFake(t, fake, filepath.Join(dir, "bin", "openspec"))
	env := append(testutil.Env(t), "DOCKET_OPENSPEC=./bin/openspec")
	r, err := openspec.New(dir, env)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(r.Cmd[0]) {
		t.Fatalf("Cmd[0] = %q, want absolute", r.Cmd[0])
	}
	if _, err := r.Run("--help"); err != nil {
		t.Fatal(err)
	}
	if got := readLog(t, log); len(got) != 2 {
		t.Fatalf("log = %q", got)
	}
}

func TestFailureCarriesStderr(t *testing.T) {
	fake, _ := testutil.FakeOpenSpec(t, "1.14.3")
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+fake, "FAKE_OPENSPEC_FAIL=new")
	dir := t.TempDir()
	r, err := openspec.New(dir, env)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Run("new", "change", "x", "--schema", "docket")
	var fe *openspec.FailedError
	if !errors.As(err, &fe) || fe.Code != 3 || !strings.Contains(fe.Stderr, "fake failure") {
		t.Fatalf("err = %#v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "openspec/changes/x")); err != nil {
		t.Fatalf("change folder should exist before failing: %v", err)
	}
}
