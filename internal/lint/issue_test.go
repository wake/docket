package lint

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	docket "github.com/wake/docket"
)

const (
	issSlug     = "2026-11-06-lint-timeout"
	issPath     = "docket/issues/" + issSlug + ".md"
	issFront    = "---\ntitle: T\nsource: s\n---\n"
	issSections = "## Current\nc\n\n## Expected\ne\n\n## Unchanged\nu\n\n## Evidence\nv\n"
)

func issTemplate(t *testing.T) []byte {
	t.Helper()
	b, err := docket.Shipped.ReadFile("shipped/issue/template.md")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// lintIss lints one issue the branch adds and checks that every finding is about it alone.
func lintIss(t *testing.T, content string) Report {
	t.Helper()
	in, _ := repo(t, map[string]string{issPath: content})
	in.IssueTemplate = issTemplate(t)
	in.Issues = []string{issPath}
	rep := run(t, in)
	for _, f := range rep.Findings {
		if f.File != issPath || f.Change != "" {
			t.Errorf("finding is not about the issue alone: %+v", f)
		}
	}
	return rep
}

func only(t *testing.T, rep Report, rule string, msg string) {
	t.Helper()
	if !has(rep, rule, issPath, -1, msg) || len(rep.Findings) != 1 {
		t.Errorf("want only %s: %+v", rule, rep.Findings)
	}
}

func TestIssueClean(t *testing.T) {
	rep := lintIss(t, issFront+"\n"+issSections)
	if len(rep.Findings) != 0 || !reflect.DeepEqual(rep.Issues, []string{issSlug}) || len(rep.Records) != 0 {
		t.Fatalf("%+v", rep)
	}
}

func TestIssueTheBranchAdds(t *testing.T) {
	only(t, lintIss(t, "---\ntitle: T\n---\n"+issSections), "issue-front-matter", "source")
}

func TestIssueMergedLeftAlone(t *testing.T) {
	in, _ := repo(t, map[string]string{issPath: issSections})
	in.IssueTemplate = issTemplate(t)
	in.Issues = []string{}
	if rep := run(t, in); len(rep.Findings) != 0 || len(rep.Issues) != 0 {
		t.Fatalf("%+v", rep)
	}
}

func TestIssueWithoutFrontMatter(t *testing.T) {
	only(t, lintIss(t, issSections), "issue-front-matter", "the issue has no front matter")
}

func TestIssueTitleTooLong(t *testing.T) {
	only(t, lintIss(t, "---\ntitle: "+strings.Repeat("a", 121)+"\nsource: s\n---\n"+issSections), "title-too-long", "121")
}

func TestIssueSectionEmpty(t *testing.T) {
	only(t, lintIss(t, issFront+"## Current\nc\n\n## Expected\ne\n\n## Unchanged\n<!-- c -->\n\n## Evidence\nv\n"), "issue-section", "Unchanged")
}

func TestIssueRecordSection(t *testing.T) {
	rep := lintIss(t, issFront+issSections+"\n## What\nw\n")
	only(t, rep, "issue-section", "What")
	if m := rep.Findings[0].Message; !strings.Contains(m, "not part of an issue") {
		t.Error(m)
	}
}

func TestIssueTooLong(t *testing.T) {
	rep := lintIss(t, issFront+"## Current\n"+strings.Repeat("a", 1441-3)+"\n\n## Expected\ne\n\n## Unchanged\nu\n\n## Evidence\nv\n")
	if !has(rep, "issue-too-long", issPath, -1, "1441") || !has(rep, "issue-too-long", issPath, -1, "1440") {
		t.Fatalf("%+v", rep.Findings)
	}
}

func TestIssueTasksSection(t *testing.T) {
	only(t, lintIss(t, issFront+issSections+"\n## Tasks\n### Task x\n"), "issue-section", "Tasks")
}

func TestIssuePlaceholder(t *testing.T) {
	comments := regexp.MustCompile(`(?s)<!--.*?-->`).FindAllString(string(issTemplate(t)), -1)
	rep := lintIss(t, issFront+"## Current\nc\n"+comments[1]+"\n\n## Expected\ne\n\n## Unchanged\nu\n\n## Evidence\nv\n")
	only(t, rep, "placeholder", "")
}

func TestNamedIssue(t *testing.T) {
	in, _ := repo(t, map[string]string{issPath: issSections})
	in.Slugs = []string{issSlug}
	rep := run(t, in)
	if !has(rep, "issue-front-matter", issPath, -1, "") || count(rep, "record-missing") != 0 ||
		!reflect.DeepEqual(rep.Issues, []string{issSlug}) || len(rep.Changes) != 0 {
		t.Fatalf("%+v", rep)
	}
}

func TestSameSlugRecordAndIssue(t *testing.T) {
	recFile := "docket/records/" + issSlug + ".md"
	in, _ := repo(t, map[string]string{issPath: issSections,
		recFile: "---\ntitle: T\nkind: chore\nsource: s\n---\n" + recSections})
	in.Slugs = []string{issSlug}
	rep := run(t, in)
	if !has(rep, "issue-front-matter", issPath, -1, "") || !has(rep, "record-kind", recFile, -1, "") ||
		count(rep, "record-missing") != 0 || len(rep.Records) != 1 || len(rep.Issues) != 1 {
		t.Fatalf("%+v", rep)
	}
}

func TestNamedNeitherMessage(t *testing.T) {
	in, _ := repo(t, map[string]string{"README.md": "x\n"})
	in.Slugs = []string{issSlug}
	rep := run(t, in)
	want := `no record or issue named "` + issSlug + `": neither docket/records/` + issSlug + `.md nor docket/issues/` + issSlug + `.md is a regular file`
	if len(rep.Findings) != 1 || rep.Findings[0].Message != want {
		t.Fatalf("%+v", rep.Findings)
	}
}
