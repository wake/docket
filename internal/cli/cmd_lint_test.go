package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

type lintOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     *lintResult
}

// TestLintCleanChangeNoRecords: a clean change has an empty, not null, records list.
func TestLintCleanChangeNoRecords(t *testing.T) {
	r := lintRepo(t, "spec-driven")
	_, _, raw, _ := doLint(t, r.Dir, r.Env, "add-x")
	if !strings.Contains(raw, `"records":[]`) || !strings.Contains(raw, `"issues":[]`) {
		t.Fatalf("%s", raw)
	}
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
	testutil.OpenSpecAnswer(t, dir, 0, `{"deltas":[{"spec":"x-cap","operation":"ADDED","requirements":[{"name":"X","text":"The system SHALL x.","scenarios":[]}]}]}`,
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
	if code != 1 || lines[0] != "Linted 1 changes, 0 records, 0 issues and 0 main spec titles." || !strings.Contains(out, "change-missing") {
		t.Fatalf("%d %q", code, out)
	}
}

// routingRepo is a repository whose main holds one commit, with a branch feat checked out.
func routingRepo(t *testing.T, files map[string]string) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	r.Commit("base", files)
	r.Git("checkout", "-q", "-b", "feat")
	return r
}

func routingFiles(m lintOut) string {
	var s []string
	for _, f := range m.Findings {
		if f.Rule == "routing" {
			s = append(s, f.File)
		}
	}
	return strings.Join(s, ",")
}

func defaultBranch(m lintOut) string {
	if m.Result == nil || m.Result.DefaultBranch == nil {
		return "<null>"
	}
	return *m.Result.DefaultBranch
}

func TestLintRoutingUncommittedPlan(t *testing.T) { // Plan written to superpowers' folder
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Write("openspec/changes/add-x/.openspec.yaml", "schema: spec-driven\ncreated: 2026-01-01\n")
	r.Write("docs/superpowers/plans/2026-11-06-x.md", "p\n")
	code, m, raw, _ := doLint(t, r.Dir, r.Env, "add-x")
	if code != 1 || routingFiles(m) != "docs/superpowers/plans/2026-11-06-x.md" || defaultBranch(m) != "main" {
		t.Fatalf("%d %s", code, raw)
	}
	f := m.Findings[0]
	if f.Change != "" || f.Line != 0 {
		t.Fatalf("%+v", f)
	}
}

