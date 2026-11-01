package cli

import (
	"bytes"
	"github.com/wake/docket"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

const lintProposal = `# Proposal

## Why

Because it helps.

## What Changes

A thing changes.

## Non-Goals

- Not this.

## Builds On

- internal/change: ParseProposal

## Capabilities

### New Capabilities
- ` + "`x`" + `: does x

## Impact

Some code.

## Implementation Contract

### Observable Behavior

It works.

### Interfaces

A command.

### Failure Modes

It fails loudly.

### Acceptance Criteria

- x#works

## Change Log

| Date | Source | Change | Impact |
|---|---|---|---|
| 2026-10-24 | review | first | none |
`

func runLint(t *testing.T, dir string, env []string, args ...string) (int, string, string) {
	t.Helper()
	var so, se bytes.Buffer
	code := Run(Options{Args: append([]string{"lint"}, args...), Dir: dir, Env: env, Stdout: &so, Stderr: &se})
	return code, so.String(), se.String()
}

func lintFixture(t *testing.T) *testutil.Repo {
	t.Helper()
	r := cleanRepo(t)
	for _, slug := range []string{"add-x", "add-y"} {
		r.Write("openspec/changes/"+slug+"/.openspec.yaml", "schema: docket\n")
		r.Write("openspec/changes/"+slug+"/proposal.md", lintProposal)
		r.Write("openspec/changes/"+slug+"/docket.yaml", "title: "+slug+"\n")
	}
	return r
}

func sourcesOf(t *testing.T, so string) []string {
	t.Helper()
	var out []string
	us, _ := decode(t, so)["unreadable"].([]any)
	for _, u := range us {
		out = append(out, u.(map[string]any)["source"].(string))
	}
	return out
}

func TestDesignWrittenToTheSuperpowersFolder(t *testing.T) {
	r := lintFixture(t)
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "changes")
	r.Write("docs/superpowers/specs/2026-10-25-x-design.md", "design\n")
	code, so, se := runLint(t, r.Dir, r.Env, "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if !hasRule(t, so, "routing") || !strings.Contains(so, "docs/superpowers/specs/2026-10-25-x-design.md") {
		t.Fatalf("no routing finding:\n%s", so)
	}
}

func TestNoDefaultBranchForRouting(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Git("checkout", "-q", "-b", "topic")
	testutil.InstallSchema(t, r.Dir)
	r.Commit("init", map[string]string{
		"openspec/changes/add-x/.openspec.yaml": "schema: docket\n",
	})
	code, so, se := runLint(t, r.Dir, r.Env, "--json")
	if code != 3 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if !hasRule(t, so, "proposal-section") {
		t.Fatalf("other rules did not run:\n%s", so)
	}
	if got := sourcesOf(t, so); !reflect.DeepEqual(got, []string{"added files"}) {
		t.Fatalf("unreadable = %v\n%s", got, so)
	}
	if !strings.Contains(so, "--default-branch") {
		t.Fatalf("the source needs the hint:\n%s", so)
	}
}

func TestNoOpenSpecInstalled(t *testing.T) {
	r := lintFixture(t)
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "changes")
	var env []string
	for _, kv := range r.Env {
		if strings.HasPrefix(kv, "PATH=") {
			continue
		}
		env = append(env, kv)
	}
	gitDir := filepath.Dir(lookGit(t))
	env = append(env, "PATH="+gitDir)
	code, so, se := runLint(t, r.Dir, env, "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

func lookGit(t *testing.T) string {
	t.Helper()
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		p := filepath.Join(d, "git")
		if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
			return p
		}
	}
	t.Skip("git not found")
	return ""
}

func TestUnreadableFileExitsThree(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads mode-000 files")
	}
	r := lintFixture(t)
	r.Write("openspec/changes/add-x/plan.md", "# plan\n")
	if err := os.Chmod(filepath.Join(r.Dir, "openspec/changes/add-x/plan.md"), 0); err != nil {
		t.Fatal(err)
	}
	r.Write("openspec/changes/add-y/proposal.md", "# Proposal\n")
	code, so, se := runLint(t, r.Dir, r.Env, "--json")
	if code != 3 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if !hasRule(t, so, "proposal-section") {
		t.Fatalf("finding missing:\n%s", so)
	}
	found := false
	for _, s := range sourcesOf(t, so) {
		if strings.HasSuffix(s, "add-x/plan.md") {
			found = true
		}
	}
	if !found {
		t.Fatalf("unreadable missing:\n%s", so)
	}
}

func TestLintEveryActiveChangeJSON(t *testing.T) {
	r := lintFixture(t)
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "changes")
	code, so, se := runLint(t, r.Dir, r.Env, "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	if got := res["changes"]; !reflect.DeepEqual(got, []any{"add-x", "add-y"}) {
		t.Fatalf("changes = %v", got)
	}
	if got := res["records"]; !reflect.DeepEqual(got, []any{}) {
		t.Fatalf("records = %#v, want []", got)
	}
}

