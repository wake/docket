package lint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket"
)

func TestUnreadableMainSpecIsListedOnce(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read a mode 000 file")
	}
	dir := t.TempDir()
	files := map[string]string{
		"openspec/changes/add-x/proposal.md":           validProposal,
		"openspec/changes/add-y/proposal.md":           validProposal,
		deltaAt("booking"):                             bookingDelta,
		"openspec/changes/add-y/specs/booking/spec.md": bookingDelta,
		"openspec/specs/booking/spec.md":               "### Requirement: Join\n\nx\n",
	}
	for p, s := range files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	spec := filepath.Join(dir, "openspec/specs/booking/spec.md")
	if err := os.Chmod(spec, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(spec, 0o644) })
	r := Run(Input{Repo: os.DirFS(dir), Templates: docket.Schema()})
	n := 0
	for _, u := range r.Unreadable {
		if u.Source == "openspec/specs/booking/spec.md" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("unreadable = %+v, want one entry for the main spec", r.Unreadable)
	}
	if len(r.Changes) != 2 {
		t.Fatalf("changes = %v, other rules must still run", r.Changes)
	}
}

func lintDelta(d string) Report {
	return runOn(repoWith(map[string]string{
		"openspec/changes/add-x/proposal.md": validProposal, deltaAt("booking"): d,
	}))
}

func TestRequirementOutsideAnyDeltaSection(t *testing.T) {
	d := "## ADDED Requirements\n\n### Requirement: A\n\nx\n\n#### Scenario: S\n\n- a\n\n## Notes\n\n### Requirement: Stray\n\nx\n"
	r := lintDelta(d)
	for _, f := range r.Findings {
		if f.Rule == "unparseable" && f.Line == lineOf(d, "### Requirement: Stray") && f.File == deltaAt("booking") {
			return
		}
	}
	t.Fatalf("findings = %+v", r.Findings)
}

func TestHeaderInsideACodeBlock(t *testing.T) {
	d := "## ADDED Requirements\n\n### Requirement: A\n\nx\n\n```\n## REMOVED Requirements\n### Requirement: Fake\n```\n\n#### Scenario: S\n\n- a\n"
	if r := lintDelta(d); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestUnpairedRename(t *testing.T) {
	d := "## RENAMED Requirements\n\n- FROM: `### Requirement: Old`\n"
	r := lintDelta(d)
	if len(r.Findings) == 0 || r.Findings[0].Rule != "unparseable" || r.Findings[0].Line != lineOf(d, "- FROM") {
		t.Fatalf("findings = %+v", r.Findings)
	}
}