func TestLintRoutingCommittedStagedUntracked(t *testing.T) {
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Commit("c", map[string]string{"docs/superpowers/specs/committed.md": "a\n"})
	r.Write("docs/superpowers/specs/staged.md", "b\n")
	r.Git("add", "docs/superpowers/specs/staged.md")
	r.Write("docs/superpowers/plans/untracked.md", "c\n")
	r.Write("docs/superpowers/other/skipped.md", "d\n")
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	want := "docs/superpowers/plans/untracked.md,docs/superpowers/specs/committed.md,docs/superpowers/specs/staged.md"
	if code != 1 || routingFiles(m) != want {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintRoutingFileOnDefaultBranch(t *testing.T) { // File already on the default branch
	r := routingRepo(t, map[string]string{"docs/superpowers/specs/old.md": "old\n"})
	r.Commit("edit", map[string]string{"docs/superpowers/specs/old.md": "new\n"})
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || routingFiles(m) != "" || defaultBranch(m) != "main" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintRoutingIgnoredFile(t *testing.T) { // Ignored file
	r := routingRepo(t, map[string]string{".gitignore": "docs/superpowers/plans/x.md\n"})
	r.Write("docs/superpowers/plans/x.md", "p\n")
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || routingFiles(m) != "" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintRoutingNothingAdded(t *testing.T) { // Clean change: default branch main
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || len(m.Findings) != 0 || defaultBranch(m) != "main" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintRoutingNoDefaultBranch(t *testing.T) { // Routing without a default branch
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "work")
	r.Commit("c", map[string]string{"docs/superpowers/plans/x.md": "p\n"})
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || routingFiles(m) != "" || defaultBranch(m) != "<null>" || !strings.Contains(raw, `"default_branch":null`) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintRoutingUnrelatedHistories(t *testing.T) {
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Git("checkout", "-q", "--orphan", "other")
	r.Commit("other", map[string]string{"docs/superpowers/plans/x.md": "p\n"})
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || routingFiles(m) != "" || defaultBranch(m) != "<null>" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintRoutingUnbornHead(t *testing.T) {
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Git("update-ref", "refs/remotes/origin/main", r.Git("rev-parse", "main"))
	r.Git("checkout", "-q", "--orphan", "fresh")
	r.Git("branch", "-D", "main", "feat")
	r.Write("docs/superpowers/plans/x.md", "p\n")
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || routingFiles(m) != "" || defaultBranch(m) != "<null>" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintRoutingText(t *testing.T) {
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.0")
	text := func(r *testutil.Repo) string {
		env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd)
		_, out, _ := runIn(t, r.Dir, env, "lint")
		return out
	}
	none := testutil.NewRepo(t)
	if out := text(none); !strings.Contains(out, "Linted 0 changes, 0 records, 0 issues and 0 main spec titles.\nNo default branch found; the routing rule and the record and issue checks were skipped.\n") {
		t.Fatalf("%q", out)
	}
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Git("checkout", "-q", "--orphan", "other")
	r.Commit("other", nil)
	if out := text(r); !strings.Contains(out, "Linted 0 changes, 0 records, 0 issues and 0 main spec titles.\nNo merge base with main; the routing rule and the record and issue checks were skipped.\n") {
		t.Fatalf("%q", out)
	}
	ok := routingRepo(t, map[string]string{"README.md": "x\n"})
	if out := text(ok); strings.Contains(out, "skipped") {
		t.Fatalf("%q", out)
	}
}

const (
	lintRecSlug = "2026-11-06-fix-x"
	lintRecPath = "docket/records/" + lintRecSlug + ".md"
	lintRecBody = "## What\nw\n\n## Why\ny\n\n## Verification\nv\n\n## Review\nr\n"
	lintRecGood = "---\ntitle: T\nkind: fix\nsource: s\n---\n\n" + lintRecBody
	lintRecBad  = "---\ntitle: T\nkind: chore\nsource: s\n---\n\n" + lintRecBody
)

func lintRecords(m lintOut) string {
	if m.Result == nil {
		return "<nil>"
	}
	return strings.Join(m.Result.Records, ",")
}

func recordFindings(m lintOut) int {
	n := 0
	for _, f := range m.Findings {
		if strings.HasPrefix(f.File, "docket/records/") {
			n++
		}
	}
	return n
}

func TestLintRecordTheBranchAdds(t *testing.T) { // Record the branch adds
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Write(lintRecPath, lintRecBad)
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != "record-kind" || m.Findings[0].File != lintRecPath ||
		m.Findings[0].Change != "" || lintRecords(m) != lintRecSlug {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintMergedRecordLeftAlone(t *testing.T) {
	r := routingRepo(t, map[string]string{lintRecPath: lintRecBad})
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || len(m.Findings) != 0 || lintRecords(m) != "" || !strings.Contains(raw, `"records":[]`) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintRecordsNoDefaultBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Write(lintRecPath, lintRecBad)
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || recordFindings(m) != 0 || !strings.Contains(raw, `"records":[]`) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintNamedChangeSkipsRecords(t *testing.T) {
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Write("openspec/changes/add-x/.openspec.yaml", "schema: spec-driven\ncreated: 2026-01-01\n")
	r.Write(lintRecPath, lintRecBad)
	code, m, raw, _ := doLint(t, r.Dir, r.Env, "add-x")
	if code != 0 || recordFindings(m) != 0 || !strings.Contains(raw, `"records":[]`) {
		t.Fatalf("%d %s", code, raw)
	}
}

// lockRecordsFolder makes docket/records unreadable until the test ends.
func lockRecordsFolder(t *testing.T, dir string) {
	t.Helper()
	folder := filepath.Join(dir, "docket/records")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(folder, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(folder, 0o755) })
}

func TestLintNamedChangeIgnoresRecordsFolder(t *testing.T) {
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Write("openspec/changes/add-x/.openspec.yaml", "schema: spec-driven\ncreated: 2026-01-01\n")
	lockRecordsFolder(t, r.Dir)
	if code, _, raw, _ := doLint(t, r.Dir, r.Env, "add-x"); code != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintUnreadableRecordsFolder(t *testing.T) {
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	lockRecordsFolder(t, r.Dir)
	code, m, raw, _ := doLint(t, r.Dir, r.Env, lintRecSlug)
	if code != 3 || len(m.Findings) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintRecordsFolderIsFile(t *testing.T) {
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Write("docket/records", "a file where the folder would be\n")
	code, m, raw, _ := doLint(t, r.Dir, r.Env, lintRecSlug)
	if code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != "record-missing" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintResultRecords(t *testing.T) { // Records in the result
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Write(lintRecPath, lintRecGood)
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || len(m.Findings) != 0 || !strings.Contains(raw, `"records":["`+lintRecSlug+`"]`) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintNamedRecord(t *testing.T) {
	r := routingRepo(t, map[string]string{lintRecPath: lintRecBad})
	code, m, raw, _ := doLint(t, r.Dir, r.Env, lintRecSlug)
	if code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != "record-kind" || len(m.Result.Changes) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
	code, m, raw, _ = doLint(t, r.Dir, r.Env, "2026-11-06-fix-q")
	if code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != "record-missing" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestRecordRoundTrip(t *testing.T) { // dkt record, then dkt lint
	r := recordRepo(t)
	if code, _, raw := doRecord(t, r, r.Dir, recArgs...); code != 0 {
		t.Fatalf("%d %s", code, raw)
	}
	const slug = "2026-10-30-fix-x"
	code, m, raw, _ := doLint(t, r.Dir, r.Env, slug)
	if code != 1 || !hasRuleIn(m.Findings, "placeholder") || !hasRuleIn(m.Findings, "record-section") {
		t.Fatalf("%d %s", code, raw)
	}
	path := filepath.Join(r.Dir, "docket", "records", slug+".md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// The first comment is the template's note above the sections: drop it, fill the rest.
	comment := regexp.MustCompile(`(?s)<!--.*?-->`)
	first := true
	filled := comment.ReplaceAllStringFunc(string(b), func(string) string {
		if first {
			first = false
			return ""
		}
		return "A sentence."
	})
	if err := os.WriteFile(path, []byte(filled), 0o644); err != nil {
		t.Fatal(err)
	}
	code, m, raw, _ = doLint(t, r.Dir, r.Env, slug)
	if code != 0 || len(m.Findings) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}

// lintTrailerRepo is a repository whose main holds onMain, with a branch feat whose one commit carries the Record trailer value.
func lintTrailerRepo(t *testing.T, onMain map[string]string, value string) *testutil.Repo {
	t.Helper()
	r := routingRepo(t, onMain)
	trailRec(r, "Record", value)
	return r
}

func TestLintTrailerNamesNoRecord(t *testing.T) { // Trailer names no record
	r := lintTrailerRepo(t, map[string]string{lintRecPath: lintRecGood}, "fix-x")
	sha := r.Git("rev-parse", "HEAD")
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 1 || ruleList(m.Findings) != "record-trailer" || m.Findings[0].Change != "" || m.Findings[0].File != "" ||
		!strings.Contains(m.Findings[0].Message, sha[:7]) || !strings.Contains(m.Findings[0].Message, "fix-x") {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintTrailerWithUnreadableFolder(t *testing.T) {
	r := lintTrailerRepo(t, map[string]string{"README.md": "y\n"}, "fix-x")
	lockRecordsFolder(t, r.Dir)
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 3 || ruleList(m.Findings) != "" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintTrailerNamesARecord(t *testing.T) { // Trailer names a record
	r := lintTrailerRepo(t, map[string]string{lintRecPath: lintRecGood}, lintRecSlug)
	trailRec(r, "Record", lintRecSlug+"/2")
	if code, _, raw, _ := doLint(t, r.Dir, r.Env); code != 0 {
		t.Fatal(raw)
	}
	trailRec(r, "Record", lintRecSlug+"/x") // Trailer with a bad task list
	if code, m, raw, _ := doLint(t, r.Dir, r.Env); code != 1 || ruleList(m.Findings) != "record-trailer" {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintTrailerNamesEarlierRecord(t *testing.T) {
	r := lintTrailerRepo(t, map[string]string{"docket/records/2026-10-01-fix-a.md": lintRecGood}, "2026-10-01-fix-a")
	if code, m, raw, _ := doLint(t, r.Dir, r.Env); code != 0 || len(m.Findings) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintMergedTrailerIgnored(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "main")
	r.Commit("base\n\nRecord: gone", map[string]string{"README.md": "x\n"})
	r.Git("checkout", "-q", "-b", "feat")
	if code, m, raw, _ := doLint(t, r.Dir, r.Env); code != 0 || len(m.Findings) != 0 {
		t.Fatalf("%d %s %v", code, raw, m.Findings)
	}
}

func TestLintNamedSkipsTrailers(t *testing.T) {
	r := lintTrailerRepo(t, map[string]string{lintRecPath: lintRecGood}, "fix-x")
	if code, m, raw, _ := doLint(t, r.Dir, r.Env, lintRecSlug); code != 0 || len(m.Findings) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}

const (
	lintIssSlug = "2026-11-06-lint-timeout"
	lintIssPath = "docket/issues/" + lintIssSlug + ".md"
	lintIssBody = "## Current\nc\n\n## Expected\ne\n\n## Unchanged\nu\n\n## Evidence\nv\n"
	lintIssBad  = "---\ntitle: T\n---\n\n" + lintIssBody
)

func TestLintIssueTheBranchAdds(t *testing.T) { // Issue the branch adds
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Write(lintIssPath, lintIssBad)
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 1 || ruleList(m.Findings) != "issue-front-matter" || m.Findings[0].File != lintIssPath ||
		strings.Join(m.Result.Issues, ",") != lintIssSlug || len(m.Result.Records) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintMergedIssueLeftAlone(t *testing.T) {
	r := routingRepo(t, map[string]string{lintIssPath: lintIssBad})
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || len(m.Findings) != 0 || !strings.Contains(raw, `"issues":[]`) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintResultIssues(t *testing.T) { // Issues in the result
	r := routingRepo(t, map[string]string{"README.md": "x\n"})
	r.Write(lintIssPath, "---\ntitle: T\nsource: s\n---\n\n"+lintIssBody)
	code, m, raw, _ := doLint(t, r.Dir, r.Env)
	if code != 0 || len(m.Findings) != 0 || !strings.Contains(raw, `"issues":["`+lintIssSlug+`"]`) {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestLintNamedIssue(t *testing.T) {
	r := routingRepo(t, map[string]string{lintIssPath: lintIssBad})
	code, m, raw, _ := doLint(t, r.Dir, r.Env, lintIssSlug)
	if code != 1 || ruleList(m.Findings) != "issue-front-matter" || len(m.Result.Changes) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}

func TestIssueRoundTripQuoting(t *testing.T) { // dkt issue with quoted text, then dkt lint
	r := recordRepo(t)
	const text = `a "q" \ é 日本`
	if code, _, raw := doIssue(t, r, r.Dir, "lint-timeout", "--title", text, "--source", text); code != 0 {
		t.Fatalf("%d %s", code, raw)
	}
	const slug = "2026-10-30-lint-timeout"
	path := filepath.Join(r.Dir, "docket", "issues", slug+".md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	comment := regexp.MustCompile(`(?s)<!--.*?-->`)
	first := true
	filled := comment.ReplaceAllStringFunc(string(b), func(string) string {
		if first {
			first = false
			return ""
		}
		return "A sentence."
	})
	if err := os.WriteFile(path, []byte(filled), 0o644); err != nil {
		t.Fatal(err)
	}
	code, m, raw, _ := doLint(t, r.Dir, r.Env, slug)
	if code != 0 || len(m.Findings) != 0 {
		t.Fatalf("%d %s", code, raw)
	}
}
