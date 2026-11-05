package cli

import (
	"bytes"
	"encoding/json"
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
	code, _, errs := runIn(t, r.Dir, r.Env, "inspect")
	if code != 1 || !strings.HasPrefix(errs, "dkt inspect: ") {
		t.Errorf("code=%d err=%q", code, errs)
	}
}
