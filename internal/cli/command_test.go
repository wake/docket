package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func runIn(t *testing.T, dir string, env []string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(Options{Args: args, Dir: dir, Env: env, Stdout: &out, Stderr: &errb})
	return code, out.String(), errb.String()
}

func TestCommandHelp(t *testing.T) {
	cases := map[string][]string{
		"init":    {"--tools", "--force", "--json"},
		"inspect": {"--json"},
	}
	for name, flags := range cases {
		for _, h := range []string{"--help", "-h"} {
			for _, args := range [][]string{{name, h}, {name, h, "--json"}, {"--json", name, h}} {
				code, out, errs := run(t, args...)
				if code != 0 || errs != "" || !strings.HasPrefix(out, "usage: dkt "+name) {
					t.Fatalf("%v: code=%d out=%q err=%q", args, code, out, errs)
				}
				for _, f := range flags {
					if !strings.Contains(out, f) {
						t.Errorf("%v: help lacks %s: %q", args, f, out)
					}
				}
			}
		}
	}
}

func TestHelpListsCommands(t *testing.T) {
	_, out, _ := run(t, "--help")
	for _, l := range []string{"  dkt init [--tools <coding agent>] [--force] [--json]", "  dkt inspect [--json]"} {
		if !strings.Contains(out, l) {
			t.Errorf("help lacks %q: %q", l, out)
		}
	}
}

func TestCommandUsageError(t *testing.T) {
	code, out, errs := run(t, "init", "--nosuch")
	if code != 2 || out != "" || !strings.Contains(errs, "dkt init:") || !strings.Contains(errs, "dkt init [--tools") {
		t.Errorf("code=%d out=%q err=%q", code, out, errs)
	}
	code, out, _ = run(t, "init", "--nosuch", "--json")
	var m struct {
		Command  string
		Findings []Finding
	}
	if err := json.Unmarshal([]byte(out), &m); err != nil || code != 2 || m.Command != "init" || len(m.Findings) != 1 || m.Findings[0].Rule != "usage" {
		t.Errorf("code=%d out=%q", code, out)
	}
	if code, _, errs = run(t, "init", "--tools"); code != 2 || !strings.Contains(errs, "dkt init:") {
		t.Errorf("missing value: code=%d err=%q", code, errs)
	}
}

func TestUnknownCommandStillUsage(t *testing.T) {
	if code, _, errs := run(t, "nosuch"); code != 2 || !strings.Contains(errs, "unknown command") {
		t.Errorf("code=%d err=%q", code, errs)
	}
}

func TestOutsideRepository(t *testing.T) {
	for _, name := range []string{"init", "inspect"} {
		code, out, _ := runIn(t, t.TempDir(), testutil.Env(t), name, "--json")
		var m struct {
			Command  string
			Repo     json.RawMessage
			Findings []Finding
		}
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatalf("%s: %q: %v", name, out, err)
		}
		if code != 4 || string(m.Repo) != "null" || m.Command != name || len(m.Findings) != 1 || m.Findings[0].Rule != "no-repository" {
			t.Errorf("%s: code=%d out=%q", name, code, out)
		}
	}
}

func TestGitMissingCommand(t *testing.T) {
	r := testutil.NewRepo(t)
	env := append(append([]string{}, r.Env...), "PATH="+t.TempDir())
	code, out, _ := runIn(t, r.Dir, env, "inspect", "--json")
	var m struct{ Findings []Finding }
	if err := json.Unmarshal([]byte(out), &m); err != nil || code != 4 || len(m.Findings) != 1 || m.Findings[0].Rule != "git-missing" {
		t.Errorf("code=%d out=%q", code, out)
	}
}

func TestCommandErrorText(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("one", map[string]string{"f": "1"})
	code, _, errs := runIn(t, r.Dir, r.Env, "init")
	if code != 2 || !strings.HasPrefix(errs, "dkt init: ") {
		t.Errorf("code=%d err=%q", code, errs)
	}
}

func TestUnexpectedArgument(t *testing.T) {
	code, out, errs := run(t, "inspect", "extra", "--json")
	var m struct {
		Command  string
		Findings []Finding
	}
	if err := json.Unmarshal([]byte(out), &m); err != nil || code != 2 || m.Command != "inspect" || len(m.Findings) != 1 || m.Findings[0].Rule != "usage" {
		t.Errorf("code=%d out=%q", code, out)
	}
	if !strings.Contains(errs, `dkt inspect: unexpected argument "extra"`) {
		t.Errorf("err=%q", errs)
	}
	if code, _, _ = run(t, "init", "extra"); code != 2 {
		t.Errorf("init extra: code=%d", code)
	}
}

