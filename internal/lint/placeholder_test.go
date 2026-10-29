package lint

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/wake/docket"
)

func templateWhyComment(t *testing.T) string {
	t.Helper()
	return templateLine(t, "<!-- Explain the motivation")
}

func templateLine(t *testing.T, prefix string) string {
	t.Helper()
	b, err := fs.ReadFile(docket.Schema(), "templates/proposal.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, prefix) {
			return l
		}
	}
	t.Fatalf("template line %q not found", prefix)
	return ""
}

func TestTemplateCommentLeft(t *testing.T) {
	comment := templateWhyComment(t)
	doc := strings.Replace(validProposal, "Because it helps.", comment, 1)
	f := wantOne(t, lintProposal(t, doc), "placeholder", lineOf(doc, comment))
	if !strings.Contains(f.Message, "templates/proposal.md") || !strings.Contains(f.Message, "Explain the motivation") {
		t.Fatalf("message = %q", f.Message)
	}
}

func TestTemplateCommentInFenceOrCodeSpan(t *testing.T) {
	comment := templateWhyComment(t)
	for name, text := range map[string]string{
		"fence": "```\n" + comment + "\n```",
		"span":  "see `" + comment + "` here",
	} {
		doc := strings.Replace(validProposal, "Because it helps.", text, 1)
		if r := lintProposal(t, doc); len(r.Findings) != 0 {
			t.Errorf("%s: findings = %+v", name, r.Findings)
		}
	}
}

func TestMultiLineCommentLeft(t *testing.T) {
	b, _ := fs.ReadFile(docket.Schema(), "templates/proposal.md")
	text := string(b)
	i := strings.Index(text, "<!-- Capabilities being introduced.")
	j := strings.Index(text[i:], "-->") + 3
	comment := text[i : i+j]
	doc := strings.Replace(validProposal, "Some code.", comment, 1)
	wantOne(t, lintProposal(t, doc), "placeholder", lineOf(doc, comment))
}

func TestAnglePlaceholderLeft(t *testing.T) {
	doc := strings.Replace(validProposal, "A thing changes.", "- `x`: <brief description of what this capability covers>", 1)
	wantOne(t, lintProposal(t, doc), "placeholder", lineOf(doc, "<brief"))
}

func TestPlaceholderInPlan(t *testing.T) {
	plan := reusePlan("| Walk | none | new | nothing fits | rg Walk |\n")
	plan = strings.Replace(plan, "Task 1: One", "Task 1: <name>", 1)
	repo := repoWith(map[string]string{
		"openspec/changes/add-x/proposal.md": validProposal,
		"openspec/changes/add-x/plan.md":     plan,
		"openspec/changes/add-x/docket.yaml": "title: X\n",
	})
	f := wantOne(t, runOn(repo), "placeholder", lineOf(plan, "<name>"))
	if f.File != "openspec/changes/add-x/plan.md" || !strings.Contains(f.Message, "<name>") {
		t.Fatalf("finding = %+v", f)
	}
}

func TestPlaceholderInOtherFile(t *testing.T) {
	comment := templateWhyComment(t)
	repo := repoWith(map[string]string{
		"openspec/changes/add-x/proposal.md":    validProposal,
		"openspec/changes/add-x/docket.yaml":    "title: X\n",
		"openspec/changes/add-x/.openspec.yaml": "x: 1\n# " + comment + "\n",
		"openspec/changes/add-x/notes/a.txt":    "\n\n" + comment,
	})
	r := runOn(repo)
	if len(r.Findings) != 2 || r.Findings[0].File != "openspec/changes/add-x/.openspec.yaml" || r.Findings[0].Line != 2 ||
		r.Findings[1].File != "openspec/changes/add-x/notes/a.txt" || r.Findings[1].Line != 3 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestOrdinaryAngleTextIsNotAPlaceholder(t *testing.T) {
	doc := strings.Replace(validProposal, "A thing changes.", "Use <b>this</b> and a < b.", 1)
	if r := lintProposal(t, doc); len(r.Findings) != 0 {
		t.Fatalf("findings = %+v", r.Findings)
	}
}
