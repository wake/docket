package lint

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wake/docket"
)

const issueSlug = "2026-11-02-b"
const issuePath = "docket/issues/" + issueSlug + ".md"

func issueBody() string {
	return section("Current", "It fails.") + section("Expected", "It works.") +
		section("Unchanged", "The rest.") + section("Found", "A review.")
}

func goodIssue() string { return "---\ntitle: B\nsource: a note\n---\n\n" + issueBody() }

func runIssue(content string, extra map[string]string) Report {
	files := map[string]string{issuePath: content}
	for k, v := range extra {
		files[k] = v
	}
	return Run(Input{
		Repo:          repoWith(files),
		Templates:     docket.Schema(),
		AddedIssues:   []string{issuePath},
		IssueTemplate: docket.IssueTemplate(),
	})
}

// wantIssue checks that exactly one finding has the rule and returns it.
func wantIssue(t *testing.T, r Report, rule string) Finding {
	t.Helper()
	fs := recFindings(r, rule)
	if len(fs) != 1 || fs[0].File != issuePath || fs[0].Change != "" {
		t.Fatalf("%s findings = %+v (all: %+v)", rule, fs, r.Findings)
	}
	return fs[0]
}

func TestGoodIssue(t *testing.T) {
	r := runIssue(goodIssue(), nil)
	if len(r.Findings) != 0 || len(r.Unreadable) != 0 || len(r.Issues) != 1 || r.Issues[0] != issueSlug {
		t.Fatalf("%+v", r)
	}
	r = runIssue("---\ntitle: B\nsource: s\ndropped: yes\nidea: docket/ideas/x.md\n---\n\n"+issueBody(),
		map[string]string{"docket/ideas/x.md": "idea\n"})
	if len(r.Findings) != 0 {
		t.Fatalf("%+v", r.Findings)
	}
}

func TestIssueFrontMatterInvalid(t *testing.T) {
	for name, front := range map[string]string{
		"idea names no file":  "title: B\nsource: s\nidea: docket/ideas/none.md\n",
		"idea names a folder": "title: B\nsource: s\nidea: docket/ideas/dir\n",
		"idea outside ideas":  "title: B\nsource: s\nidea: README.md\n",
		"dropped false":       "title: B\nsource: s\ndropped: false\n",
		"dropped not a bool":  "title: B\nsource: s\ndropped: maybe\n",
		"no source":           "title: B\n",
	} {
		t.Run(name, func(t *testing.T) {
			r := runIssue("---\n"+front+"---\n\n"+issueBody(), map[string]string{"README.md": "x", "docket/ideas/dir/a.md": "x"})
			wantIssue(t, r, "issue-front-matter")
		})
	}
}

func TestIssueSectionEmpty(t *testing.T) {
	body := strings.Replace(issueBody(), "## Unchanged\n\nThe rest.\n\n", "## Unchanged\n\n", 1)
	f := wantIssue(t, runIssue("---\ntitle: B\nsource: s\n---\n\n"+body, nil), "issue-section")
	if !strings.Contains(f.Message, "Unchanged") {
		t.Fatal(f.Message)
	}
}

func TestIssueTemplateTextLeft(t *testing.T) {
	tpl := docket.IssueTemplate()
	i := strings.Index(tpl, "## Found\n\n") + len("## Found\n\n")
	comment := tpl[i : i+strings.Index(tpl[i:], "-->")+3]
	body := strings.Replace(issueBody(), "A review.", comment, 1)
	wantIssue(t, runIssue("---\ntitle: B\nsource: s\n---\n\n"+body, nil), "placeholder")
}

func TestIssueTooLong(t *testing.T) {
	body := strings.Replace(issueBody(), "It fails.", strings.TrimSpace(strings.Repeat("word ", 61)), 1)
	f := wantIssue(t, runIssue("---\ntitle: B\nsource: s\n---\n\n"+body, nil), "issue-too-long")
	if !strings.Contains(f.Message, "Current") || !strings.Contains(f.Message, "61") {
		t.Fatal(f.Message)
	}
}

