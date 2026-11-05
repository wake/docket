package lint

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	docket "github.com/wake/docket"
	"github.com/wake/docket/internal/openspec"
)

type stubOpenSpec struct{}

func (stubOpenSpec) ShowChange(string) ([]openspec.Delta, error)     { return nil, nil }
func (stubOpenSpec) ShowSpec(string) ([]openspec.Requirement, error) { return nil, nil }
func (stubOpenSpec) Validate(string) ([]openspec.Issue, error)       { return nil, nil }

const goodFront = "---\ntitle: X\nkind: feature\nsource: s\n---\n\n"
const goodSections = "# Proposal\n\n## Why\n\nw\n\n## What Changes\n\nc\n\n## Non-Goals\n\n- n\n\n## Capabilities\n\nc\n\n## Impact\n\ni\n"
const goodProposal = goodFront + goodSections

// repo writes files under a temporary directory and returns an Input over it
// and a counter of OpenSpec constructions.
func repo(t *testing.T, files map[string]string) (Input, *int) {
	t.Helper()
	dir := t.TempDir()
	for p, c := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return open(t, dir)
}

func open(t *testing.T, dir string) (Input, *int) {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	tpl, err := fs.Sub(docket.Shipped, "shipped/schema")
	if err != nil {
		t.Fatal(err)
	}
	calls := new(int)
	return Input{Root: root.FS(), Templates: tpl, OpenSpec: func() (OpenSpec, error) {
		*calls++
		return stubOpenSpec{}, nil
	}}, calls
}

func run(t *testing.T, in Input) Report {
	t.Helper()
	rep, err := Run(in)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

// has reports whether a finding matches rule, file, line (-1 for any) and a part of the message.
func has(rep Report, rule, file string, line int, msg string) bool {
	for _, f := range rep.Findings {
		if f.Rule == rule && f.File == file && (line < 0 || f.Line == line) && strings.Contains(f.Message, msg) {
			return true
		}
	}
	return false
}

func count(rep Report, rule string) int {
	n := 0
	for _, f := range rep.Findings {
		if f.Rule == rule {
			n++
		}
	}
	return n
}

func skipRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("running as root")
	}
}

func TestEveryActiveChangeAndSpecs(t *testing.T) {
	in, calls := repo(t, map[string]string{
		"openspec/changes/add-y/proposal.md":        goodProposal,
		"openspec/changes/add-x/proposal.md":        goodProposal,
		"openspec/changes/archive/old/proposal.md":  "junk",
		"openspec/changes/add-x/.openspec.yaml":     "schema: docket\n",
		"openspec/changes/other/.openspec.yaml":     "schema: spec-driven\n",
		"openspec/specs/booking/spec.md":            "# Booking Specification\n",
		"openspec/specs/identity/user-auth/spec.md": "# User Auth Specification\n",
		"openspec/changes/README.md":                "not a dir",
	})
	rep := run(t, in)
	want := []Linted{{"add-x", "docket"}, {"add-y", "docket"}, {"other", "spec-driven"}}
	if !reflect.DeepEqual(rep.Changes, want) {
		t.Errorf("Changes = %v, want %v", rep.Changes, want)
	}
	if rep.Specs != 2 {
		t.Errorf("Specs = %d, want 2", rep.Specs)
	}
	if *calls != 1 {
		t.Errorf("OpenSpec called %d times, want 1", *calls)
	}
	if len(rep.Findings) != 0 {
		t.Errorf("findings: %v", rep.Findings)
	}
}

func TestNoChangesDirectory(t *testing.T) {
	in, _ := repo(t, map[string]string{"openspec/specs/a/spec.md": "# A Specification\n"})
	rep := run(t, in)
	if len(rep.Changes) != 0 || len(rep.Unreadable) != 0 || rep.Specs != 1 {
		t.Errorf("%+v", rep)
	}
}