func TestUsageBeforeRepository(t *testing.T) {
	if code, _, _ := runIn(t, t.TempDir(), testutil.Env(t), "init", "--nosuch"); code != 2 {
		t.Errorf("code=%d", code)
	}
}

func TestInitHelpFlagLine(t *testing.T) {
	_, out, _ := run(t, "init", "--help")
	if !strings.Contains(out, "\n  --tools") {
		t.Errorf("out=%q", out)
	}
}

func TestFlagValueReachesRun(t *testing.T) {
	var got string
	cmds := []func() Command{func() Command {
		var tools string
		return Command{
			Name: "x", Usage: "x",
			Define: func(fs *flag.FlagSet) { fs.StringVar(&tools, "tools", "", "") },
			Run:    func(*Context, []string) (Outcome, error) { got = tools; return Outcome{}, nil },
		}
	}}
	r := testutil.NewRepo(t)
	var out, errb bytes.Buffer
	code := RunWith(Options{Args: []string{"x", "--tools", "claude"}, Dir: r.Dir, Env: r.Env, Stdout: &out, Stderr: &errb}, cmds)
	if code != 0 || got != "claude" {
		t.Errorf("code=%d got=%q err=%q", code, got, errb.String())
	}
}

func positionalCmd(gotArgs *[]string, gotTools *string) []func() Command {
	return []func() Command{func() Command {
		var tools string
		return Command{
			Name: "x", Usage: "x <slug> [--tools <t>]", Args: 1,
			Define: func(fs *flag.FlagSet) { fs.StringVar(&tools, "tools", "", "") },
			Run: func(_ *Context, a []string) (Outcome, error) {
				*gotArgs, *gotTools = a, tools
				return Outcome{}, nil
			},
		}
	}}
}

func TestPositionalPlacement(t *testing.T) {
	for _, args := range [][]string{
		{"x", "s", "--tools", "c"},
		{"x", "--tools", "c", "s"},
		{"x", "--tools", "c", "s", "--json"},
		{"x", "--json", "s", "--tools", "c"},
		{"x", "--tools", "c", "--", "s"},
	} {
		var a []string
		var tools string
		r := testutil.NewRepo(t)
		var out, errb bytes.Buffer
		code := RunWith(Options{Args: args, Dir: r.Dir, Env: r.Env, Stdout: &out, Stderr: &errb}, positionalCmd(&a, &tools))
		if code != 0 || len(a) != 1 || a[0] != "s" || tools != "c" {
			t.Errorf("%v: code=%d args=%v tools=%q err=%q", args, code, a, tools, errb.String())
		}
	}
}

func TestPositionalCountWrong(t *testing.T) {
	for _, tc := range []struct {
		args []string
		msg  string
	}{
		{[]string{"x"}, "missing argument"},
		{[]string{"x", "--tools", "c"}, "missing argument"},
		{[]string{"x", "a", "b"}, `unexpected argument "b"`},
		{[]string{"x", "a", "--tools", "c", "b"}, `unexpected argument "b"`},
	} {
		for _, j := range []bool{false, true} {
			args := tc.args
			if j {
				args = append(append([]string{}, args...), "--json")
			}
			var a []string
			var tools string
			var out, errb bytes.Buffer
			code := RunWith(Options{Args: args, Dir: t.TempDir(), Stdout: &out, Stderr: &errb}, positionalCmd(&a, &tools))
			if code != 2 || a != nil || !strings.Contains(errb.String(), tc.msg) || !strings.Contains(errb.String(), "usage: dkt x <slug>") {
				t.Errorf("%v: code=%d args=%v err=%q", args, code, a, errb.String())
			}
			if j != strings.HasPrefix(out.String(), "{") {
				t.Errorf("%v: json=%v out=%q", args, j, out.String())
			}
		}
	}
}

func TestZeroArgCommandsRefusePositional(t *testing.T) {
	for _, name := range []string{"init", "inspect"} {
		for _, args := range [][]string{{name, "a"}, {name, "--json", "a"}, {name, "--", "a"}} {
			if code, _, errs := run(t, args...); code != 2 || !strings.Contains(errs, `unexpected argument "a"`) {
				t.Errorf("%v: code=%d err=%q", args, code, errs)
			}
		}
	}
}