func TestRecordSourceNamesAnIssue(t *testing.T) {
	rec := func(src string) Report {
		in := recInput(map[string]string{
			recPath(goodSlug): strings.Replace(goodRecord(), "source: a note", "source: "+src, 1),
			issuePath:         goodIssue(),
		}, recPath(goodSlug))
		return Run(in)
	}
	f := wantRec(t, rec("docket/issues/2026-11-02-none.md"), "issue-source")
	if !strings.Contains(f.Message, "docket/issues/2026-11-02-none.md") {
		t.Fatal(f.Message)
	}
	for _, src := range []string{"'  " + issuePath + "  '", issuePath, "see " + "docket/issues/2026-11-02-none.md"} {
		if r := rec(src); len(recFindings(r, "issue-source")) != 0 {
			t.Fatalf("%q: %+v", src, r.Findings)
		}
	}
}

func TestIssueIdeaLinkOrDotDot(t *testing.T) {
	repo := repoWith(map[string]string{issuePath: "", "README.md": "x", "docket/ideas/real.md": "idea\n"})
	repo["docket/ideas/link.md"] = &fstest.MapFile{Data: []byte("real.md"), Mode: fs.ModeSymlink}
	for _, idea := range []string{"docket/ideas/link.md", "docket/ideas/../README.md"} {
		repo[issuePath] = &fstest.MapFile{Data: []byte("---\ntitle: B\nsource: s\nidea: " + idea + "\n---\n\n" + issueBody())}
		r := Run(Input{Repo: repo, Templates: docket.Schema(), AddedIssues: []string{issuePath}, IssueTemplate: docket.IssueTemplate()})
		wantIssue(t, r, "issue-front-matter")
	}
}

func TestNamedRunLintsNoIssue(t *testing.T) {
	bad := "---\ntitle: B\nsource: s\n---\n\n" + strings.Replace(issueBody(), "The rest.", "", 1)
	repo := repoWith(map[string]string{issuePath: bad, "openspec/changes/add-x/proposal.md": "x"})
	for _, name := range []string{"add-x", "2026-10-30-fix-b"} {
		r := Run(Input{Repo: repo, Templates: docket.Schema(), Slugs: []string{name}, AddedIssues: []string{issuePath}})
		if len(r.Issues) != 0 || len(recFindings(r, "issue-section")) != 0 {
			t.Fatalf("%s: %+v", name, r)
		}
	}
}

func TestRecordSourceSpacedNamesNoIssue(t *testing.T) {
	in := recInput(map[string]string{
		recPath(goodSlug): strings.Replace(goodRecord(), "source: a note", `source: "  docket/issues/2026-11-02-none.md "`, 1),
	}, recPath(goodSlug))
	f := wantRec(t, Run(in), "issue-source")
	if !strings.Contains(f.Message, `"docket/issues/2026-11-02-none.md"`) {
		t.Fatal(f.Message)
	}
}

func TestChangeSourceNamesNoIssue(t *testing.T) {
	repo := repoWith(map[string]string{
		"openspec/changes/add-x/proposal.md": "x",
		"openspec/changes/add-x/docket.yaml": "title: add-x\nsource: docket/issues/2026-11-02-none.md\n",
	})
	r := Run(Input{Repo: repo, Templates: docket.Schema()})
	if len(recFindings(r, "issue-source")) != 1 {
		t.Fatalf("%+v", r.Findings)
	}
}

func TestSourceWithANoteNamesNoIssue(t *testing.T) {
	src := "docket/issues/2026-11-02-x.md, from review of #40"
	run := func(files map[string]string) Report {
		files[recPath(goodSlug)] = strings.Replace(goodRecord(), "source: a note", "source: \""+src+"\"", 1)
		return Run(recInput(files, recPath(goodSlug)))
	}
	f := wantRec(t, run(map[string]string{}), "issue-source")
	if !strings.Contains(f.Message, src) {
		t.Fatal(f.Message)
	}
	if r := run(map[string]string{"docket/issues/2026-11-02-x.md": goodIssue()}); len(recFindings(r, "issue-source")) != 0 {
		t.Fatalf("%+v", r)
	}
}
