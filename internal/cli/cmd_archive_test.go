package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket/internal/archive"
	"github.com/wake/docket/internal/testutil"
)

const archiveOrigin = "https://example.test/org/repo.git"

const archiveDelta = "## ADDED Requirements\n\n### Requirement: Do x\nThe system SHALL do x.\n\n#### Scenario: Works\n- **WHEN** a\n- **THEN** b\n"

const archiveShow = `{"deltas":[{"spec":"x","operation":"ADDED","requirement":{"name":"Do x","scenarios":[{"name":"Works"}]}}]}`

const archivePlan = "# Plan\n\n## Reuse\n\n| Need | Existing (path:symbol) | Decision | Reason | Evidence |\n|---|---|---|---|---|\n" +
	"| X | `internal/x.go` | reuse | | rg X |\n\n### Task 1: One\n\n**Scenarios:** x#works\n\n- [x] **Step 1: a**\n- [x] **Step 2: b**\n"

func runArchive(t *testing.T, r *testutil.Repo, env []string, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	env = append(append([]string{}, env...), "XDG_DATA_HOME="+dataHome(r.Dir))
	code := Run(Options{Args: append([]string{"archive"}, args...), Dir: r.Dir, Env: env, Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

func archiveFixture(t *testing.T) *testutil.Repo {
	t.Helper()
	return archiveFixtureWith(t, true)
}

// archiveFixtureWith is archiveFixture with or without a project copy of the
// schema.
func archiveFixtureWith(t *testing.T, projectCopy bool) *testutil.Repo {
	t.Helper()
	r := cleanRepoWith(t, projectCopy)
	r.Git("remote", "add", "origin", archiveOrigin)
	r.Commit("code", map[string]string{"internal/x.go": "package x\n", "openspec/config.yaml": "schema: docket\n"})
	base := r.Git("rev-parse", "HEAD")
	r.Commit("add the change", map[string]string{
		"openspec/changes/add-x/.openspec.yaml":      "schema: docket\n",
		"openspec/changes/add-x/proposal.md":         lintProposal,
		"openspec/changes/add-x/docket.yaml":         "title: X\nkind: feature\nsource: test\nbase: " + base + "\n",
		"openspec/changes/add-x/specs/x/docket.yaml": "title: X\n",
		"openspec/changes/add-x/specs/x/spec.md":     archiveDelta,
		"openspec/changes/add-x/plan.md":             archivePlan,
	})
	r.Commit("work\n\nChange: add-x\nTask: 1", map[string]string{"a.txt": "a\n"})
	code, so, se := runLint(t, r.Dir, r.Env, "add-x")
	if code != 0 {
		t.Fatalf("fixture does not lint clean: exit %d\n%s\n%s", code, so, se)
	}
	return r
}

func archiveEnv(t *testing.T, r *testutil.Repo) []string {
	t.Helper()
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.3")
	show := filepath.Join(t.TempDir(), "show.json")
	if err := os.WriteFile(show, []byte(archiveShow), 0o644); err != nil {
		t.Fatal(err)
	}
	return append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd, "FAKE_OPENSPEC_SHOW="+show)
}

func TestArchiveCommandRefusesUncommitted(t *testing.T) {
	r := archiveFixture(t)
	r.Write("dirty.txt", "x\n")
	code, so, se := runArchive(t, r, archiveEnv(t, r), "add-x", "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if !hasRule(t, so, "uncommitted-changes") {
		t.Fatalf("no finding:\n%s", so)
	}
	res := decode(t, so)["result"].(map[string]any)
	if !reflect.DeepEqual(res["stopped"], map[string]any{"change": "", "step": "preconditions"}) {
		t.Fatalf("stopped = %v", res["stopped"])
	}
	for _, k := range []string{"archived", "drifts"} {
		if a, ok := res[k].([]any); !ok || len(a) != 0 {
			t.Fatalf("%s = %v", k, res[k])
		}
	}
	if res["readings"] != nil {
		t.Fatalf("readings = %v", res["readings"])
	}
}

func TestArchiveCommandArchives(t *testing.T) {
	r := archiveFixture(t)
	head := r.Git("rev-parse", "HEAD")
	code, so, se := runArchive(t, r, archiveEnv(t, r), "add-x", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	want := []any{map[string]any{"change": "add-x", "folder": "openspec/changes/archive/2026-01-02-add-x",
		"evidence": "openspec/changes/archive/2026-01-02-add-x/evidence.md"}}
	if !reflect.DeepEqual(res["archived"], want) || res["stopped"] != nil {
		t.Fatalf("result %v", res)
	}
	b, err := os.ReadFile(filepath.Join(r.Dir, "openspec/changes/archive/2026-01-02-add-x/evidence.md"))
	if err != nil {
		t.Fatal(err)
	}
	task := r.Git("log", "-1", "--format=%H", "--grep=Task: 1")
	for _, w := range []string{archiveOrigin, task[:7], "1.14.3"} {
		if !strings.Contains(string(b), w) {
			t.Errorf("evidence.md lacks %q:\n%s", w, b)
		}
	}
	if got := r.Git("rev-parse", "HEAD"); got != head {
		t.Errorf("HEAD moved: %s -> %s", head, got)
	}
}

func TestArchiveHumanLines(t *testing.T) {
	r := archiveFixture(t)
	code, so, se := runArchive(t, r, archiveEnv(t, r), "add-x")
	if code != 0 || !strings.Contains(so, "archived add-x as 2026-01-02-add-x") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestArchiveHumanStoppedAtPreconditions(t *testing.T) {
	r := archiveFixture(t)
	r.Write("dirty.txt", "x\n")
	code, so, se := runArchive(t, r, archiveEnv(t, r), "add-x")
	if code != 1 || !strings.Contains(so, "stopped at preconditions") || strings.Contains(so, " for ") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestArchiveHumanStoppedAtCrossCheck(t *testing.T) {
	r := archiveFixture(t)
	env := archiveEnv(t, r)
	show := filepath.Join(t.TempDir(), "other.json")
	other := `{"deltas":[{"spec":"x","operation":"ADDED","requirement":{"name":"Do y","scenarios":[{"name":"Works"}]}}]}`
	if err := os.WriteFile(show, []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	env = append(env, "FAKE_OPENSPEC_SHOW="+show)
	code, so, se := runArchive(t, r, env, "add-x")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	for _, w := range []string{"stopped at cross-check for add-x", "\ndocket:\n  ", "\nopenspec:\n  "} {
		if !strings.Contains(so, w) {
			t.Errorf("output lacks %q:\n%s", w, so)
		}
	}
}

func TestArchiveHumanDriftAndReadings(t *testing.T) {
	rep := archive.ArchiveReport{
		Drifts:   []archive.Drift{{Capability: "x", Requirement: "Do x", Base: "", Now: "line1\nline2\n"}},
		Readings: &archive.Readings{Docket: []string{"a", "b"}, OpenSpec: nil},
	}
	got := archiveHuman(rep)
	want := []string{"x / Do x at the base:", "  (absent)", "now:", "  line1", "  line2",
		"docket:", "  a", "  b", "openspec:", "  (absent)"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestArchiveNeedsOpenSpec(t *testing.T) {
	r := archiveFixture(t)
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC=/nonexistent/openspec")
	code, so, se := runArchive(t, r, env, "add-x")
	if code != 4 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

// The OpenSpec check comes before the default-branch check: with both
// unusable, the finding names OpenSpec.
func TestArchiveVersionCheckComesFirst(t *testing.T) {
	r := archiveFixture(t)
	r.Git("branch", "-m", "trunk")
	r.Git("remote", "remove", "origin")
	old, _ := testutil.FakeOpenSpec(t, "1.13.0")
	for name, cmd := range map[string]string{"missing": "/nonexistent/openspec", "old": old} {
		env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd)
		code, so, se := runArchive(t, r, env, "add-x", "--json")
		if code != 4 || !hasRule(t, so, "environment") || !strings.Contains(strings.ToLower(so), "openspec") ||
			strings.Contains(strings.ToLower(so), "default branch") {
			t.Fatalf("%s: exit %d\n%s\n%s", name, code, so, se)
		}
	}
	// control: with a usable OpenSpec the same repo fails on the branch.
	code, so, se := runArchive(t, r, archiveEnv(t, r), "add-x", "--json")
	if code != 4 || !strings.Contains(strings.ToLower(so), "default branch") {
		t.Fatalf("control: exit %d\n%s\n%s", code, so, se)
	}
}

func TestArchiveNeedsASlug(t *testing.T) {
	r := archiveFixture(t)
	cmd, log := testutil.FakeOpenSpec(t, "1.14.3")
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd)
	code, so, se := runArchive(t, r, env, "--json")
	if code != 2 || !strings.Contains(so, "archive needs at least one slug") || !hasRule(t, so, "usage") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if b, err := os.ReadFile(log); err == nil && len(b) > 0 {
		t.Fatalf("OpenSpec was called:\n%s", b)
	}
}

func TestArchiveKeepsItsReportOnAnInfrastructureError(t *testing.T) {
	r := archiveFixture(t)
	base := r.Git("rev-parse", "HEAD~2")
	r.Commit("add the second change", map[string]string{
		"openspec/changes/add-y/.openspec.yaml":      "schema: docket\n",
		"openspec/changes/add-y/proposal.md":         lintProposal,
		"openspec/changes/add-y/docket.yaml":         "title: Y\nkind: feature\nsource: test\nbase: " + base + "\n",
		"openspec/changes/add-y/specs/x/docket.yaml": "title: Y\n",
		"openspec/changes/add-y/specs/x/spec.md":     archiveDelta,
		"openspec/changes/add-y/plan.md":             archivePlan,
	})
	r.Commit("more work\n\nChange: add-y\nTask: 1", map[string]string{"b.txt": "b\n"})
	if code, so, se := runLint(t, r.Dir, r.Env, "add-y"); code != 0 {
		t.Fatalf("second change does not lint clean: exit %d\n%s\n%s", code, so, se)
	}
	// A wrapper around the fake OpenSpec that loses its execute bit once it
	// has archived, so the next OpenSpec call fails to start (not an exit
	// failure).
	fake, _ := testutil.FakeOpenSpec(t, "1.14.3")
	wrapper := filepath.Join(t.TempDir(), "openspec")
	script := "#!/bin/sh\nout=$('" + fake + "' \"$@\")\nrc=$?\nprintf '%s\\n' \"$out\"\nif [ \"$1\" = archive ]; then chmod -x \"$0\"; fi\nexit $rc\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	show := filepath.Join(t.TempDir(), "show.json")
	if err := os.WriteFile(show, []byte(archiveShow), 0o644); err != nil {
		t.Fatal(err)
	}
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+wrapper, "FAKE_OPENSPEC_SHOW="+show)
	code, so, se := runArchive(t, r, env, "add-x", "add-y", "--json")
	if code != 4 {
		t.Fatalf("exit %d, want 4 (OpenSpec became unusable)\n%s\n%s", code, so, se)
	}
	res, _ := decode(t, so)["result"].(map[string]any)
	archived, _ := res["archived"].([]any)
	if len(archived) != 1 || archived[0].(map[string]any)["change"] != "add-x" {
		t.Fatalf("result lost the archived change: %v\n%s", res, so)
	}
}

func TestArchiveRefusesRepeatedSlugs(t *testing.T) {
	r := archiveFixture(t)
	cmd, log := testutil.FakeOpenSpec(t, "1.14.3")
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd)
	code, so, se := runArchive(t, r, env, "add-x", "add-y", "add-x")
	if code != 2 || !strings.Contains(se, `slug "add-x" is given twice`) {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if b, err := os.ReadFile(log); err == nil && len(b) > 0 {
		t.Fatalf("OpenSpec was called:\n%s", b)
	}
}

const realSpec = "# booking\n\n## Purpose\nBookings.\n\n## Requirements\n\n### Requirement: Cancel booking\nThe system SHALL cancel a booking.\n\n#### Scenario: Cancelled\n- **WHEN** a guest cancels\n- **THEN** it is cancelled\n"

func realDelta(then string) string {
	return "## MODIFIED Requirements\n\n### Requirement: Cancel booking\nThe system SHALL cancel a booking.\n\n#### Scenario: Cancelled\n- **WHEN** a guest cancels\n- **THEN** " + then + "\n"
}

func TestArchiveWithOpenSpec(t *testing.T) {
	real := os.Getenv("DOCKET_TEST_OPENSPEC")
	if real == "" {
		t.Skip("DOCKET_TEST_OPENSPEC is not set")
	}
	r := cleanRepo(t)
	r.Commit("main spec", map[string]string{
		"internal/x.go":                  "package x\n",
		"openspec/config.yaml":           "schema: docket\n",
		"openspec/specs/booking/spec.md": realSpec,
	})
	base := r.Git("rev-parse", "HEAD")
	plan := strings.Replace(archivePlan, "x#works", "booking#cancelled", 1)
	for i, slug := range []string{"add-x", "add-y"} {
		then := []string{"it is cancelled at once", "it is cancelled late"}[i]
		r.Commit("add "+slug, map[string]string{
			"openspec/changes/" + slug + "/.openspec.yaml":            "schema: docket\n",
			"openspec/changes/" + slug + "/proposal.md":               strings.Replace(lintProposal, "`x`", "`booking`", 1),
			"openspec/changes/" + slug + "/docket.yaml":               "title: " + slug + "\nkind: feature\nsource: x\nbase: " + base + "\n",
			"openspec/changes/" + slug + "/specs/booking/spec.md":     realDelta(then),
			"openspec/changes/" + slug + "/specs/booking/docket.yaml": "title: Booking\n",
			"openspec/changes/" + slug + "/plan.md":                   plan,
		})
		r.Commit("work\n\nChange: "+slug+"\nTask: 1", map[string]string{slug + ".txt": "a\n"})
	}
	head := r.Git("rev-parse", "HEAD")
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+real)
	code, so, se := runArchive(t, r, env, "add-x", "add-y", "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	if !reflect.DeepEqual(res["stopped"], map[string]any{"change": "add-y", "step": "base"}) {
		t.Fatalf("result %v\n%s", res, so)
	}
	arch := res["archived"].([]any)
	if len(arch) != 1 || arch[0].(map[string]any)["change"] != "add-x" {
		t.Fatalf("archived %v", arch)
	}
	ev := arch[0].(map[string]any)["evidence"].(string)
	if _, err := os.Stat(filepath.Join(r.Dir, ev)); err != nil {
		t.Fatal(err)
	}
	spec, _ := os.ReadFile(filepath.Join(r.Dir, "openspec/specs/booking/spec.md"))
	if !strings.Contains(string(spec), "it is cancelled at once") {
		t.Fatalf("specs not merged:\n%s", spec)
	}
	if title, _ := os.ReadFile(filepath.Join(r.Dir, "openspec/specs/booking/docket.yaml")); string(title) != "title: Booking\n" {
		t.Fatalf("spec title = %q", title)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "openspec/changes/add-y/plan.md")); err != nil {
		t.Fatalf("add-y touched: %v", err)
	}
	if got := r.Git("rev-parse", "HEAD"); got != head {
		t.Errorf("HEAD moved")
	}
}

func TestArchiveSchemaNotInstalled(t *testing.T) {
	r := archiveFixtureWith(t, false)
	cmd, log := testutil.FakeOpenSpec(t, "1.14.3")
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd)
	before := r.Git("status", "--porcelain")
	code, so, se := runArchive(t, r, env, "add-x", "--json")
	if code != 1 || !hasRule(t, so, "schema-missing") || !strings.Contains(so, "the docket schema is not installed; run dkt init") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	if !reflect.DeepEqual(res["stopped"], map[string]any{"change": "", "step": "preconditions"}) || res["schema"] != nil {
		t.Fatalf("result = %v", res)
	}
	if b, _ := os.ReadFile(log); len(b) != 0 {
		t.Errorf("openspec was run: %q", b)
	}
	if after := r.Git("status", "--porcelain"); after != before {
		t.Errorf("working tree changed: %q -> %q", before, after)
	}
}

func TestArchiveSchemaDiffers(t *testing.T) {
	r := archiveFixtureWith(t, false)
	testutil.InstallDataSchema(t, dataHome(r.Dir))
	editMachineSchema(t, r.Dir)
	code, so, se := runArchive(t, r, archiveEnv(t, r), "add-x")
	if code != 0 || !strings.Contains(so, "archived add-x as 2026-01-02-add-x") ||
		!strings.Contains(so, "differs from the shipped one; dkt init --force overwrites it") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestArchiveSchemaInResult(t *testing.T) {
	r := archiveFixtureWith(t, false)
	testutil.InstallDataSchema(t, dataHome(r.Dir))
	editMachineSchema(t, r.Dir)
	code, so, se := runArchive(t, r, archiveEnv(t, r), "add-x", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	s := schemaOf(t, so)
	samePath(t, s["path"].(string), filepath.Join(dataHome(r.Dir), "openspec/schemas/docket"))
	if s["matches"] != false {
		t.Errorf("schema = %v", s)
	}
}

func TestArchiveRelativeDataHomeWithCopy(t *testing.T) {
	r := archiveFixture(t)
	env := append(archiveEnv(t, r), "XDG_DATA_HOME=data")
	code, so, se := runArchive(t, r, env, "add-x", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	if a, _ := res["archived"].([]any); len(a) != 1 {
		t.Fatalf("result %v", res)
	}
}

func TestArchiveRelativeDataHome(t *testing.T) {
	r := archiveFixtureWith(t, false)
	cmd, log := testutil.FakeOpenSpec(t, "1.14.3")
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd)
	before := r.Git("status", "--porcelain")
	var so, se bytes.Buffer
	env = append(env, "XDG_DATA_HOME=data")
	code := Run(Options{Args: []string{"archive", "add-x"}, Dir: r.Dir, Env: env, Stdout: &so, Stderr: &se})
	if code != 4 {
		t.Fatalf("exit %d\n%s\n%s", code, so.String(), se.String())
	}
	if b, _ := os.ReadFile(log); len(b) != 0 {
		t.Errorf("openspec was run: %q", b)
	}
	if after := r.Git("status", "--porcelain"); after != before {
		t.Errorf("working tree changed: %q -> %q", before, after)
	}
}
