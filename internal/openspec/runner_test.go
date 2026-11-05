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

// readLog returns one entry per call: the OPENSPEC_TELEMETRY field, then args.
func readLog(t *testing.T, log string) [][]string {
	t.Helper()
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	var out [][]string
	if len(b) == 0 {
		return out
	}
	for _, l := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		out = append(out, strings.Split(l, "\t"))
	}
	return out
}

func newRunner(t *testing.T, version string, extra ...string) (*openspec.Runner, string) {
	t.Helper()
	fake, log := testutil.FakeOpenSpec(t, version)
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+fake)
	r, err := openspec.New(t.TempDir(), append(env, extra...))
	if err != nil {
		t.Fatal(err)
	}
	return r, log
}

func TestOverrideWithNpx(t *testing.T) {
	fake, log := testutil.FakeOpenSpec(t, "1.7.0")
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+fake+" -y @fission-ai/openspec@1.7.0")
	r, err := openspec.New(t.TempDir(), env)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("init", "--tools", "claude"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"OPENSPEC_TELEMETRY=0", "-y", "@fission-ai/openspec@1.7.0", "--version"},
		{"OPENSPEC_TELEMETRY=0", "-y", "@fission-ai/openspec@1.7.0", "init", "--tools", "claude"},
	}
	if got := readLog(t, log); !reflect.DeepEqual(got, want) {
		t.Fatalf("log = %q, want %q", got, want)
	}
}

func TestDefaultFromPath(t *testing.T) {
	fake, log := testutil.FakeOpenSpec(t, "1.7.0")
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
	if len(got) != 2 || !reflect.DeepEqual(got[1], []string{"OPENSPEC_TELEMETRY=0", "--help"}) {
		t.Fatalf("log = %q", got)
	}
}

func TestTelemetryOffOnEveryInvocation(t *testing.T) {
	r, log := newRunner(t, "1.7.0", "OPENSPEC_TELEMETRY=1")
	if _, err := r.Run("--help"); err != nil {
		t.Fatal(err)
	}
	got := readLog(t, log)
	if len(got) != 2 {
		t.Fatalf("log = %q", got)
	}
	for _, c := range got {
		if c[0] != "OPENSPEC_TELEMETRY=0" {
			t.Errorf("call %q: telemetry not off", c)
		}
	}
}

func TestTooOld(t *testing.T) {
	r, log := newRunner(t, "1.6.2")
	_, err := r.Run("init", "--tools", "claude")
	var ue *openspec.UnsupportedError
	if !errors.As(err, &ue) || ue.Found != "1.6.2" {
		t.Fatalf("err = %#v", err)
	}
	if got := readLog(t, log); len(got) != 1 {
		t.Fatalf("only --version should run: %q", got)
	}
	for _, want := range []string{"1.6.2", "1.7.0", "DOCKET_OPENSPEC"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q lacks %q", err.Error(), want)
		}
	}
}

func TestVersionAcceptance(t *testing.T) {
	cases := map[string]bool{
		"1.7.0": true, "1.7.1": true, "1.10.0": true, "2.0.0": true, "10.0.0": true,
		"1.6.9": false, "1.7": false, "abc": false, "v1.7.0": false, "1.7.0-beta": false,
		"1.7.x": false, "": false,
		"1.7.0\n": true, "  1.7.0  \n": true, "1.7.0\nupdate available": false,
	}
	for v, ok := range cases {
		r, _ := newRunner(t, v)
		got, err := r.CheckVersion()
		if ok && (err != nil || got != strings.TrimSpace(v)) {
			t.Errorf("%q: got %q, %v", v, got, err)
		}
		var ue *openspec.UnsupportedError
		if !ok && (!errors.As(err, &ue) || ue.Found != v) {
			t.Errorf("%q: err = %#v, want UnsupportedError", v, err)
		}
	}
}

func TestFailingVersionIsUnsupported(t *testing.T) {
	r, _ := newRunner(t, "1.7.0", "FAKE_OPENSPEC_FAIL=--version")
	_, err := r.CheckVersion()
	var ue *openspec.UnsupportedError
	var ua *openspec.UnavailableError
	if !errors.As(err, &ue) || errors.As(err, &ua) || !strings.Contains(ue.Found, "fake failure") {
		t.Fatalf("err = %#v", err)
	}
}

