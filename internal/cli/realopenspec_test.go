package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/setup"
	"github.com/wake/docket/internal/testutil"
)

// TestRealOpenSpec runs dkt init, dkt new, dkt inspect and dkt lint against the real OpenSpec CLI
// named by DOCKET_OPENSPEC. CI runs it for OpenSpec 1.14.0 and the latest release.
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

	r.Git("add", "-A") // as a user commits what init created
	r.Git("commit", "-q", "-m", "set up")

	code, out, errs = runIn(t, r.Dir, env, "new", "add-x", "--json", "--kind", "feature", "--source", "x", "--title", "Add x")
	if code != 0 {
		t.Fatalf("new: code=%d out=%q err=%q", code, out, errs)
	}
	var nw newOut
	if err := json.Unmarshal([]byte(out), &nw); err != nil {
		t.Fatalf("%q: %v", out, err)
	}
	if nw.Result.Slug != "add-x" || len(nw.Result.Used) != 0 {
		t.Errorf("new result = %+v, want slug add-x and no used", nw.Result)
	}
	meta, err := os.ReadFile(filepath.Join(r.Dir, "openspec/changes/add-x/.openspec.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains("\n"+string(meta), "\nschema: docket\n") {
		t.Errorf(".openspec.yaml lacks schema: docket: %q", meta)
	}
	prop, err := os.ReadFile(filepath.Join(r.Dir, "openspec/changes/add-x/proposal.md"))
	if err != nil {
		t.Fatal(err)
	}
	if want := change.FrontMatter("Add x", "feature", "x"); string(prop) != string(want) {
		t.Errorf("proposal.md = %q, want %q", prop, want)
	}

	rn, err := openspec.New(r.Dir, env)
	if err != nil {
		t.Fatal(err)
	}
	var steps [][]string
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

	// dkt lint: a clean change, then an unclaimed scenario, then a scenario-less requirement.
	base := filepath.Join(r.Dir, "openspec/changes/add-x")
	write := func(rel, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(base, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fm := strings.TrimSuffix(string(change.FrontMatter("Add x", "feature", "x")), "---\n")
	write("proposal.md", fm+"specs:\n  rooms: Rooms\n---\n\n# Proposal\n\n## Why\n\nPeople need to see the rooms.\n\n"+
		"## What Changes\n\nA room list.\n\n## Non-Goals\n\nNo booking.\n\n## Capabilities\n\nRooms.\n\n## Impact\n\nNone.\n")
	scenario := "#### Scenario: List\n- **WHEN** a user asks for rooms\n- **THEN** the rooms are listed\n"
	delta := "## ADDED Requirements\n\n### Requirement: Room list\nThe system SHALL list rooms.\n\n"
	write("specs/rooms/spec.md", delta+scenario)
	plan := func(scenarios string) string {
		return "# Implementation Plan\n\n## Reuse\n\n| Need | Existing (path:symbol) | Decision | Reason | Evidence |\n|---|---|---|---|---|\n" +
			"| Config | openspec/config.yaml | reuse | Already there | read it |\n\n### Task 1: Room list\n\n**Scenarios:** " + scenarios + "\n\n- [ ] **Step 1: List rooms**\n"
	}
	write("plan.md", plan("rooms#list"))

	lint := func(wantCode int, args ...string) lintOut {
		t.Helper()
		code, out, errs := runIn(t, r.Dir, env, append([]string{"lint", "--json"}, args...)...)
		var l lintOut
		if err := json.Unmarshal([]byte(out), &l); err != nil {
			t.Fatalf("lint: %q %q: %v", out, errs, err)
		}
		if code != wantCode {
			t.Fatalf("lint %v: code=%d, want %d: out=%q err=%q", args, code, wantCode, out, errs)
		}
		return l
	}
	hasRule := func(l lintOut, rule, sub string) bool {
		for _, f := range l.Findings {
			if f.Rule == rule && strings.Contains(f.Message, sub) {
				return true
			}
		}
		return false
	}

	l := lint(0, "add-x")
	if len(l.Findings) != 0 || l.Result == nil || len(l.Result.Changes) != 1 || l.Result.Changes[0] != (lintChange{Slug: "add-x", Schema: "docket"}) || l.Result.Specs != 0 {
		t.Errorf("clean lint: findings=%+v result=%+v", l.Findings, l.Result)
	}

	write("plan.md", plan("none"))
	if l = lint(1, "add-x"); !hasRule(l, "scenario-unclaimed", "rooms#list") {
		t.Errorf("want scenario-unclaimed for rooms#list: %+v", l.Findings)
	}

	write("plan.md", plan("rooms#list"))
	write("specs/rooms/spec.md", delta)
	if l = lint(1, "add-x"); !hasRule(l, "openspec-invalid", "") {
		t.Errorf("want openspec-invalid: %+v", l.Findings)
	}

	l = lint(1)
	if l.Result == nil || len(l.Result.Changes) != 1 || l.Result.Changes[0].Slug != "add-x" || l.Result.Specs != 0 {
		t.Errorf("lint without a name: result=%+v", l.Result)
	}
}

// TestRealOpenSpecConflicts runs dkt conflicts against the real OpenSpec CLI: a rename in one
// worktree and a modification, with other spacing in the name, in another.
func TestRealOpenSpecConflicts(t *testing.T) {
	if os.Getenv("DOCKET_TEST_OPENSPEC") != "1" {
		t.Skip("set DOCKET_TEST_OPENSPEC=1 to run against the real OpenSpec")
	}
	r := testutil.NewRepo(t)
	r.Commit("base", map[string]string{"README.md": "x"})
	env := append([]string{}, r.Env...)
	for _, k := range []string{"DOCKET_OPENSPEC", "PATH", "npm_config_cache"} {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	other := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "-b", "other", other)

	put(t, r.Dir, map[string]string{
		"openspec/changes/add-x/proposal.md":           string(change.FrontMatter("Add", "feature", "x")),
		"openspec/changes/add-x/specs/booking/spec.md": "## RENAMED Requirements\n\n- FROM: `### Requirement: Cancel booking`\n- TO: `### Requirement: Cancel reservation`\n",
	})
	put(t, other, map[string]string{
		"openspec/changes/add-y/specs/booking/spec.md": "## MODIFIED Requirements\n\n### Requirement: Cancel  Booking\nThe system SHALL cancel a booking.\n\n" +
			"#### Scenario: Cancel\n- **WHEN** a user cancels\n- **THEN** the booking is cancelled\n",
	})

	code, out, errs := runIn(t, r.Dir, env, "conflicts", "--json")
	var m conflictsOut
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("%q %q: %v", out, errs, err)
	}
	if code != 1 || m.Result == nil || len(m.Result.Overlaps) != 1 {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
	got := map[string][]string{}
	for _, c := range m.Result.Overlaps[0].Changes {
		got[c.Slug] = c.Operations
	}
	if !slices.Equal(got["add-x"], []string{"RENAMED-FROM"}) || !slices.Equal(got["add-y"], []string{"MODIFIED"}) {
		t.Errorf("overlap = %+v", m.Result.Overlaps[0])
	}
}
