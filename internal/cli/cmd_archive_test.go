package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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
	code := Run(Options{Args: append([]string{"archive"}, args...), Dir: r.Dir, Env: env, Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

func archiveFixture(t *testing.T) *testutil.Repo {
	t.Helper()
	r := cleanRepo(t)
	r.Git("remote", "add", "origin", archiveOrigin)
	r.Commit("code", map[string]string{"internal/x.go": "package x\n", "openspec/config.yaml": "schema: docket\n"})
	base := r.Git("rev-parse", "HEAD")
	r.Commit("add the change", map[string]string{
		"openspec/changes/add-x/.openspec.yaml":  "schema: docket\nx-base: " + base + "\n",
		"openspec/changes/add-x/proposal.md":     lintProposal,
		"openspec/changes/add-x/specs/x/spec.md": archiveDelta,
		"openspec/changes/add-x/plan.md":         archivePlan,
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

func TestArchiveNeedsOpenSpec(t *testing.T) {
	r := archiveFixture(t)
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC=/nonexistent/openspec")
	code, so, se := runArchive(t, r, env, "add-x")
	if code != 4 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func TestArchiveNeedsASlug(t *testing.T) {
	r := archiveFixture(t)
	code, so, se := runArchive(t, r, archiveEnv(t, r))
	if code != 2 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
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
			"openspec/changes/" + slug + "/.openspec.yaml":        "schema: docket\nx-base: " + base + "\n",
			"openspec/changes/" + slug + "/proposal.md":           strings.Replace(lintProposal, "`x`", "`booking`", 1),
			"openspec/changes/" + slug + "/specs/booking/spec.md": realDelta(then),
			"openspec/changes/" + slug + "/plan.md":               plan,
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
	if _, err := os.Stat(filepath.Join(r.Dir, "openspec/changes/add-y/plan.md")); err != nil {
		t.Fatalf("add-y touched: %v", err)
	}
	if got := r.Git("rev-parse", "HEAD"); got != head {
		t.Errorf("HEAD moved")
	}
}
