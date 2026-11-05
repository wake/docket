package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

type lintOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     *lintResult
}

// lintRepo is a repository with a change add-x of the given schema.
func lintRepo(t *testing.T, schema string) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Write("openspec/changes/add-x/.openspec.yaml", "schema: "+schema+"\ncreated: 2026-01-01\n")
	return r
}

// doLint runs dkt lint --json with the fake OpenSpec (version 1.14.0) and returns the exit code,
// the decoded JSON, the raw output and the fake's call log.
func doLint(t *testing.T, dir string, env []string, args ...string) (int, lintOut, string, string) {
	t.Helper()
	cmd, log := testutil.FakeOpenSpec(t, "1.14.0")
	env = append(append([]string{}, env...), "DOCKET_OPENSPEC="+cmd, "FAKE_OPENSPEC_JSON="+t.TempDir())
	code, out, errs := runIn(t, dir, env, append([]string{"lint", "--json"}, args...)...)
	var m lintOut
	if out != "" {
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatalf("%q: %v", out, err)
		}
	}
	calls, _ := os.ReadFile(log)
	return code, m, out + errs, string(calls)
}

func TestLintClean(t *testing.T) { // Clean change
	r := lintRepo(t, "spec-driven")
	code, m, raw, _ := doLint(t, r.Dir, r.Env, "add-x")
	if code != 0 || len(m.Findings) != 0 || m.Result == nil || m.Result.Specs != 0 ||
		len(m.Result.Changes) != 1 || m.Result.Changes[0] != (lintChange{Slug: "add-x", Schema: "spec-driven"}) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintNothingLinted(t *testing.T) {
	r := testutil.NewRepo(t)
	code, _, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || !strings.Contains(raw, `"changes":[]`) || !strings.Contains(raw, `"specs":0`) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintOutsideRepository(t *testing.T) { // Lint outside a repository
	code, m, raw, _ := doLint(t, t.TempDir(), testutil.Env(t))
	if code != 4 || ruleList(m.Findings) != "no-repository" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintBadNamesReadNothing(t *testing.T) {
	r := lintRepo(t, "docket")
	for _, name := range []string{"", "a/b", `a\b`, ".", ".."} {
		code, _, raw, calls := doLint(t, r.Dir, r.Env, name)
		if code != 2 || !strings.Contains(raw, "not a change name") || calls != "" {
			t.Fatalf("%q: %d %s calls=%q", name, code, raw, calls)
		}
	}
	empty := testutil.NewRepo(t)
	if code, _, raw, _ := doLint(t, empty.Dir, empty.Env, "a/b"); code != 2 {
		t.Fatalf("%d %s", code, raw)
	}
	if _, err := os.Stat(filepath.Join(empty.Dir, "openspec")); err == nil {
		t.Fatal("openspec/ was created")
	}
}

func lintWithoutOpenSpec(t *testing.T, r *testutil.Repo) (int, lintOut, string) {
	t.Helper()
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC=/nonexistent/openspec")
	code, out, errs := runIn(t, r.Dir, env, "lint", "--json")
	var m lintOut
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("%q %q: %v", out, errs, err)
	}
	return code, m, out + errs
}

func TestLintOpenSpecMissing(t *testing.T) {
	code, m, raw := lintWithoutOpenSpec(t, lintRepo(t, "docket"))
	if code != 4 || ruleList(m.Findings) != "openspec-missing" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintMainSpecsNeedNoOpenSpec(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write("openspec/specs/foo/spec.md", "# Foo Bar Specification\n\n## Purpose\nx\n")
	code, m, raw := lintWithoutOpenSpec(t, r)
	if code != 0 || m.Result == nil || m.Result.Specs != 1 {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads everything")
	}
	r := lintRepo(t, "docket")
	r.Write("openspec/changes/add-x/proposal.md", "x\n")
	if err := os.Chmod(filepath.Join(r.Dir, "openspec/changes/add-x/proposal.md"), 0); err != nil {
		t.Fatal(err)
	}
	code, m, raw, _ := doLint(t, r.Dir, r.Env, "add-x")
	if code != 3 || len(m.Unreadable) == 0 {
		t.Fatalf("%d %s", code, raw)
	}
}

const lintProposal = `---
title: Add x
kind: feature
source: test
specs:
  x-cap: X capability
---

# Proposal

## Why

Because.

## What Changes

Things.

## Non-Goals

- Nothing else.

## Capabilities

### New Capabilities
- ` + "`x-cap`" + `: x

### Modified Capabilities

## Impact

None.
`

// lintDocket runs dkt lint add-x --json on a docket change with the given proposal and the fake's answers.
func lintDocket(t *testing.T, proposal string) (int, lintOut, string) {
	t.Helper()
	r := lintRepo(t, "docket")
	r.Write("openspec/changes/add-x/proposal.md", proposal)
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.0")
	dir := t.TempDir()
	testutil.OpenSpecAnswer(t, dir, 0, `{"deltas":[{"spec":"x-cap","operation":"ADDED","requirements":[{"text":"The system SHALL x.","scenarios":[]}]}]}`,
		"show", "add-x", "--type", "change", "--json", "--deltas-only")
	testutil.OpenSpecAnswer(t, dir, 0, `{"items":[{"id":"add-x","valid":true,"issues":[]}]}`,
		"validate", "add-x", "--type", "change", "--json")
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd, "FAKE_OPENSPEC_JSON="+dir)
	code, out, errs := runIn(t, r.Dir, env, "lint", "add-x", "--json")
	var m lintOut
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("%q %q: %v", out, errs, err)
	}
	return code, m, out
}

func TestLintCleanDocketChange(t *testing.T) { // Clean change, docket schema
	code, m, out := lintDocket(t, lintProposal)
	if code != 0 || len(m.Findings) != 0 || m.Result == nil || m.Result.Specs != 0 ||
		len(m.Result.Changes) != 1 || m.Result.Changes[0] != (lintChange{Slug: "add-x", Schema: "docket"}) {
		t.Fatalf("%d %s", code, out)
	}
}

func TestLintFindingFields(t *testing.T) {
	code, m, out := lintDocket(t, strings.Replace(lintProposal, "kind: feature", "kind: fix", 1))
	var got *Finding
	for i, f := range m.Findings {
		if f.Rule == "metadata-invalid" {
			got = &m.Findings[i]
		}
	}
	if code != 1 || got == nil || got.Change != "add-x" || got.File != "openspec/changes/add-x/proposal.md" || got.Line != 3 {
		t.Fatalf("%d %s", code, out)
	}
}

func TestLintText(t *testing.T) {
	r := lintRepo(t, "spec-driven")
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.0")
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd)
	code, out, _ := runIn(t, r.Dir, env, "lint", "add-x", "add-z")
	lines := strings.Split(out, "\n")
	if code != 1 || lines[0] != "Linted 1 changes and 0 main spec titles." || !strings.Contains(out, "change-missing") {
		t.Fatalf("%d %q", code, out)
	}
}