func TestLintNamedChanges(t *testing.T) {
	r := lintFixture(t)
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "changes")
	code, so, se := runLint(t, r.Dir, r.Env, "add-x", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	if got := res["changes"]; !reflect.DeepEqual(got, []any{"add-x"}) {
		t.Fatalf("changes = %v", got)
	}
}

func TestLintCleanExit(t *testing.T) {
	r := lintFixture(t)
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "changes")
	code, so, se := runLint(t, r.Dir, r.Env)
	if code != 0 || !strings.HasSuffix(strings.TrimRight(so, "\n"), "dkt lint: ok") {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
}

const lintRecord = "---\ntitle: T\nkind: fix\nsource: s\n---\n\n" +
	"## What\n\na\n\n## Why\n\nb\n\n## Verification\n\nc\n\n## Review\n\nd\n"

func TestLintRecordsTheBranchAdds(t *testing.T) {
	r := cleanRepo(t)
	r.Commit("old record", map[string]string{"docket/records/2026-10-01-fix-a.md": "bad\n"})
	r.Git("checkout", "-q", "-b", "topic")
	r.Commit("record", map[string]string{"docket/records/2026-10-30-fix-b.md": lintRecord})
	r.Write("docket/records/2026-10-30-fix-c.md", lintRecord)
	code, so, se := runLint(t, r.Dir, r.Env, "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	if got := res["records"]; !reflect.DeepEqual(got, []any{"2026-10-30-fix-b", "2026-10-30-fix-c"}) {
		t.Fatalf("records = %v\n%s", got, so)
	}
	if strings.Contains(so, "fix-a") {
		t.Fatalf("an old record was linted:\n%s", so)
	}
}

func TestLintRecordTemplateTextLeft(t *testing.T) {
	tpl := docket.RecordTemplate()
	i := strings.Index(tpl, "## Why\n\n") + len("## Why\n\n")
	comment := tpl[i : i+strings.Index(tpl[i:], "-->")+3]
	rec := strings.Replace(lintRecord, "## Why\n\nb\n", "## Why\n\n"+comment+"\n", 1)
	r := cleanRepo(t)
	r.Git("checkout", "-q", "-b", "topic")
	r.Commit("record", map[string]string{"docket/records/2026-10-30-fix-b.md": rec})
	_, so, _ := runLint(t, r.Dir, r.Env, "--json")
	for _, f := range decode(t, so)["findings"].([]any) {
		m := f.(map[string]any)
		if m["rule"] == "placeholder" && m["file"] == "docket/records/2026-10-30-fix-b.md" {
			return
		}
	}
	t.Fatalf("no placeholder finding:\n%s", so)
}

func TestLintOneUnreadableEntry(t *testing.T) {
	r := lintFixture(t)
	r.Git("branch", "-m", "main", "trunk")
	code, so, se := runLint(t, r.Dir, r.Env, "--json")
	if code != 3 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got := sourcesOf(t, so); !reflect.DeepEqual(got, []string{"added files"}) {
		t.Fatalf("unreadable = %v\n%s", got, so)
	}
}

func TestLintRecordTrailer(t *testing.T) {
	r := cleanRepo(t)
	r.Git("checkout", "-q", "-b", "topic")
	r.Commit("record\n\nRecord: fix-x", map[string]string{"docket/records/2026-11-01-fix-x.md": lintRecord})
	code, so, se := runLint(t, r.Dir, r.Env, "--json")
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if !hasRule(t, so, "record-trailer") {
		t.Fatalf("no record-trailer:\n%s", so)
	}
}

const lintIssue = "---\ntitle: T\nsource: s\n---\n\n" +
	"## Current\n\na\n\n## Expected\n\nb\n\n## Unchanged\n\nc\n\n## Found\n\nd\n"

func TestLintIssuesTheBranchAdds(t *testing.T) {
	r := cleanRepo(t)
	r.Commit("old issue", map[string]string{"docket/issues/2026-10-01-a.md": "bad\n"})
	r.Git("checkout", "-q", "-b", "topic")
	r.Commit("issue", map[string]string{"docket/issues/2026-11-02-b.md": lintIssue})
	code, so, se := runLint(t, r.Dir, r.Env, "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	res := decode(t, so)["result"].(map[string]any)
	if got := res["issues"]; !reflect.DeepEqual(got, []any{"2026-11-02-b"}) {
		t.Fatalf("issues = %v\n%s", got, so)
	}
}

func TestLintIssuesWithoutDefaultBranch(t *testing.T) {
	r := lintFixture(t)
	r.Git("branch", "-m", "main", "trunk")
	r.Write("docket/issues/2026-11-02-b.md", "bad\n")
	code, so, se := runLint(t, r.Dir, r.Env, "--json")
	if code != 3 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if got := sourcesOf(t, so); !reflect.DeepEqual(got, []string{"added files"}) {
		t.Fatalf("unreadable = %v\n%s", got, so)
	}
	if res := decode(t, so)["result"].(map[string]any); !reflect.DeepEqual(res["issues"], []any{}) {
		t.Fatalf("issues = %v", res["issues"])
	}
}
