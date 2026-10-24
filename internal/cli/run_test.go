package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

// probeCmd returns a command set with one command, probe, that records its
// positionals and returns the given outcome and error.
func probeCmd(out Outcome, err error, got *[]string) []func() Command {
	return []func() Command{func() Command {
		return Command{
			Name:   "probe",
			Usage:  "probe [args]",
			Define: func(fs *flag.FlagSet) {},
			Run: func(c *Context, args []string) (Outcome, error) {
				if got != nil {
					*got = args
				}
				return out, err
			},
		}
	}}
}

func run(t *testing.T, dir string, cmds []func() Command, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	code := RunWith(Options{Args: args, Dir: dir, Env: testutil.Env(t), Stdout: &so, Stderr: &se}, cmds)
	return code, so.String(), se.String()
}

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("stdout is not one JSON object: %v\n%s", err, s)
	}
	return m
}

func repoOf(t *testing.T, so string) map[string]any {
	t.Helper()
	repo, ok := decode(t, so)["repo"].(map[string]any)
	if !ok {
		t.Fatalf("repo is not an object: %s", so)
	}
	return repo
}

func TestEnvelopeFieldsPresent(t *testing.T) {
	r := testutil.NewRepo(t)
	code, so, _ := run(t, r.Dir, probeCmd(Outcome{}, nil, nil), "probe", "--json")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	m := decode(t, so)
	if len(m) != 8 {
		t.Fatalf("want 8 keys, got %v", m)
	}
	for _, k := range []string{"tool", "version", "command", "repo", "ok", "findings", "unreadable", "result"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %s", k)
		}
	}
	if m["tool"] != "dkt" || m["command"] != "probe" || m["ok"] != true {
		t.Errorf("unexpected envelope: %v", m)
	}
	for _, k := range []string{"findings", "unreadable"} {
		a, ok := m[k].([]any)
		if !ok || len(a) != 0 {
			t.Errorf("%s should be an empty array, got %#v", k, m[k])
		}
	}
}

func TestDetachedHead(t *testing.T) {
	r := testutil.NewRepo(t)
	sha := r.Commit("one", map[string]string{"a.txt": "a"})
	r.Git("checkout", "--detach", sha)
	_, so, _ := run(t, r.Dir, probeCmd(Outcome{}, nil, nil), "probe", "--json")
	repo := repoOf(t, so)
	if repo["branch"] != nil {
		t.Errorf("branch should be null, got %v", repo["branch"])
	}
	if h, _ := repo["head"].(string); !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(h) || h != sha {
		t.Errorf("head = %v, want %s", repo["head"], sha)
	}
}

func TestNoOriginRemote(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a.txt": "a"})
	code, so, _ := run(t, r.Dir, probeCmd(Outcome{}, nil, nil), "probe", "--json")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if repoOf(t, so)["origin"] != nil {
		t.Errorf("origin should be null")
	}
}

func TestNotAGitRepository(t *testing.T) {
	code, so, _ := run(t, t.TempDir(), probeCmd(Outcome{}, nil, nil), "probe", "--json")
	if code != 4 {
		t.Fatalf("exit %d, want 4", code)
	}
	m := decode(t, so)
	if m["repo"] != nil {
		t.Errorf("repo should be null")
	}
	f, _ := m["findings"].([]any)
	if len(f) != 1 || f[0].(map[string]any)["rule"] != "environment" {
		t.Errorf("findings = %v", m["findings"])
	}
	if m["ok"] != false {
		t.Errorf("ok should be false")
	}
}

func TestUnbornHeadIdentity(t *testing.T) {
	r := testutil.NewRepo(t)
	_, so, _ := run(t, r.Dir, probeCmd(Outcome{}, nil, nil), "probe", "--json")
	repo := repoOf(t, so)
	if repo["head"] != nil || repo["branch"] != "main" {
		t.Errorf("repo = %v", repo)
	}
}

func TestFlagsAfterPositionals(t *testing.T) {
	r := testutil.NewRepo(t)
	var got []string
	code, so, _ := run(t, r.Dir, probeCmd(Outcome{}, nil, &got), "probe", "one", "--json", "two")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Join(got, " ") != "one two" {
		t.Errorf("positionals = %v", got)
	}
	decode(t, so)
}

