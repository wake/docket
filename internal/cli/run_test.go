package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func usageEnvelope(t *testing.T, args ...string) {
	t.Helper()
	code, out, errs := run(t, args...)
	if code != 2 {
		t.Fatalf("%v: code=%d", args, code)
	}
	if !strings.Contains(errs, "usage:") {
		t.Errorf("%v: stderr lacks usage: %q", args, errs)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &m); err != nil || strings.Count(strings.TrimSpace(out), "\n") != 0 {
		t.Fatalf("%v: stdout not one object: %q", args, out)
	}
	if len(m) != 8 || string(m["ok"]) != "false" || string(m["command"]) != `""` {
		t.Errorf("%v: bad envelope: %s", args, out)
	}
	var fs []Finding
	if err := json.Unmarshal(m["findings"], &fs); err != nil || len(fs) != 1 || fs[0].Rule != "usage" {
		t.Errorf("%v: findings = %s", args, m["findings"])
	}
}

func TestUsageErrorsJSON(t *testing.T) {
	for _, args := range [][]string{{"--json", "nosuch"}, {"nosuch", "--json"}, {"--json"}} {
		usageEnvelope(t, args...)
	}
}

func TestUsageErrorsText(t *testing.T) {
	for _, args := range [][]string{{}, {"nosuch"}} {
		code, out, errs := run(t, args...)
		if code != 2 || out != "" || !strings.Contains(errs, "usage:") {
			t.Errorf("%v: code=%d out=%q err=%q", args, code, out, errs)
		}
	}
}

func TestJSONAfterTerminatorIsPositional(t *testing.T) {
	code, out, errs := run(t, "nosuch", "--", "--json")
	if code != 2 || out != "" || !strings.Contains(errs, "usage:") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestHelpIgnoresJSON(t *testing.T) {
	for _, args := range [][]string{{"--help", "--json"}, {"--json", "--help"}, {"-h", "--json"}} {
		code, out, errs := run(t, args...)
		if code != 0 || !strings.HasPrefix(out, "usage:") || errs != "" {
			t.Errorf("%v: code=%d out=%q err=%q", args, code, out, errs)
		}
	}
}

func TestHelp(t *testing.T) {
	for _, a := range []string{"--help", "-h"} {
		code, out, errs := run(t, a)
		if code != 0 || !strings.Contains(out, "usage:") || errs != "" {
			t.Errorf("%s: code=%d out=%q err=%q", a, code, out, errs)
		}
	}
}
