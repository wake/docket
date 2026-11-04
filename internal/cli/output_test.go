package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(Options{Args: args, Dir: t.TempDir(), Stdout: &out, Stderr: &errb})
	return code, out.String(), errb.String()
}

func TestVersionText(t *testing.T) {
	code, out, _ := run(t, "--version")
	if code != 0 || out != "dkt dev\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestVersionJSONEnvelope(t *testing.T) {
	for _, args := range [][]string{{"--version", "--json"}, {"--json", "--version"}} {
		code, out, _ := run(t, args...)
		if code != 0 {
			t.Fatalf("%v: code=%d", args, code)
		}
		if strings.Count(strings.TrimSpace(out), "\n") != 0 {
			t.Fatalf("%v: not one line: %q", args, out)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if len(m) != 8 {
			t.Fatalf("%v: want 8 fields, got %d: %s", args, len(m), out)
		}
		want := map[string]string{
			"tool": `"dkt"`, "version": `"dev"`, "command": `"version"`, "repo": `null`,
			"ok": `true`, "findings": `[]`, "unreadable": `[]`, "result": `{"version":"dev"}`,
		}
		for k, v := range want {
			if string(m[k]) != v {
				t.Errorf("%v: %s = %s, want %s", args, k, m[k], v)
			}
		}
	}
}
