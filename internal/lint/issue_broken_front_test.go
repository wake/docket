package lint

import "testing"

func TestBrokenFrontKeepsSectionChecks(t *testing.T) {
	rep := lintIss(t, "---\ntitle: \"T\nsource: s\n---\n\nloose text\n")
	if count(rep, "issue-front-matter") != 1 || count(rep, "issue-section") != 5 || len(rep.Findings) != 6 {
		t.Errorf("local: %+v", rep.Findings)
	}
	only(t, lintIss(t, "---\nexternal: \"https://x/1\"\ntitle: \"T\n---\n"), "issue-front-matter", "")
}

func TestCardDuplicateBrokenFront(t *testing.T) {
	in, _ := repo(t, map[string]string{issPath: card, otherPath: "---\nexternal: \"https://x/1\"\ntitle: \"T\n---\n"})
	in.IssueTemplate = issTemplate(t)
	in.Issues = []string{issPath}
	rep := run(t, in)
	if len(rep.Unreadable) != 1 || rep.Unreadable[0].Source != otherPath {
		t.Fatalf("%+v", rep)
	}
}