func TestUnknownCommand(t *testing.T) {
	code, so, se := run(t, t.TempDir(), probeCmd(Outcome{}, nil, nil), "nope")
	if code != 2 {
		t.Fatalf("exit %d", code)
	}
	if so != "" || !strings.Contains(se, "usage") {
		t.Errorf("stdout %q stderr %q", so, se)
	}
	if code, _, _ := run(t, t.TempDir(), nil); code != 2 {
		t.Errorf("no args: exit %d", code)
	}
}

func TestBadFlagIsUsageError(t *testing.T) {
	r := testutil.NewRepo(t)
	code, so, _ := run(t, r.Dir, probeCmd(Outcome{}, nil, nil), "probe", "--bogus", "--json")
	if code != 2 {
		t.Fatalf("exit %d", code)
	}
	if decode(t, so)["repo"] != nil {
		t.Errorf("repo should be null before identification")
	}
	assertUsageEnvelope(t, so, "probe")
}

func assertUsageEnvelope(t *testing.T, so, command string) {
	t.Helper()
	m := decode(t, so)
	if m["command"] != command || m["ok"] != false {
		t.Errorf("command/ok = %v/%v: %s", m["command"], m["ok"], so)
	}
	f, _ := m["findings"].([]any)
	if len(f) != 1 || f[0].(map[string]any)["rule"] != "usage" {
		t.Errorf("findings = %v", m["findings"])
	}
}

func TestJSONBeforeCommand(t *testing.T) {
	r := testutil.NewRepo(t)
	code, so, _ := run(t, r.Dir, probeCmd(Outcome{}, nil, nil), "--json", "probe")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if decode(t, so)["command"] != "probe" {
		t.Errorf("stdout %s", so)
	}
}

func TestUnknownCommandJSON(t *testing.T) {
	code, so, se := run(t, t.TempDir(), probeCmd(Outcome{}, nil, nil), "nosuch", "--json")
	if code != 2 || !strings.Contains(se, "usage") {
		t.Fatalf("exit %d stderr %q", code, se)
	}
	assertUsageEnvelope(t, so, "nosuch")
}

func TestNoCommandJSON(t *testing.T) {
	code, so, se := run(t, t.TempDir(), probeCmd(Outcome{}, nil, nil), "--json")
	if code != 2 || !strings.Contains(se, "usage") {
		t.Fatalf("exit %d stderr %q", code, se)
	}
	assertUsageEnvelope(t, so, "")
}

func TestJSONEqualsTrue(t *testing.T) {
	r := testutil.NewRepo(t)
	code, so, _ := run(t, r.Dir, probeCmd(Outcome{}, nil, nil), "probe", "--json=true")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	decode(t, so)
	code, so, _ = run(t, r.Dir, probeCmd(Outcome{}, nil, nil), "probe", "--bogus", "--json=true")
	if code != 2 {
		t.Fatalf("exit %d", code)
	}
	assertUsageEnvelope(t, so, "probe")
}

func TestTextOutput(t *testing.T) {
	r := testutil.NewRepo(t)
	out := Outcome{Findings: []Finding{{Rule: "r1", Message: "bad", File: "f.md", Line: 3}}}
	code, so, _ := run(t, r.Dir, probeCmd(out, nil, nil), "probe")
	if code != 1 || !strings.Contains(so, "r1: bad") || !strings.Contains(so, "f.md:3") {
		t.Errorf("exit %d output %q", code, so)
	}
	_, so, _ = run(t, r.Dir, probeCmd(Outcome{Human: []string{"hello"}}, nil, nil), "probe")
	if so != "hello\ndkt probe: ok\n" {
		t.Errorf("output %q", so)
	}
}

func TestInternalError(t *testing.T) {
	r := testutil.NewRepo(t)
	code, so, _ := run(t, r.Dir, probeCmd(Outcome{}, errors.New("boom"), nil), "probe", "--json")
	f := decode(t, so)["findings"].([]any)
	if code != 1 || len(f) != 1 || f[0].(map[string]any)["rule"] != "internal" {
		t.Errorf("exit %d findings %v", code, f)
	}
}

