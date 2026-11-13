package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pf = "openspec/changes/add-x/proposal.md"

func lintX(t *testing.T, files map[string]string) Report {
	t.Helper()
	m := map[string]string{}
	for p, c := range files {
		m["openspec/changes/add-x/"+p] = c
	}
	in, _ := repo(t, m)
	in.Slugs = []string{"add-x"}
	return run(t, in)
}

func TestFrontMatterRules(t *testing.T) {
	type w struct {
		rule string
		line int
		msg  string
	}
	tests := []struct {
		name  string
		front string
		want  []w
		rules map[string]int
	}{
		{"no title", "---\nkind: feature\nsource: s\n---\n", []w{{"title-missing", 0, ""}}, map[string]int{"title-missing": 1, "metadata-invalid": 0}},
		{"blank title", "---\ntitle: \"   \"\nkind: feature\nsource: s\n---\n", []w{{"title-missing", 2, ""}}, map[string]int{"title-missing": 1, "metadata-invalid": 0}},
		{"no kind", "---\ntitle: T\nsource: s\n---\n", []w{{"metadata-invalid", 0, "kind"}}, map[string]int{"title-missing": 0, "metadata-invalid": 1}},
		{"unknown kind", "---\ntitle: T\nkind: fix\nsource: s\n---\n", []w{{"metadata-invalid", 3, "kind"}}, map[string]int{"title-missing": 0, "metadata-invalid": 1}},
		{"no source", "---\ntitle: T\nkind: feature\n---\n", []w{{"metadata-invalid", 0, "source"}}, map[string]int{"title-missing": 0, "metadata-invalid": 1}},
		{"multiline source", "---\ntitle: T\nkind: feature\nsource: \"a\\nb\"\n---\n", []w{{"metadata-invalid", 4, "source"}}, map[string]int{"title-missing": 0, "metadata-invalid": 1}},
		{"error", "---\ntitle: T\ntitle: U\n---\n", []w{{"metadata-invalid", 0, ""}}, map[string]int{"title-missing": 0, "metadata-invalid": 1}},
		{"no front matter", "# Proposal\n", []w{{"title-missing", 0, ""}, {"metadata-invalid", 0, "kind"}, {"metadata-invalid", 0, "source"}}, map[string]int{"title-missing": 1, "metadata-invalid": 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rep := lintX(t, map[string]string{"proposal.md": tc.front + "\n" + goodSections})
			for _, x := range tc.want {
				if !has(rep, x.rule, pf, x.line, x.msg) {
					t.Errorf("missing %+v in %v", x, rep.Findings)
				}
			}
			for r, n := range tc.rules {
				if got := count(rep, r); got != n {
					t.Errorf("%s = %d, want %d: %v", r, got, n, rep.Findings)
				}
			}
		})
	}
}

func TestLintTitleTooLong(t *testing.T) { // Change title too long
	front := "---\ntitle: " + strings.Repeat("a", 121) + "\nkind: feature\nsource: s\n---\n"
	rep := lintX(t, map[string]string{"proposal.md": front + "\n" + goodSections})
	if !has(rep, "title-too-long", pf, 2, "121") || !has(rep, "title-too-long", pf, 2, "120") || count(rep, "title-missing") != 0 {
		t.Errorf("%v", rep.Findings)
	}
}