func TestMissingProgram(t *testing.T) {
	env := append(testutil.Env(t), "DOCKET_OPENSPEC=/nonexistent/openspec")
	_, err := openspec.New(t.TempDir(), env)
	var ua *openspec.UnavailableError
	if !errors.As(err, &ua) || !strings.Contains(err.Error(), "DOCKET_OPENSPEC") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnstartableProgramIsUnavailable(t *testing.T) {
	dir := t.TempDir()
	prog := filepath.Join(dir, "openspec")
	if err := os.WriteFile(prog, []byte("#!/nonexistent/interpreter\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := openspec.New(dir, append(testutil.Env(t), "DOCKET_OPENSPEC="+prog))
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Run("--help")
	var ua *openspec.UnavailableError
	if !errors.As(err, &ua) {
		t.Fatalf("err = %#v", err)
	}
}

func TestFailureCarriesCommandCodeStderr(t *testing.T) {
	r, _ := newRunner(t, "1.7.0")
	_, err := r.Run("init", "--tools", "nosuch")
	var fe *openspec.FailedError
	if !errors.As(err, &fe) || fe.Code != 1 || !strings.Contains(fe.Stderr, "nosuch") {
		t.Fatalf("err = %#v", err)
	}
	if got := strings.Join(fe.Command[len(fe.Command)-3:], " "); got != "init --tools nosuch" {
		t.Errorf("command = %q", fe.Command)
	}
	for _, want := range []string{"init --tools nosuch", "exited with 1", "unknown tool 'nosuch'"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q lacks %q", err.Error(), want)
		}
	}
}

func TestFakeInitWritesFiles(t *testing.T) {
	r, _ := newRunner(t, "1.7.0")
	if _, err := r.Run("init", "--tools", "claude"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"openspec/config.yaml", "openspec/specs", "openspec/changes/archive", ".claude/skills/openspec-explore/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(r.Dir, p)); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	b, _ := os.ReadFile(filepath.Join(r.Dir, "openspec/config.yaml"))
	if !strings.HasPrefix(string(b), "schema: spec-driven\n\n# Project context") {
		t.Errorf("config = %q", b)
	}
}

func TestEnvironmentPassesThrough(t *testing.T) {
	fake, log := testutil.FakeOpenSpec(t, "1.7.0")
	env := append(testutil.Env(t), "DOCKET_OPENSPEC="+fake, "FOO=bar")
	r, err := openspec.New(t.TempDir(), env)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("--help"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(log + ".foo")
	if err != nil || string(b) != "bar\nbar\n" {
		t.Fatalf("FOO log = %q, %v", b, err)
	}
}

func TestFakeCreatesEmptyLog(t *testing.T) {
	_, log := testutil.FakeOpenSpec(t, "1.7.0")
	if got := readLog(t, log); len(got) != 0 {
		t.Fatalf("log = %q", got)
	}
}

func TestRepoOpenspecNotFoundThroughPath(t *testing.T) {
	fake, _ := testutil.FakeOpenSpec(t, "1.7.0")
	b, err := os.ReadFile(fake)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{":" + t.TempDir(), ".", "bin"} {
		repo := t.TempDir()
		for _, p := range []string{"openspec", "bin/openspec"} {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, p)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(repo, p), b, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		_, err := openspec.New(repo, append(testutil.Env(t), "DOCKET_OPENSPEC=", "PATH="+path))
		var ua *openspec.UnavailableError
		if !errors.As(err, &ua) {
			t.Errorf("PATH=%q: err = %v, want UnavailableError", path, err)
		}
	}
}

func TestRelativeOverrideResolvesAgainstDir(t *testing.T) {
	fake, _ := testutil.FakeOpenSpec(t, "1.7.0")
	b, _ := os.ReadFile(fake)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin/openspec"), b, 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := openspec.New(dir, append(testutil.Env(t), "DOCKET_OPENSPEC=./bin/openspec"))
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(r.Cmd[0]) {
		t.Fatalf("Cmd[0] = %q", r.Cmd[0])
	}
}
