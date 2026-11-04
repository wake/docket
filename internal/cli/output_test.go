package cli

import (
	"bytes"
	"encoding/json"
	"errors"
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
	code, out, errs := run(t, "--version")
	if code != 0 || out != "dkt dev\n" || errs != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestVersionJSONEnvelope(t *testing.T) {
	for _, args := range [][]string{{"--version", "--json"}, {"--json", "--version"}} {
		code, out, errs := run(t, args...)
		if code != 0 || errs != "" {
			t.Fatalf("%v: code=%d err=%q", args, code, errs)
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

func TestFindingOptionalFieldsJSON(t *testing.T) {
	var buf bytes.Buffer
	o := Outcome{Findings: []Finding{{Rule: "r", Message: "m", Change: "c", File: "f.md", Line: 7}, {Rule: "r", Message: "m"}}}
	if err := writeJSON(&buf, "x", nil, o, 1); err != nil {
		t.Fatal(err)
	}
	var m struct{ Findings []map[string]any }
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil || len(m.Findings) != 2 {
		t.Fatalf("%v %s", err, buf.String())
	}
	full, bare := m.Findings[0], m.Findings[1]
	if full["change"] != "c" || full["file"] != "f.md" || full["line"] != float64(7) {
		t.Errorf("full = %v", full)
	}
	for _, k := range []string{"change", "file", "line"} {
		if _, ok := bare[k]; ok {
			t.Errorf("bare has %s: %v", k, bare)
		}
	}
}

func TestFindingTextFormat(t *testing.T) {
	var buf bytes.Buffer
	o := Outcome{Findings: []Finding{{Rule: "r", Message: "m", Change: "c", File: "f.md", Line: 7}, {Rule: "r", Message: "m", File: "f.md"}}}
	if err := writeText(&buf, "x", o, 1); err != nil {
		t.Fatal(err)
	}
	if want := "r: m (f.md:7) [c]\nr: m (f.md)\n"; buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestWriteJSONMarshalFailure(t *testing.T) {
	var buf bytes.Buffer
	err := writeJSON(&buf, "x", nil, Outcome{Result: make(chan int)}, 0)
	if err == nil {
		t.Fatal("want error")
	}
	var m struct {
		OK       bool
		Findings []Finding
	}
	if e := json.Unmarshal(buf.Bytes(), &m); e != nil || m.OK || len(m.Findings) != 1 || m.Findings[0].Rule != "internal" {
		t.Fatalf("out=%q err=%v", buf.String(), e)
	}
}

func TestRunWriterErrorIsNonZero(t *testing.T) {
	var errb bytes.Buffer
	if code := Run(Options{Args: []string{"--version"}, Stdout: failWriter{}, Stderr: &errb}); code == 0 {
		t.Fatal("want non-zero")
	}
}