func TestNewCapabilities(t *testing.T) {
	spec := "## ADDED Requirements\n"
	front := func(specs string) string {
		return "---\ntitle: T\nkind: feature\nsource: s\n" + specs + "---\n\n" + goodSections
	}
	t.Run("unnamed with specs line", func(t *testing.T) {
		rep := lintX(t, map[string]string{"proposal.md": front("specs:\n  other: Other\n"), "specs/booking/spec.md": spec})
		if !has(rep, "title-missing", pf, 5, "booking: <name>") {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("no specs key", func(t *testing.T) {
		rep := lintX(t, map[string]string{"proposal.md": front(""), "specs/booking/spec.md": spec})
		if !has(rep, "title-missing", pf, 0, "booking") {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("blank name", func(t *testing.T) {
		rep := lintX(t, map[string]string{"proposal.md": front("specs:\n  booking: \"  \"\n"), "specs/booking/spec.md": spec})
		if !has(rep, "title-missing", pf, 5, "booking") {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("named", func(t *testing.T) {
		rep := lintX(t, map[string]string{"proposal.md": front("specs:\n  booking: Booking\n"), "specs/booking/spec.md": spec})
		if len(rep.Findings) != 0 {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("nested path", func(t *testing.T) {
		rep := lintX(t, map[string]string{"proposal.md": front(""), "specs/identity/user-auth/spec.md": spec})
		if !has(rep, "title-missing", pf, 0, "identity/user-auth: <name>") {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("existing main spec", func(t *testing.T) {
		in, _ := repo(t, map[string]string{
			pf: goodProposal, "openspec/changes/add-x/specs/booking/spec.md": spec,
			"openspec/specs/booking/spec.md": "# Booking Specification\n",
		})
		in.Slugs = []string{"add-x"}
		if rep := run(t, in); len(rep.Findings) != 0 {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("no proposal", func(t *testing.T) {
		rep := lintX(t, map[string]string{"specs/booking/spec.md": spec})
		if !has(rep, "title-missing", pf, 0, "booking") || count(rep, "proposal-section") != 1 {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("docket front matter error is the only finding", func(t *testing.T) {
		rep := lintX(t, map[string]string{"proposal.md": "---\ntitle: T\nkind: feature\n" + goodSections, "specs/booking/spec.md": spec})
		if len(rep.Findings) != 1 || rep.Findings[0].Rule != "metadata-invalid" {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("other schema front matter error means no names", func(t *testing.T) {
		rep := lintX(t, map[string]string{".openspec.yaml": "schema: spec-driven\n", "proposal.md": "---\ntitle: T\ntitle: U\n---\n" + goodSections, "specs/booking/spec.md": spec})
		if len(rep.Findings) != 1 || !has(rep, "title-missing", pf, 0, "booking") {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("level three Why is not a section", func(t *testing.T) {
		rep := lintX(t, map[string]string{"proposal.md": strings.Replace(goodProposal, "## Why", "### Why", 1)})
		if !has(rep, "proposal-section", pf, 0, "Why") || count(rep, "proposal-section") != 1 {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("symbolic link in a change is not read", func(t *testing.T) {
		dir := t.TempDir()
		for p, c := range map[string]string{
			pf:           goodProposal,
			"outside.md": "<!-- Explain the motivation for this change. What problem does this solve? Why now? -->\n",
		} {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, p), []byte(c), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Symlink("../../../outside.md", filepath.Join(dir, "openspec/changes/add-x/notes.md")); err != nil {
			t.Fatal(err)
		}
		in, _ := open(t, dir)
		in.Slugs = []string{"add-x"}
		if rep := run(t, in); len(rep.Findings) != 0 {
			t.Errorf("%v", rep.Findings)
		}
	})
}

func TestProposalSections(t *testing.T) {
	t.Run("non-goals missing", func(t *testing.T) {
		rep := lintX(t, map[string]string{"proposal.md": goodFront + "# P\n\n## Why\n\nw\n\n## What Changes\n\nc\n\n## Capabilities\n\nc\n\n## Impact\n\ni\n"})
		if !has(rep, "proposal-section", pf, 0, "Non-Goals") || count(rep, "proposal-section") != 1 {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("non-goals only comment", func(t *testing.T) {
		p := strings.Replace(goodProposal, "- n\n", "<!-- keep me out -->\n", 1)
		rep := lintX(t, map[string]string{"proposal.md": p})
		line := 0
		for i, l := range strings.Split(p, "\n") {
			if l == "## Non-Goals" {
				line = i + 1
			}
		}
		if !has(rep, "proposal-section", pf, line, "Non-Goals") {
			t.Errorf("want line %d: %v", line, rep.Findings)
		}
	})
	t.Run("case and trim, heading in fence ignored", func(t *testing.T) {
		p := goodFront + "# P\n\n## why \n\nw\n\n## WHAT CHANGES\n\nc\n\n## non-goals\n\n- n\n\n## Capabilities\n\nc\n\n```\n## Impact\n```\n"
		rep := lintX(t, map[string]string{"proposal.md": p})
		if !has(rep, "proposal-section", pf, 0, "Impact") || count(rep, "proposal-section") != 1 {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("missing proposal", func(t *testing.T) {
		rep := lintX(t, map[string]string{"design.md": "d\n"})
		if count(rep, "proposal-section") != 1 || !has(rep, "proposal-section", pf, 0, "proposal.md") {
			t.Errorf("%v", rep.Findings)
		}
	})
}

func TestPlaceholders(t *testing.T) {
	comment := "<!-- Explain the motivation for this change. What problem does this solve? Why now? -->"
	t.Run("template comment left", func(t *testing.T) {
		p := strings.Replace(goodProposal, "## Why\n\nw", "## Why\n\n"+comment, 1)
		rep := lintX(t, map[string]string{"proposal.md": p})
		if !has(rep, "placeholder", pf, 11, "proposal.md") || count(rep, "placeholder") != 1 {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("angle placeholder, code span, fence, foreign comment, yaml", func(t *testing.T) {
		d := "uses <capability-path> here\n`<capability-path>` in a span\n```\n<capability-path>\n```\n<!-- my own note -->\n"
		rep := lintX(t, map[string]string{
			"proposal.md":  goodProposal,
			"design.md":    d,
			"notes.yaml":   "x: <capability-path>\n",
			"deep/more.md": "<capability-path>\n",
		})
		if !has(rep, "placeholder", "openspec/changes/add-x/design.md", 1, "<capability-path>") || count(rep, "placeholder") != 2 {
			t.Errorf("%v", rep.Findings)
		}
		if !has(rep, "placeholder", "openspec/changes/add-x/deep/more.md", 1, "") {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("angle tag inside a foreign comment", func(t *testing.T) {
		rep := lintX(t, map[string]string{"proposal.md": goodProposal, "design.md": "<!-- see <capability-path> -->\n" + comment + "\n"})
		if count(rep, "placeholder") != 1 || !has(rep, "placeholder", "openspec/changes/add-x/design.md", 2, "Explain") {
			t.Errorf("%v", rep.Findings)
		}
	})
	t.Run("templates unreadable once", func(t *testing.T) {
		in, _ := repo(t, map[string]string{"openspec/changes/a/proposal.md": goodProposal, "openspec/changes/b/proposal.md": goodProposal})
		in.Templates = nil
		rep := run(t, in)
		if len(rep.Unreadable) != 1 {
			t.Errorf("%v", rep.Unreadable)
		}
	})
}

func TestMainSpecTitles(t *testing.T) {
	in, _ := repo(t, map[string]string{
		"openspec/specs/booking/spec.md":       "# booking Specification\n",
		"openspec/specs/waitlist/spec.md":      "# Booking Waitlist Specification\n",
		"openspec/specs/identity/auth/spec.md": "# Auth Specification\n",
		"openspec/specs/identity/other.md":     "ignored\n",
	})
	rep := run(t, in)
	if rep.Specs != 3 {
		t.Errorf("Specs = %d", rep.Specs)
	}
	if !has(rep, "title-missing", "openspec/specs/booking/spec.md", 1, "") || count(rep, "title-missing") != 1 {
		t.Errorf("%v", rep.Findings)
	}
	if rep.Findings[0].Change != "" {
		t.Errorf("%v", rep.Findings)
	}
}

func TestFenceInComment(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		file  string
		line  int
		n     int
	}{
		{"marker left in plan", map[string]string{"proposal.md": goodProposal, "plan.md": "<!--\n```\n-->\n### Task 2: Real\n"}, "openspec/changes/add-x/plan.md", 2, 1},
		{"whole block in design", map[string]string{"proposal.md": goodProposal, "design.md": "<!--\n```\nx\n```\n-->\n"}, "", 0, 0},
		{"comment marker in fenced code", map[string]string{"proposal.md": goodProposal + "\n```\n<!--\n```\n"}, "", 0, 0},
		{"nested file", map[string]string{"proposal.md": goodProposal, "deep/more.md": "<!--\n~~~\n-->\n"}, "openspec/changes/add-x/deep/more.md", 2, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rep := lintX(t, tc.files)
			if got := count(rep, "fence-in-comment"); got != tc.n {
				t.Fatalf("count = %d: %+v", got, rep.Findings)
			}
			if tc.n > 0 && !has(rep, "fence-in-comment", tc.file, tc.line, "") {
				t.Errorf("%+v", rep.Findings)
			}
		})
	}
}

func TestMainSpecFenceInComment(t *testing.T) {
	in, _ := repo(t, map[string]string{
		"openspec/specs/orders/spec.md": "# Orders Specification\n<!--\n```\n-->\n### Requirement: Persist orders\n",
	})
	rep := run(t, in)
	if !has(rep, "fence-in-comment", "openspec/specs/orders/spec.md", 3, "") || count(rep, "fence-in-comment") != 1 {
		t.Errorf("%v", rep.Findings)
	}
	fnd, _, _ := SpecTitles(in.Root)
	for _, f := range fnd {
		if f.Rule == "fence-in-comment" {
			t.Errorf("SpecTitles reported %v", f)
		}
	}
}
