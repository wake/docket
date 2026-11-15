package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wake/docket/internal/testutil"
)

func actRun(t *testing.T, env []string, dir, stdin string, args ...string) (int, string) {
	t.Helper()
	var out, errb bytes.Buffer
	o := Options{Args: args, Dir: dir, Env: env, Stdout: &out, Stderr: &errb}
	if stdin != "" {
		o.Stdin = strings.NewReader(stdin)
	}
	code := Run(o)
	return code, out.String() + errb.String()
}

func actState(t *testing.T) ([]string, string) {
	t.Helper()
	s := t.TempDir()
	return []string{"XDG_STATE_HOME=" + s, "HOME=" + s}, filepath.Join(s, "docket", "sessions")
}

func count(dir string) int {
	es, _ := os.ReadDir(dir)
	return len(es)
}

func TestActivityInputNeverClosed(t *testing.T) {
	env, dir := actState(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	w.WriteString("{")
	done := make(chan int, 1)
	go func() {
		var out, errb bytes.Buffer
		done <- Run(Options{Args: []string{"activity", "working"}, Dir: t.TempDir(), Env: env, Stdin: r, Stdout: &out, Stderr: &errb})
	}()
	select {
	case code := <-done:
		if code != 0 || count(dir) != 1 {
			t.Fatalf("%d %d", code, count(dir))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("dkt activity waited for its input")
	}
}

func TestActivityWrites(t *testing.T) {
	env, dir := actState(t)
	code, out := actRun(t, env, t.TempDir(), `{"session_id":"a","cwd":"/w"}`, "activity", "working")
	if code != 0 || out != "" || count(dir) != 1 {
		t.Fatalf("%d %q %d", code, out, count(dir))
	}
}

func TestActivitySilentZero(t *testing.T) {
	for _, args := range [][]string{{"activity", "busy"}, {"activity"}, {"activity", "working", "extra"}, {"--json", "activity", "working"}} {
		env, dir := actState(t)
		code, out := actRun(t, env, t.TempDir(), "", args...)
		if code != 0 || out != "" {
			t.Errorf("%v: %d %q", args, code, out)
		}
		// --json before activity is a global flag, so that call still records, silently
		if args[0] != "--json" && count(dir) != 0 {
			t.Errorf("%v wrote a record", args)
		}
	}
	env, dir := actState(t)
	if code, out := actRun(t, env, t.TempDir(), "", "activity", "working", "--json"); code != 0 || out != "" || count(dir) != 0 {
		t.Errorf("--json after: %d %q", code, out)
	}
	if code, out := actRun(t, env, t.TempDir(), "not json", "activity", "working"); code != 0 || out != "" {
		t.Errorf("bad input: %d %q", code, out)
	}
	if code, out := actRun(t, nil, t.TempDir(), "", "activity", "working"); code != 0 || out != "" {
		t.Errorf("no home: %d %q", code, out)
	}
}

func TestActivityUnwritableFolder(t *testing.T) {
	s := t.TempDir()
	os.WriteFile(filepath.Join(s, "docket"), []byte("f"), 0o600)
	if code, out := actRun(t, []string{"XDG_STATE_HOME=" + s}, t.TempDir(), "", "activity", "working"); code != 0 || out != "" {
		t.Errorf("%d %q", code, out)
	}
}

func TestActivityNoGit(t *testing.T) {
	_, dir := actState(t)
	env := []string{"PATH=", "XDG_STATE_HOME=" + filepath.Dir(filepath.Dir(dir))}
	code, out := actRun(t, env, t.TempDir(), `{"session_id":"a"}`, "activity", "working")
	if code != 0 || out != "" || count(dir) != 1 {
		t.Fatalf("%d %q %d", code, out, count(dir))
	}
}

func TestActivityDoesNotRegister(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("base", snFiles())
	state := t.TempDir()
	env := append(append([]string(nil), r.Env...), "XDG_STATE_HOME="+state)
	code, out := actRun(t, env, r.Dir, "", "activity", "working")
	if code != 0 || out != "" {
		t.Fatalf("%d %q", code, out)
	}
	if _, err := os.Stat(filepath.Join(state, "docket", "projects.json")); err == nil {
		t.Error("index written")
	}
}

func TestActivityHelpAndUsage(t *testing.T) {
	code, out, _ := run(t, "--help")
	if code != 0 || !strings.Contains(out, "activity <working|waiting|ended>") {
		t.Errorf("usage lacks activity: %d %q", code, out)
	}
	code, out = actRun(t, nil, t.TempDir(), "", "activity", "--help")
	if code != 0 || !strings.Contains(out, "usage: dkt activity") {
		t.Errorf("%d %q", code, out)
	}
	code, out = actRun(t, nil, t.TempDir(), "", "activity", "-h")
	if code != 0 || !strings.Contains(out, "usage: dkt activity") {
		t.Errorf("-h: %d %q", code, out)
	}
}

func TestIsActivity(t *testing.T) {
	for args, want := range map[string]bool{"activity working": true, "--json activity": true, "snapshot": false, "": false} {
		if got := IsActivity(strings.Fields(args)); got != want {
			t.Errorf("%q: %v", args, got)
		}
	}
}