func TestNamedMissing(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{"openspec/changes/archive", "openspec/changes/real", "openspec/specs/a"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("real", filepath.Join(dir, "openspec/changes/linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "openspec/specs/a/spec.md"), []byte("no title\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in, _ := open(t, dir)
	in.Slugs = []string{"add-z", "archive", "linked", "real", "real"}
	rep := run(t, in)
	if got := count(rep, "change-missing"); got != 3 {
		t.Errorf("change-missing = %d: %v", got, rep.Findings)
	}
	for _, name := range []string{"add-z", "archive", "linked"} {
		ok := false
		for _, f := range rep.Findings {
			if f.Rule == "change-missing" && f.Change == "" && strings.Contains(f.Message, name) {
				ok = true
			}
		}
		if !ok {
			t.Errorf("no change-missing naming %s", name)
		}
	}
	if len(rep.Changes) != 1 || rep.Changes[0].Slug != "real" {
		t.Errorf("Changes = %v", rep.Changes)
	}
	if rep.Specs != 0 || has(rep, "title-missing", "openspec/specs/a/spec.md", -1, "") {
		t.Errorf("main specs read when a change is named: %+v", rep)
	}
}

func TestOpenSpecFailureReturnsNothing(t *testing.T) {
	in, _ := repo(t, map[string]string{"openspec/changes/a/proposal.md": "x"})
	boom := errors.New("boom")
	in.OpenSpec = func() (OpenSpec, error) { return nil, boom }
	rep, err := Run(in)
	if err != boom {
		t.Errorf("err = %v", err)
	}
	if !reflect.DeepEqual(rep, Report{}) {
		t.Errorf("report = %+v", rep)
	}
}

func TestOtherSchemaNeverNeedsOpenSpec(t *testing.T) {
	in, calls := repo(t, map[string]string{
		"openspec/changes/a/.openspec.yaml":        "schema: spec-driven\n",
		"openspec/changes/a/specs/booking/spec.md": "## ADDED Requirements\n",
		"openspec/changes/a/proposal.md":           "no front matter and no sections\n",
		"openspec/changes/a/notes.md":              "<!-- Explain the motivation for this change. What problem does this solve? Why now? -->\n",
	})
	in.OpenSpec = func() (OpenSpec, error) { *calls++; return nil, errors.New("must not be called") }
	rep := run(t, in)
	if *calls != 0 {
		t.Error("OpenSpec called")
	}
	if len(rep.Findings) != 1 || rep.Findings[0].Rule != "title-missing" || !strings.Contains(rep.Findings[0].Message, "booking") {
		t.Errorf("findings: %v", rep.Findings)
	}
}

func TestUnreadable(t *testing.T) {
	skipRoot(t)
	dir := t.TempDir()
	write := func(p, c string, mode os.FileMode) {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(full, mode); err != nil {
			t.Fatal(err)
		}
	}
	write("openspec/changes/a/proposal.md", "x", 0)
	write("openspec/changes/a/design.md", "d", 0o644)
	write("openspec/changes/a/.openspec.yaml", "schema: spec-driven\n", 0)
	write("openspec/changes/b/proposal.md", goodProposal, 0o644)
	locked := filepath.Join(dir, "openspec/changes/c/locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	in, _ := open(t, dir)
	rep := run(t, in)
	srcs := map[string]bool{}
	for _, u := range rep.Unreadable {
		srcs[u.Source] = true
	}
	for _, s := range []string{"openspec/changes/a/proposal.md", "openspec/changes/a/.openspec.yaml", "openspec/changes/c/locked"} {
		if !srcs[s] {
			t.Errorf("%s not unreadable: %v", s, rep.Unreadable)
		}
	}
	want := []Linted{{"a", "docket"}, {"b", "docket"}, {"c", "docket"}}
	if !reflect.DeepEqual(rep.Changes, want) {
		t.Errorf("Changes = %v", rep.Changes)
	}
	for _, f := range rep.Findings {
		if f.Change == "a" {
			t.Errorf("finding for a: %v", f)
		}
	}
}

func TestFindingsSorted(t *testing.T) {
	in, _ := repo(t, map[string]string{
		"openspec/changes/b/proposal.md": "no front\n",
		"openspec/changes/a/proposal.md": goodFront + "# Proposal\n\n## Why\n\n<!-- Explain the motivation for this change. What problem does this solve? Why now? -->\n",
		"openspec/specs/z/spec.md":       "bad\n",
		"openspec/specs/y/spec.md":       "bad\n",
	})
	rep := run(t, in)
	if len(rep.Findings) < 5 {
		t.Fatalf("too few findings: %v", rep.Findings)
	}
	for i := 1; i < len(rep.Findings); i++ {
		a, b := rep.Findings[i-1], rep.Findings[i]
		bad := false
		switch {
		case a.Change != b.Change:
			bad = a.Change > b.Change
		case a.File != b.File:
			bad = a.File > b.File
		case a.Line != b.Line:
			bad = a.Line > b.Line
		case a.Rule != b.Rule:
			bad = a.Rule > b.Rule
		default:
			bad = a.Message > b.Message
		}
		if bad {
			t.Errorf("unsorted: %v before %v", a, b)
		}
	}
}
