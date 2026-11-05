package lint

import "testing"

func routingFindings(t *testing.T, added []string, names ...string) []Finding {
	t.Helper()
	in, _ := repo(t, map[string]string{"openspec/changes/add-x/.openspec.yaml": "schema: spec-driven\ncreated: 2026-01-01\n"})
	in.Added, in.Slugs = added, names
	rep, err := Run(in)
	if err != nil {
		t.Fatal(err)
	}
	var out []Finding
	for _, f := range rep.Findings {
		if f.Rule == "routing" {
			out = append(out, f)
		}
	}
	return out
}

func TestRoutingFindsBothFolders(t *testing.T) {
	got := routingFindings(t, []string{"docs/superpowers/specs/a.md", "docs/superpowers/plans/b.md"})
	want := []Finding{
		{Rule: "routing", File: "docs/superpowers/plans/b.md",
			Message: "written to docs/superpowers/plans; write it into the change folder instead (proposal.md, specs/, design.md or plan.md)"},
		{Rule: "routing", File: "docs/superpowers/specs/a.md",
			Message: "written to docs/superpowers/specs; write it into the change folder instead (proposal.md, specs/, design.md or plan.md)"},
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("%+v", got)
	}
}

func TestRoutingNilAndEmptyReportNothing(t *testing.T) {
	if got := routingFindings(t, nil); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
	if got := routingFindings(t, []string{}); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestRoutingWithAndWithoutNames(t *testing.T) {
	for _, names := range [][]string{nil, {"add-x"}} {
		got := routingFindings(t, []string{"docs/superpowers/plans/b.md"}, names...)
		if len(got) != 1 || got[0].Change != "" || got[0].Line != 0 {
			t.Fatalf("%v: %+v", names, got)
		}
	}
}
