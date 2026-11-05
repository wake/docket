package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/setup"
	"github.com/wake/docket/internal/testutil"
)

// TestRealOpenSpec runs dkt init and dkt inspect against the real OpenSpec CLI
// named by DOCKET_OPENSPEC. CI runs it for OpenSpec 1.7.0 and the latest release.
func TestRealOpenSpec(t *testing.T) {
	if os.Getenv("DOCKET_TEST_OPENSPEC") != "1" {
		t.Skip("set DOCKET_TEST_OPENSPEC=1 to run against the real OpenSpec")
	}
	r := testutil.NewRepo(t)
	env := append([]string{}, r.Env...)
	for _, k := range []string{"DOCKET_OPENSPEC", "PATH", "npm_config_cache"} {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}

	code, out, errs := runIn(t, r.Dir, env, "init", "--json", "--tools", "claude")
	if code != 0 {
		t.Fatalf("init: code=%d out=%q err=%q", code, out, errs)
	}
	var m initOut
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("%q: %v", out, err)
	}
	if m.Result.OpenspecInit == nil || *m.Result.OpenspecInit != "claude" {
		t.Errorf("openspec_init = %v, want claude", m.Result.OpenspecInit)
	}
	var config, skill bool
	for _, f := range m.Result.Files {
		config = config || f.Path == "openspec/config.yaml"
		skill = skill || strings.HasPrefix(f.Path, ".claude/skills/openspec-")
		if f.Kind != string(setup.KindOpenSpec) && f.Status != "created" {
			t.Errorf("%s: status %q, want created", f.Path, f.Status)
		}
	}
	if !config || !skill {
		t.Errorf("files lack openspec/config.yaml (%v) or .claude/skills/openspec-* (%v): %+v", config, skill, m.Result.Files)
	}

	rn, err := openspec.New(r.Dir, env)
	if err != nil {
		t.Fatal(err)
	}
	steps := [][]string{{"new", "change", "add-x"}}
	for _, a := range []string{"proposal", "specs", "design", "plan"} {
		steps = append(steps, []string{"instructions", a, "--change", "add-x"})
	}
	steps = append(steps, []string{"status", "--change", "add-x"})
	for _, args := range steps {
		got, err := rn.Run(args...)
		if err != nil {
			t.Fatalf("openspec %v: %v", args, err)
		}
		if args[1] == "plan" && !strings.Contains(got, "plan.md") {
			t.Errorf("instructions plan does not name plan.md: %q", got)
		}
	}

	code, out, errs = runIn(t, r.Dir, env, "inspect", "--json")
	if code != 0 {
		t.Fatalf("inspect: code=%d out=%q err=%q", code, out, errs)
	}
	var in initOut
	if err := json.Unmarshal([]byte(out), &in); err != nil {
		t.Fatalf("%q: %v", out, err)
	}
	if len(in.Findings) != 0 {
		t.Errorf("inspect findings: %+v", in.Findings)
	}
}
