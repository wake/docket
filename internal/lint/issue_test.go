package lint

import (
	"os"
	"path/filepath"
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
	only(t, rep, "placeholder", "shipped/issue/template.md")
}

// A folder that cannot be read may hold the named file, so nothing is reported missing.
func TestNamedMissingWithUnreadableIssues(t *testing.T) {
	dir := t.TempDir()
	locked := filepath.Join(dir, "docket", "issues")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	in, _ := open(t, dir)
	in.Slugs = []string{issSlug}
	rep := run(t, in)
	if count(rep, "record-missing") != 0 || len(rep.Unreadable) == 0 {
		t.Fatalf("%+v", rep)
	}
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

const card = "---\ntitle: T\nsource: s\nexternal: \"https://x/1\"\nexternal-id: \"#1\"\n---\n"

func withField(base, field string) string {
	return strings.Replace(base, "\n---\n", "\n"+field+"\n---\n", 1)
}

func TestCardClean(t *testing.T) {
	for _, c := range []string{card, card + "\n<!-- note -->\n", withField(card, "unsynced: []")} {
		if rep := lintIss(t, c); len(rep.Findings) != 0 {
			t.Errorf("%q: %+v", c, rep.Findings)
		}
	}
}

func TestCardFrontMatter(t *testing.T) {
	only(t, lintIss(t, strings.Replace(card, "external-id: \"#1\"\n", "", 1)), "issue-front-matter", "external-id")
	only(t, lintIss(t, strings.Replace(card, "\"#1\"", "#1", 1)), "issue-front-matter", "quote")
	only(t, lintIss(t, strings.Replace(card, "\"https://x/1\"", "\"\"", 1)), "issue-front-matter", "external")
	only(t, lintIss(t, withField(card, "dropped: yes")), "issue-front-matter", "dropped")
}

func TestExternalIDWithoutAddress(t *testing.T) {
	rep := lintIss(t, withField(issFront, "external-id: \"#1\""))
	if !has(rep, "issue-front-matter", issPath, -1, "external") || count(rep, "issue-section") != 4 || len(rep.Findings) != 5 {
		t.Errorf("%+v", rep.Findings)
	}
}

func TestCardContent(t *testing.T) {
	rep := lintIss(t, card+"\n## Current\nc\n")
	only(t, rep, "issue-section", "Current")
	if !strings.Contains(rep.Findings[0].Message, "other tool") {
		t.Error(rep.Findings[0].Message)
	}
	only(t, lintIss(t, card+"\nloose text\n"), "issue-section", "outside")
}

func TestIssueDropped(t *testing.T) {
	dropped := withField(card, "dropped: true")
	if rep := lintIss(t, dropped+"\n## Dropped\nClosed in the tracker as a duplicate.\n"); len(rep.Findings) != 0 {
		t.Errorf("%+v", rep.Findings)
	}
	only(t, lintIss(t, withField(issFront, "dropped: true")+issSections), "issue-section", "Dropped")
	only(t, lintIss(t, dropped+"\n## Dropped\n<!-- why -->\n"), "issue-section", "Dropped")
	only(t, lintIss(t, card+"\n## Dropped\nwhy\n"), "issue-section", "Dropped")
}

func TestIssueUnsynced(t *testing.T) {
	if rep := lintIss(t, withField(card, "unsynced: [taken, pr]")); len(rep.Findings) != 0 {
		t.Errorf("%+v", rep.Findings)
	}
	if rep := lintIss(t, withField(issFront, "unsynced: [open]")+issSections); len(rep.Findings) != 0 {
		t.Errorf("%+v", rep.Findings)
	}
	only(t, lintIss(t, withField(card, "unsynced: [open]")), "issue-unsynced", "open")
	only(t, lintIss(t, withField(issFront, "unsynced: [taken]")+issSections), "issue-unsynced", "taken")
	only(t, lintIss(t, withField(card, "unsynced: [later]")), "issue-unsynced", "later")
	only(t, lintIss(t, withField(card, "unsynced:\n  - taken")), "issue-front-matter", "")
}

const otherPath = "docket/issues/2026-11-01-other.md"

func TestCardDuplicate(t *testing.T) {
	for ext, dup := range map[string]bool{"https://x/1": true, "https://x/1/": false} {
		in, _ := repo(t, map[string]string{issPath: card, otherPath: strings.Replace(card, "https://x/1", ext, 1)})
		in.IssueTemplate = issTemplate(t)
		in.Issues = []string{issPath}
		rep := run(t, in)
		if dup && (!has(rep, "issue-external", issPath, -1, "2026-11-01-other") || len(rep.Findings) != 1) || !dup && len(rep.Findings) != 0 {
			t.Errorf("%s: %+v", ext, rep.Findings)
		}
	}
}

func TestCardDuplicateUnreadable(t *testing.T) {
	skipRoot(t)
	dir := t.TempDir()
	for p, c := range map[string]string{issPath: card, otherPath: card} {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(filepath.Join(dir, filepath.FromSlash(otherPath)), 0); err != nil {
		t.Fatal(err)
	}
	in, _ := open(t, dir)
	in.IssueTemplate = issTemplate(t)
	in.Issues = []string{issPath}
	rep := run(t, in)
	if count(rep, "issue-external") != 0 || len(rep.Unreadable) != 1 || rep.Unreadable[0].Source != otherPath {
		t.Fatalf("%+v", rep)
	}
}

func TestLintIssueFenceInComment(t *testing.T) {
	rep := lintIss(t, issFront+"## Current\nc\n<!--\n~~~\n-->\n\n## Expected\ne\n\n## Unchanged\nu\n\n## Evidence\nv\n")
	if !has(rep, "fence-in-comment", issPath, 8, "") || count(rep, "fence-in-comment") != 1 {
		t.Errorf("%+v", rep.Findings)
	}
}
