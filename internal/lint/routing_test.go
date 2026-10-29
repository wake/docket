package lint

import (
	"errors"
	"testing"

	"github.com/wake/docket"
)

func TestRoutingFinding(t *testing.T) {
	rep := Run(Input{
		Repo:      repoWith(nil),
		Templates: docket.Schema(),
		Added: []string{
			"docs/superpowers/specs/2026-10-25-x-design.md",
			"docs/superpowers/plans/2026-10-25-x.md",
			"docs/other/notes.md",
		},
	})
	if len(rep.Findings) != 2 {
		t.Fatalf("findings = %+v", rep.Findings)
	}
	for _, f := range rep.Findings {
		if f.Rule != "routing" || f.Change != "" || f.File == "" {
			t.Fatalf("finding = %+v", f)
		}
	}
	if rep.Findings[0].File != "docs/superpowers/plans/2026-10-25-x.md" {
		t.Fatalf("findings = %+v", rep.Findings)
	}
}

func TestRoutingUnreadable(t *testing.T) {
	repo := repoWith(map[string]string{"openspec/changes/add-x/.openspec.yaml": "schema: docket\n", "openspec/changes/add-x/docket.yaml": "title: X\n"})
	rep := Run(Input{Repo: repo, Templates: docket.Schema(), AddedErr: errors.New("no default branch")})
	if len(rep.Unreadable) != 1 || rep.Unreadable[0].Source != "routing" || rep.Unreadable[0].Reason != "no default branch" {
		t.Fatalf("unreadable = %+v", rep.Unreadable)
	}
	found := false
	for _, f := range rep.Findings {
		if f.Rule == "proposal-section" {
			found = true
		}
	}
	if !found || len(rep.Changes) != 1 {
		t.Fatalf("other rules did not run: %+v", rep)
	}
}
