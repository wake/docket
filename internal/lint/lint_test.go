package lint

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wake/docket"
)

const validProposal = `# Proposal

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

func repoWith(files map[string]string) fstest.MapFS {
	m := fstest.MapFS{}
	for p, s := range files {
		m[p] = &fstest.MapFile{Data: []byte(s)}
	}
	return m
}

func runOn(repo fstest.MapFS, slugs ...string) Report {
	return Run(Input{Repo: repo, Templates: docket.Schema(), Slugs: slugs})
}

func rules(r Report) []string {
	var out []string
	for _, f := range r.Findings {
		out = append(out, f.Rule)
	}
	return out
}

func wantOne(t *testing.T, r Report, rule string, line int) Finding {
	t.Helper()
	if len(r.Findings) != 1 {
		t.Fatalf("findings = %+v, want exactly one %s", r.Findings, rule)
	}
	f := r.Findings[0]
	if f.Rule != rule || f.Line != line {
		t.Fatalf("finding = %+v, want rule %s line %d", f, rule, line)
	}
	return f
}

func TestLintEveryActiveChange(t *testing.T) {
	repo := repoWith(map[string]string{
		"openspec/changes/add-y/proposal.md":                    validProposal,
		"openspec/changes/add-y/docket.yaml":                    "title: Y\n",
		"openspec/changes/add-x/proposal.md":                    validProposal,
		"openspec/changes/add-x/docket.yaml":                    "title: X\n",
		"openspec/changes/archive/2026-01-01-add-z/proposal.md": "junk",
	})
	r := runOn(repo)
	if want := []string{"add-x", "add-y"}; !reflect.DeepEqual(r.Changes, want) {
		t.Fatalf("Changes = %v, want %v", r.Changes, want)
	}
	if len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestNoChangesDirectory(t *testing.T) {
	r := runOn(repoWith(map[string]string{"README.md": "x"}))
	if len(r.Changes) != 0 || len(r.Findings) != 0 || len(r.Unreadable) != 0 {
		t.Fatalf("report = %+v", r)
	}
}

func TestNamedChangeMissing(t *testing.T) {
	repo := repoWith(map[string]string{"openspec/changes/add-x/proposal.md": validProposal, "openspec/changes/add-x/docket.yaml": "title: X\n"})
	for _, slug := range []string{"add-z", "../x", "a/b", "archive"} {
		r := runOn(repo, slug)
		f := wantOne(t, r, "change-missing", 0)
		if f.Change != "" || !strings.Contains(f.Message, slug) {
			t.Errorf("%q: finding = %+v", slug, f)
		}
		if len(r.Changes) != 0 {
			t.Errorf("%q: Changes = %v", slug, r.Changes)
		}
	}
}

func TestNamedChangeLintsOnlyThat(t *testing.T) {
	repo := repoWith(map[string]string{
		"openspec/changes/add-x/proposal.md": validProposal,
		"openspec/changes/add-x/docket.yaml": "title: X\n",
		"openspec/changes/add-y/proposal.md": "",
	})
	r := runOn(repo, "add-x")
	if !reflect.DeepEqual(r.Changes, []string{"add-x"}) || len(r.Findings) != 0 {
		t.Fatalf("report = %+v", r)
	}
}

func TestValidProposalIsClean(t *testing.T) {
	r := runOn(repoWith(map[string]string{"openspec/changes/add-x/proposal.md": validProposal, "openspec/changes/add-x/docket.yaml": "title: X\n"}))
	if len(r.Findings) != 0 || len(r.Unreadable) != 0 {
		t.Fatalf("report = %+v", r)
	}
}

func TestPlanNotWrittenYet(t *testing.T) {
	repo := repoWith(map[string]string{"openspec/changes/add-x/proposal.md": validProposal, "openspec/changes/add-x/docket.yaml": "title: X\n"})
	for _, f := range runOn(repo).Findings {
		t.Errorf("unexpected %+v", f)
	}
}

func TestCRLFProposal(t *testing.T) {
	crlf := "\xef\xbb\xbf" + strings.ReplaceAll(validProposal, "\n", "\r\n")
	r := runOn(repoWith(map[string]string{"openspec/changes/add-x/proposal.md": crlf, "openspec/changes/add-x/docket.yaml": "title: X\n"}))
	if len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
	broken := strings.Replace(crlf, "| 2026-10-24 | review", "| 2026-10-24 |  ", 1)
	want := strings.Count(validProposal[:strings.Index(validProposal, "| 2026-10-24")], "\n") + 1
	wantOne(t, runOn(repoWith(map[string]string{"openspec/changes/add-x/proposal.md": broken, "openspec/changes/add-x/docket.yaml": "title: X\n"})), "change-log", want)
}

func osRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs permission bits and a non-root user")
	}
	root := t.TempDir()
	for p, s := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestUnreadableFile(t *testing.T) {
	root := osRepo(t, map[string]string{
		"openspec/changes/add-x/proposal.md": validProposal,
		"openspec/changes/add-x/docket.yaml": "title: X\n",
		"openspec/changes/add-x/plan.md":     "# Plan\n",
		"openspec/changes/add-y/docket.yaml": "title: Y\n",
		"openspec/changes/add-y/proposal.md": strings.Replace(validProposal, "## Non-Goals", "## Other", 1),
	})
	plan := filepath.Join(root, "openspec/changes/add-x/plan.md")
	if err := os.Chmod(plan, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(plan, 0o644) })
	r := Run(Input{Repo: os.DirFS(root), Templates: docket.Schema()})
	if len(r.Unreadable) != 1 || r.Unreadable[0].Source != "openspec/changes/add-x/plan.md" {
		t.Fatalf("Unreadable = %+v", r.Unreadable)
	}
	if !reflect.DeepEqual(r.Changes, []string{"add-x", "add-y"}) {
		t.Fatalf("Changes = %v", r.Changes)
	}
	if got := rules(r); !reflect.DeepEqual(got, []string{"proposal-section"}) || r.Findings[0].Change != "add-y" {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestUnreadableChangeDirectory(t *testing.T) {
	root := osRepo(t, map[string]string{
		"openspec/changes/add-x/proposal.md": validProposal,
		"openspec/changes/add-y/proposal.md": validProposal,
		"openspec/changes/add-y/docket.yaml": "title: Y\n",
	})
	dir := filepath.Join(root, "openspec/changes/add-x")
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	r := Run(Input{Repo: os.DirFS(root), Templates: docket.Schema()})
	if len(r.Unreadable) != 1 || r.Unreadable[0].Source != "openspec/changes/add-x" {
		t.Fatalf("Unreadable = %+v", r.Unreadable)
	}
	if !reflect.DeepEqual(r.Changes, []string{"add-y"}) || len(r.Findings) != 0 {
		t.Fatalf("report = %+v", r)
	}
}