func TestLaterProbeFailureIsEnvironment(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"a.txt": "a"})
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nfor a in \"$@\"; do\n  if [ \"$a\" = symbolic-ref ]; then echo 'fatal: probe broken' >&2; exit 128; fi\ndone\nexec '" + realGit + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	var so, se bytes.Buffer
	env := append(testutil.Env(t), "PATH="+bin+":"+filepath.Dir(realGit)+":/usr/bin:/bin")
	code := RunWith(Options{Args: []string{"probe", "--json"}, Dir: r.Dir, Env: env, Stdout: &so, Stderr: &se}, probeCmd(Outcome{}, nil, nil))
	if code != 4 {
		t.Fatalf("exit %d, want 4; stderr %s", code, se.String())
	}
	m := decode(t, so.String())
	f, _ := m["findings"].([]any)
	if m["repo"] != nil || len(f) != 1 || f[0].(map[string]any)["rule"] != "environment" {
		t.Errorf("envelope = %s", so.String())
	}
}

func TestDoubleDashEndsFlags(t *testing.T) {
	r := testutil.NewRepo(t)
	var got []string
	code, so, _ := run(t, r.Dir, probeCmd(Outcome{}, nil, &got), "probe", "--", "--json")
	if code != 0 || strings.Join(got, " ") != "--json" {
		t.Fatalf("exit %d positionals %q", code, got)
	}
	if !strings.Contains(so, "dkt probe: ok") || strings.HasPrefix(so, "{") {
		t.Errorf("want text output, got %q", so)
	}
	code, _, _ = run(t, r.Dir, probeCmd(Outcome{}, nil, &got), "probe", "--", "-x")
	if code != 0 || strings.Join(got, " ") != "-x" {
		t.Fatalf("exit %d positionals %q", code, got)
	}
	code, _, _ = run(t, r.Dir, probeCmd(Outcome{}, nil, &got), "probe", "a", "--", "-x", "--", "b")
	if code != 0 || strings.Join(got, " ") != "a -x -- b" {
		t.Fatalf("exit %d positionals %q", code, got)
	}
	code, so, _ = run(t, r.Dir, probeCmd(Outcome{}, nil, &got), "probe", "--", "a", "--json")
	if code != 0 || strings.Join(got, " ") != "a --json" || strings.HasPrefix(so, "{") {
		t.Fatalf("exit %d positionals %q", code, got)
	}
}

func TestDoubleDashAsFlagValue(t *testing.T) {
	r := testutil.NewRepo(t)
	var got []string
	code, so, _ := run(t, r.Dir, probeCmd(Outcome{}, nil, &got), "probe", "--default-branch", "--", "arg", "--json")
	if code != 0 || strings.Join(got, " ") != "arg" || !strings.HasPrefix(so, "{") {
		t.Fatalf("exit %d positionals %q stdout %q", code, got, so)
	}
	code, so, _ = run(t, r.Dir, probeCmd(Outcome{}, nil, &got), "probe", "--default-branch", "--json", "--", "-x")
	if code != 0 || strings.Join(got, " ") != "-x" || strings.HasPrefix(so, "{") {
		t.Fatalf("exit %d positionals %q stdout %q", code, got, so)
	}
}

func TestMarshalFailureStillWritesEnvelope(t *testing.T) {
	r := testutil.NewRepo(t)
	code, so, _ := run(t, r.Dir, probeCmd(Outcome{Result: make(chan int)}, nil, nil), "probe", "--json")
	if code == 0 {
		t.Fatalf("exit 0 despite marshal failure")
	}
	m := decode(t, so)
	f, _ := m["findings"].([]any)
	if m["ok"] != false || m["result"] != nil || len(f) != 1 || f[0].(map[string]any)["rule"] != "internal" {
		t.Errorf("envelope = %s", so)
	}
}

func TestExitCodePrecedence(t *testing.T) {
	f := []Finding{{Rule: "x"}}
	u := []Unreadable{{Source: "s"}}
	cases := []struct {
		name string
		o    Outcome
		err  error
		want int
	}{
		{"clean", Outcome{}, nil, 0},
		{"findings", Outcome{Findings: f}, nil, 1},
		{"unreadable", Outcome{Unreadable: u}, nil, 3},
		{"both", Outcome{Findings: f, Unreadable: u}, nil, 3},
		{"usage", Outcome{}, &UsageError{"x"}, 2},
		{"env", Outcome{}, &EnvError{"x"}, 4},
		{"other", Outcome{}, errors.New("x"), 1},
	}
	for _, c := range cases {
		if got := ExitCode(c.o, c.err); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}
